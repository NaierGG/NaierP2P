package federation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/naier/backend/internal/config"
)

func TestSignAndVerifyEventWithSeedPrivateKey(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	service := &Service{
		config: config.FederationConfig{
			ServerDomain:     "local.naier",
			ServerPrivateKey: base64.RawStdEncoding.EncodeToString(privateKey.Seed()),
		},
	}

	payload, err := json.Marshal(map[string]string{
		"channel_id": "33333333-3333-3333-3333-333333333333",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	event := FederatedEvent{
		EventID:   "evt-sign-verify",
		Type:      EventChannelStateSync,
		ServerID:  "local.naier",
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	signature, err := service.signEvent(event)
	if err != nil {
		t.Fatalf("sign event: %v", err)
	}
	event.Signature = signature

	valid, err := verifyEventSignature(event, base64.RawStdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatalf("verify event signature: %v", err)
	}
	if !valid {
		t.Fatal("expected signature verification to succeed")
	}
}

func TestDecodePrivateKeySupportsSeedAndFullPrivateKey(t *testing.T) {
	t.Parallel()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate keypair: %v", err)
	}

	seedEncoded := base64.RawStdEncoding.EncodeToString(privateKey.Seed())
	fullEncoded := base64.RawStdEncoding.EncodeToString(privateKey)

	seedDecoded, err := decodePrivateKey(seedEncoded)
	if err != nil {
		t.Fatalf("decode seed private key: %v", err)
	}
	fullDecoded, err := decodePrivateKey(fullEncoded)
	if err != nil {
		t.Fatalf("decode full private key: %v", err)
	}

	if string(seedDecoded) != string(privateKey) {
		t.Fatal("expected decoded seed key to expand to the original private key")
	}
	if string(fullDecoded) != string(privateKey) {
		t.Fatal("expected decoded full key to match the original private key")
	}
}

func TestJoinURL(t *testing.T) {
	t.Parallel()

	got := joinURL("http://backend-remote:8080", "/_federation/v1/events")
	if got != "http://backend-remote:8080/_federation/v1/events" {
		t.Fatalf("expected joined url, got %q", got)
	}
}
