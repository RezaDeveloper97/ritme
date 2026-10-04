package usage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/ai/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// MaxRangeDays bounds the admin report window.
const MaxRangeDays = 92

// Route registers one admin route (internal/http routes_admin_ai.go).
type Route func(method, path string, chain httpadmin.Chain)

// AdminHandlers serve GET /api/admin/v1/ai/usage (any active admin: aggregates only, no user ids, no content).
type AdminHandlers struct {
	q      Store
	budget *Budget
	clock  clock.Clock
}

// NewAdminHandlers wires the handlers; budget reports today's spend against the cap.
func NewAdminHandlers(db *sql.DB, budget *Budget, c clock.Clock) *AdminHandlers {
	return NewAdminHandlersWith(store.New(db), budget, c)
}

// NewAdminHandlersWith wires the handlers on a store (tests).
func NewAdminHandlersWith(q Store, budget *Budget, c clock.Clock) *AdminHandlers {
	if c == nil {
		c = clock.Real{}
	}
	return &AdminHandlers{q: q, budget: budget, clock: c}
}

// Routes registers the routes.
func (h *AdminHandlers) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/ai/usage", kit.Admin(h.Usage))
}

func usd(micros int64) float64 { return float64(micros) / 1e6 }

// window is [from, to] inclusive Tehran days from ?from=&to= (default: the last 30 days up to today).
func (h *AdminHandlers) window(c fiber.Ctx) (civildate.Date, civildate.Date, error) {
	today := civildate.InTehran(clock.FromContext(c, h.clock).Now())
	to, from := today, today.AddDays(-29)
	if s := c.Query("to"); s != "" {
		d, err := civildate.Parse(s)
		if err != nil {
			return from, to, httpadmin.FieldError("to", "The to field must be a date in the format Y-m-d.")
		}
		to = d
		if c.Query("from") == "" {
			from = to.AddDays(-29)
		}
	}
	if s := c.Query("from"); s != "" {
		d, err := civildate.Parse(s)
		if err != nil {
			return from, to, httpadmin.FieldError("from", "The from field must be a date in the format Y-m-d.")
		}
		from = d
	}
	if from.After(to) {
		return from, to, httpadmin.FieldError("from", "The from field must be a date before or equal to to.")
	}
	if from.DiffDays(to) >= MaxRangeDays {
		return from, to, httpadmin.FieldError("from", fmt.Sprintf("The range may cover at most %d days.", MaxRangeDays))
	}
	return from, to, nil
}

func dayString(v any) string {
	switch d := v.(type) {
	case string:
		return d
	case []byte:
		return string(d)
	case time.Time:
		return d.Format("2006-01-02")
	}
	return ""
}

// Usage is GET /ai/usage?from=YYYY-MM-DD&to=YYYY-MM-DD: today's spend against the cap, totals, per day and per
// feature / provider / model in the window.
func (h *AdminHandlers) Usage(c fiber.Ctx) error {
	from, to, err := h.window(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	fromAt := sql.NullTime{Time: from.TehranMidnight(), Valid: true}
	toAt := sql.NullTime{Time: to.AddDays(1).TehranMidnight(), Valid: true}
	days, err := h.q.UsageByDay(ctx, store.UsageByDayParams{FromAt: fromAt, ToAt: toAt})
	if err != nil {
		return fmt.Errorf("ai usage: by day: %w", err)
	}
	feats, err := h.q.UsageByFeature(ctx, store.UsageByFeatureParams{FromAt: fromAt, ToAt: toAt})
	if err != nil {
		return fmt.Errorf("ai usage: by feature: %w", err)
	}
	var calls, okCalls, in, out, cost int64
	byDay := make([]any, len(days))
	for i, d := range days {
		calls, okCalls, in, out, cost = calls+d.Calls, okCalls+d.OkCalls, in+d.InputTokens, out+d.OutputTokens, cost+d.CostMicros
		byDay[i] = jsonx.Obj("day", dayString(d.Day), "calls", d.Calls, "ok_calls", d.OkCalls,
			"input_tokens", d.InputTokens, "output_tokens", d.OutputTokens, "cost_micros", d.CostMicros, "cost_usd", usd(d.CostMicros))
	}
	byFeature := make([]any, len(feats))
	for i, f := range feats {
		byFeature[i] = jsonx.Obj("feature", f.Feature, "provider", f.Provider, "model", f.Model, "calls", f.Calls,
			"ok_calls", f.OkCalls, "users", f.Users, "input_tokens", f.InputTokens, "output_tokens", f.OutputTokens,
			"audio_bytes", f.AudioBytes, "image_bytes", f.ImageBytes, "avg_latency_ms", f.AvgLatencyMs,
			"cost_micros", f.CostMicros, "cost_usd", usd(f.CostMicros))
	}
	today := jsonx.Obj("spent_micros", nil, "spent_usd", nil, "cap_micros", nil, "cap_usd", nil, "exhausted", nil)
	if h.budget != nil {
		spent, err := h.budget.Spent(ctx)
		if err != nil {
			return err
		}
		capM := h.budget.CapMicros()
		today = jsonx.Obj("spent_micros", spent, "spent_usd", float64(spent)/1e6, "cap_micros", capM,
			"cap_usd", float64(capM)/1e6, "exhausted", spent >= capM)
	}
	return httpadmin.OK(c, jsonx.Obj(
		"from", from.String(), "to", to.String(),
		"today", today,
		"totals", jsonx.Obj("calls", calls, "ok_calls", okCalls, "input_tokens", in, "output_tokens", out,
			"cost_micros", cost, "cost_usd", usd(cost)),
		"by_day", byDay,
		"by_feature", byFeature,
	))
}
