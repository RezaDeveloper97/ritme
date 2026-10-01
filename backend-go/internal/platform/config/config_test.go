package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookup(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func minimal() map[string]string {
	return map[string]string{
		"DB_HOST":      "mysql",
		"DB_DATABASE":  "ritme_salamat",
		"DB_USERNAME":  "ritme",
		"REDIS_HOST":   "redis",
		"STORAGE_PATH": "/var/www/html/storage/",
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)

	assert.Equal(t, "production", cfg.App.Env)
	assert.False(t, cfg.App.Debug)
	assert.Equal(t, "Asia/Tehran", cfg.App.Location.String())
	assert.Equal(t, 3306, cfg.DB.Port)
	assert.Equal(t, "redis:6379", cfg.Redis.Addr())
	assert.Equal(t, "ritme-go:", cfg.Redis.Prefix)
	assert.Equal(t, DefaultCORSOrigins, cfg.CORS.AllowedOrigins)
	assert.Equal(t, 365, cfg.Passport.TokenLifetimeDays)
	assert.Equal(t, 30, cfg.Passport.RefreshWindowDays)
	assert.Equal(t, 511293, cfg.SMS.SMSIR.TemplateID)
	assert.Equal(t, "1507703", cfg.SMS.Kavenegar.TemplateLoginOTP)
	assert.Equal(t, "/var/www/html/storage", cfg.StoragePath)
	assert.Equal(t, ":8020", cfg.HTTP.Addr)
	assert.False(t, cfg.RunMigrations, "goose never runs unless asked (prod: Laravel owns the schema)")
}

func TestLoad_RunMigrations(t *testing.T) {
	env := minimal()
	env["RUN_MIGRATIONS"] = "true"
	cfg, err := LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.True(t, cfg.RunMigrations)

	env["RUN_MIGRATIONS"] = "sometimes"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "RUN_MIGRATIONS")
}

func TestLoad_LaravelEnvSemantics(t *testing.T) {
	env := minimal()
	env["APP_ENV"] = "local"
	env["APP_DEBUG"] = "(true)"
	env["REDIS_PASSWORD"] = "null"
	env["CORS_ALLOWED_ORIGINS"] = " https://web.ritme.app , ,http://localhost:3000"
	env["KAVENEGAR_SENDER"] = `"1000"`

	cfg, err := LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.True(t, cfg.App.Debug)
	assert.Empty(t, cfg.Redis.Password)
	assert.Equal(t, []string{"https://web.ritme.app", "http://localhost:3000"}, cfg.CORS.AllowedOrigins)
	assert.Equal(t, "1000", cfg.SMS.Kavenegar.Sender)
}

func TestLoad_FailsFastWithEveryProblem(t *testing.T) {
	env := map[string]string{"DB_PORT": "abc", "APP_DEBUG": "maybe", "APP_TIMEZONE": "Mars/Base"}
	_, err := LoadFrom(lookup(env))
	require.Error(t, err)
	for _, want := range []string{"DB_HOST is required", "DB_DATABASE is required", "DB_USERNAME is required",
		"REDIS_HOST is required", "STORAGE_PATH is required", "DB_PORT", "APP_DEBUG", "APP_TIMEZONE"} {
		assert.Contains(t, err.Error(), want)
	}
}

func TestLoad_ProductionRefusesDebug(t *testing.T) {
	env := minimal()
	env["APP_DEBUG"] = "true"
	_, err := LoadFrom(lookup(env))
	require.ErrorContains(t, err, "APP_DEBUG must be false")
}

func TestLoad_PaymentProvider(t *testing.T) {
	// Production without PAYMENT_PROVIDER: no provider (checkout answers 503), never the fake.
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, PaymentProviderNone, cfg.Payment.Provider)
	assert.Equal(t, 15*time.Second, cfg.Payment.Timeout)

	// Outside production the fake is the default; the return allow-list starts with PLUS_CALLBACK_URL.
	env := minimal()
	env["APP_ENV"] = "staging"
	env["APP_URL"] = "https://stage.example"
	env["PLUS_CALLBACK_URL"] = "https://stage.example/plus/return"
	env["PAYMENT_RETURN_URLS"] = "https://stage.example/shop/return"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, PaymentProviderFake, cfg.Payment.Provider)
	assert.Equal(t, "https://stage.example", cfg.Payment.CallbackBaseURL)
	assert.Equal(t, []string{"https://stage.example/plus/return", "https://stage.example/shop/return"}, cfg.Payment.ReturnURLs)

	// The fake is refused in production, at start-up.
	env = minimal()
	env["PAYMENT_PROVIDER"] = "fake"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "PAYMENT_PROVIDER=fake is not allowed")

	// Unknown provider; a real provider in production needs https URLs.
	env = minimal()
	env["PAYMENT_PROVIDER"] = "paypal"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "PAYMENT_PROVIDER")
	env["PAYMENT_PROVIDER"] = "zarinpal"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "https")
	env["APP_URL"] = "https://api.example"
	env["PLUS_CALLBACK_URL"] = "https://web.example/plus/return"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, PaymentProviderZarinpal, cfg.Payment.Provider)
}
