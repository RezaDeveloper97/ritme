package aicredits

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Model form limits.
const (
	// MaxPriceMilli bounds a price in milli-USD per 1M tokens (= $100,000 per 1M tokens).
	MaxPriceMilli = 100_000_000
	maxCodeLen    = 64
	maxUpstream   = 128
	maxTitle      = 128
)

const modelCodePattern = "/^[a-z0-9][a-z0-9_-]*$/"

// capabilities decodes a model's capabilities column (an unreadable column = none).
func capabilities(m *store.AiModel) []ai.Capability {
	var caps []ai.Capability
	_ = json.Unmarshal(m.Capabilities, &caps)
	return caps
}

// serves reports whether m has every capability in needs.
func serves(m *store.AiModel, needs []ai.Capability) bool {
	caps := capabilities(m)
	for _, n := range needs {
		if !slices.Contains(caps, n) {
			return false
		}
	}
	return true
}

func capStrings(caps []ai.Capability) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = string(c)
	}
	return out
}

func (h *Handlers) configured() []string {
	if h.gateway == nil {
		return []string{}
	}
	if p := h.gateway.ConfiguredProviders(); p != nil {
		return p
	}
	return []string{}
}

func modelJSON(m *store.AiModel, configured []string, slots int64) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", m.ID,
		"code", m.Code,
		"provider", m.Provider,
		"upstream_model", m.UpstreamModel,
		"title", m.Title,
		"capabilities", capStrings(capabilities(m)),
		"input_price_milli", m.InputPriceMilli,
		"output_price_milli", m.OutputPriceMilli,
		"audio_price_milli", m.AudioPriceMilli,
		"is_active", m.IsActive,
		"configured", slices.Contains(configured, m.Provider),
		"slots_count", slots,
		"created_at", httpadmin.Time(m.CreatedAt),
		"updated_at", httpadmin.Time(m.UpdatedAt),
	)
}

// ListModels is GET /ai/models: the whole catalogue (it is small; not paginated), with whether each model's provider
// adapter is configured on this server (its key is set) and how many feature slots use it.
func (h *Handlers) ListModels(c fiber.Ctx) error {
	rows, err := h.q.AdminListAiModels(c.Context())
	if err != nil {
		return err
	}
	configured := h.configured()
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, modelJSON(&rows[i].AiModel, configured, rows[i].SlotsCount))
	}
	return httpadmin.OK(c, jsonx.Obj(
		"items", jsonx.List(items),
		"providers", ai.Providers(),
		"configured_providers", configured,
		"capabilities", capStrings(ai.Capabilities()),
		"limits", jsonx.Obj("max_price_milli", MaxPriceMilli),
	))
}

func (h *Handlers) findModel(c fiber.Ctx) (store.AiModel, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.AiModel{}, httpadmin.NotFound("Model")
	}
	m, err := h.q.AdminGetAiModel(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return m, httpadmin.NotFound("Model")
	}
	return m, err
}

func (h *Handlers) respondModel(c fiber.Ctx, id uint64, created bool, msg string) error {
	m, err := h.q.AdminGetAiModel(c.Context(), id)
	if err != nil {
		return err
	}
	slots, err := h.q.AdminAiModelSlots(c.Context(), store.AdminAiModelSlotsParams{ID: nullU64(id)})
	if err != nil {
		return err
	}
	body := jsonx.Obj("model", modelJSON(&m, h.configured(), int64(len(slots))), "slots", slots)
	var msgs []string
	if msg != "" {
		msgs = []string{msg}
	}
	if created {
		return httpadmin.Created(c, body, msgs...)
	}
	return httpadmin.OK(c, body, msgs...)
}

// ShowModel is GET /ai/models/:id: the model and the keys of the slots that use it.
func (h *Handlers) ShowModel(c fiber.Ctx) error {
	m, err := h.findModel(c)
	if err != nil {
		return err
	}
	return h.respondModel(c, m.ID, false, "")
}

func modelRules(create bool) validation.Rules {
	price := "nullable|integer|min:0|max:" + strconv.Itoa(MaxPriceMilli)
	caps := capStrings(ai.Capabilities())
	var r validation.Rules
	if create {
		r = append(r, validation.F("code", "required|string|max:"+strconv.Itoa(maxCodeLen), validation.Regex(modelCodePattern)))
	}
	return append(r,
		validation.F("provider", "required|string", validation.In(ai.Providers()...)),
		validation.F("upstream_model", "required|string|max:"+strconv.Itoa(maxUpstream)),
		validation.F("title", "required|string|max:"+strconv.Itoa(maxTitle)),
		validation.F("capabilities", "required|array|min:1"),
		validation.F("capabilities.*", "required|string", validation.In(caps...)),
		validation.F("input_price_milli", price),
		validation.F("output_price_milli", price),
		validation.F("audio_price_milli", price),
		validation.F("is_active", "nullable|boolean"),
	)
}

// uniqueness: the code (create) and the (provider, upstream_model) pair are unique.
func (h *Handlers) modelChecks(c fiber.Ctx, exceptID uint64, create bool) form.Check {
	return func(in phpval.Map, add form.Add) error {
		if create {
			if code, ok := in.Get("code"); ok {
				if s, isStr := code.(string); isStr && s != "" {
					taken, err := h.q.AdminAiModelCodeTaken(c.Context(), store.AdminAiModelCodeTakenParams{Code: s, ExceptID: 0})
					if err != nil {
						return err
					}
					if taken {
						add("code", form.Msg(c, "validation.unique", "code"))
					}
				}
			}
		}
		prov, _ := in.Get("provider")
		up, _ := in.Get("upstream_model")
		ps, ok1 := prov.(string)
		us, ok2 := up.(string)
		if ok1 && ok2 && ps != "" && us != "" {
			taken, err := h.q.AdminAiModelUpstreamTaken(c.Context(), store.AdminAiModelUpstreamTakenParams{
				Provider: ps, UpstreamModel: strings.TrimSpace(us), ExceptID: exceptID})
			if err != nil {
				return err
			}
			if taken {
				add("upstream_model", form.Msg(c, "validation.unique", "upstream_model"))
			}
		}
		return nil
	}
}

// capsOf is the validated capability list, deduplicated, in the canonical order of ai.Capabilities().
func capsOf(data phpval.Map) []ai.Capability {
	raw, _ := data.Get("capabilities")
	_, vals := phpval.Entries(raw)
	sent := map[string]bool{}
	for _, v := range vals {
		sent[phpval.ToString(v)] = true
	}
	out := []ai.Capability{}
	for _, c := range ai.Capabilities() {
		if sent[string(c)] {
			out = append(out, c)
		}
	}
	return out
}

func priceOr(data phpval.Map, key string, def uint32) uint32 {
	if !form.Has(data, key) {
		return def
	}
	v, _ := data.Get(key)
	return uint32(max(0, phpval.ToFloat(v))) //nolint:gosec // G115: validated ≤ MaxPriceMilli
}

func boolOr(data phpval.Map, key string, def bool) bool {
	if !form.Has(data, key) {
		return def
	}
	return httpadmin.Bool(data, key)
}

// StoreModel is POST /ai/models (super).
func (h *Handlers) StoreModel(c fiber.Ctx) error {
	data, err := form.Validate(c, modelRules(true), h.modelChecks(c, 0, true))
	if err != nil {
		return err
	}
	p := store.AdminCreateAiModelParams{
		Code: httpadmin.String(data, "code"), Provider: httpadmin.String(data, "provider"),
		UpstreamModel:   strings.TrimSpace(httpadmin.String(data, "upstream_model")),
		Title:           strings.TrimSpace(httpadmin.String(data, "title")),
		Capabilities:    form.JSON(capsOf(data)),
		InputPriceMilli: priceOr(data, "input_price_milli", 0), OutputPriceMilli: priceOr(data, "output_price_milli", 0),
		AudioPriceMilli: priceOr(data, "audio_price_milli", 0),
		IsActive:        boolOr(data, "is_active", true), Now: httpadmin.DBTime(httpadmin.Now(c)),
	}
	lastID, err := h.q.AdminCreateAiModel(c.Context(), p)
	if err != nil {
		return err
	}
	id := uint64(lastID) //nolint:gosec // G115: auto-increment ids are positive
	httpadmin.Audit(c, h.logger, "ai_model.create", "ai_model", id,
		slog.String("code", p.Code), slog.String("provider", p.Provider))
	return h.respondModel(c, id, true, "Model created.")
}

// UpdateModel is PUT /ai/models/:id (super). The code is fixed at creation. Removing a capability that a slot using
// the model needs is refused (422 ai_capability_in_use, `slots`); switching the model off is allowed (the slots fall
// back to their other model or stop) — the admin panel confirms it first.
func (h *Handlers) UpdateModel(c fiber.Ctx) error {
	cur, err := h.findModel(c)
	if err != nil {
		return err
	}
	data, err := form.Validate(c, modelRules(false), h.modelChecks(c, cur.ID, false))
	if err != nil {
		return err
	}
	caps := capsOf(data)
	if err := h.checkStillServes(c, cur.ID, caps); err != nil {
		return err
	}
	p := store.AdminUpdateAiModelParams{
		Provider: httpadmin.String(data, "provider"), UpstreamModel: strings.TrimSpace(httpadmin.String(data, "upstream_model")),
		Title: strings.TrimSpace(httpadmin.String(data, "title")), Capabilities: form.JSON(caps),
		InputPriceMilli:  priceOr(data, "input_price_milli", cur.InputPriceMilli),
		OutputPriceMilli: priceOr(data, "output_price_milli", cur.OutputPriceMilli),
		AudioPriceMilli:  priceOr(data, "audio_price_milli", cur.AudioPriceMilli),
		IsActive:         boolOr(data, "is_active", cur.IsActive), Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}
	if err := h.q.AdminUpdateAiModel(c.Context(), p); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "ai_model.update", "ai_model", cur.ID, slog.Bool("is_active", p.IsActive))
	return h.respondModel(c, cur.ID, false, "Model updated.")
}

// checkStillServes refuses a capability set that no longer covers a declared slot assigned to the model.
func (h *Handlers) checkStillServes(c fiber.Ctx, modelID uint64, caps []ai.Capability) error {
	slots, err := h.q.AdminAiModelSlots(c.Context(), store.AdminAiModelSlotsParams{ID: nullU64(modelID)})
	if err != nil {
		return err
	}
	var broken []string
	var missing []string
	for _, key := range slots {
		spec, ok := ai.SpecOf(ai.Feature(key))
		if !ok {
			continue
		}
		lacks := false
		for _, n := range spec.Needs {
			if !slices.Contains(caps, n) {
				lacks = true
				if !slices.Contains(missing, string(n)) {
					missing = append(missing, string(n))
				}
			}
		}
		if lacks {
			broken = append(broken, key)
		}
	}
	if len(broken) == 0 {
		return nil
	}
	return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeCapabilityStillNeeded,
		"Feature slots using this model need the capabilities being removed.", "slots", broken, "missing", missing)
}

// DestroyModel is DELETE /ai/models/:id (super). Refused with 422 ai_model_in_use while a feature slot points at the model
// (`slots`: their keys) — assign another model first, or switch it off with PUT. Usage logs keep the provider and
// model names as text, so a deleted model leaves the history intact.
func (h *Handlers) DestroyModel(c fiber.Ctx) error {
	cur, err := h.findModel(c)
	if err != nil {
		return err
	}
	n, err := h.q.AdminDeleteUnusedAiModel(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		slots, err := h.q.AdminAiModelSlots(c.Context(), store.AdminAiModelSlotsParams{ID: nullU64(cur.ID)})
		if err != nil {
			return err
		}
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse,
			"Feature slots still use this model; assign another model first.", "slots_count", len(slots), "slots", slots)
	}
	httpadmin.Audit(c, h.logger, "ai_model.delete", "ai_model", cur.ID, slog.String("code", cur.Code))
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID, "deleted", true), "Model deleted.")
}

// SyncModels is POST /ai/models/sync {provider} (super): asks the provider's gateway which models it serves and adds
// the ones the catalogue lacks as INACTIVE rows (capabilities ["chat"], zero prices) for the admin to review, price
// and switch on. 422 ai_provider_not_configured when this server has no key for the provider, 422
// ai_provider_unavailable when the gateway cannot list its models (add them by hand then).
func (h *Handlers) SyncModels(c fiber.Ctx) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("provider", "required|string", validation.In(ai.Providers()...)),
	})
	if err != nil {
		return err
	}
	provider := httpadmin.String(data, "provider")
	if !slices.Contains(h.configured(), provider) {
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeProviderNotConfigured,
			"This server has no key for the provider.", "provider", provider)
	}
	names, err := h.gateway.ProviderModels(c.Context(), provider)
	if err != nil {
		h.logger.WarnContext(c.Context(), "ai model sync: gateway cannot list its models",
			slog.String("provider", provider), slog.String("error", err.Error()))
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeProviderUnavailable,
			"The provider's gateway did not list its models; add them by hand.", "provider", provider)
	}
	known, err := h.q.AdminAiModelUpstreams(c.Context(), provider)
	if err != nil {
		return err
	}
	codes, err := h.q.AdminAiModelCodes(c.Context())
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, k := range known {
		have[k] = true
	}
	taken := map[string]bool{}
	for _, k := range codes {
		taken[k] = true
	}
	stamp := httpadmin.DBTime(httpadmin.Now(c))
	added := []any{}
	existing, skipped := 0, 0
	for _, name := range names {
		name = strings.TrimSpace(name)
		switch {
		case name == "":
			continue
		case have[name]:
			existing++
			continue
		case utf8.RuneCountInString(name) > maxUpstream:
			skipped++
			continue
		}
		have[name] = true
		code := uniqueCode(slug(name), taken)
		taken[code] = true
		n, err := h.q.AdminInsertAiModelIfMissing(c.Context(), store.AdminInsertAiModelIfMissingParams{
			Code: code, Provider: provider, UpstreamModel: name, Title: name,
			Capabilities: json.RawMessage(`["chat"]`), Now: stamp,
		})
		if err != nil {
			return err
		}
		if n == 0 { // raced with another writer: it exists now
			existing++
			continue
		}
		added = append(added, jsonx.Obj("code", code, "upstream_model", name))
	}
	httpadmin.Audit(c, h.logger, "ai_model.sync", "ai_model", 0,
		slog.String("provider", provider), slog.Int("added", len(added)), slog.Int("existing", existing))
	return httpadmin.OK(c, jsonx.Obj("provider", provider, "added", added, "existing", existing, "skipped", skipped),
		"Models synced.")
}

// slug is a catalogue code for an upstream model name: lower-case [a-z0-9], everything else collapsed to '-'.
func slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimRight(b.String(), "-")
	if len(s) > maxCodeLen-4 {
		s = strings.TrimRight(s[:maxCodeLen-4], "-")
	}
	if s == "" {
		s = "model"
	}
	return s
}

// uniqueCode is base, or base-2, base-3, … when taken.
func uniqueCode(base string, taken map[string]bool) string {
	if !taken[base] {
		return base
	}
	for i := 2; ; i++ {
		c := base + "-" + strconv.Itoa(i)
		if !taken[c] {
			return c
		}
	}
}
