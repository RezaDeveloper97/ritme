package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/store"
)

// Store is the persistence the settings endpoints need (profile store, scoped by user_id in SQL).
type Store interface {
	Getter
	UpsertNotificationPreferences(ctx context.Context, arg store.UpsertNotificationPreferencesParams) error
}

// Handlers are GET/PUT /api/v1/profile/notification-settings. Mount behind locale + auth RequireUser.
type Handlers struct {
	q     Store
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(q Store, base clock.Clock) *Handlers { return &Handlers{q: q, clock: base} }

func userID(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// SettingsJSON is the response body: groups in screen order, quiet hours "HH:MM", neutral copy.
func SettingsJSON(p Preferences) *jsonx.OrderedMap {
	groups := make([]any, 0, len(Groups))
	for _, g := range Groups {
		items := make([]any, 0, len(g.Categories))
		for _, c := range g.Categories {
			items = append(items, jsonx.Obj("code", string(c), "enabled", p.Enabled(c)))
		}
		groups = append(groups, jsonx.Obj("code", g.Code, "items", items))
	}
	return jsonx.Obj(
		"groups", groups,
		"quiet_hours", jsonx.Obj(
			"enabled", p.QuietEnabled,
			"start", FormatClock(p.QuietStart),
			"end", FormatClock(p.QuietEnd),
		),
		"neutral_copy", p.NeutralCopy,
	)
}

// Show is GET /profile/notification-settings (defaults when the user never saved).
func (h *Handlers) Show(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	p, err := Load(c, h.q, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, SettingsJSON(p))
}

// updateRules validate a partial PUT body; keys outside these are ignored.
func updateRules() validation.Rules {
	F := validation.F
	rules := validation.Rules{
		F("categories", "sometimes", "array"),
		F("quiet_hours", "sometimes", "array"),
		F("quiet_hours.enabled", "sometimes", "boolean"),
		F("quiet_hours.start", "sometimes", "date_format:H:i"),
		F("quiet_hours.end", "sometimes", "date_format:H:i"),
		F("neutral_copy", "sometimes", "boolean"),
	}
	for _, g := range Groups {
		for _, cat := range g.Categories {
			rules = append(rules, F("categories."+string(cat), "sometimes", "boolean"))
		}
	}
	return rules
}

// Apply validates body against p and returns the updated preferences (controller-style 422 on failure).
func Apply(p Preferences, body phpval.Map, locale string, now time.Time) (Preferences, error) {
	v := validation.Make(lang.Default(), locale, body, updateRules(),
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return p, httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", v.ErrorBag())
	}
	out := p
	out.Categories = make(map[Category]bool, len(p.Categories))
	for k, val := range p.Categories {
		out.Categories[k] = val
	}
	for _, g := range Groups {
		for _, cat := range g.Categories {
			if val, ok := phpval.Get(body, "categories."+string(cat)); ok {
				out.Categories[cat] = phpval.Truthy(val)
			}
		}
	}
	if val, ok := phpval.Get(body, "quiet_hours.enabled"); ok {
		out.QuietEnabled = phpval.Truthy(val)
	}
	if val, ok := phpval.Get(body, "quiet_hours.start"); ok {
		if m, ok := ParseClock(phpval.ToString(val)); ok {
			out.QuietStart = m
		}
	}
	if val, ok := phpval.Get(body, "quiet_hours.end"); ok {
		if m, ok := ParseClock(phpval.ToString(val)); ok {
			out.QuietEnd = m
		}
	}
	if val, ok := phpval.Get(body, "neutral_copy"); ok {
		out.NeutralCopy = phpval.Truthy(val)
	}
	return out, nil
}

// Update is PUT /profile/notification-settings: a partial update (omitted keys keep their value), then the settings.
func (h *Handlers) Update(c fiber.Ctx) error {
	id, err := userID(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	now := clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
	cur, err := Load(c, h.q, id)
	if err != nil {
		return err
	}
	next, err := Apply(cur, validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.q.UpsertNotificationPreferences(c, store.UpsertNotificationPreferencesParams{
		UserID:            id,
		Categories:        db.NullRawJSON{V: next.CategoriesJSON(), Valid: true},
		QuietHoursEnabled: next.QuietEnabled,
		QuietStart:        dbClock(next.QuietStart),
		QuietEnd:          dbClock(next.QuietEnd),
		NeutralCopy:       next.NeutralCopy,
		Now:               sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return fmt.Errorf("notifications: save: %w", err)
	}
	return httpx.OK(c, SettingsJSON(next), T("messages.saved", locale))
}
