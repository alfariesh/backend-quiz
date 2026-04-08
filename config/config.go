package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	GoogleOAuth GoogleOAuthConfig
	R2 R2Config
}

type ServerConfig struct {
	Port         int           `envconfig:"SERVER_PORT" default:"8080"`
	Environment  string        `envconfig:"ENVIRONMENT" default:"development"`
	ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"30s"`
}

type DatabaseConfig struct {
	URL             string        `envconfig:"DATABASE_URL" required:"true"`
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
}

type RedisConfig struct {
	Addr     string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	Password string `envconfig:"REDIS_PASSWORD" default:""`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

type JWTConfig struct {
	Secret          string        `envconfig:"JWT_SECRET" required:"true"`
	AccessDuration  time.Duration `envconfig:"JWT_ACCESS_DURATION" default:"15m"`
	RefreshDuration time.Duration `envconfig:"JWT_REFRESH_DURATION" default:"720h"`
}

type GoogleOAuthConfig struct {
	ClientID     string `envconfig:"GOOGLE_CLIENT_ID"`
	ClientSecret string `envconfig:"GOOGLE_CLIENT_SECRET"`
	RedirectURL  string `envconfig:"GOOGLE_REDIRECT_URL"`
}

type R2Config struct {
	AccountID      string `envconfig:"R2_ACCOUNT_ID"`
	AccessKeyID    string `envconfig:"R2_ACCESS_KEY_ID"`
	SecretAccessKey string `envconfig:"R2_SECRET_ACCESS_KEY"`
	BucketName     string `envconfig:"R2_BUCKET_NAME"`
	PublicURL      string `envconfig:"R2_PUBLIC_URL"`
	MaxFileSizeMB  int    `envconfig:"R2_MAX_FILE_SIZE_MB" default:"10"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
