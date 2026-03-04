package federation

import "testing"

func TestParseTXTRecord(t *testing.T) {
	t.Parallel()

	publicKey, endpoint := parseTXTRecord("v=mc1;key=test-public-key;endpoint=http://backend-remote:8080")
	if publicKey != "test-public-key" {
		t.Fatalf("expected public key to be parsed, got %q", publicKey)
	}
	if endpoint != "http://backend-remote:8080" {
		t.Fatalf("expected endpoint to be parsed, got %q", endpoint)
	}
}

func TestParseTXTRecordInvalidEndpointFallsBackEmpty(t *testing.T) {
	t.Parallel()

	publicKey, endpoint := parseTXTRecord("v=mc1;key=test-public-key;endpoint=not a url")
	if publicKey != "test-public-key" {
		t.Fatalf("expected public key to be parsed, got %q", publicKey)
	}
	if endpoint != "" {
		t.Fatalf("expected invalid endpoint to be cleared, got %q", endpoint)
	}
}

func TestParseTXTRecordRejectsWrongVersion(t *testing.T) {
	t.Parallel()

	publicKey, endpoint := parseTXTRecord("v=mc2;key=test-public-key;endpoint=http://backend-remote:8080")
	if publicKey != "" || endpoint != "" {
		t.Fatalf("expected invalid version record to be ignored, got publicKey=%q endpoint=%q", publicKey, endpoint)
	}
}

func TestBuildDefaultEndpoint(t *testing.T) {
	t.Parallel()

	if got := buildDefaultEndpoint("remote3.test"); got != "https://remote3.test" {
		t.Fatalf("expected https://remote3.test, got %q", got)
	}
}
