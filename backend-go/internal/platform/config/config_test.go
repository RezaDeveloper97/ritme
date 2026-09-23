package config

import (
	"testing"

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
