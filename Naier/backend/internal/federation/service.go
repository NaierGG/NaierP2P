package federation

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naier/backend/internal/auth"
	"github.com/naier/backend/internal/config"
	"github.com/naier/backend/internal/message"
)

const (
	EventMessageForward = "MESSAGE_FORWARD"
	EventUserSync       = "USER_SYNC"
	EventChannelStateSync = "CHANNEL_STATE_SYNC"
	maxClockSkew        = 10 * time.Minute
)

type Service struct {
	db         *pgxpool.Pool
	resolver   *Resolver
	httpClient *http.Client
	config     config.FederationConfig
}

func NewService(db *pgxpool.Pool, resolver *Resolver, cfg config.FederationConfig) *Service {
	return &Service{
		db:       db,
		resolver: resolver,
		config:   cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) SendEvent(ctx context.Context, targetDomain string, event FederatedEvent) error {
	resolved, err := s.resolver.ResolveServer(ctx, targetDomain)
	if err != nil {
		return err
	}

	if strings.TrimSpace(event.EventID) == "" {
		event.EventID = uuid.NewString()
	}
	if strings.TrimSpace(event.ServerID) == "" {
		event.ServerID = s.config.ServerDomain
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(event.Signature) == "" {
		signature, signErr := s.signEvent(event)
		if signErr != nil {
			return signErr
		}
		event.Signature = signature
	}

	body, err := json.Marshal(EventEnvelope{Event: event})
	if err != nil {
		return fmt.Errorf("marshal federated event: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		joinURL(resolved.Endpoint, "/_federation/v1/events"),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create federated request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send federated event to %s: %w", targetDomain, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("federated event rejected with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (s *Service) VerifyAndProcessEvent(ctx context.Context, event FederatedEvent) (EventProcessingResult, error) {
	if err := s.validateIncomingEvent(event); err != nil {
		return EventProcessingResult{}, err
	}

	publicKey, err := s.resolver.GetServerKey(ctx, event.ServerID)
	if err != nil {
		return EventProcessingResult{}, err
	}

	valid, err := verifyEventSignature(event, publicKey)
	if err != nil {
		return EventProcessingResult{}, err
	}
	if !valid {
		return EventProcessingResult{}, fmt.Errorf("invalid federated event signature")
	}

	payloadHash := hashFederatedPayload(event.Payload)
	duplicate, err := s.recordIncomingEvent(ctx, event, payloadHash)
	if err != nil {
		return EventProcessingResult{}, err
	}
	if duplicate {
		return EventProcessingResult{
			Processed: false,
			Duplicate: true,
		}, nil
	}

	if err := s.resolver.UpsertServerKey(ctx, event.ServerID, publicKey); err != nil {
		return EventProcessingResult{}, err
	}

	if _, err := s.db.Exec(ctx, `
		UPDATE federated_servers
		SET last_ping = NOW(), status = 'active'
		WHERE domain = $1
	`, strings.ToLower(event.ServerID)); err != nil {
		return EventProcessingResult{}, fmt.Errorf("update federated server heartbeat: %w", err)
	}

	if err := s.processEvent(ctx, event); err != nil {
		_ = s.discardIncomingEvent(ctx, event)
		return EventProcessingResult{}, err
	}
	if err := s.markEventProcessed(ctx, event); err != nil {
		return EventProcessingResult{}, err
	}

	return EventProcessingResult{
		Processed: true,
		Duplicate: false,
	}, nil
}

func (s *Service) ForwardMessageToServer(ctx context.Context, msg message.MessageDTO, targetServer string) error {
	payload, err := json.Marshal(MessageForwardPayload{Message: msg})
	if err != nil {
		return fmt.Errorf("marshal message forward payload: %w", err)
	}

	return s.SendEvent(ctx, targetServer, FederatedEvent{
		Type:    EventMessageForward,
		Payload: payload,
	})
}

func (s *Service) FetchRemoteUser(ctx context.Context, username, domain string) (*auth.UserDTO, error) {
	resolved, err := s.resolver.ResolveServer(ctx, domain)
	if err != nil {
		return nil, err
	}

	requestURL := joinURL(resolved.Endpoint, "/_federation/v1/users/"+url.PathEscape(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create remote user request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch remote user from %s: %w", domain, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, auth.ErrInvalidCredentials
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("remote user request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var response RemoteUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode remote user response: %w", err)
	}

	if err := s.upsertRemoteUser(ctx, response.User, domain); err != nil {
		return nil, err
	}

	return &response.User, nil
}

func (s *Service) GetLocalUser(ctx context.Context, username string) (*auth.UserDTO, error) {
	var user auth.UserDTO
	err := s.db.QueryRow(ctx, `
		SELECT id::text, username, display_name, public_key,
		       identity_signing_key, identity_exchange_key,
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at
		FROM users
		WHERE username = $1
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
		&user.IdentitySigningKey,
		&user.IdentityExchangeKey,
		&user.AvatarURL,
		&user.Bio,
		&user.ServerID,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("get local user: %w", err)
	}

	return &user, nil
}

func (s *Service) GetLocalChannelState(ctx context.Context, channelID string) (FederatedChannelStatePayload, error) {
	var state FederatedChannelStatePayload
	err := s.db.QueryRow(ctx, `
		SELECT id::text, type, COALESCE(name, ''), COALESCE(description, ''), is_encrypted, max_members
		FROM channels
		WHERE id::text = $1
	`, channelID).Scan(
		&state.ChannelID,
		&state.ChannelType,
		&state.Name,
		&state.Description,
		&state.IsEncrypted,
		&state.MaxMembers,
	)
	if err != nil {
		return FederatedChannelStatePayload{}, fmt.Errorf("get local channel state: %w", err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT u.id::text, u.username, u.display_name, u.public_key,
		       u.identity_signing_key, u.identity_exchange_key,
		       COALESCE(u.avatar_url, ''), COALESCE(u.bio, ''), u.server_id, u.created_at,
		       cm.role, cm.joined_at, cm.is_muted
		FROM channel_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.channel_id::text = $1
		ORDER BY CASE cm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, cm.joined_at
	`, channelID)
	if err != nil {
		return FederatedChannelStatePayload{}, fmt.Errorf("list local channel members: %w", err)
	}
	defer rows.Close()

	state.Members = make([]FederatedChannelMember, 0)
	for rows.Next() {
		var member FederatedChannelMember
		if err := rows.Scan(
			&member.User.ID,
			&member.User.Username,
			&member.User.DisplayName,
			&member.User.PublicKey,
			&member.User.IdentitySigningKey,
			&member.User.IdentityExchangeKey,
			&member.User.AvatarURL,
			&member.User.Bio,
			&member.User.ServerID,
			&member.User.CreatedAt,
			&member.Role,
			&member.JoinedAt,
			&member.IsMuted,
		); err != nil {
			return FederatedChannelStatePayload{}, fmt.Errorf("scan local channel member: %w", err)
		}
		state.Members = append(state.Members, member)
	}
	if err := rows.Err(); err != nil {
		return FederatedChannelStatePayload{}, err
	}
	state.MemberCount = len(state.Members)

	return state, nil
}

func (s *Service) SendChannelStateToServer(ctx context.Context, actorID, channelID uuid.UUID, targetDomain string) error {
	if err := s.ensureActorIsChannelMember(ctx, actorID, channelID); err != nil {
		return err
	}

	state, err := s.GetLocalChannelState(ctx, channelID.String())
	if err != nil {
		return err
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal channel state payload: %w", err)
	}

	return s.SendEvent(ctx, targetDomain, FederatedEvent{
		Type:    EventChannelStateSync,
		Payload: payload,
	})
}

func (s *Service) PullChannelStateFromServer(ctx context.Context, actorID, channelID uuid.UUID, targetDomain string) (FederatedChannelStatePayload, error) {
	if err := s.ensureActorIsChannelMember(ctx, actorID, channelID); err != nil {
		return FederatedChannelStatePayload{}, err
	}

	resolved, err := s.resolver.ResolveServer(ctx, targetDomain)
	if err != nil {
		return FederatedChannelStatePayload{}, err
	}

	requestURL := joinURL(resolved.Endpoint, "/_federation/v1/channels/"+url.PathEscape(channelID.String())+"/state")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return FederatedChannelStatePayload{}, fmt.Errorf("create remote channel state request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return FederatedChannelStatePayload{}, fmt.Errorf("pull remote channel state from %s: %w", targetDomain, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return FederatedChannelStatePayload{}, fmt.Errorf("remote channel state request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var response ChannelStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return FederatedChannelStatePayload{}, fmt.Errorf("decode remote channel state response: %w", err)
	}
	if err := s.upsertRemoteChannelState(ctx, response.Channel, targetDomain); err != nil {
		return FederatedChannelStatePayload{}, err
	}

	return response.Channel, nil
}

func (s *Service) GetServerKeyResponse() ServerKeyResponse {
	return ServerKeyResponse{
		Domain:    s.config.ServerDomain,
		PublicKey: s.config.ServerPublicKey,
	}
}

func (s *Service) GetWellKnownResponse() WellKnownResponse {
	return WellKnownResponse{
		Version:   "mc1",
		Domain:    s.config.ServerDomain,
		PublicKey: s.config.ServerPublicKey,
		Endpoint:  buildDefaultEndpoint(s.config.ServerDomain),
	}
}

func (s *Service) processEvent(ctx context.Context, event FederatedEvent) error {
	switch event.Type {
	case EventMessageForward:
		var payload MessageForwardPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode message forward payload: %w", err)
		}
		if payload.Message.ID == "" || payload.Message.ChannelID == "" || payload.Message.SenderID == "" {
			return fmt.Errorf("invalid message forward payload")
		}
		return nil
	case EventUserSync:
		var payload RemoteUserResponse
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode user sync payload: %w", err)
		}
		if payload.User.Username == "" || payload.User.IdentitySigningKey == "" || payload.User.IdentityExchangeKey == "" {
			return fmt.Errorf("invalid user sync payload")
		}
		return s.upsertRemoteUser(ctx, payload.User, event.ServerID)
	case EventChannelStateSync:
		var payload FederatedChannelStatePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("decode channel state payload: %w", err)
		}
		if payload.ChannelID == "" || payload.ChannelType == "" {
			return fmt.Errorf("invalid channel state payload")
		}
		return s.upsertRemoteChannelState(ctx, payload, event.ServerID)
	default:
		if !json.Valid(event.Payload) {
			return fmt.Errorf("invalid federated payload")
		}
		return nil
	}
}

func (s *Service) validateIncomingEvent(event FederatedEvent) error {
	if strings.TrimSpace(event.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(event.Type) == "" {
		return fmt.Errorf("type is required")
	}
	if strings.TrimSpace(event.ServerID) == "" {
		return fmt.Errorf("server_id is required")
	}
	if strings.TrimSpace(event.Signature) == "" {
		return fmt.Errorf("signature is required")
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	now := time.Now().UTC()
	if event.Timestamp.Before(now.Add(-maxClockSkew)) || event.Timestamp.After(now.Add(maxClockSkew)) {
		return fmt.Errorf("event timestamp outside allowed skew")
	}

	if !json.Valid(event.Payload) {
		return fmt.Errorf("payload must be valid json")
	}

	return nil
}

func (s *Service) recordIncomingEvent(ctx context.Context, event FederatedEvent, payloadHash string) (bool, error) {
	commandTag, err := s.db.Exec(ctx, `
		INSERT INTO federated_events (event_id, origin_server, event_type, payload_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (origin_server, event_id) DO NOTHING
	`, event.EventID, strings.ToLower(event.ServerID), event.Type, payloadHash)
	if err != nil {
		return false, fmt.Errorf("record federated event: %w", err)
	}

	if commandTag.RowsAffected() > 0 {
		return false, nil
	}

	var existingHash string
	err = s.db.QueryRow(ctx, `
		SELECT payload_hash
		FROM federated_events
		WHERE origin_server = $1 AND event_id = $2
	`, strings.ToLower(event.ServerID), event.EventID).Scan(&existingHash)
	if err != nil {
		return false, fmt.Errorf("load federated event dedupe record: %w", err)
	}
	if existingHash != payloadHash {
		return false, fmt.Errorf("federated event id replayed with different payload")
	}

	return true, nil
}

func (s *Service) markEventProcessed(ctx context.Context, event FederatedEvent) error {
	_, err := s.db.Exec(ctx, `
		UPDATE federated_events
		SET processed_at = NOW()
		WHERE origin_server = $1 AND event_id = $2
	`, strings.ToLower(event.ServerID), event.EventID)
	if err != nil {
		return fmt.Errorf("mark federated event processed: %w", err)
	}

	return nil
}

func (s *Service) discardIncomingEvent(ctx context.Context, event FederatedEvent) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM federated_events
		WHERE origin_server = $1 AND event_id = $2 AND processed_at IS NULL
	`, strings.ToLower(event.ServerID), event.EventID)
	if err != nil {
		return fmt.Errorf("discard federated event: %w", err)
	}

	return nil
}

func (s *Service) signEvent(event FederatedEvent) (string, error) {
	privateKey, err := decodePrivateKey(s.config.ServerPrivateKey)
	if err != nil {
		return "", err
	}

	payload, err := payloadToSign(event)
	if err != nil {
		return "", err
	}

	signature := ed25519.Sign(privateKey, payload)
	return base64.RawStdEncoding.EncodeToString(signature), nil
}

func verifyEventSignature(event FederatedEvent, publicKeyB64 string) (bool, error) {
	publicKey, err := decodePublicKey(publicKeyB64)
	if err != nil {
		return false, err
	}

	signature, err := decodeBase64(signatureString(event.Signature))
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}

	payload, err := payloadToSign(event)
	if err != nil {
		return false, err
	}

	return ed25519.Verify(publicKey, payload, signature), nil
}

func payloadToSign(event FederatedEvent) ([]byte, error) {
	type signableEvent struct {
		EventID   string          `json:"event_id"`
		Type      string          `json:"type"`
		ServerID  string          `json:"server_id"`
		Timestamp time.Time       `json:"timestamp"`
		Payload   json.RawMessage `json:"payload"`
	}

	return json.Marshal(signableEvent{
		EventID:   event.EventID,
		Type:      event.Type,
		ServerID:  event.ServerID,
		Timestamp: event.Timestamp.UTC(),
		Payload:   event.Payload,
	})
}

func hashFederatedPayload(payload json.RawMessage) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *Service) ensureActorIsChannelMember(ctx context.Context, actorID, channelID uuid.UUID) error {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM channel_members
			WHERE channel_id = $1 AND user_id = $2
		)
	`, channelID, actorID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check local channel membership: %w", err)
	}
	if !exists {
		return fmt.Errorf("actor is not a member of this channel")
	}

	return nil
}

func (s *Service) upsertRemoteUser(ctx context.Context, user auth.UserDTO, domain string) error {
	if strings.TrimSpace(domain) == "" {
		return fmt.Errorf("remote user domain is required")
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO remote_users (
			remote_user_id, domain, username, display_name, public_key,
			identity_signing_key, identity_exchange_key, avatar_url, bio, last_synced_at, updated_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, NULLIF($8, ''), NULLIF($9, ''), NOW(), NOW())
		ON CONFLICT (domain, username) DO UPDATE
		SET remote_user_id = EXCLUDED.remote_user_id,
		    display_name = EXCLUDED.display_name,
		    public_key = EXCLUDED.public_key,
		    identity_signing_key = EXCLUDED.identity_signing_key,
		    identity_exchange_key = EXCLUDED.identity_exchange_key,
		    avatar_url = EXCLUDED.avatar_url,
		    bio = EXCLUDED.bio,
		    last_synced_at = NOW(),
		    updated_at = NOW()
	`, user.ID, strings.ToLower(domain), user.Username, user.DisplayName, user.PublicKey, user.IdentitySigningKey, user.IdentityExchangeKey, user.AvatarURL, user.Bio)
	if err != nil {
		return fmt.Errorf("upsert remote user: %w", err)
	}

	return nil
}

func (s *Service) upsertRemoteChannelState(ctx context.Context, state FederatedChannelStatePayload, originServer string) error {
	originServer = strings.ToLower(strings.TrimSpace(originServer))
	if originServer == "" {
		return fmt.Errorf("origin server is required")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin remote channel state transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO remote_channels (
			origin_server, remote_channel_id, channel_type, name, description,
			is_encrypted, max_members, member_count, last_synced_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (origin_server, remote_channel_id) DO UPDATE
		SET channel_type = EXCLUDED.channel_type,
		    name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    is_encrypted = EXCLUDED.is_encrypted,
		    max_members = EXCLUDED.max_members,
		    member_count = EXCLUDED.member_count,
		    last_synced_at = NOW()
	`, originServer, state.ChannelID, state.ChannelType, state.Name, state.Description, state.IsEncrypted, state.MaxMembers, state.MemberCount)
	if err != nil {
		return fmt.Errorf("upsert remote channel: %w", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM remote_channel_memberships
		WHERE origin_server = $1 AND remote_channel_id = $2
	`, originServer, state.ChannelID)
	if err != nil {
		return fmt.Errorf("clear remote channel memberships: %w", err)
	}

	for _, member := range state.Members {
		if member.User.Username == "" || member.User.IdentitySigningKey == "" || member.User.IdentityExchangeKey == "" {
			return fmt.Errorf("remote channel member payload missing identity fields")
		}
		if err := s.upsertRemoteUser(ctx, member.User, originServer); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO remote_channel_memberships (
				origin_server, remote_channel_id, remote_user_id, username,
				display_name, role, joined_at, is_muted, last_synced_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`, originServer, state.ChannelID, member.User.ID, member.User.Username, member.User.DisplayName, member.Role, member.JoinedAt, member.IsMuted)
		if err != nil {
			return fmt.Errorf("insert remote channel membership: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit remote channel state transaction: %w", err)
	}

	return nil
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := decodeBase64(value)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(decoded), nil
}

func decodePrivateKey(value string) (ed25519.PrivateKey, error) {
	decoded, err := decodeBase64(value)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	switch len(decoded) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(decoded), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	default:
		return nil, fmt.Errorf("private key must be %d-byte key or %d-byte seed", ed25519.PrivateKeySize, ed25519.SeedSize)
	}
}

func decodeBase64(value string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(strings.TrimSpace(value))
		if err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("invalid base64 value")
}

func signatureString(value string) string {
	return strings.TrimSpace(value)
}

func joinURL(baseURL, suffix string) string {
	if baseURL == "" {
		return suffix
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + suffix
	}

	parsed.Path = path.Join(parsed.Path, suffix)
	return strings.TrimRight(parsed.String(), "/")
}
