package config

import (
	"strings"
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

// CB-LOSS-01: PRIVATE_NOTE_KEY is base64 of 32 bytes; missing in production disables the private notes.
func TestLoad_PrivateNoteKey(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Empty(t, cfg.PrivateNotes.Key)
	assert.True(t, cfg.PrivateNotes.Missing(cfg.App))

	env := minimal()
	env["PRIVATE_NOTE_KEY"] = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // 32 bytes
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Len(t, cfg.PrivateNotes.Key, PrivateNoteKeyLen)
	assert.False(t, cfg.PrivateNotes.Missing(cfg.App))

	for envName, missing := range map[string]bool{"local": false, "testing": false, "staging": true, "contract": true} {
		env = minimal()
		env["APP_ENV"] = envName
		cfg, err = LoadFrom(lookup(env))
		require.NoError(t, err)
		assert.Equal(t, missing, cfg.PrivateNotes.Missing(cfg.App), "the development key only in local/testing: "+envName)
	}

	env = minimal()
	env["PRIVATE_NOTE_KEY_PREVIOUS"] = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=, YWJjZGVmMDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODk="
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Len(t, cfg.PrivateNotes.PreviousKeys, 2)
	env["PRIVATE_NOTE_KEY_PREVIOUS"] = "c2hvcnQ="
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "PRIVATE_NOTE_KEY_PREVIOUS")

	for _, bad := range []string{"not base64!", "c2hvcnQ="} {
		env = minimal()
		env["PRIVATE_NOTE_KEY"] = bad
		_, err = LoadFrom(lookup(env))
		require.ErrorContains(t, err, "PRIVATE_NOTE_KEY", bad)
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

func TestLoad_AIProvider(t *testing.T) {
	// Production without AI_PROVIDER: none (AI features answer 503), never the fake.
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, AIProviderNone, cfg.AI.Provider)
	assert.Equal(t, 30*time.Second, cfg.AI.Timeout)
	assert.Equal(t, "gemini-flash-latest", cfg.AI.Gemini.Model)
	assert.Empty(t, cfg.AI.Gemini.APIKey)

	// Outside production the fake is the default.
	env := minimal()
	env["APP_ENV"] = "staging"
	env["APP_URL"] = "https://stage.example"
	env["PLUS_CALLBACK_URL"] = "https://stage.example/plus/return"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, AIProviderFake, cfg.AI.Provider)

	// The fake is refused in production; unknown providers are refused anywhere.
	env = minimal()
	env["AI_PROVIDER"] = "fake"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "AI_PROVIDER=fake is not allowed")
	env["AI_PROVIDER"] = "openai"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "AI_PROVIDER")
	env["AI_PROVIDER"] = "Gemini"
	env["GEMINI_API_KEY"] = "k"
	env["GEMINI_BASE_URL"] = "http://127.0.0.1:1/"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "GEMINI_BASE_URL", "https only in production")
	env["GEMINI_BASE_URL"] = "https://user:pw@gemini.example"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "GEMINI_BASE_URL")
	env["GEMINI_BASE_URL"] = "https://gemini-proxy.example/"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, AIProviderGemini, cfg.AI.Provider)
	assert.Equal(t, "https://gemini-proxy.example", cfg.AI.Gemini.BaseURL)
	// http is fine outside production (local proxy / tests)
	env["APP_ENV"] = "local"
	env["GEMINI_BASE_URL"] = "http://127.0.0.1:1"
	_, err = LoadFrom(lookup(env))
	require.NoError(t, err)
}

func TestLoad_Companion(t *testing.T) {
	// Outside production: the fake invite SMS is the default, no pepper needed.
	env := minimal()
	env["APP_ENV"] = "local"
	cfg, err := LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, CompanionSMSFake, cfg.Companion.SMSProvider)
	assert.Equal(t, "companion-invite", cfg.Companion.InviteTemplate)
	assert.False(t, cfg.Companion.PepperMissing(cfg.App))

	// Production: none by default; the missing pepper does not stop start-up but disables invites (fail closed).
	cfg, err = LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, CompanionSMSNone, cfg.Companion.SMSProvider)
	assert.True(t, cfg.Companion.PepperMissing(cfg.App))

	env = minimal()
	env["COMPANION_CODE_PEPPER"] = strings.Repeat("p", MinCompanionPepperLen)
	env["COMPANION_SMS_PROVIDER"] = "gateway"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.False(t, cfg.Companion.PepperMissing(cfg.App))
	assert.Equal(t, CompanionSMSGateway, cfg.Companion.SMSProvider)

	// Refused: the fake in production, a short pepper in production, an unknown provider.
	env = minimal()
	env["COMPANION_SMS_PROVIDER"] = "fake"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "COMPANION_SMS_PROVIDER=fake")
	env = minimal()
	env["COMPANION_CODE_PEPPER"] = "short"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "COMPANION_CODE_PEPPER")
	env = minimal()
	env["APP_ENV"] = "local"
	env["COMPANION_SMS_PROVIDER"] = "pigeon"
	_, err = LoadFrom(lookup(env))
	require.ErrorContains(t, err, "COMPANION_SMS_PROVIDER")
}

func TestLoad_AICostCapAndPrices(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.InDelta(t, 5.0, cfg.AI.DailyCostCapUSD, 1e-9)
	assert.InDelta(t, 0.25, cfg.AI.UserDailyCostCapUSD, 1e-9, "B-N6-05b per-user cap default")
	require.Contains(t, cfg.AI.Prices, "gemini-flash-latest")
	assert.Equal(t, AIPrice{InputPerMTok: 0.30, OutputPerMTok: 2.50, AudioPerMTok: 1.00}, cfg.AI.Prices["gemini-flash-latest"])

	env := minimal()
	env["AI_DAILY_COST_CAP_USD"] = "0"
	env["AI_PRICES"] = "m1=1/2, m2=0.5/4/3"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Zero(t, cfg.AI.DailyCostCapUSD, "0 is allowed: every call refused (fail closed)")
	assert.Equal(t, AIPrice{InputPerMTok: 1, OutputPerMTok: 2, AudioPerMTok: 1}, cfg.AI.Prices["m1"], "audio defaults to input")
	assert.Equal(t, AIPrice{InputPerMTok: 0.5, OutputPerMTok: 4, AudioPerMTok: 3}, cfg.AI.Prices["m2"])

	for _, bad := range []map[string]string{
		{"AI_DAILY_COST_CAP_USD": "-1"},
		{"AI_DAILY_COST_CAP_USD": "lots"},
		{"AI_USER_DAILY_COST_CAP_USD": "-0.1"},
		{"AI_PRICES": "m1"},
		{"AI_PRICES": "m1=1"},
		{"AI_PRICES": "m1=a/b"},
		{"AI_PRICES": "m1=-1/2"},
		{"AI_PRICES": "=1/2"},
		{"AI_PRICES": ","},
	} {
		env := minimal()
		for k, v := range bad {
			env[k] = v
		}
		_, err := LoadFrom(lookup(env))
		require.Error(t, err, bad)
	}
}
