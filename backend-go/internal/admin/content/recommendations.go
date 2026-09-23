package content

import (
	"context"
	"database/sql"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// PhaseGeneral is the ?phase= value for recommendations without a phase.
const PhaseGeneral = "general"

func recommendationJSON(r *store.Recommendation) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"key", httpadmin.NullString(r.Key),
		"type", r.Type,
		"title", form.NullRaw(r.Title),
		"text", form.Raw(r.Text),
		"cycle_phase", httpadmin.NullString(r.CyclePhase),
		"cycle_subphases", form.NullRaw(r.CycleSubphases),
		"symptom_trigger", httpadmin.NullString(r.SymptomTrigger),
		"is_active", r.IsActive,
		"sort_order", r.SortOrder,
		"created_at", httpadmin.Time(r.CreatedAt),
		"updated_at", httpadmin.Time(r.UpdatedAt),
	)
}

func (h *Handlers) recommendations() resource[store.Recommendation] {
	return resource[store.Recommendation]{
		name: "Recommendation", key: "recommendation", get: h.q.GetRecommendation, json: recommendationJSON,
		id: func(r *store.Recommendation) uint64 { return r.ID },
		toggle: func(ctx context.Context, now sql.NullTime, id uint64) error {
			return h.q.ToggleRecommendation(ctx, store.ToggleRecommendationParams{Now: now, ID: id})
		},
		del: h.q.DeleteRecommendation,
	}
}

// ListRecommendations is GET /recommendations?phase=general|<phase>&type=<type>
// (phase-less rows last, then phase, sort_order, id).
func (h *Handlers) ListRecommendations(c fiber.Ctx) error {
	phase, typ := c.Query("phase"), c.Query("type")
	phasePattern := form.Exact(phase)
	if phase == PhaseGeneral {
		phasePattern = "" // IFNULL(cycle_phase, '') LIKE '' ⇔ cycle_phase IS NULL
	}
	pp := sql.NullString{String: phasePattern, Valid: true}
	total, err := h.q.CountAdminRecommendations(c.Context(), store.CountAdminRecommendationsParams{
		PhasePattern: pp, TypePattern: form.Exact(typ),
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminRecommendations(c.Context(), store.ListAdminRecommendationsParams{
		PhasePattern: pp, TypePattern: form.Exact(typ),
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, recommendationJSON(&rows[i]))
	}
	nullable := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("phase", nullable(phase), "type", nullable(typ)))
	return httpadmin.OK(c, page)
}

// RecommendationOptions is GET /recommendations/options: phases, the sub-phases a day can
// report (with the phase each belongs to, for the picker), types and symptom triggers.
func (h *Handlers) RecommendationOptions(c fiber.Ctx) error {
	loc := locale(c)
	subs := []enums.Option{}
	for _, s := range enums.CyclePhaseAllSubphases() {
		subs = append(subs, enums.Option{Value: string(s), Label: s.Label(loc)})
	}
	subPhase := jsonx.NewObject()
	for _, p := range enums.CyclePhaseCases() {
		for _, s := range p.Subphases() {
			subPhase.Set(string(s), string(p))
		}
	}
	return httpadmin.OK(c, jsonx.Obj(
		"phases", options(enums.CyclePhaseOptions(loc)),
		"subphases", subs,
		"subphase_phases", subPhase,
		"types", options(enums.RecommendationTypeOptions(loc)),
		"triggers", options(enums.RecommendationTriggerOptions(loc)),
	))
}

// ShowRecommendation is GET /recommendations/:id.
func (h *Handlers) ShowRecommendation(c fiber.Ctx) error { return h.recommendations().show(c) }

// recommendationRules: cycle_subphases.* is limited to the sub-phases the chosen phase
// can reach (every emittable one when no phase is chosen).
func recommendationRules(c fiber.Ctx) validation.Rules {
	phase := ""
	if v, ok := validation.Input(c).Get("cycle_phase"); ok && v != nil {
		phase = phpval.ToString(v)
	}
	rules := validation.Rules{validation.F("type", "required", validation.In(enums.RecommendationTypeValues()...))}
	rules = append(rules, form.Translatable(c, "title", false, "max:255")...)
	rules = append(rules, form.Translatable(c, "text", true, "max:2000")...)
	return append(rules,
		validation.F("cycle_phase", "nullable", validation.In(enums.CyclePhaseValues()...)),
		validation.F("cycle_subphases", "nullable|array"),
		validation.F("cycle_subphases.*", validation.In(enums.CyclePhaseSubphaseValuesFor(phase)...)),
		validation.F("symptom_trigger", "nullable", validation.In(enums.RecommendationTriggerValues()...)),
		validation.F("sort_order", "nullable|integer"),
		validation.F("is_active", "nullable"),
	)
}

// subphasesOf is `array_values($data['cycle_subphases'] ?? []) ?: null`.
func subphasesOf(data phpval.Map) db.NullRawJSON {
	v, _ := data.Get("cycle_subphases")
	_, vals := phpval.Entries(v)
	if len(vals) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(vals), Valid: true}
}

func recommendationParams(data phpval.Map) store.CreateRecommendationParams {
	return store.CreateRecommendationParams{
		Type: httpadmin.String(data, "type"), Title: form.Clean(data, "title"), Text: form.ReqJSON(data, "text"),
		CyclePhase: form.Str(data, "cycle_phase"), SymptomTrigger: form.Str(data, "symptom_trigger"),
		CycleSubphases: subphasesOf(data),
		IsActive:       httpadmin.Bool(data, "is_active"), SortOrder: form.Int32(data, "sort_order", 0),
	}
}

// StoreRecommendation is POST /recommendations.
func (h *Handlers) StoreRecommendation(c fiber.Ctx) error {
	data, err := form.Validate(c, recommendationRules(c))
	if err != nil {
		return err
	}
	p := recommendationParams(data)
	p.Now = h.now(c)
	res, err := h.q.CreateRecommendation(c.Context(), p)
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "recommendation.create", "recommendation", id)
	return h.recommendations().respond(c, id, true, "Recommendation created.")
}

// UpdateRecommendation is PUT /recommendations/:id.
func (h *Handlers) UpdateRecommendation(c fiber.Ctx) error {
	cur, err := find(c, "Recommendation", h.q.GetRecommendation)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, recommendationRules(c))
	if err != nil {
		return err
	}
	p := recommendationParams(data)
	if err := h.q.UpdateRecommendation(c.Context(), store.UpdateRecommendationParams{
		Type: p.Type, Title: p.Title, Text: p.Text,
		CyclePhase:     form.KeepStr(data, "cycle_phase", cur.CyclePhase),
		CycleSubphases: p.CycleSubphases,
		SymptomTrigger: form.KeepStr(data, "symptom_trigger", cur.SymptomTrigger),
		IsActive:       p.IsActive, SortOrder: p.SortOrder, Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "recommendation.update", "recommendation", cur.ID)
	return h.recommendations().respond(c, cur.ID, false, "Recommendation updated.")
}

// DestroyRecommendation is DELETE /recommendations/:id.
func (h *Handlers) DestroyRecommendation(c fiber.Ctx) error {
	return h.recommendations().destroy(c, h.logger, "Recommendation deleted.")
}

// ToggleRecommendation is POST /recommendations/:id/toggle (is_active).
func (h *Handlers) ToggleRecommendation(c fiber.Ctx) error {
	return h.recommendations().doToggle(c, h.logger, "Status changed.")
}
