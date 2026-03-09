package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type smokeConfig struct {
	BaseURL         string
	RemoteBaseURL   string
	AdminToken      string
	LocalDomain     string
	RemoteDomain    string
	LocalPostgres   string
	RemotePostgres  string
	RequestTimeout  time.Duration
}

type authTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID string `json:"id"`
	} `json:"user"`
}

type inviteCreateResponse struct {
	Invite struct {
		Code string `json:"code"`
	} `json:"invite"`
}

type challengeResponse struct {
	Challenge string `json:"challenge"`
}

type deviceListResponse struct {
	Devices []struct {
		ID      string `json:"id"`
		Trusted bool   `json:"trusted"`
	} `json:"devices"`
}

type createChannelResponse struct {
	ID string `json:"id"`
}

type syncResponse struct {
	Events []json.RawMessage `json:"events"`
}

type serverKeyResponse struct {
	Domain    string `json:"domain"`
	PublicKey string `json:"public_key"`
}

func main() {
	cfg := smokeConfig{
		BaseURL:        envOr("NAIER_SMOKE_BASE_URL", "http://127.0.0.1:8080"),
		RemoteBaseURL:  envOr("NAIER_SMOKE_REMOTE_BASE_URL", "http://127.0.0.1:8081"),
		AdminToken:     envOr("NAIER_SMOKE_ADMIN_TOKEN", envOr("MESH_ADMIN_API_TOKEN", "development-admin-token")),
		LocalDomain:    envOr("NAIER_SMOKE_LOCAL_DOMAIN", "local.naier"),
		RemoteDomain:   envOr("NAIER_SMOKE_REMOTE_DOMAIN", "remote3.test"),
		LocalPostgres:  envOr("NAIER_SMOKE_LOCAL_POSTGRES_DSN", "postgres://mesh:mesh@127.0.0.1:5432/naier?sslmode=disable"),
		RemotePostgres: envOr("NAIER_SMOKE_REMOTE_POSTGRES_DSN", "postgres://mesh:mesh@127.0.0.1:5433/naier_remote?sslmode=disable"),
		RequestTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: cfg.RequestTimeout}
	mustStatusOK(ctx, client, cfg.BaseURL+"/health", "local health")
	mustStatusOK(ctx, client, cfg.RemoteBaseURL+"/health", "remote health")

	localDB, err := pgxpool.New(ctx, cfg.LocalPostgres)
	must(err)
	defer localDB.Close()

	remoteDB, err := pgxpool.New(ctx, cfg.RemotePostgres)
	must(err)
	defer remoteDB.Close()

	remoteKey := fetchRemoteServerKey(ctx, client, cfg.RemoteBaseURL)
	upsertFederatedServer(ctx, localDB, cfg.RemoteDomain, remoteKey.PublicKey, cfg.RemoteBaseURL)

	inviteCode := createInvite(ctx, client, cfg.BaseURL, cfg.AdminToken)
	localKeys := mustGenerateKeyBundle()
	username := fmt.Sprintf("smoke%s", time.Now().UTC().Format("150405"))
	challenge := requestChallenge(ctx, client, cfg.BaseURL, username, localKeys.deviceSigningPublic)
	deviceSignature := signBase64(localKeys.deviceSigningPrivate, challenge)
	deviceProofSignature := signBase64(localKeys.identitySigningPrivate, localKeys.deviceSigningPublic+":"+localKeys.deviceExchangePublic)

	auth := registerUser(ctx, client, cfg.BaseURL, inviteCode, username, localKeys, deviceSignature, deviceProofSignature)
	accessToken := auth.AccessToken

	getAuthenticated(ctx, client, cfg.BaseURL+"/api/v1/auth/me", accessToken)

	secondKeys := mustGenerateKeyBundle()
	pendingDeviceID := createPendingDevice(ctx, client, cfg.BaseURL, accessToken, secondKeys)
	approveDevice(ctx, client, cfg.BaseURL, accessToken, pendingDeviceID)
	devices := listDevices(ctx, client, cfg.BaseURL, accessToken)
	if len(devices.Devices) < 2 {
		fail("expected at least 2 devices after approval")
	}

	channelID := createChannel(ctx, client, cfg.BaseURL, accessToken)
	sendMessage(ctx, client, cfg.BaseURL, accessToken, channelID)
	verifySync(ctx, client, cfg.BaseURL, accessToken)

	pushChannelState(ctx, client, cfg.BaseURL, accessToken, channelID, cfg.RemoteDomain)
	verifyRemoteShadow(ctx, remoteDB, cfg.LocalDomain, channelID)

	createRemoteMirror(ctx, remoteDB, channelID)
	pullChannelState(ctx, client, cfg.BaseURL, accessToken, channelID, cfg.RemoteDomain)
	verifyLocalRemoteShadow(ctx, localDB, cfg.RemoteDomain, channelID)

	fmt.Println("integration smoke passed")
}

type keyBundle struct {
	identitySigningPrivate ed25519.PrivateKey
	identitySigningPublic  string
	identityExchangePublic string
	deviceSigningPrivate   ed25519.PrivateKey
	deviceSigningPublic    string
	deviceExchangePublic   string
}

func mustGenerateKeyBundle() keyBundle {
	identityPublic, identityPrivate, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	identityExchangePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	must(err)
	devicePublic, devicePrivate, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	deviceExchangePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	must(err)

	return keyBundle{
		identitySigningPrivate: identityPrivate,
		identitySigningPublic:  rawBase64(identityPublic),
		identityExchangePublic: rawBase64(identityExchangePrivate.PublicKey().Bytes()),
		deviceSigningPrivate:   devicePrivate,
		deviceSigningPublic:    rawBase64(devicePublic),
		deviceExchangePublic:   rawBase64(deviceExchangePrivate.PublicKey().Bytes()),
	}
}

func createInvite(ctx context.Context, client *http.Client, baseURL, adminToken string) string {
	payload := map[string]any{
		"note":     "integration smoke",
		"max_uses": 1,
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/admin/invites", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", adminToken)
	req.Header.Set("X-Admin-Actor", "integration-smoke")

	var response inviteCreateResponse
	doJSON(client, req, http.StatusCreated, &response)
	if strings.TrimSpace(response.Invite.Code) == "" {
		fail("expected invite code in admin response")
	}

	return response.Invite.Code
}

func requestChallenge(ctx context.Context, client *http.Client, baseURL, username, deviceSigningKey string) string {
	payload := map[string]any{
		"username":           username,
		"device_signing_key": deviceSigningKey,
		"device_name":        "Integration smoke device",
		"platform":           "web",
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/challenge", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Content-Type", "application/json")

	var response challengeResponse
	doJSON(client, req, http.StatusOK, &response)
	return response.Challenge
}

func registerUser(ctx context.Context, client *http.Client, baseURL, inviteCode, username string, keys keyBundle, deviceSignature, deviceProofSignature string) authTokens {
	payload := map[string]any{
		"username":                      username,
		"display_name":                  "Integration Smoke",
		"invite_code":                   inviteCode,
		"identity_signing_key":          keys.identitySigningPublic,
		"identity_exchange_key":         keys.identityExchangePublic,
		"device_signing_key":            keys.deviceSigningPublic,
		"device_exchange_key":           keys.deviceExchangePublic,
		"device_signature":              deviceSignature,
		"identity_signature_over_device": deviceProofSignature,
		"device_name":                   "Integration smoke device",
		"platform":                      "web",
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Content-Type", "application/json")

	var response authTokens
	doJSON(client, req, http.StatusCreated, &response)
	if response.AccessToken == "" || response.RefreshToken == "" {
		fail("expected register response tokens")
	}

	return response
}

func getAuthenticated(ctx context.Context, client *http.Client, url, accessToken string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	doStatus(client, req, http.StatusOK)
}

func createPendingDevice(ctx context.Context, client *http.Client, baseURL, accessToken string, keys keyBundle) string {
	payload := map[string]any{
		"device_signing_key":  keys.deviceSigningPublic,
		"device_exchange_key": keys.deviceExchangePublic,
		"device_name":         "Secondary integration device",
		"platform":            "web",
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/devices/pending", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	var response struct {
		Device struct {
			ID string `json:"id"`
		} `json:"device"`
	}
	doJSON(client, req, http.StatusCreated, &response)
	return response.Device.ID
}

func approveDevice(ctx context.Context, client *http.Client, baseURL, accessToken, deviceID string) {
	payload := map[string]any{"device_id": deviceID}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/devices/approve", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	doStatus(client, req, http.StatusNoContent)
}

func listDevices(ctx context.Context, client *http.Client, baseURL, accessToken string) deviceListResponse {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/auth/devices", nil)
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	var response deviceListResponse
	doJSON(client, req, http.StatusOK, &response)
	return response
}

func createChannel(ctx context.Context, client *http.Client, baseURL, accessToken string) string {
	payload := map[string]any{
		"type":         "group",
		"name":         "Integration Room",
		"description":  "smoke",
		"is_encrypted": true,
		"max_members":  8,
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/channels", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	var response createChannelResponse
	doJSON(client, req, http.StatusCreated, &response)
	return response.ID
}

func sendMessage(ctx context.Context, client *http.Client, baseURL, accessToken, channelID string) {
	payload := map[string]any{
		"type":            "text",
		"content":         "integration smoke message",
		"client_event_id": uuid.NewString(),
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/channels/"+channelID+"/messages", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	doStatus(client, req, http.StatusCreated)
}

func verifySync(ctx context.Context, client *http.Client, baseURL, accessToken string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/events/sync?limit=50", nil)
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	var response syncResponse
	doJSON(client, req, http.StatusOK, &response)
	if len(response.Events) == 0 {
		fail("expected sync events")
	}
}

func fetchRemoteServerKey(ctx context.Context, client *http.Client, baseURL string) serverKeyResponse {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/_federation/v1/server-key", nil)
	must(err)
	var response serverKeyResponse
	doJSON(client, req, http.StatusOK, &response)
	return response
}

func pushChannelState(ctx context.Context, client *http.Client, baseURL, accessToken, channelID, targetDomain string) {
	payload := map[string]any{"target_domain": targetDomain}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/federation/channels/"+channelID+"/sync", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	doStatus(client, req, http.StatusAccepted)
}

func pullChannelState(ctx context.Context, client *http.Client, baseURL, accessToken, channelID, targetDomain string) {
	payload := map[string]any{"target_domain": targetDomain}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/federation/channels/"+channelID+"/pull", bytes.NewReader(reqBody))
	must(err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	doStatus(client, req, http.StatusOK)
}

func upsertFederatedServer(ctx context.Context, db *pgxpool.Pool, domain, publicKey, endpoint string) {
	_, err := db.Exec(ctx, `
		INSERT INTO federated_servers (domain, public_key, endpoint, status, last_ping)
		VALUES ($1, $2, $3, 'active', NOW())
		ON CONFLICT (domain)
		DO UPDATE SET public_key = EXCLUDED.public_key,
		              endpoint = EXCLUDED.endpoint,
		              status = 'active',
		              last_ping = NOW()
	`, domain, publicKey, endpoint)
	must(err)
}

func verifyRemoteShadow(ctx context.Context, db *pgxpool.Pool, originServer, channelID string) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM remote_channels
			WHERE origin_server = $1 AND remote_channel_id = $2
		)
	`, originServer, channelID).Scan(&exists)
	must(err)
	if !exists {
		fail("expected remote shadow channel after sync")
	}
}

func createRemoteMirror(ctx context.Context, db *pgxpool.Pool, channelID string) {
	username := "remote-smoke"
	var userID uuid.UUID
	identityPublic, _, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	exchangePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	must(err)

	err = db.QueryRow(ctx, `
		INSERT INTO users (id, username, display_name, public_key, identity_signing_key, identity_exchange_key, server_id)
		VALUES ($1, $2, $3, $4, $5, $6, 'remote3.test')
		ON CONFLICT (username) DO UPDATE
		SET display_name = EXCLUDED.display_name
		RETURNING id
	`, uuid.New(), username, "Remote Smoke", rawBase64(identityPublic), rawBase64(identityPublic), rawBase64(exchangePrivate.PublicKey().Bytes())).Scan(&userID)
	must(err)

	_, err = db.Exec(ctx, `
		INSERT INTO channels (id, type, name, description, owner_id, is_encrypted, max_members)
		VALUES ($1, 'group', 'Remote Mirror', 'integration smoke', $2, TRUE, 8)
		ON CONFLICT (id) DO NOTHING
	`, channelID, userID)
	must(err)

	_, err = db.Exec(ctx, `
		INSERT INTO channel_members (channel_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (channel_id, user_id) DO NOTHING
	`, channelID, userID)
	must(err)
}

func verifyLocalRemoteShadow(ctx context.Context, db *pgxpool.Pool, originServer, channelID string) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM remote_channels
			WHERE origin_server = $1 AND remote_channel_id = $2
		)
	`, originServer, channelID).Scan(&exists)
	must(err)
	if !exists {
		fail("expected local remote shadow channel after pull")
	}
}

func signBase64(privateKey ed25519.PrivateKey, message string) string {
	return rawBase64(ed25519.Sign(privateKey, []byte(message)))
}

func rawBase64(value []byte) string {
	return base64.RawStdEncoding.EncodeToString(value)
}

func mustStatusOK(ctx context.Context, client *http.Client, url, label string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	must(err)
	doStatus(client, req, http.StatusOK)
}

func doStatus(client *http.Client, req *http.Request, want int) {
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		fail(fmt.Sprintf("%s %s returned %d: %s", req.Method, req.URL, resp.StatusCode, strings.TrimSpace(string(body))))
	}
}

func doJSON(client *http.Client, req *http.Request, want int, target any) {
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		fail(fmt.Sprintf("%s %s returned %d: %s", req.Method, req.URL, resp.StatusCode, strings.TrimSpace(string(body))))
	}
	if target == nil {
		return
	}
	must(json.NewDecoder(resp.Body).Decode(target))
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
