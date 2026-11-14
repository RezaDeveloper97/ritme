package credits

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Handlers serve the user's credit screens: GET /api/v1/ai/credits and GET /api/v1/ai/credits/usage. They only
// read — spending happens in the AI gate (internal/ai/access), buying in internal/plus (credit packs).
type Handlers struct {
	svc     *Service
	catalog *Catalog
	clock   clock.Clock
	// external: calls leave the server (a real provider). The in-process fake answers for every slot, so a slot
	// without a model is still available while it is false.
	external bool
}

// NewHandlers wires the handlers; base is the fallback clock (X-Test-Now pins it per request in tests).
func NewHandlers(svc *Service, catalog *Catalog, external bool, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, catalog: catalog, external: external, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// titles resolves feature keys to their localized titles (the key itself for a slot without a title).
type titles struct {
	byKey    map[ai.Feature]Feature
	locale   string
	fallback string
}

func (h *Handlers) titles(c fiber.Ctx) (titles, []Feature, error) {
	feats, err := h.catalog.Features(c)
	if err != nil {
		return titles{}, nil, err
	}
	t := titles{byKey: make(map[ai.Feature]Feature, len(feats)), locale: i18n.Locale(c), fallback: i18n.LanguagesOf(c).DefaultCode()}
	for _, f := range feats {
		t.byKey[f.Key] = f
	}
	return t, feats, nil
}

func (t titles) of(key string) string {
	if f, ok := t.byKey[ai.Feature(key)]; ok {
		if s := i18n.PickString(f.Title, t.locale, t.fallback); s != "" {
			return s
		}
	}
	return key
}

// BalanceJSON is the balance object of GET /ai/credits (and of a 402's context on the clients).
func BalanceJSON(b Balance) *jsonx.OrderedMap {
	var period, ends any
	if b.Active {
		period, ends = b.Period, jsonx.ISO8601(b.PeriodEndsAt)
	}
	return jsonx.Obj(
		"can_use", b.Active,
		"period", period,
		"period_ends_at", ends,
		"plan", jsonx.Obj("total", b.PlanTotal, "used", b.PlanTotal-b.PlanRemaining, "remaining", b.PlanRemaining),
		// Extra credits never expire; `locked` = they are kept but cannot be spent until a subscription runs again.
		"extra", jsonx.Obj("remaining", b.Extra, "locked", !b.Active && b.Extra > 0),
		"spendable", b.Spendable(),
	)
}

// Show is GET /ai/credits: the balance and what one use of each feature costs.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	b, err := h.svc.Balance(c, userID, h.now(c))
	if err != nil {
		return err
	}
	t, feats, err := h.titles(c)
	if err != nil {
		return err
	}
	list := make([]any, 0, len(feats))
	for _, f := range feats {
		if _, declared := ai.SpecOf(f.Key); !declared {
			continue // a slot no code uses any more
		}
		available := f.Active && (len(f.Models) > 0 || !h.external)
		list = append(list, jsonx.Obj("key", string(f.Key), "title", t.of(string(f.Key)), "credit_cost", f.Cost,
			"available", available, "affordable", available && b.Active && b.Spendable() >= f.Cost))
	}
	return httpx.OK(c, BalanceJSON(b).Set("features", list))
}

// UsageDays is the window of the usage summary.
const UsageDays = 30

// Usage is GET /ai/credits/usage?before=<id>: the last UsageDays days by feature and by day, and a page of the
// history (newest first; `next` is the `before` of the following page, null at the end).
func (h *Handlers) Usage(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	before, _ := strconv.ParseUint(c.Query("before"), 10, 64) // absent / unreadable = the first page
	now := h.now(c)
	since := civildate.FromTime(now).AddDays(-(UsageDays - 1)).TehranMidnight()
	byFeature, byDay, err := h.svc.Summary(c, userID, since)
	if err != nil {
		return err
	}
	items, next, err := h.svc.History(c, userID, before)
	if err != nil {
		return err
	}
	t, _, err := h.titles(c)
	if err != nil {
		return err
	}
	var total int64
	feats := make([]any, len(byFeature))
	for i, f := range byFeature {
		total += f.Credits
		feats[i] = jsonx.Obj("feature", f.Feature, "title", t.of(f.Feature), "uses", f.Uses, "credits", f.Credits)
	}
	days := make([]any, len(byDay))
	for i, d := range byDay {
		days[i] = jsonx.Obj("date", d.Day, "credits", d.Credits)
	}
	history := make([]any, len(items))
	for i, s := range items {
		history[i] = jsonx.Obj("id", s.ID, "feature", s.Feature, "title", t.of(s.Feature), "credits", s.Credits, "at", jsonx.ISO8601(s.At))
	}
	var cursor any
	if next > 0 {
		cursor = next
	}
	return httpx.OK(c, jsonx.Obj(
		"days", UsageDays,
		"since", jsonx.ISO8601(since),
		"total", total,
		"by_feature", feats,
		"by_day", days,
		"history", jsonx.Obj("items", history, "next", cursor),
	))
}
