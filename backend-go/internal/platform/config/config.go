// Package config loads the service configuration from the environment.
//
// Variable names match the Laravel backend (backend/config/*.php, docker-compose.yml)
// so both stacks can be fed from the same .env during the strangler period. Go-only
// variables are HTTP_ADDR, STORAGE_PATH and REDIS_PREFIX; RUN_MIGRATIONS is shared (each
// service's entrypoint reads its own value).
//
// Load fails fast: every missing required variable and every malformed value is
// reported in one error, and cmd/api refuses to start.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultCORSOrigins mirrors the fallback list in backend/config/cors.php.
var DefaultCORSOrigins = []string{
	"https://web.ritme.app",
	"http://localhost:3000",
	"http://127.0.0.1:3000",
}

// Config is the whole service configuration. It is immutable after Load.
type Config struct {
	App      App
	HTTP     HTTP
	DB       DB
	Redis    Redis
	CORS     CORS
	Passport Passport
	SMS      SMS
	Telegram Telegram
	Swagger  Swagger
	// StoragePath is the mounted Laravel storage/ directory (backend-storage volume):
	// Passport keys, translations, public uploads.
	StoragePath string
	// RunMigrations (RUN_MIGRATIONS, default false) makes cmd/api run db.MigrateOnStart before
	// serving. True only where goose owns the schema (stage, T-M2-28); prod keeps it false until
	// T-M2-27 — Laravel's entrypoint migrates there.
	RunMigrations bool
}

// App holds the APP_* settings.
type App struct {
	Name     string
	Env      string
	Debug    bool
	URL      string
	Timezone string
	Location *time.Location
	Locale   string
}

// IsProduction reports whether APP_ENV is "production".
func (a App) IsProduction() bool { return a.Env == "production" }

// HTTP holds the listener settings.
type HTTP struct {
	Addr string
}

// DB holds the MariaDB connection settings (DB_*).
type DB struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// Redis holds the Redis connection settings (REDIS_*).
type Redis struct {
	Host     string
	Port     int
	Username string
	Password string
	DB       int
	Prefix   string
}

// Addr returns host:port.
func (r Redis) Addr() string { return fmt.Sprintf("%s:%d", r.Host, r.Port) }

// CORS holds the allowed browser origins (CORS_ALLOWED_ORIGINS, comma separated).
type CORS struct {
	AllowedOrigins []string
}

// Passport holds the session-token lifetime rules (see backend/CLAUDE.MD).
type Passport struct {
	TokenLifetimeDays    int
	RefreshWindowDays    int
	PasswordClientID     string
	PasswordClientSecret string
}

// SMS holds the OTP provider settings.
type SMS struct {
	Provider  string
	Kavenegar Kavenegar
	SMSIR     SMSIR
}

// Kavenegar holds KAVENEGAR_*.
type Kavenegar struct {
	APIKey           string
	Sender           string
	TemplateLoginOTP string
}

// SMSIR holds SMSIR_*.
type SMSIR struct {
	APIKey     string
	TemplateID int
	LineNumber string
}

// Telegram holds TELEGRAM_* (empty token = notifier disabled, as in prod today).
type Telegram struct {
	BotToken string
	ChatID   string
	Timeout  time.Duration
}

// Swagger holds the Basic-auth credentials guarding /docs.
type Swagger struct {
	User     string
	Password string
}

// Load reads the process environment.
func Load() (*Config, error) { return LoadFrom(os.LookupEnv) }

// LoadFrom reads configuration through lookup (os.LookupEnv in production, a map in tests).
func LoadFrom(lookup func(string) (string, bool)) (*Config, error) {
	e := &env{lookup: lookup}

	cfg := &Config{
		App: App{
			Name:     e.str("APP_NAME", "Ritme"),
			Env:      e.str("APP_ENV", "production"),
			Debug:    e.boolean("APP_DEBUG", false),
			URL:      e.str("APP_URL", "http://localhost"),
			Timezone: e.str("APP_TIMEZONE", "Asia/Tehran"),
			Locale:   e.str("APP_LOCALE", "en"),
		},
		HTTP: HTTP{Addr: e.str("HTTP_ADDR", ":8020")},
		DB: DB{
			Host:     e.required("DB_HOST"),
			Port:     e.integer("DB_PORT", 3306),
			Database: e.required("DB_DATABASE"),
			Username: e.required("DB_USERNAME"),
			Password: e.str("DB_PASSWORD", ""),
		},
		Redis: Redis{
			Host:     e.required("REDIS_HOST"),
			Port:     e.integer("REDIS_PORT", 6379),
			Username: e.str("REDIS_USERNAME", ""),
			Password: e.str("REDIS_PASSWORD", ""),
			DB:       e.integer("REDIS_DB", 0),
			Prefix:   e.str("REDIS_PREFIX", "ritme-go:"),
		},
		CORS: CORS{AllowedOrigins: e.list("CORS_ALLOWED_ORIGINS", DefaultCORSOrigins)},
		Passport: Passport{
			TokenLifetimeDays:    e.integer("PASSPORT_TOKEN_LIFETIME_DAYS", 365),
			RefreshWindowDays:    e.integer("PASSPORT_REFRESH_WINDOW_DAYS", 30),
			PasswordClientID:     e.str("PASSPORT_PASSWORD_CLIENT_ID", ""),
			PasswordClientSecret: e.str("PASSPORT_PASSWORD_CLIENT_SECRET", ""),
		},
		SMS: SMS{
			Provider: e.str("SMS_PROVIDER", "kavenegar"),
			Kavenegar: Kavenegar{
				APIKey:           e.str("KAVENEGAR_API_KEY", ""),
				Sender:           e.str("KAVENEGAR_SENDER", ""),
				TemplateLoginOTP: e.str("KAVENEGAR_TEMPLATE_LOGIN_OTP", "1507703"),
			},
			SMSIR: SMSIR{
				APIKey:     e.str("SMSIR_API_KEY", ""),
				TemplateID: e.integer("SMSIR_TEMPLATE_ID", 511293),
				LineNumber: e.str("SMSIR_LINE_NUMBER", ""),
			},
		},
		Telegram: Telegram{
			BotToken: e.str("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   e.str("TELEGRAM_CHAT_ID", ""),
			Timeout:  time.Duration(e.integer("TELEGRAM_TIMEOUT", 5)) * time.Second,
		},
		Swagger: Swagger{
			User:     e.str("SWAGGER_USER", "ritme"),
			Password: e.str("SWAGGER_PASSWORD", ""),
		},
		StoragePath:   strings.TrimRight(e.required("STORAGE_PATH"), "/"),
		RunMigrations: e.boolean("RUN_MIGRATIONS", false),
	}

	if cfg.App.Timezone != "" {
		loc, err := time.LoadLocation(cfg.App.Timezone)
		if err != nil {
			e.fail("APP_TIMEZONE: %v", err)
		}
		cfg.App.Location = loc
	}
	if len(cfg.CORS.AllowedOrigins) == 0 {
		e.fail("CORS_ALLOWED_ORIGINS: no origins after parsing")
	}
	if cfg.App.IsProduction() && cfg.App.Debug {
		e.fail("APP_DEBUG must be false when APP_ENV=production")
	}

	if len(e.errs) > 0 {
		return nil, fmt.Errorf("config: %w", errors.Join(e.errs...))
	}
	return cfg, nil
}

// env reads variables with Laravel's env() semantics: "null"/"(null)" and
// "empty"/"(empty)" mean empty, surrounding quotes are stripped, and an empty
// value falls back to the default.
type env struct {
	lookup func(string) (string, bool)
	errs   []error
}

func (e *env) fail(format string, args ...any) {
	e.errs = append(e.errs, fmt.Errorf(format, args...))
}

func (e *env) raw(key string) string {
	v, ok := e.lookup(key)
	if !ok {
		return ""
	}
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
		v = v[1 : len(v)-1]
	}
	switch strings.ToLower(v) {
	case "null", "(null)", "empty", "(empty)":
		return ""
	}
	return v
}

func (e *env) str(key, def string) string {
	if v := e.raw(key); v != "" {
		return v
	}
	return def
}

func (e *env) required(key string) string {
	v := e.raw(key)
	if v == "" {
		e.fail("%s is required", key)
	}
	return v
}

func (e *env) integer(key string, def int) int {
	v := e.raw(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		e.fail("%s: %q is not an integer", key, v)
		return def
	}
	return n
}

func (e *env) boolean(key string, def bool) bool {
	switch strings.ToLower(e.raw(key)) {
	case "":
		return def
	case "true", "(true)", "1", "on", "yes":
		return true
	case "false", "(false)", "0", "off", "no":
		return false
	default:
		e.fail("%s: %q is not a boolean", key, e.raw(key))
		return def
	}
}

// list mirrors cors.php: explode on ",", trim, drop empties.
func (e *env) list(key string, def []string) []string {
	v := e.raw(key)
	if v == "" {
		return append([]string(nil), def...)
	}
	out := make([]string, 0)
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
