// Package overview is GET /overview of the admin API: the latest metrics snapshot for the dashboard (B-N9-03, D-85).
// Aggregates only; small cohorts are already suppressed in the snapshot.
package overview

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/metrics"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Handlers serve the overview.
type Handlers struct {
	svc    *metrics.Service
	logger *slog.Logger
	okr    func(ctx context.Context, now time.Time) (any, error)
}

// WithOKR fills the overview's `okr` block (B-N9-09) from fn at read time (the snapshot stores null there: the OKR
// progress is live, not a daily value).
func (h *Handlers) WithOKR(fn func(ctx context.Context, now time.Time) (any, error)) *Handlers {
	h.okr = fn
	return h
}

// New wires the handlers.
func New(svc *metrics.Service, logger *slog.Logger) *Handlers {
	return &Handlers{svc: svc, logger: logger}
}

// Route registers an admin route.
type Route func(method, path string, chain httpadmin.Chain)

// Routes mounts the module (capability «dashboard» by path).
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/overview", kit.Admin(h.Show))
	route(fiber.MethodGet, "/kpi", kit.Admin(h.KPI)) // B-N9-08
}

// Show is GET /overview: {day, computed_at, metrics} of the newest snapshot, `metrics` null before the first run.
func (h *Handlers) Show(c fiber.Ctx) error {
	sn, err := h.svc.Latest(c.Context())
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.OK(c, jsonx.Obj("day", nil, "computed_at", nil, "metrics", nil))
	}
	if err != nil {
		return err
	}
	payload := sn.Payload
	if h.okr != nil {
		sum, err := h.okr(c.Context(), httpadmin.Now(c))
		if err != nil {
			return err
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(payload, &m) == nil {
			b, _ := json.Marshal(sum)
			m["okr"] = b
			if merged, err := json.Marshal(m); err == nil {
				payload = merged
			}
		}
	}
	return httpadmin.OK(c, jsonx.Obj("day", sn.Day, "computed_at", httpadmin.Time(sql.NullTime{Time: sn.ComputedAt, Valid: true}), "metrics", payload))
}

// KPI is GET /kpi?days=30|60|90&event=any|log_saved (B-N9-08, D-89): the activation funnel, weekly signup cohorts with
// D0-D60 retention, engine accuracy and the AI indicators, computed on demand from the analytics events. Aggregates
// only, groups under k = 10 suppressed.
func (h *Handlers) KPI(c fiber.Ctx) error {
	o := metrics.KPIOptions{Days: 90, Event: "any"}
	if v := c.Query("days"); v != "" {
		d := 0
		for _, ch := range v {
			if ch < '0' || ch > '9' || d > 1000 {
				d = -1
				break
			}
			d = d*10 + int(ch-'0')
		}
		if !slices.Contains(metrics.FunnelDays, d) {
			return httpadmin.Invalid(invalid("days", "days must be 30, 60 or 90"))
		}
		o.Days = d
	}
	if v := c.Query("event"); v != "" {
		if v != "any" && v != "log_saved" {
			return httpadmin.Invalid(invalid("event", "event must be any or log_saved"))
		}
		o.Event = v
	}
	k, err := h.svc.KPI(c.Context(), time.Now(), o)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, k)
}

func invalid(field, msg string) *httpx.ValidationError {
	ve := httpx.NewValidationError()
	ve.Add(field, msg)
	return ve
}
