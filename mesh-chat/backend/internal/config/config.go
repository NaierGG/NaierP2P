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
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
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

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.mode", "release")

	v.SetDefault("database.postgres_dsn", "postgres://mesh:mesh@postgres:5432/meshchat?sslmode=disable")
	v.SetDefault("database.redis_addr", "redis:6379")
	v.SetDefault("database.redis_password", "")

	v.SetDefault("auth.jwt_secret", "change-me-in-production")
	v.SetDefault("auth.jwt_expiry", "15m")
	v.SetDefault("auth.refresh_expiry", "720h")

	v.SetDefault("media.minio_endpoint", "minio:9000")
	v.SetDefault("media.minio_bucket", "meshchat")
	v.SetDefault("media.minio_access_key", "minioadmin")
	v.SetDefault("media.minio_secret_key", "minioadmin")

	v.SetDefault("federation.server_domain", "local.meshchat")
	v.SetDefault("federation.server_public_key", "")
	v.SetDefault("federation.server_private_key", "")
}
