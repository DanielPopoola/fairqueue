// Package config loads and validates runtime configuration.
package config

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Env          string         `koanf:"env"      validate:"required,oneof=development staging production"`
	Server       ServerConfig   `koanf:"server"   validate:"required"`
	Database     DatabaseConfig `koanf:"database" validate:"required"`
	Redis        RedisConfig    `koanf:"redis"    validate:"required"`
	Auth         AuthConfig     `koanf:"auth"     validate:"required"`
	Paystack     PaystackConfig `koanf:"paystack" validate:"required"`
	GatewayRetry RetryConfig    `koanf:"gatewayretry" validate:"required"`
	Workers      WorkersConfig  `koanf:"workers"  validate:"required"`
	Logger       LoggerConfig   `koanf:"logger"`
}

type ServerConfig struct {
	Port         string        `koanf:"port"          validate:"required"`
	ReadTimeout  time.Duration `koanf:"read_timeout"  validate:"required"`
	WriteTimeout time.Duration `koanf:"write_timeout" validate:"required"`
	IdleTimeout  time.Duration `koanf:"idle_timeout"  validate:"required"`
}

type DatabaseConfig struct {
	Host            string        `koanf:"host" validate:"required"`
	Port            int           `koanf:"port" validate:"required"`
	User            string        `koanf:"user" validate:"required"`
	Password        string        `koanf:"password" validate:"required"`
	Name            string        `koanf:"name" validate:"required"`
	SSLMode         string        `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int           `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int           `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time" validate:"required"`
}

type RedisConfig struct {
	Host     string `koanf:"host"     validate:"required"`
	Port     int    `koanf:"port"     validate:"required"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

type AuthConfig struct {
	TokenSecret string        `koanf:"token_secret"  validate:"required,min=32"`
	TokenTTL    time.Duration `koanf:"token_ttl"     validate:"required"`
}

type PaystackConfig struct {
	SecretKey string `koanf:"secret_key"      validate:"required"`
	BaseURL   string `koanf:"base_url"        validate:"required,url"`
}

type RetryConfig struct {
	MaxAttempts int           `koanf:"max_attempts" validate:"required"`
	BaseDelay   time.Duration `koanf:"base_delay"   validate:"required"`
	MaxDelay    time.Duration `koanf:"max_delay"    validate:"required"`
}

// WorkersConfig gives each worker its own interval and batch size
// because they have meaningfully different cadences:
// admission runs every few seconds, reconciliation and expiry every 30s.
type WorkersConfig struct {
	Admission      AdmissionWorkerConfig      `koanf:"admission"      validate:"required"`
	Expiry         ExpiryWorkerConfig         `koanf:"expiry"         validate:"required"`
	Reconciliation ReconciliationWorkerConfig `koanf:"reconciliation" validate:"required"`
}

type AdmissionWorkerConfig struct {
	Interval  time.Duration `koanf:"interval"   validate:"required"`
	BatchSize int           `koanf:"batch_size" validate:"required"`
}

type ExpiryWorkerConfig struct {
	Interval  time.Duration `koanf:"interval"   validate:"required"`
	BatchSize int           `koanf:"batch_size" validate:"required"`
}

type ReconciliationWorkerConfig struct {
	Interval           time.Duration `koanf:"interval"               validate:"required"`
	StalePaymentAge    time.Duration `koanf:"stale_payment_age"      validate:"required"`
	StaleQueueEntryAge time.Duration `koanf:"stale_queue_entry_age"  validate:"required"`
}

type LoggerConfig struct {
	Level string `koanf:"level"`
}

// Load reads configuration from environment variables and validates it.
func Load() (*Config, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	k := koanf.New(".")

	err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(s), "__", ".")
	}), nil)
	if err != nil {
		logger.Error("failed to load environment variables", "error", err)
		return nil, err
	}

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		logger.Error("failed to unmarshal config", "error", err)
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		logger.Error("config validation failed", "error", err)
		return nil, err
	}

	return cfg, nil
}
