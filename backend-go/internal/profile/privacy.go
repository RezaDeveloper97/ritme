package profile

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/store"
)

// Consent codes of the Privacy screen (B-N1-12, `nbl_Me_Privacy` «داده‌ها و هوش مصنوعی»), in screen order.
// Every consent is opt-in: no row, or a row with granted = 0, means «no».
const (
	ConsentAILabAnalysis    = "ai_lab_analysis"   // lab results may be read by the AI analysis (B-N6)
	ConsentAssistantProfile = "assistant_profile" // the health assistant may use age, mode and medications (B-N7)
	ConsentAnonymousStats   = "anonymous_stats"   // anonymous, aggregated usage statistics
)

// ConsentCodes is the screen order of the consents.
var ConsentCodes = []string{ConsentAILabAnalysis, ConsentAssistantProfile, ConsentAnonymousStats}

// The privacy/support copy (controller messages, validation attribute names) is data: lang/<code>/privacy.json,
// read through the platform translator. A language without the file (or a key) falls back to English.
//
//go:embed lang/*/*.json
var privacyLangFS embed.FS

var privacyTranslator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(privacyLangFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// PT is the privacy line for key ("messages.consents_saved") in locale.
func PT(key, locale string) string { return privacyTranslator().Trans("privacy."+key, nil, locale) }

// privacyAttributes are the validation attribute names for locale, as flat "key", "name" pairs.
func privacyAttributes(locale string) []string {
	line, ok := privacyTranslator().Get("privacy.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// PrivacyStore is the persistence of the consent and support endpoints (profile store; user_id in every query).
type PrivacyStore interface {
	ListUserConsents(ctx context.Context, userID uint64) ([]store.ListUserConsentsRow, error)
	GrantUserConsent(ctx context.Context, arg store.GrantUserConsentParams) error
	RevokeUserConsent(ctx context.Context, arg store.RevokeUserConsentParams) error
	CreateSupportReport(ctx context.Context, arg store.CreateSupportReportParams) (int64, error)
	CountRecentSupportReports(ctx context.Context, arg store.CountRecentSupportReportsParams) (int64, error)
}

// PrivacyHandlers are the Go-only B-N1-12 endpoints: GET/PUT /profile/consents and POST /support/reports.
// Mount behind locale + auth RequireUser.
type PrivacyHandlers struct {
	q           PrivacyStore
	clock       clock.Clock
	storagePath string // STORAGE_PATH; "" = screenshots are refused (the report itself is still stored)
	logger      *slog.Logger
}

// PrivacyOptions wires PrivacyHandlers.
type PrivacyOptions struct {
	Store       PrivacyStore
	Clock       clock.Clock // fallback when the request carries no test clock
	StoragePath string
	Logger      *slog.Logger
}

// NewPrivacyHandlers builds the handlers.
func NewPrivacyHandlers(o PrivacyOptions) *PrivacyHandlers {
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	return &PrivacyHandlers{q: o.Store, clock: o.Clock, storagePath: o.StoragePath, logger: o.Logger}
}

func (h *PrivacyHandlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func privacyUserID(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func isoOrNull(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.ISO8601(t.Time)
}

// consentsJSON is {consents: [{code, granted, granted_at, revoked_at}]} in screen order; unknown stored codes
// (a consent retired later) are not listed.
func consentsJSON(rows []store.ListUserConsentsRow) *jsonx.OrderedMap {
	byCode := make(map[string]store.ListUserConsentsRow, len(rows))
	for _, r := range rows {
		byCode[r.Consent] = r
	}
	list := make([]any, 0, len(ConsentCodes))
	for _, code := range ConsentCodes {
		r, ok := byCode[code]
		list = append(list, jsonx.Obj(
			"code", code,
			"granted", ok && r.Granted,
			"granted_at", isoOrNull(r.GrantedAt),
			"revoked_at", isoOrNull(r.RevokedAt),
		))
	}
	return jsonx.Obj("consents", list)
}

func (h *PrivacyHandlers) loadConsents(ctx context.Context, userID uint64) (*jsonx.OrderedMap, error) {
	rows, err := h.q.ListUserConsents(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile: consents: %w", err)
	}
	return consentsJSON(rows), nil
}

// Consents is GET /profile/consents.
func (h *PrivacyHandlers) Consents(c fiber.Ctx) error {
	id, err := privacyUserID(c)
	if err != nil {
		return err
	}
	out, err := h.loadConsents(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, out)
}

func consentRules() validation.Rules {
	rules := validation.Rules{validation.F("consents", "required", "array")}
	for _, code := range ConsentCodes {
		rules = append(rules, validation.F("consents."+code, "sometimes", "boolean"))
	}
	return rules
}

// UpdateConsents is PUT /profile/consents {consents: {<code>: bool, …}}: a partial update (omitted codes keep their
// answer, unknown codes are ignored), each change stamped with the time; then the consents.
func (h *PrivacyHandlers) UpdateConsents(c fiber.Ctx) error {
	id, err := privacyUserID(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, consentRules(),
		validation.Now(now), validation.Attributes(privacyAttributes(locale)...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, PT("messages.validation_failed", locale), "errors", v.ErrorBag())
	}
	at := sql.NullTime{Time: now, Valid: true}
	for _, code := range ConsentCodes {
		val, ok := phpval.Get(body, "consents."+code)
		if !ok {
			continue
		}
		if phpval.Truthy(val) {
			err = h.q.GrantUserConsent(c, store.GrantUserConsentParams{UserID: id, Consent: code, Now: at})
		} else {
			err = h.q.RevokeUserConsent(c, store.RevokeUserConsentParams{UserID: id, Consent: code, Now: at})
		}
		if err != nil {
			return fmt.Errorf("profile: save consent %s: %w", code, err)
		}
	}
	out, err := h.loadConsents(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, out, PT("messages.consents_saved", locale))
}

type privacyExport struct {
	consents, reports, notifications any
}

// exportPrivacy is the B-N1-12 part of GET /profile/export: every consent row with its timestamps, the support
// reports (text, status, date, whether a screenshot was attached — never the file path) and the effective
// notification settings.
func exportPrivacy(ctx context.Context, q *store.Queries, userID uint64) (privacyExport, error) {
	consentRows, err := q.ExportUserConsents(ctx, userID)
	if err != nil {
		return privacyExport{}, fmt.Errorf("profile: export consents: %w", err)
	}
	consents := make([]any, len(consentRows))
	for i, r := range consentRows {
		consents[i] = jsonx.Obj(
			"code", r.Consent,
			"granted", r.Granted,
			"granted_at", isoOrNull(r.GrantedAt),
			"revoked_at", isoOrNull(r.RevokedAt),
			"created_at", isoOrNull(r.CreatedAt),
			"updated_at", isoOrNull(r.UpdatedAt),
		)
	}
	reportRows, err := q.ExportSupportReports(ctx, userID)
	if err != nil {
		return privacyExport{}, fmt.Errorf("profile: export support reports: %w", err)
	}
	reports := make([]any, len(reportRows))
	for i, r := range reportRows {
		reports[i] = jsonx.Obj(
			"id", r.ID,
			"message", r.Message,
			"status", r.Status,
			"has_screenshot", r.HasScreenshot == 1,
			"created_at", isoOrNull(r.CreatedAt),
		)
	}
	prefs, err := notifications.Load(ctx, q, userID)
	if err != nil {
		return privacyExport{}, fmt.Errorf("profile: export notification settings: %w", err)
	}
	return privacyExport{consents: consents, reports: reports, notifications: notifications.SettingsJSON(prefs)}, nil
}
