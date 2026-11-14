package aicredits

import (
	"database/sql"
	"errors"
	"log/slog"
	"slices"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Feature slot limits.
const (
	MaxCreditCost = 100_000
	maxSortOrder  = 100_000
)

// needsOf is what a slot's model must be able to do (nil when the key is no longer declared in code).
func needsOf(key string) ([]ai.Capability, bool) {
	spec, ok := ai.SpecOf(ai.Feature(key))
	if !ok {
		return nil, false
	}
	return spec.Needs, true
}

func modelRef(id sql.NullInt64, byID map[uint64]*store.AiModel) any {
	if !id.Valid {
		return nil
	}
	m, ok := byID[uint64(id.Int64)] //nolint:gosec // G115: ids are positive
	if !ok {
		return nil
	}
	return jsonx.Obj("id", m.ID, "code", m.Code, "title", m.Title, "provider", m.Provider, "is_active", m.IsActive)
}

// usableModel: the slot may route to m (active and capable of what the slot needs).
func usableModel(m *store.AiModel, needs []ai.Capability) bool { return m.IsActive && serves(m, needs) }

func featureJSON(f *store.AiFeature, models []store.AiModel, byID map[uint64]*store.AiModel) *jsonx.OrderedMap {
	needs, declared := needsOf(f.Key)
	usableIDs := []uint64{}
	for i := range models {
		if usableModel(&models[i], needs) {
			usableIDs = append(usableIDs, models[i].ID)
		}
	}
	routes := 0 // the models the slot routes to right now (primary + fallback that are usable)
	for _, id := range []sql.NullInt64{f.ModelID, f.FallbackModelID} {
		if !id.Valid {
			continue
		}
		if m, ok := byID[uint64(id.Int64)]; ok && usableModel(m, needs) { //nolint:gosec // G115: ids are positive
			routes++
		}
	}
	return jsonx.Obj(
		"id", f.ID,
		"key", f.Key,
		"title", form.Raw(f.Title),
		"needs", capStrings(needs),
		"declared", declared,
		"model_id", nullID(f.ModelID),
		"fallback_model_id", nullID(f.FallbackModelID),
		"model", modelRef(f.ModelID, byID),
		"fallback_model", modelRef(f.FallbackModelID, byID),
		"credit_cost", f.CreditCost,
		"is_active", f.IsActive,
		"sort_order", f.SortOrder,
		// usable: what the gate sees — switched on, declared, and at least one assigned model can serve it.
		"usable", f.IsActive && declared && routes > 0,
		"routes", routes,
		"usable_model_ids", usableIDs,
		"created_at", httpadmin.Time(f.CreatedAt),
		"updated_at", httpadmin.Time(f.UpdatedAt),
	)
}

func (h *Handlers) modelIndex(c fiber.Ctx) ([]store.AiModel, map[uint64]*store.AiModel, error) {
	rows, err := h.q.AdminListAiModels(c.Context())
	if err != nil {
		return nil, nil, err
	}
	models := make([]store.AiModel, len(rows))
	byID := make(map[uint64]*store.AiModel, len(rows))
	for i := range rows {
		models[i] = rows[i].AiModel
	}
	for i := range models {
		byID[models[i].ID] = &models[i]
	}
	return models, byID, nil
}

// ListFeatures is GET /ai/features: every slot (display order; not paginated — a few dozen to ~100 rows) with what
// it needs, whether it is still declared in code, its models, and the ids of every model that could serve it; plus
// the short model list the panel's selects use.
func (h *Handlers) ListFeatures(c fiber.Ctx) error {
	feats, err := h.q.AdminListAiFeatures(c.Context())
	if err != nil {
		return err
	}
	models, byID, err := h.modelIndex(c)
	if err != nil {
		return err
	}
	configured := h.configured()
	items := make([]*jsonx.OrderedMap, 0, len(feats))
	for i := range feats {
		items = append(items, featureJSON(&feats[i], models, byID))
	}
	opts := make([]*jsonx.OrderedMap, 0, len(models))
	for i := range models {
		m := &models[i]
		opts = append(opts, jsonx.Obj("id", m.ID, "code", m.Code, "title", m.Title, "provider", m.Provider,
			"upstream_model", m.UpstreamModel, "capabilities", capStrings(capabilities(m)), "is_active", m.IsActive,
			"configured", slices.Contains(configured, m.Provider)))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", jsonx.List(items),
		"models", jsonx.List(opts),
		"capabilities", capStrings(ai.Capabilities()),
		"limits", jsonx.Obj("max_credit_cost", MaxCreditCost),
	))
}

// optID reads a nullable id field: (value, sent). Absent = keep; null = clear.
func optID(data phpval.Map, key string, cur sql.NullInt64) sql.NullInt64 {
	v, sent := data.Get(key)
	if !sent {
		return cur
	}
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(phpval.ToFloat(v)), Valid: true}
}

// UpdateFeature is PUT /ai/features/:id (super): title (translatable; absent = unchanged), model_id /
// fallback_model_id (absent = unchanged, null = none), credit_cost (0–100000), is_active, sort_order. A model must
// exist (422 validation_failed), be active (422 ai_model_inactive) and have every capability the slot needs (422
// ai_model_lacks_capability, `missing`); the fallback must differ from the model (422 ai_fallback_same_as_model).
// Slots are created from code, so there is no POST / DELETE.
func (h *Handlers) UpdateFeature(c fiber.Ctx) error {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return httpadmin.NotFound("Feature")
	}
	cur, err := h.q.AdminGetAiFeature(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.NotFound("Feature")
	}
	if err != nil {
		return err
	}
	rules := form.Translatable(c, "title", false, "max:128")
	rules = append(rules,
		validation.F("model_id", "nullable|integer|min:1"),
		validation.F("fallback_model_id", "nullable|integer|min:1"),
		validation.F("credit_cost", "required|integer|min:0|max:"+strconv.Itoa(MaxCreditCost)),
		validation.F("is_active", "nullable|boolean"),
		validation.F("sort_order", "nullable|integer|min:0|max:"+strconv.Itoa(maxSortOrder)),
	)
	data, err := form.Validate(c, rules)
	if err != nil {
		return err
	}
	p := store.AdminUpdateAiFeatureParams{
		Title:           cur.Title,
		ModelID:         optID(data, "model_id", cur.ModelID),
		FallbackModelID: optID(data, "fallback_model_id", cur.FallbackModelID),
		CreditCost:      uint32(form.Int(data, "credit_cost", 0)), //nolint:gosec // G115: validated 0–100000
		IsActive:        boolOr(data, "is_active", cur.IsActive),
		SortOrder:       cur.SortOrder,
		Now:             httpadmin.DBTime(httpadmin.Now(c)),
		ID:              cur.ID,
	}
	if _, sent := data.Get("title"); sent {
		p.Title = []byte("{}")
		if t := form.Clean(data, "title"); t.Valid {
			p.Title = t.V
		}
	}
	if form.Has(data, "sort_order") {
		p.SortOrder = uint32(form.Int(data, "sort_order", 0)) //nolint:gosec // G115: validated ≤ maxSortOrder
	}
	needs, _ := needsOf(cur.Key)
	if err := h.checkSlotModel(c, "model_id", p.ModelID, needs); err != nil {
		return err
	}
	if err := h.checkSlotModel(c, "fallback_model_id", p.FallbackModelID, needs); err != nil {
		return err
	}
	if p.ModelID.Valid && p.FallbackModelID.Valid && p.ModelID.Int64 == p.FallbackModelID.Int64 {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeFallbackSameAsModel,
			"The fallback model must differ from the model.", "field", "fallback_model_id")
	}
	if err := h.q.AdminUpdateAiFeature(c.Context(), p); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "ai_feature.update", "ai_feature", cur.ID, slog.String("key", cur.Key),
		slog.Int64("credit_cost", int64(p.CreditCost)), slog.Bool("is_active", p.IsActive),
		slog.Int64("model_id", p.ModelID.Int64), slog.Int64("fallback_model_id", p.FallbackModelID.Int64))
	feat, err := h.q.AdminGetAiFeature(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	models, byID, err := h.modelIndex(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("feature", featureJSON(&feat, models, byID)), "Feature updated.")
}

// checkSlotModel validates one model assignment of a slot.
func (h *Handlers) checkSlotModel(c fiber.Ctx, field string, id sql.NullInt64, needs []ai.Capability) error {
	if !id.Valid {
		return nil
	}
	m, err := h.q.AdminGetAiModel(c.Context(), uint64(id.Int64)) //nolint:gosec // G115: validated ≥ 1
	if errors.Is(err, sql.ErrNoRows) {
		return httpadmin.FieldError(field, form.Msg(c, "validation.exists", field))
	}
	if err != nil {
		return err
	}
	if !m.IsActive {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeModelInactive,
			"The selected model is switched off.", "field", field)
	}
	have := capabilities(&m)
	missing := []string{}
	for _, n := range needs {
		if !slices.Contains(have, n) {
			missing = append(missing, string(n))
		}
	}
	if len(missing) > 0 {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeModelLacksCapability,
			"The selected model cannot do what this feature needs.", "field", field, "missing", missing)
	}
	return nil
}
