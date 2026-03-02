package federation

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
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
	"github.com/meshchat/backend/internal/auth"
	"github.com/meshchat/backend/internal/config"
	"github.com/meshchat/backend/internal/message"
)

const (
	EventMessageForward = "MESSAGE_FORWARD"
	EventUserSync       = "USER_SYNC"
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

func (s *Service) VerifyAndProcessEvent(ctx context.Context, event FederatedEvent) error {
	if err := s.validateIncomingEvent(event); err != nil {
		return err
	}

	publicKey, err := s.resolver.GetServerKey(ctx, event.ServerID)
	if err != nil {
		return err
	}

	valid, err := verifyEventSignature(event, publicKey)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid federated event signature")
	}

	if err := s.resolver.UpsertServerKey(ctx, event.ServerID, publicKey); err != nil {
		return err
	}

	if _, err := s.db.Exec(ctx, `
		UPDATE federated_servers
		SET last_ping = NOW(), status = 'active'
		WHERE domain = $1
	`, strings.ToLower(event.ServerID)); err != nil {
		return fmt.Errorf("update federated server heartbeat: %w", err)
	}

	return s.processEvent(ctx, event)
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

	return &response.User, nil
}

func (s *Service) GetLocalUser(ctx context.Context, username string) (*auth.UserDTO, error) {
	var user auth.UserDTO
	err := s.db.QueryRow(ctx, `
		SELECT id::text, username, display_name, public_key,
		       COALESCE(avatar_url, ''), COALESCE(bio, ''), server_id, created_at
		FROM users
		WHERE username = $1
	`, username).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.PublicKey,
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
		if payload.User.Username == "" || payload.User.PublicKey == "" {
			return fmt.Errorf("invalid user sync payload")
		}
		return nil
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
