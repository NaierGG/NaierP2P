package config

import "testing"

func TestNormalizeOrigins(t *testing.T) {
	origins := normalizeOrigins([]string{
		" https://app.example.com ",
		"",
		"https://app.example.com",
		"https://staging.example.com",
	})

	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if origins[0] != "https://app.example.com" {
		t.Fatalf("unexpected first origin %q", origins[0])
	}
}

func TestValidateConfigRelease(t *testing.T) {
	cfg := Config{
		Server: ServerConfig{
			Mode:           "release",
			AllowedOrigins: []string{"https://app.example.com"},
		},
		Auth: AuthConfig{
			JWTSecret: "super-secret",
		},
		Media: MediaConfig{
			MinIOEndpoint:  "minio.internal:9000",
			MinIOBucket:    "naier",
			MinIOAccessKey: "release-access",
			MinIOSecretKey: "release-secret",
		},
		Federation: FederationConfig{
			ServerDomain:     "api.example.com",
			ServerPublicKey:  "pub",
			ServerPrivateKey: "priv",
		},
		Beta: BetaConfig{
			InviteOnly: true,
		},
		Admin: AdminConfig{
			APIToken: "admin-token",
		},
	}

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("expected release config to validate, got %v", err)
	}
}
