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
	R2          R2Config
	Mailer      MailerConfig
	App         AppConfig
	AuthPolicy  AuthPolicyConfig
	Security    SecurityConfig
	Captcha     CaptchaConfig
	RAG         RAGConfig
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
	ClientID      string `envconfig:"GOOGLE_CLIENT_ID"`
	ClientSecret  string `envconfig:"GOOGLE_CLIENT_SECRET"`
	RedirectURL   string `envconfig:"GOOGLE_REDIRECT_URL"`
	SuccessRedirect string `envconfig:"GOOGLE_SUCCESS_REDIRECT" default:"/"`
	FailureRedirect string `envconfig:"GOOGLE_FAILURE_REDIRECT" default:"/"`
}

func (g GoogleOAuthConfig) Enabled() bool {
	return g.ClientID != "" && g.ClientSecret != "" && g.RedirectURL != ""
}

type R2Config struct {
	AccountID      string `envconfig:"R2_ACCOUNT_ID"`
	AccessKeyID    string `envconfig:"R2_ACCESS_KEY_ID"`
	SecretAccessKey string `envconfig:"R2_SECRET_ACCESS_KEY"`
	BucketName     string `envconfig:"R2_BUCKET_NAME"`
	PublicURL      string `envconfig:"R2_PUBLIC_URL"`
	MaxFileSizeMB  int    `envconfig:"R2_MAX_FILE_SIZE_MB" default:"10"`
}

type MailerConfig struct {
	ResendAPIKey string `envconfig:"RESEND_API_KEY"`
	FromAddress  string `envconfig:"MAIL_FROM" default:"Surau Quiz <noreply@example.com>"`
	ReplyTo      string `envconfig:"MAIL_REPLY_TO"`
}

type AppConfig struct {
	Name        string `envconfig:"APP_NAME" default:"Surau Quiz"`
	FrontendURL string `envconfig:"APP_FRONTEND_URL" default:"http://localhost:3000"`
}

type AuthPolicyConfig struct {
	OTPLength            int           `envconfig:"AUTH_OTP_LENGTH" default:"6"`
	OTPTTL               time.Duration `envconfig:"AUTH_OTP_TTL" default:"15m"`
	OTPMaxAttempts       int           `envconfig:"AUTH_OTP_MAX_ATTEMPTS" default:"5"`
	ResetTokenTTL        time.Duration `envconfig:"AUTH_RESET_TOKEN_TTL" default:"15m"`
	LoginLockoutWindow   time.Duration `envconfig:"AUTH_LOGIN_LOCKOUT_WINDOW" default:"15m"`
	LoginLockoutMaxFails int           `envconfig:"AUTH_LOGIN_LOCKOUT_MAX_FAILS" default:"5"`
	RequireEmailVerified bool          `envconfig:"AUTH_REQUIRE_EMAIL_VERIFIED" default:"false"`

	PasswordMinLength    int           `envconfig:"AUTH_PASSWORD_MIN_LENGTH" default:"8"`
	PasswordMaxLength    int           `envconfig:"AUTH_PASSWORD_MAX_LENGTH" default:"72"`
	PasswordRequireMixed bool          `envconfig:"AUTH_PASSWORD_REQUIRE_MIXED" default:"false"`
	PasswordHIBPCheck    bool          `envconfig:"AUTH_PASSWORD_HIBP_CHECK" default:"true"`
	DeletionGracePeriod  time.Duration `envconfig:"AUTH_DELETION_GRACE_PERIOD" default:"720h"`
}

type SecurityConfig struct {
	// Comma-separated list of allowed origins. Use "*" for any (dev only). Empty = no CORS headers.
	AllowedOrigins []string `envconfig:"SECURITY_ALLOWED_ORIGINS" default:"http://localhost:3000"`
	// Whether to trust X-Forwarded-For / X-Real-IP. Only enable behind a trusted reverse proxy.
	TrustForwardedFor bool `envconfig:"SECURITY_TRUST_FORWARDED_FOR" default:"false"`
	// HSTS max-age in seconds. 0 disables. Recommended: 31536000 (1 year) in production.
	HSTSMaxAge int `envconfig:"SECURITY_HSTS_MAX_AGE" default:"0"`
	// Content-Security-Policy. Empty string disables. Only set if serving HTML.
	ContentSecurityPolicy string `envconfig:"SECURITY_CSP" default:""`
	// Rate-limit settings — stricter limits for public auth endpoints.
	AuthRateLimitRPS   float64 `envconfig:"SECURITY_AUTH_RL_RPS" default:"0.5"`
	AuthRateLimitBurst int     `envconfig:"SECURITY_AUTH_RL_BURST" default:"5"`
}

type CaptchaConfig struct {
	// Cloudflare Turnstile secret key. Empty → captcha disabled.
	TurnstileSecret string `envconfig:"TURNSTILE_SECRET_KEY"`
}

func (c CaptchaConfig) Enabled() bool { return c.TurnstileSecret != "" }

type RAGConfig struct {
	ServiceURL    string        `envconfig:"RAG_SERVICE_URL" default:"http://localhost:8001"`
	Timeout       time.Duration `envconfig:"RAG_TIMEOUT" default:"120s"`
	InternalToken string        `envconfig:"RAG_INTERNAL_TOKEN"`
	StreamTimeout time.Duration `envconfig:"RAG_STREAM_TIMEOUT" default:"3m"`
	UserRPS       float64       `envconfig:"RAG_USER_RPS" default:"0.5"`
	UserBurst     int           `envconfig:"RAG_USER_BURST" default:"3"`
	DailyLimit    int           `envconfig:"RAG_DAILY_LIMIT" default:"100"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
