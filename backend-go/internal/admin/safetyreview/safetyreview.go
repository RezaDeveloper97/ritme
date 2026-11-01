// Package safetyreview is /safety of the admin API «ایمنی و محتوای بالینی» (bloom B-N9-10, D-90): aggregated urgent
// signals, the unified clinical review queue with claim / release, and the urgent-path health checks. Capability
// «clinical» by path. Responses carry counts, titles, ids and ages only — never a user id or a health value; approve /
// reject stay on each source's own screen (the queue deep-links there).
package safetyreview

import (
	"database/sql"
	"log/slog"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/safety"
)

// Error codes.
const (
	CodeAlreadyClaimed = "already_claimed"
	CodeNotClaimer     = "not_claimer"
)

// Handlers serve /safety.
type Handlers struct {
	svc    *safety.Service
	logger *slog.Logger
}

// New wires the handlers.
func New(db *sql.DB, logger *slog.Logger) *Handlers {
	return &Handlers{svc: safety.New(db), logger: logger}
}

// Route registers an admin route.
type Route func(method, path string, chain httpadmin.Chain)

// Routes mounts the module.
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/safety", kit.Admin(h.Show))
	route(fiber.MethodPost, "/safety/queue/:source/:id/claim", kit.Admin(h.Claim))
	route(fiber.MethodPost, "/safety/queue/:source/:id/release", kit.Admin(h.Release))
}

func at(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return httpadmin.Time(sql.NullTime{Time: t, Valid: true})
}

func (h *Handlers) queue(c fiber.Ctx, now time.Time) ([]safety.Item, error) {
	return h.svc.Queue(c.Context(), now, i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode())
}

// Show is GET /safety: {window_days, signals[], queue{pending, by_source, sla_hours, items[]}, checks{…}}.
func (h *Handlers) Show(c fiber.Ctx) error {
	now := httpadmin.Now(c)
	sigs, err := h.svc.Signals(c.Context(), now)
	if err != nil {
		return err
	}
	signals := make([]any, 0, len(sigs))
	for _, s := range sigs {
		signals = append(signals, jsonx.Obj("key", s.Key, "count", s.Count, "previous", s.Previous, "delta", s.Count-s.Previous))
	}
	items, err := h.queue(c, now)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	rows := make([]any, 0, len(items))
	overdue := 0
	for _, it := range items {
		counts[it.Source] += it.Items
		age := it.AgeHours(now)
		if age > safety.SLAHours {
			overdue++
		}
		var claim any
		if it.Claim != nil {
			claim = jsonx.Obj("id", it.Claim.AdminID, "name", it.Claim.Name, "at", at(it.Claim.At))
		}
		rows = append(rows, jsonx.Obj(
			"source", it.Source, "id", it.ID, "title", it.Title, "ref", it.Ref, "status", it.Status,
			"approvals", it.Approvals, "required_approvals", it.Required, "items", it.Items,
			"submitted_by", nullable(it.SubmittedBy), "submitted_at", at(it.SubmittedAt),
			"age_hours", age, "overdue", age > safety.SLAHours, "claimed_by", claim))
	}
	bySource := jsonx.Obj()
	for _, src := range safety.Sources {
		bySource.Set(src, counts[src])
	}
	unapproved, err := h.svc.UnapprovedUrgentCopy(c.Context())
	if err != nil {
		return err
	}
	var scenarios any
	last, err := h.svc.LatestRun(c.Context()) // T-M10-24: the persisted safety.nightly run (any process)
	if err != nil {
		return err
	}
	if last != nil {
		scenarios = jsonx.Obj("ran_at", at(last.RanAt), "passed", last.Passed, "total", last.Total, "failed", nonNil(last.Failed))
	}
	checks := jsonx.Obj(
		// Not instrumented: an analytics event would tie a user id to an urgent health event (D-90).
		"modal_latency_p95_ms", nil,
		"call_115_ctr", nil,
		"text_parity", jsonx.Obj("ok", unapproved == 0, "unapproved", unapproved, "groups", safety.UrgentCopyGroups),
		"scenarios", scenarios,
	)
	return httpadmin.OK(c, jsonx.Obj(
		"window_days", safety.WindowDays,
		"today", civildate.InTehran(now).String(),
		"signals", signals,
		"queue", jsonx.Obj("pending", len(items), "overdue", overdue, "sla_hours", safety.SLAHours,
			"by_source", bySource, "items", rows),
		"checks", checks,
	))
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// target resolves :source/:id to a current queue item (404 when it is not in the queue any more).
func (h *Handlers) target(c fiber.Ctx, now time.Time) (*safety.Item, error) {
	src := c.Params("source")
	id, ok := httpadmin.ID(c, "id")
	if !ok || !slices.Contains(safety.Sources, src) {
		return nil, httpadmin.NotFound("review item")
	}
	items, err := h.queue(c, now)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Source == src && items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, httpadmin.NotFound("review item")
}

// Claim is POST /safety/queue/:source/:id/claim: the current admin takes the item (409 when another admin holds it).
func (h *Handlers) Claim(c fiber.Ctx) error {
	now := httpadmin.Now(c)
	it, err := h.target(c, now)
	if err != nil {
		return err
	}
	me := httpadmin.CurrentAdmin(c)
	if it.Claim != nil && it.Claim.AdminID != me.ID {
		return httpadmin.Fail(fiber.StatusConflict, CodeAlreadyClaimed, "Another admin is reviewing this item.")
	}
	if it.Claim == nil {
		httpadmin.Audit(c, h.logger, safety.ActionClaim, safety.ClaimTarget(it.Source), it.ID)
	}
	return httpadmin.OK(c, jsonx.Obj("claimed_by", jsonx.Obj("id", me.ID, "name", me.Name, "at", at(now))), "Claimed.")
}

// Release is POST /safety/queue/:source/:id/release: the claimer (or a super admin) hands the item back.
func (h *Handlers) Release(c fiber.Ctx) error {
	now := httpadmin.Now(c)
	it, err := h.target(c, now)
	if err != nil {
		return err
	}
	me := httpadmin.CurrentAdmin(c)
	if it.Claim != nil {
		if it.Claim.AdminID != me.ID && me.Role != httpadmin.RoleSuper {
			return httpadmin.Fail(fiber.StatusForbidden, CodeNotClaimer, "Only the admin who claimed this item can release it.")
		}
		httpadmin.Audit(c, h.logger, safety.ActionRelease, safety.ClaimTarget(it.Source), it.ID)
	}
	return httpadmin.OK(c, jsonx.Obj("claimed_by", nil), "Released.")
}
