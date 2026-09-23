// Package pregnancy is the admin API of the pregnancy v2 content (T-M7-06,
// docs/pregnancy-v2/README.md § What the admin defines, docs/go-migration/admin-api.md §13):
//
//   - /pregnancy-week-details and /pregnancy-weeks/:n/details — the structured week page
//     (pregnancy_week_details, one row per week 1–42, upserted);
//   - /pregnancy-care-items — the care plan (CRUD, reorder, toggle; delete refused while
//     appointments reference the item's key);
//   - /pregnancy-alert-rules — the alert rules: behaviour (enabled, level, window, typed params)
//     and per-locale texts, persisted as message_contents rows of group pregnancy_alert.
//
// Everything is editor + super (kit.Admin); mutations need the CSRF token and are audited.
// The user API reads the same rows without a cache, so an edit is live on the next request.
package pregnancy

import (
	"database/sql"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve the pregnancy v2 admin endpoints.
type Handlers struct {
	db     *sql.DB
	q      *store.Queries
	logger *slog.Logger
}

// New builds the handlers.
func New(db *sql.DB, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{db: db, q: store.New(db), logger: logger}
}

// Routes registers the endpoints (editor and super admins); static paths before params.
// The week details live under /pregnancy-weeks/:n/details next to the v1 text editor
// (/pregnancy-weeks/:id, a row id); the list and options use their own prefix so they never
// collide with that :id route.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	a := kit.Admin
	route(fiber.MethodGet, "/pregnancy-week-details", a(h.ListWeekDetails))
	route(fiber.MethodGet, "/pregnancy-week-details/options", a(h.WeekDetailsOptions))
	route(fiber.MethodGet, "/pregnancy-weeks/:n/details", a(h.ShowWeekDetails))
	route(fiber.MethodPut, "/pregnancy-weeks/:n/details", a(h.UpdateWeekDetails))

	route(fiber.MethodGet, "/pregnancy-care-items", a(h.ListCareItems))
	route(fiber.MethodGet, "/pregnancy-care-items/options", a(h.CareItemOptions))
	route(fiber.MethodPost, "/pregnancy-care-items/reorder", a(h.ReorderCareItems))
	route(fiber.MethodPost, "/pregnancy-care-items", a(h.StoreCareItem))
	route(fiber.MethodGet, "/pregnancy-care-items/:id", a(h.ShowCareItem))
	route(fiber.MethodPut, "/pregnancy-care-items/:id", a(h.UpdateCareItem))
	route(fiber.MethodDelete, "/pregnancy-care-items/:id", a(h.DestroyCareItem))
	route(fiber.MethodPost, "/pregnancy-care-items/:id/toggle", a(h.ToggleCareItem))

	route(fiber.MethodGet, "/pregnancy-alert-rules", a(h.ListAlertRules))
	route(fiber.MethodGet, "/pregnancy-alert-rules/options", a(h.AlertRuleOptions))
	route(fiber.MethodGet, "/pregnancy-alert-rules/:key", a(h.ShowAlertRule))
	route(fiber.MethodPut, "/pregnancy-alert-rules/:key", a(h.UpdateAlertRule))
}

// ---------------------------------------------------------------------------
// Shared helpers

// keyPattern: lowercase snake case starting with a letter (keys are stored in user data).
const keyPattern = "/^[a-z][a-z0-9_]*$/"

// sent reports whether the raw input carries key with a non-null value.
func sent(in phpval.Map, key string) bool {
	v, ok := phpval.Get(in, key)
	return ok && v != nil
}

// translatable is a {code: text} field. When required (or sent, for an optional one) every
// active language needs text — the content is shown in each of them (as §12 checkups).
func translatable(field string, required bool, codes []string, maxLen int) validation.Rules {
	p := "nullable"
	if required {
		p = "required"
	}
	rules := validation.Rules{validation.F(field, p+"|array")}
	for _, code := range codes {
		rules = append(rules, validation.F(field+"."+code, p+"|string|max:"+strconv.Itoa(maxLen)))
	}
	return rules
}

// translatedObj is {code: text} over the active languages that have text (nil when none).
func translatedObj(v any, codes []string) *jsonx.OrderedMap {
	kv := make([]any, 0, 2*len(codes))
	for _, code := range codes {
		t, ok := phpval.Get(v, code)
		if !ok || t == nil {
			continue
		}
		if s := phpval.ToString(t); s != "" {
			kv = append(kv, code, s)
		}
	}
	if len(kv) == 0 {
		return nil
	}
	return jsonx.Obj(kv...)
}

func nullJSON(v any) db.NullRawJSON {
	switch x := v.(type) {
	case nil:
		return db.NullRawJSON{}
	case *jsonx.OrderedMap:
		if x == nil {
			return db.NullRawJSON{}
		}
	}
	return db.NullRawJSON{V: form.JSON(v), Valid: true}
}

// listOrEmpty decodes a JSON array column; NULL is [] so admin-web can always map it.
func listOrEmpty(col db.NullRawJSON) any {
	v := form.NullRaw(col)
	if v == nil {
		return []any{}
	}
	return v
}

// distinctKeys adds a validation.distinct error for a repeated `key` in a list of objects.
func distinctKeys(c fiber.Ctx, field string) form.Check {
	return func(in phpval.Map, add form.Add) error {
		v, _ := phpval.Get(in, field)
		if !phpval.IsArray(v) {
			return nil
		}
		keys, items := phpval.Entries(v)
		seen := map[string]bool{}
		for i, item := range items {
			k, _ := phpval.Get(item, "key")
			s, isStr := k.(string)
			if !isStr || s == "" {
				continue
			}
			if seen[s] {
				f := field + "." + keys[i] + ".key"
				add(f, form.Msg(c, "validation.distinct", f))
			}
			seen[s] = true
		}
		return nil
	}
}
