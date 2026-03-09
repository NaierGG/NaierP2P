package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Media      MediaConfig      `mapstructure:"media"`
	Federation FederationConfig `mapstructure:"federation"`
	Beta       BetaConfig       `mapstructure:"beta"`
	Admin      AdminConfig      `mapstructure:"admin"`
}

type ServerConfig struct {
	Port           string   `mapstructure:"port"`
	Host           string   `mapstructure:"host"`
	Mode           string   `mapstructure:"mode"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type DatabaseConfig struct {
	PostgresDSN   string `mapstructure:"postgres_dsn"`
	RedisAddr     string `mapstructure:"redis_addr"`
	RedisPassword string `mapstructure:"redis_password"`
}

type AuthConfig struct {
	JWTSecret     string        `mapstructure:"jwt_secret"`
	JWTExpiry     time.Duration `mapstructure:"jwt_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

type MediaConfig struct {
	MinIOEndpoint  string `mapstructure:"minio_endpoint"`
	MinIOBucket    string `mapstructure:"minio_bucket"`
	MinIOAccessKey string `mapstructure:"minio_access_key"`
	MinIOSecretKey string `mapstructure:"minio_secret_key"`
}

type FederationConfig struct {
	ServerDomain     string `mapstructure:"server_domain"`
	ServerPublicKey  string `mapstructure:"server_public_key"`
	ServerPrivateKey string `mapstructure:"server_private_key"`
}

type BetaConfig struct {
	InviteOnly bool `mapstructure:"invite_only"`
}

type AdminConfig struct {
	APIToken string `mapstructure:"api_token"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	v.AddConfigPath("/app")
	v.SetEnvPrefix("MESH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	_ = v.ReadInConfig()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Auth.JWTExpiry = v.GetDuration("auth.jwt_expiry")
	cfg.Auth.RefreshExpiry = v.GetDuration("auth.refresh_expiry")
	cfg.Server.AllowedOrigins = normalizeOrigins(v.GetStringSlice("server.allowed_origins"))
	if len(cfg.Server.AllowedOrigins) == 0 {
		cfg.Server.AllowedOrigins = normalizeOrigins(strings.Split(v.GetString("server.allowed_origins"), ","))
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "release")
	v.SetDefault("server.allowed_origins", []string{
		"http://localhost:4173",
		"http://127.0.0.1:4173",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	})

	v.SetDefault("database.postgres_dsn", "postgres://mesh:mesh@postgres:5432/naier?sslmode=disable")
	v.SetDefault("database.redis_addr", "redis:6379")
	v.SetDefault("database.redis_password", "")

	v.SetDefault("auth.jwt_secret", "change-me-in-production")
	v.SetDefault("auth.jwt_expiry", "15m")
	v.SetDefault("auth.refresh_expiry", "720h")

	v.SetDefault("media.minio_endpoint", "minio:9000")
	v.SetDefault("media.minio_bucket", "naier")
	v.SetDefault("media.minio_access_key", "minioadmin")
	v.SetDefault("media.minio_secret_key", "minioadmin")

	v.SetDefault("federation.server_domain", "local.naier")
	v.SetDefault("federation.server_public_key", "")
	v.SetDefault("federation.server_private_key", "")

	v.SetDefault("beta.invite_only", false)
	v.SetDefault("admin.api_token", "")
}

func normalizeOrigins(origins []string) []string {
	normalized := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func validateConfig(cfg Config) error {
	if cfg.Server.Mode != "release" {
		return nil
	}

	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" || cfg.Auth.JWTSecret == "change-me-in-production" || cfg.Auth.JWTSecret == "change-me" {
		return fmt.Errorf("release mode requires a non-default auth.jwt_secret")
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		return fmt.Errorf("release mode requires server.allowed_origins")
	}
	if strings.TrimSpace(cfg.Media.MinIOEndpoint) == "" ||
		strings.TrimSpace(cfg.Media.MinIOBucket) == "" ||
		strings.TrimSpace(cfg.Media.MinIOAccessKey) == "" ||
		strings.TrimSpace(cfg.Media.MinIOSecretKey) == "" {
		return fmt.Errorf("release mode requires media MinIO endpoint, bucket, access key, and secret key")
	}
	if cfg.Media.MinIOAccessKey == "minioadmin" || cfg.Media.MinIOSecretKey == "minioadmin" || cfg.Media.MinIOSecretKey == "minioadmin123" {
		return fmt.Errorf("release mode requires non-default MinIO credentials")
	}
	if strings.TrimSpace(cfg.Federation.ServerDomain) == "" ||
		strings.TrimSpace(cfg.Federation.ServerPublicKey) == "" ||
		strings.TrimSpace(cfg.Federation.ServerPrivateKey) == "" {
		return fmt.Errorf("release mode requires federation server domain and keypair")
	}
	if cfg.Beta.InviteOnly && strings.TrimSpace(cfg.Admin.APIToken) == "" {
		return fmt.Errorf("invite-only beta requires admin.api_token")
	}

	return nil
}
