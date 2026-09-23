package content

import (
	"context"
	"database/sql"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

func affirmationJSON(a *store.Affirmation) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", a.ID,
		"text", form.Raw(a.Text),
		"cycle_phase", httpadmin.NullString(a.CyclePhase),
		"is_active", a.IsActive,
		"sort_order", a.SortOrder,
		"created_at", httpadmin.Time(a.CreatedAt),
		"updated_at", httpadmin.Time(a.UpdatedAt),
	)
}

func (h *Handlers) affirmations() resource[store.Affirmation] {
	return resource[store.Affirmation]{
		name: "Affirmation", key: "affirmation", get: h.q.GetAffirmation, json: affirmationJSON,
		id: func(a *store.Affirmation) uint64 { return a.ID },
		toggle: func(ctx context.Context, now sql.NullTime, id uint64) error {
			return h.q.ToggleAffirmation(ctx, store.ToggleAffirmationParams{Now: now, ID: id})
		},
		del: h.q.DeleteAffirmation,
	}
}

// ListAffirmations is GET /affirmations (sort_order, newest first).
func (h *Handlers) ListAffirmations(c fiber.Ctx) error {
	total, err := h.q.CountAdminAffirmations(c.Context())
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminAffirmations(c.Context(), store.ListAdminAffirmationsParams{
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, affirmationJSON(&rows[i]))
	}
	return httpadmin.OK(c, httpadmin.Page(items, p, int(total)))
}

// AffirmationOptions is GET /affirmations/options.
func (h *Handlers) AffirmationOptions(c fiber.Ctx) error {
	return httpadmin.OK(c, jsonx.Obj("phases", options(enums.CyclePhaseOptions(locale(c)))))
}

// ShowAffirmation is GET /affirmations/:id.
func (h *Handlers) ShowAffirmation(c fiber.Ctx) error { return h.affirmations().show(c) }

func affirmationRules(c fiber.Ctx) validation.Rules {
	rules := form.Translatable(c, "text", true)
	return append(rules,
		validation.F("cycle_phase", "nullable", validation.In(enums.CyclePhaseValues()...)),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
}

// StoreAffirmation is POST /affirmations {text{…}, cycle_phase?, sort_order?, is_active?}.
func (h *Handlers) StoreAffirmation(c fiber.Ctx) error {
	data, err := form.Validate(c, affirmationRules(c))
	if err != nil {
		return err
	}
	res, err := h.q.CreateAffirmation(c.Context(), store.CreateAffirmationParams{
		Text: form.ReqJSON(data, "text"), CyclePhase: form.Str(data, "cycle_phase"),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0), Now: h.now(c),
	})
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "affirmation.create", "affirmation", id)
	return h.affirmations().respond(c, id, true, "Affirmation created.")
}

// UpdateAffirmation is PUT /affirmations/:id.
func (h *Handlers) UpdateAffirmation(c fiber.Ctx) error {
	cur, err := find(c, "Affirmation", h.q.GetAffirmation)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, affirmationRules(c))
	if err != nil {
		return err
	}
	if err := h.q.UpdateAffirmation(c.Context(), store.UpdateAffirmationParams{
		Text: form.ReqJSON(data, "text"), CyclePhase: form.KeepStr(data, "cycle_phase", cur.CyclePhase),
		IsActive: httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0),
		Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "affirmation.update", "affirmation", cur.ID)
	return h.affirmations().respond(c, cur.ID, false, "Affirmation updated.")
}

// DestroyAffirmation is DELETE /affirmations/:id.
func (h *Handlers) DestroyAffirmation(c fiber.Ctx) error {
	return h.affirmations().destroy(c, h.logger, "Affirmation deleted.")
}

// ToggleAffirmation is POST /affirmations/:id/toggle (is_active).
func (h *Handlers) ToggleAffirmation(c fiber.Ctx) error {
	return h.affirmations().doToggle(c, h.logger, "Status changed.")
}
