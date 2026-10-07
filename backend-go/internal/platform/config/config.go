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
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/url"
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
	Plus     Plus
	Payment  Payment
	AI       AI
	// Companion is «همدم» (bloom B-N4-02): invite-code pepper and the invite SMS adapter.
	Companion Companion
	// Learning is the courses domain (bloom B-N8-01): the «دوره برایت باز شد» SMS adapter.
	Learning Learning
	// PrivateNotes is the key of the encrypted private notes (CB-LOSS-01 loss path).
	PrivateNotes PrivateNotes
	// LabFiles is the key of the encrypted lab-sheet uploads (bloom B-N6-06, internal/labs/files).
	LabFiles LabFiles
	// Files is the key of the generic encrypted file storage (CB-CORE-05, internal/files).
	Files Files
	// Media is the lesson media pipeline (bloom B-N8-02, internal/media): signed playback URL key and media root.
	Media Media
	// Sharing is the 24h doctor code pepper and the QR origin (CB-REC-03, internal/sharelinks).
	Sharing Sharing
	// Telemed is the visit booking (bloom B-N7-03, internal/telemed): hold TTL, gateway return page, Plus discount.
	Telemed Telemed
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

// Plus holds the Ritme Plus checkout settings (B-N2-04). Amounts are rials; the VAT rate is in basis points
// (1000 = 10 %) and is snapshotted onto every invoice at checkout.
type Plus struct {
	VATRateBps  int           // PLUS_VAT_RATE_BPS, 0–10000 (default 1000 = 10 %)
	TrialDays   int           // PLUS_TRIAL_DAYS, 1–90 (default 7)
	InvoiceTTL  time.Duration // PLUS_INVOICE_TTL_MINUTES (default 30): how long a pending checkout holds a discount
	CallbackURL string        // PLUS_CALLBACK_URL: where the gateway sends the user back (the web app's return page)
}

// Telemed holds the visit-booking settings (bloom B-N7-03). Amounts are rials.
type Telemed struct {
	HoldTTL             time.Duration // TELEMED_HOLD_MINUTES (default 10, 2–60): how long a picked slot is held for payment
	CallbackURL         string        // TELEMED_CALLBACK_URL: the web app's booking return page (default {PLUS_CALLBACK_URL origin}/services/bookings/return; always on the payment allow-list)
	PlusDiscountPercent int           // TELEMED_PLUS_DISCOUNT_PERCENT (default 20, 0–90): Plus «تخفیف ویزیت» on the visit price
}

// Payment provider ids accepted by PAYMENT_PROVIDER (internal/payments implements them).
const (
	PaymentProviderNone     = "none"     // no provider: checkout/verify answer 503 payment_unavailable
	PaymentProviderFake     = "fake"     // local TEST gateway (success/fail page served by Go); never in production
	PaymentProviderZarinpal = "zarinpal" // Zarinpal v4 REST web gateway
)

// Payment holds the payment-gateway adapter settings (B-N2-05). No secret has a default; the merchant id lives only
// in the server's .env.
type Payment struct {
	// Provider is PAYMENT_PROVIDER: none | fake | zarinpal. Unset → fake outside production, none in production.
	// PAYMENT_PROVIDER=fake with APP_ENV=production is refused at start-up.
	Provider string
	// CallbackBaseURL (PAYMENT_CALLBACK_BASE_URL, default APP_URL) is the public origin of this API; the gateway sends
	// the user back to {CallbackBaseURL}/api/v1/payments/{provider}/return.
	CallbackBaseURL string
	// ReturnURLs is the allow-list of web-app pages a payment may finally redirect to: PLUS_CALLBACK_URL plus
	// PAYMENT_RETURN_URLS (comma separated). Compared on scheme, host and path; anything else falls back to the first.
	ReturnURLs []string
	// Timeout (PAYMENT_HTTP_TIMEOUT_SECONDS, default 15) bounds every server-to-server gateway call.
	Timeout  time.Duration
	Zarinpal Zarinpal
}

// Zarinpal holds ZARINPAL_* (empty merchant id = provider disabled, payments answer 503).
type Zarinpal struct {
	MerchantID string // ZARINPAL_MERCHANT_ID (secret: server .env only)
	Sandbox    bool   // ZARINPAL_SANDBOX: sandbox.zarinpal.com instead of payment.zarinpal.com
	BaseURL    string // ZARINPAL_BASE_URL: override of the gateway origin (tests); empty = from Sandbox
}

// AI provider ids accepted by AI_PROVIDER (internal/ai implements them).
const (
	AIProviderNone   = "none"   // no provider: AI features answer 503 ai_unavailable
	AIProviderFake   = "fake"   // deterministic fixtures (dev, tests, stage); never in production
	AIProviderGemini = "gemini" // Google Gemini REST (server-side key)
)

// AI holds the AI adapter settings (B-N3-05, extended by B-N6-05). No secret has a default: the provider key lives
// only in the server's .env and is never logged.
type AI struct {
	// Provider is AI_PROVIDER: none | fake | gemini. Unset → fake outside production, none in production.
	// AI_PROVIDER=fake with APP_ENV=production is refused at start-up (it would invent health data).
	Provider string
	// Timeout (AI_HTTP_TIMEOUT_SECONDS, default 30) bounds every non-streaming provider call and the wait for the
	// response headers of a stream (a streamed chat is bounded by ai.MaxChatDuration and an idle timeout instead).
	Timeout time.Duration
	// DailyCostCapUSD (AI_DAILY_COST_CAP_USD, default 5) is the global budget of one Tehran calendar day: once the
	// estimated cost of the day's calls (ai_usage_logs) reaches it, every AI feature answers 503
	// ai_budget_exhausted until midnight. 0 refuses every call (fail closed); it is never «unlimited» (B-N6-05).
	DailyCostCapUSD float64
	// UserDailyCostCapUSD (AI_USER_DAILY_COST_CAP_USD, default 0.25) is the per-user budget of one Tehran day
	// (B-N6-05b): once one user's calls cost that much, her AI features answer 429 ai_user_budget_exhausted until
	// midnight, so one account cannot spend the global cap alone. 0 refuses every call (fail closed).
	UserDailyCostCapUSD float64
	// Prices (AI_PRICES) is the price table the cost estimate uses: comma-separated
	// `model=input/output[/audio]` in USD per 1M tokens (audio = audio input tokens; default = input). A model
	// missing from the table is priced at the table's highest rates (never free). The fake is always free.
	Prices map[string]AIPrice
	Gemini Gemini
}

// AIPrice is one model's rates in USD per 1M tokens.
type AIPrice struct {
	InputPerMTok  float64
	OutputPerMTok float64
	AudioPerMTok  float64 // audio input tokens (Gemini bills them at a higher rate than text)
}

// DefaultAIPrices is the AI_PRICES default (Gemini paid tier list prices, 2026; check before going live).
const DefaultAIPrices = "gemini-flash-latest=0.30/2.50/1.00,gemini-2.5-flash=0.30/2.50/1.00,gemini-2.5-pro=1.25/10.00/1.25"

// ParseAIPrices parses an AI_PRICES value.
func ParseAIPrices(v string) (map[string]AIPrice, error) {
	out := map[string]AIPrice{}
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		model, rates, ok := strings.Cut(part, "=")
		model = strings.TrimSpace(model)
		if !ok || model == "" {
			return nil, fmt.Errorf("%q is not model=input/output[/audio]", part)
		}
		fields := strings.Split(rates, "/")
		if len(fields) < 2 || len(fields) > 3 {
			return nil, fmt.Errorf("%q is not model=input/output[/audio]", part)
		}
		nums := make([]float64, len(fields))
		for i, f := range fields {
			n, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
			if err != nil || n < 0 || n > 1000 {
				return nil, fmt.Errorf("%q: rate %q is not a number between 0 and 1000", part, f)
			}
			nums[i] = n
		}
		p := AIPrice{InputPerMTok: nums[0], OutputPerMTok: nums[1], AudioPerMTok: nums[0]}
		if len(nums) == 3 {
			p.AudioPerMTok = nums[2]
		}
		out[model] = p
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no model prices")
	}
	return out, nil
}

// Gemini holds GEMINI_* (empty key = provider disabled, AI features answer 503).
type Gemini struct {
	APIKey  string // GEMINI_API_KEY (secret: server .env only)
	Model   string // GEMINI_MODEL (default gemini-flash-latest)
	BaseURL string // GEMINI_BASE_URL: override of the API origin (tests); default generativelanguage.googleapis.com
}

// Companion invite SMS provider ids accepted by COMPANION_SMS_PROVIDER (internal/sms implements them).
const (
	CompanionSMSNone    = "none"    // no SMS: the owner shares the code herself (sms_sent=false)
	CompanionSMSFake    = "fake"    // logs a masked number, sends nothing (dev, tests, stage); never in production
	CompanionSMSGateway = "gateway" // Kavenegar verify/lookup with the invite template (KAVENEGAR_* credentials)
)

// MinCompanionPepperLen is the shortest COMPANION_CODE_PEPPER accepted in production (bytes).
const MinCompanionPepperLen = 32

// Companion holds the «همدم» settings (B-N4-02).
type Companion struct {
	// CodePepper (COMPANION_CODE_PEPPER, secret) keys the HMAC of invite codes at rest. Required in production: when it
	// is empty there, invite creation and acceptance answer 503 (fail closed) instead of hashing with the public dev
	// default. Changing it invalidates the open (≤ 24 h) invites.
	CodePepper string
	// SMSProvider is COMPANION_SMS_PROVIDER: none | fake | gateway. Unset → fake outside production, none in
	// production. fake with APP_ENV=production is refused at start-up (it claims an SMS went out).
	SMSProvider string
	// InviteTemplate (KAVENEGAR_TEMPLATE_COMPANION_INVITE, default companion-invite) is the gateway template whose
	// single token is the invite code.
	InviteTemplate string
	// InviteTemplateNeutral (KAVENEGAR_TEMPLATE_COMPANION_INVITE_NEUTRAL, default empty) is the neutral-wording
	// variant sent when the inviting owner has «اعلان‌های محرمانه» on (CB-PRIV-01); empty = such invites are not sent.
	InviteTemplateNeutral string
}

// Learning holds the courses settings (B-N8-01).
type Learning struct {
	// SMSProvider is LEARNING_SMS_PROVIDER: none | fake | gateway (the CompanionSMS* ids). Unset → fake outside
	// production, none in production. fake with APP_ENV=production is refused at start-up.
	SMSProvider string
	// UnlockTemplate (KAVENEGAR_TEMPLATE_COURSE_UNLOCKED, default course-unlocked) is the gateway template of the
	// «دوره برایت باز شد» SMS; its single token is the number of courses opened.
	UnlockTemplate string
}

// PepperMissing reports whether production runs without a pepper (companion invites are then disabled).
func (c Companion) PepperMissing(app App) bool { return app.IsProduction() && c.CodePepper == "" }

// PrivateNoteKeyLen is the decoded length of PRIVATE_NOTE_KEY (AES-256).
const PrivateNoteKeyLen = 32

// PrivateNotes holds PRIVATE_NOTE_KEY (CB-LOSS-01): the AES-256-GCM key of the private notes stored at rest (the
// loss path's «یادداشت خصوصی»). A secret: base64 of 32 random bytes (openssl rand -base64 32), server .env only.
type PrivateNotes struct {
	// Key is the decoded key; empty → a public development key only when APP_ENV is local or testing, anywhere else
	// the note routes answer 503 (fail closed) instead of encrypting with it.
	Key []byte
	// PreviousKeys (PRIVATE_NOTE_KEY_PREVIOUS, comma-separated base64) still open notes sealed before a rotation;
	// new notes are always sealed with Key.
	PreviousKeys [][]byte
}

// devNoteEnvs are the APP_ENV values in which the public development key may stand in for PRIVATE_NOTE_KEY.
var devNoteEnvs = map[string]bool{"local": true, "testing": true}

// Missing reports whether the private notes are unavailable: no key outside a local / testing environment.
func (p PrivateNotes) Missing(app App) bool { return len(p.Key) == 0 && !devNoteEnvs[app.Env] }

// LabFileKeyLen is the decoded length of LAB_FILE_KEY (AES-256).
const LabFileKeyLen = 32

// LabFiles holds LAB_FILE_KEY (bloom B-N6-06): the AES-256-GCM key of the uploaded lab sheets stored at rest
// (STORAGE_PATH/app/private/labs). A secret: base64 of 32 random bytes (openssl rand -base64 32), server .env only.
type LabFiles struct {
	// Key is the decoded key; empty → a public development key only when APP_ENV is local, testing or contract;
	// anywhere else (stage included) lab uploads and file reads answer 503 (fail closed, B-N6-06b) instead of
	// encrypting with the public key.
	Key []byte
	// PreviousKeys (LAB_FILE_KEY_PREVIOUS, comma-separated base64) still open files sealed before a rotation; new
	// files are always sealed with Key.
	PreviousKeys [][]byte
}

// devLabFileEnvs are the APP_ENV values in which the public development key may stand in for LAB_FILE_KEY.
var devLabFileEnvs = map[string]bool{"local": true, "testing": true, "contract": true}

// Missing reports whether lab files are unavailable: no key outside a local / testing / contract environment.
func (l LabFiles) Missing(app App) bool { return len(l.Key) == 0 && !devLabFileEnvs[app.Env] }

// Files holds FILE_KEY (CB-CORE-05): the AES-256-GCM key of the generic encrypted file storage (internal/files —
// record / claim documents, place photos / licences, product images) and the root of its signed-URL key. A secret:
// base64 of 32 random bytes (openssl rand -base64 32), server .env only. Optional: without it LAB_FILE_KEY is used.
type Files struct {
	// Key is the decoded FILE_KEY (empty → LAB_FILE_KEY, see Resolve).
	Key []byte
	// PreviousKeys (FILE_KEY_PREVIOUS, comma-separated base64) still open files sealed before a rotation.
	PreviousKeys [][]byte
}

// Resolve picks the generic file keys: current = FILE_KEY, else LAB_FILE_KEY; previous = FILE_KEY_PREVIOUS plus
// LAB_FILE_KEY and LAB_FILE_KEY_PREVIOUS (decrypt only — so setting FILE_KEY later keeps the files sealed under
// LAB_FILE_KEY readable). dev = no key at all and APP_ENV local / testing / contract (the caller uses the public
// development key); missing = no key anywhere else (fail closed: the routes answer 503).
func (f Files) Resolve(app App, labs LabFiles) (current []byte, previous [][]byte, dev, missing bool) {
	current = f.Key
	previous = append(previous, f.PreviousKeys...)
	if len(current) == 0 {
		current = labs.Key
	} else if len(labs.Key) > 0 {
		previous = append(previous, labs.Key)
	}
	previous = append(previous, labs.PreviousKeys...)
	if len(current) == 0 {
		dev = devLabFileEnvs[app.Env] // the lab key policy (B-N6-06b): local / testing / contract only
		missing = !dev
	}
	return current, previous, dev, missing
}

// MediaURLKeyLen is the decoded length of MEDIA_URL_KEY.
const MediaURLKeyLen = 32

// Media holds the lesson media pipeline settings (bloom B-N8-02, internal/media, D-81).
type Media struct {
	// URLKey is the decoded MEDIA_URL_KEY (base64 of 32 random bytes, a secret from the server .env only): the HMAC
	// key of the short-lived playback URLs. Empty → a public development key only for APP_ENV local / testing /
	// contract; anywhere else playback answers 503 (fail closed). Uploads do not need it.
	URLKey []byte
	// StoragePath (MEDIA_STORAGE_PATH) is the media root (its own volume in the compose files); empty →
	// STORAGE_PATH/app/private/media.
	StoragePath string
}

// ResolveKey picks the playback URL key: dev = no key and a local / testing / contract environment (the caller uses
// the public development key); missing = no key anywhere else.
func (m Media) ResolveKey(app App) (key []byte, dev, missing bool) {
	if len(m.URLKey) > 0 {
		return m.URLKey, false, false
	}
	dev = devLabFileEnvs[app.Env]
	return nil, dev, !dev
}

// Root is the media root directory ("" when neither MEDIA_STORAGE_PATH nor STORAGE_PATH is set).
func (m Media) Root(storagePath string) string {
	if m.StoragePath != "" {
		return m.StoragePath
	}
	if storagePath == "" {
		return ""
	}
	return strings.TrimRight(storagePath, "/") + "/app/private/media"
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
		Plus: Plus{
			VATRateBps:  e.integer("PLUS_VAT_RATE_BPS", 1000),
			TrialDays:   e.integer("PLUS_TRIAL_DAYS", 7),
			InvoiceTTL:  time.Duration(e.integer("PLUS_INVOICE_TTL_MINUTES", 30)) * time.Minute,
			CallbackURL: e.str("PLUS_CALLBACK_URL", "http://localhost:3000/plus/return"),
		},
		Payment: Payment{
			Provider: strings.ToLower(e.str("PAYMENT_PROVIDER", "")),
			Timeout:  time.Duration(e.integer("PAYMENT_HTTP_TIMEOUT_SECONDS", 15)) * time.Second,
			Zarinpal: Zarinpal{
				MerchantID: e.str("ZARINPAL_MERCHANT_ID", ""),
				Sandbox:    e.boolean("ZARINPAL_SANDBOX", false),
				BaseURL:    strings.TrimRight(e.str("ZARINPAL_BASE_URL", ""), "/"),
			},
		},
		AI: AI{
			Provider:        strings.ToLower(e.str("AI_PROVIDER", "")),
			Timeout:         time.Duration(e.integer("AI_HTTP_TIMEOUT_SECONDS", 30)) * time.Second,
			DailyCostCapUSD: e.float("AI_DAILY_COST_CAP_USD", 5),
			// B-N6-05b: per-user daily cost cap.
			UserDailyCostCapUSD: e.float("AI_USER_DAILY_COST_CAP_USD", 0.25),
			Gemini: Gemini{
				APIKey:  e.str("GEMINI_API_KEY", ""),
				Model:   e.str("GEMINI_MODEL", "gemini-flash-latest"),
				BaseURL: strings.TrimRight(e.str("GEMINI_BASE_URL", ""), "/"),
			},
		},
		Companion: Companion{
			CodePepper:            e.str("COMPANION_CODE_PEPPER", ""),
			SMSProvider:           strings.ToLower(e.str("COMPANION_SMS_PROVIDER", "")),
			InviteTemplate:        e.str("KAVENEGAR_TEMPLATE_COMPANION_INVITE", "companion-invite"),
			InviteTemplateNeutral: e.str("KAVENEGAR_TEMPLATE_COMPANION_INVITE_NEUTRAL", ""),
		},
		Learning: Learning{
			SMSProvider:    strings.ToLower(e.str("LEARNING_SMS_PROVIDER", "")),
			UnlockTemplate: e.str("KAVENEGAR_TEMPLATE_COURSE_UNLOCKED", "course-unlocked"),
		},
		Telemed: Telemed{
			HoldTTL:             time.Duration(e.integer("TELEMED_HOLD_MINUTES", 10)) * time.Minute,
			CallbackURL:         e.str("TELEMED_CALLBACK_URL", ""), // default: PLUS_CALLBACK_URL's origin (below)
			PlusDiscountPercent: e.integer("TELEMED_PLUS_DISCOUNT_PERCENT", 20),
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
	if cfg.Plus.VATRateBps < 0 || cfg.Plus.VATRateBps > 10000 {
		e.fail("PLUS_VAT_RATE_BPS: must be between 0 and 10000")
	}
	if cfg.Plus.TrialDays < 1 || cfg.Plus.TrialDays > 90 {
		e.fail("PLUS_TRIAL_DAYS: must be between 1 and 90")
	}
	if cfg.Plus.InvoiceTTL < time.Minute {
		e.fail("PLUS_INVOICE_TTL_MINUTES: must be at least 1")
	}
	cfg.Payment.Provider = DefaultPaymentProvider(cfg.Payment.Provider, cfg.App)
	cfg.Payment.CallbackBaseURL = strings.TrimRight(e.str("PAYMENT_CALLBACK_BASE_URL", cfg.App.URL), "/")
	cfg.Payment.ReturnURLs = append([]string{cfg.Plus.CallbackURL}, e.list("PAYMENT_RETURN_URLS", nil)...)
	if cfg.Telemed.CallbackURL == "" { // B-N7-03: same web origin as the Plus return page
		if u, err := url.Parse(cfg.Plus.CallbackURL); err == nil && u.Host != "" {
			cfg.Telemed.CallbackURL = u.Scheme + "://" + u.Host + "/services/bookings/return"
		}
	}
	if cfg.Telemed.CallbackURL != "" { // the booking return page is always allowed
		cfg.Payment.ReturnURLs = append(cfg.Payment.ReturnURLs, cfg.Telemed.CallbackURL)
	}
	if cfg.Telemed.HoldTTL < 2*time.Minute || cfg.Telemed.HoldTTL > time.Hour {
		e.fail("TELEMED_HOLD_MINUTES: must be between 2 and 60")
	}
	if cfg.Telemed.PlusDiscountPercent < 0 || cfg.Telemed.PlusDiscountPercent > 90 {
		e.fail("TELEMED_PLUS_DISCOUNT_PERCENT: must be between 0 and 90")
	}
	switch cfg.Payment.Provider {
	case PaymentProviderNone, PaymentProviderZarinpal:
	case PaymentProviderFake:
		if cfg.App.IsProduction() {
			e.fail("PAYMENT_PROVIDER=fake is not allowed when APP_ENV=production (it grants Plus without payment)")
		}
	default:
		e.fail("PAYMENT_PROVIDER: %q is not one of none, fake, zarinpal", cfg.Payment.Provider)
	}
	if cfg.Payment.Timeout < time.Second {
		e.fail("PAYMENT_HTTP_TIMEOUT_SECONDS: must be at least 1")
	}
	if cfg.Payment.Provider != PaymentProviderNone { // the URLs are only used while a provider is active
		for _, u := range append([]string{cfg.Payment.CallbackBaseURL}, cfg.Payment.ReturnURLs...) {
			if err := checkPublicURL(u, cfg.App.IsProduction()); err != nil {
				e.fail("payment URL %q (PAYMENT_CALLBACK_BASE_URL / PLUS_CALLBACK_URL / PAYMENT_RETURN_URLS): %v", u, err)
			}
		}
	}
	cfg.AI.Provider = DefaultAIProvider(cfg.AI.Provider, cfg.App)
	switch cfg.AI.Provider {
	case AIProviderNone, AIProviderGemini:
	case AIProviderFake:
		if cfg.App.IsProduction() {
			e.fail("AI_PROVIDER=fake is not allowed when APP_ENV=production (it returns canned fixtures)")
		}
	default:
		e.fail("AI_PROVIDER: %q is not one of none, fake, gemini", cfg.AI.Provider)
	}
	if cfg.AI.Timeout < time.Second {
		e.fail("AI_HTTP_TIMEOUT_SECONDS: must be at least 1")
	}
	if cfg.AI.DailyCostCapUSD < 0 {
		e.fail("AI_DAILY_COST_CAP_USD: must be 0 or more")
	}
	if cfg.AI.UserDailyCostCapUSD < 0 {
		e.fail("AI_USER_DAILY_COST_CAP_USD: must be 0 or more")
	}
	if prices, err := ParseAIPrices(e.str("AI_PRICES", DefaultAIPrices)); err != nil {
		e.fail("AI_PRICES: %v", err)
	} else {
		cfg.AI.Prices = prices
	}
	if u := cfg.AI.Gemini.BaseURL; u != "" {
		if err := checkPublicURL(u, cfg.App.IsProduction()); err != nil {
			e.fail("GEMINI_BASE_URL %q: %v", u, err)
		}
	}
	cfg.Companion.SMSProvider = DefaultCompanionSMSProvider(cfg.Companion.SMSProvider, cfg.App)
	switch cfg.Companion.SMSProvider {
	case CompanionSMSNone, CompanionSMSGateway:
	case CompanionSMSFake:
		if cfg.App.IsProduction() {
			e.fail("COMPANION_SMS_PROVIDER=fake is not allowed when APP_ENV=production (it reports invites as sent)")
		}
	default:
		e.fail("COMPANION_SMS_PROVIDER: %q is not one of none, fake, gateway", cfg.Companion.SMSProvider)
	}
	cfg.Learning.SMSProvider = DefaultCompanionSMSProvider(cfg.Learning.SMSProvider, cfg.App)
	switch cfg.Learning.SMSProvider {
	case CompanionSMSNone, CompanionSMSGateway:
	case CompanionSMSFake:
		if cfg.App.IsProduction() {
			e.fail("LEARNING_SMS_PROVIDER=fake is not allowed when APP_ENV=production (it reports SMS as sent)")
		}
	default:
		e.fail("LEARNING_SMS_PROVIDER: %q is not one of none, fake, gateway", cfg.Learning.SMSProvider)
	}
	if p := cfg.Companion.CodePepper; p != "" && cfg.App.IsProduction() && len(p) < MinCompanionPepperLen {
		e.fail("COMPANION_CODE_PEPPER: must be at least %d bytes in production", MinCompanionPepperLen)
	}
	if raw := e.str("PRIVATE_NOTE_KEY", ""); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != PrivateNoteKeyLen {
			e.fail("PRIVATE_NOTE_KEY: must be the base64 encoding of %d bytes", PrivateNoteKeyLen)
		} else {
			cfg.PrivateNotes.Key = key
		}
	}
	if raw := e.str("LAB_FILE_KEY", ""); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != LabFileKeyLen {
			e.fail("LAB_FILE_KEY: must be the base64 encoding of %d bytes", LabFileKeyLen)
		} else {
			cfg.LabFiles.Key = key
		}
	}
	for _, raw := range strings.Split(e.str("LAB_FILE_KEY_PREVIOUS", ""), ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != LabFileKeyLen {
			e.fail("LAB_FILE_KEY_PREVIOUS: every entry must be the base64 encoding of %d bytes", LabFileKeyLen)
			continue
		}
		cfg.LabFiles.PreviousKeys = append(cfg.LabFiles.PreviousKeys, key)
	}
	for _, raw := range strings.Split(e.str("PRIVATE_NOTE_KEY_PREVIOUS", ""), ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != PrivateNoteKeyLen {
			e.fail("PRIVATE_NOTE_KEY_PREVIOUS: every entry must be the base64 encoding of %d bytes", PrivateNoteKeyLen)
			continue
		}
		cfg.PrivateNotes.PreviousKeys = append(cfg.PrivateNotes.PreviousKeys, key)
	}
	if raw := e.str("FILE_KEY", ""); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != LabFileKeyLen {
			e.fail("FILE_KEY: must be the base64 encoding of %d bytes", LabFileKeyLen)
		} else {
			cfg.Files.Key = key
		}
	}
	for _, raw := range strings.Split(e.str("FILE_KEY_PREVIOUS", ""), ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != LabFileKeyLen {
			e.fail("FILE_KEY_PREVIOUS: every entry must be the base64 encoding of %d bytes", LabFileKeyLen)
			continue
		}
		cfg.Files.PreviousKeys = append(cfg.Files.PreviousKeys, key)
	}
	if raw := e.str("MEDIA_URL_KEY", ""); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(key) != MediaURLKeyLen {
			e.fail("MEDIA_URL_KEY: must be the base64 encoding of %d bytes", MediaURLKeyLen)
		} else {
			cfg.Media.URLKey = key
		}
	}
	cfg.Media.StoragePath = e.str("MEDIA_STORAGE_PATH", "")
	cfg.Sharing = loadSharing(e, cfg.App)
	if cfg.App.IsProduction() && cfg.App.Debug {
		e.fail("APP_DEBUG must be false when APP_ENV=production")
	}

	if len(e.errs) > 0 {
		return nil, fmt.Errorf("config: %w", errors.Join(e.errs...))
	}
	return cfg, nil
}

// DefaultPaymentProvider resolves an unset PAYMENT_PROVIDER: the fake outside production, none in production (so a
// production deploy without a configured gateway keeps answering 503 instead of granting Plus for free).
func DefaultPaymentProvider(provider string, app App) string {
	if provider != "" {
		return provider
	}
	if app.IsProduction() {
		return PaymentProviderNone
	}
	return PaymentProviderFake
}

// DefaultAIProvider resolves an unset AI_PROVIDER: the fake outside production, none in production.
func DefaultAIProvider(provider string, app App) string {
	if provider != "" {
		return provider
	}
	if app.IsProduction() {
		return AIProviderNone
	}
	return AIProviderFake
}

// DefaultCompanionSMSProvider resolves an unset COMPANION_SMS_PROVIDER: the fake outside production, none in production.
func DefaultCompanionSMSProvider(provider string, app App) string {
	if provider != "" {
		return provider
	}
	if app.IsProduction() {
		return CompanionSMSNone
	}
	return CompanionSMSFake
}

// checkPublicURL accepts an absolute http(s) URL with a host and no credentials; https only in production.
func checkPublicURL(raw string, production bool) error {
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return err
	case u.Scheme != "https" && (production || u.Scheme != "http"):
		return errors.New("must be an absolute https URL (http allowed outside production)")
	case u.Host == "" || u.User != nil:
		return errors.New("must have a host and no user info")
	}
	return nil
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

func (e *env) float(key string, def float64) float64 {
	v := e.raw(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		e.fail("%s: %q is not a number", key, v)
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
