// Package config loads and validates all application configuration from
// environment variables.  The .env.dev file is loaded automatically in
// non-production environments via godotenv.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is the root configuration object passed through the dependency graph.
type Config struct {
	App        AppConfig
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Cloudinary CloudinaryConfig
	Log        LogConfig
}

// AppConfig holds global application-level settings.
type AppConfig struct {
	// Env is the runtime environment: development | production | test.
	Env string
	// MaxFileSizeBytes is the maximum allowed multipart upload size.
	MaxFileSizeBytes int64
}

// ServerConfig controls the HTTP server.
type ServerConfig struct {
	// Port is the TCP port the server listens on (e.g. "8080").
	Port string
	// GinMode is passed to gin.SetMode: debug | release | test.
	GinMode string
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	// DSN is the full libpq-compatible connection string.
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	CacheTTL time.Duration
}

// JWTConfig holds token signing parameters.
type JWTConfig struct {
	Secret          string
	AccessTokenExp  time.Duration
	RefreshTokenExp time.Duration
}

// CloudinaryConfig holds credentials for the Cloudinary storage provider.
type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
}

// LogConfig controls the file logger.
type LogConfig struct {
	Dir        string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
}

// Load reads configuration from environment variables.
// In non-production environments it first loads .env.dev (if it exists).
func Load() (*Config, error) {
	env := getStr("APP_ENV", "development")

	if !strings.EqualFold(env, "production") {
		// Best-effort: ignore error if the file is absent (e.g. in CI).
		_ = godotenv.Load(".env.dev")
	}

	// Rebuild env after potential godotenv load.
	env = getStr("APP_ENV", "development")

	maxFileMB := getInt64("APP_MAX_FILE_SIZE_MB", 5)

	dbDSN, err := buildDSN()
	if err != nil {
		return nil, err
	}

	jwtSecret := mustStr("JWT_SECRET")

	return &Config{
		App: AppConfig{
			Env:              env,
			MaxFileSizeBytes: maxFileMB * 1024 * 1024,
		},
		Server: ServerConfig{
			Port:    getStr("APP_PORT", "8080"),
			GinMode: getStr("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			DSN:             dbDSN,
			MaxConns:        int32(getInt64("POSTGRES_MAX_CONNS", 25)),
			MinConns:        int32(getInt64("POSTGRES_MIN_CONNS", 5)),
			MaxConnLifetime: getDuration("POSTGRES_MAX_CONN_LIFETIME_MINUTES", 30) * time.Minute,
		},
		Redis: RedisConfig{
			Addr:     getStr("REDIS_ADDR", "localhost:6379"),
			Password: getStr("REDIS_PASSWORD", ""),
			DB:       int(getInt64("REDIS_DB", 0)),
			CacheTTL: getDuration("REDIS_CACHE_TTL_MINUTES", 15) * time.Minute,
		},
		JWT: JWTConfig{
			Secret:          jwtSecret,
			AccessTokenExp:  getDuration("JWT_ACCESS_TOKEN_EXP_MINUTES", 60) * time.Minute,
			RefreshTokenExp: getDuration("JWT_REFRESH_TOKEN_EXP_DAYS", 7) * 24 * time.Hour,
		},
		Cloudinary: CloudinaryConfig{
			CloudName: getStr("CLOUDINARY_CLOUD_NAME", ""),
			APIKey:    getStr("CLOUDINARY_API_KEY", ""),
			APISecret: getStr("CLOUDINARY_API_SECRET", ""),
			Folder:    getStr("CLOUDINARY_FOLDER", "simple-employees-crud/avatars"),
		},
		Log: LogConfig{
			Dir:        getStr("LOG_DIR", "./logs"),
			MaxSizeMB:  int(getInt64("LOG_MAX_SIZE_MB", 100)),
			MaxBackups: int(getInt64("LOG_MAX_BACKUPS", 7)),
			MaxAgeDays: int(getInt64("LOG_MAX_AGE_DAYS", 30)),
		},
	}, nil
}

// buildDSN constructs the PostgreSQL DSN from individual environment variables
// or returns DATABASE_URL directly if it is set.
func buildDSN() (string, error) {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url, nil
	}

	host := mustStrOrErr("POSTGRES_HOST")
	port := getStr("POSTGRES_PORT", "5432")
	user := mustStrOrErr("POSTGRES_USER")
	pass := mustStrOrErr("POSTGRES_PASSWORD")
	db := getStr("POSTGRES_DB", "employee_mgmt")
	ssl := getStr("POSTGRES_SSL_MODE", "disable")

	var errs []string
	for k, v := range map[string]string{"POSTGRES_HOST": host, "POSTGRES_USER": user, "POSTGRES_PASSWORD": pass} {
		if v == "" || v == "REPLACE-ME" {
			errs = append(errs, k)
		}
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("config: missing required env vars: %s", strings.Join(errs, ", "))
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, pass, host, port, db, ssl,
	), nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func getStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustStr(key string) string {
	v := os.Getenv(key)
	if v == "" || v == "REPLACE-ME" {
		panic(fmt.Sprintf("config: required env var %q is not set", key))
	}
	return v
}

func mustStrOrErr(key string) string {
	return os.Getenv(key)
}

func getInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getDuration(key string, fallbackUnits int64) time.Duration {
	return time.Duration(getInt64(key, fallbackUnits))
}

// IsDevelopment reports whether the app is running in development mode.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.App.Env, "development")
}

// Validate checks that mandatory config fields are populated. Call this after
// Load() to get a single aggregated error instead of panicking on first miss.
func (c *Config) Validate() error {
	var errs []string

	if c.JWT.Secret == "" {
		errs = append(errs, "JWT_SECRET")
	}
	if c.Database.DSN == "" {
		errs = append(errs, "DATABASE_URL or POSTGRES_* vars")
	}

	if len(errs) > 0 {
		return errors.New("config: missing required configuration: " + strings.Join(errs, ", "))
	}
	return nil
}
