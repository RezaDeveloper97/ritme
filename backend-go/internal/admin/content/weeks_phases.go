package content

import (
	"log/slog"
	"slices"

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

// PregnancyWeekFields are the ten translatable modules of a week (PregnancyWeekController::FIELDS).
var PregnancyWeekFields = []string{
	"fetal_development", "mother_body_changes", "dos_and_donts", "care_plan", "body_adaptation",
	"emotional_status", "key_nutrition", "physical_activity", "tests_and_checkups", "faq",
}

// PhaseContentFields are the nine translatable sections of a phase (PhaseContentController::FIELDS).
var PhaseContentFields = []string{
	"symptom_prediction", "vaginal_discharge", "fertility", "hormonal_changes", "sex_tips",
	"nutrition", "exercise", "skin_care", "sleep",
}

// Weeks shown by the list (the tracker's range); rows above it (validation allows 42)
// are listed too so they stay reachable.
const pregnancyWeeks = 40

// columns pairs field names with pointers into a row, so the JSON output and the
// keep-if-absent update share one list.
func weekColumns(w *store.PregnancyWeeklyContent) []*db.NullRawJSON {
	return []*db.NullRawJSON{&w.FetalDevelopment, &w.MotherBodyChanges, &w.DosAndDonts, &w.CarePlan,
		&w.BodyAdaptation, &w.EmotionalStatus, &w.KeyNutrition, &w.PhysicalActivity, &w.TestsAndCheckups, &w.Faq}
}

func phaseColumns(p *store.PhaseContent) []*db.NullRawJSON {
	return []*db.NullRawJSON{&p.SymptomPrediction, &p.VaginalDischarge, &p.Fertility, &p.HormonalChanges,
		&p.SexTips, &p.Nutrition, &p.Exercise, &p.SkinCare, &p.Sleep}
}

func pregnancyWeekJSON(w *store.PregnancyWeeklyContent) *jsonx.OrderedMap {
	o := jsonx.Obj("id", w.ID, "week_number", w.WeekNumber)
	for i, col := range weekColumns(w) {
		o.Set(PregnancyWeekFields[i], form.NullRaw(*col))
	}
	o.Set("created_at", httpadmin.Time(w.CreatedAt))
	o.Set("updated_at", httpadmin.Time(w.UpdatedAt))
	return o
}

func phaseContentJSON(p *store.PhaseContent) *jsonx.OrderedMap {
	o := jsonx.Obj("id", p.ID, "phase", p.Phase)
	for i, col := range phaseColumns(p) {
		o.Set(PhaseContentFields[i], form.NullRaw(*col))
	}
	o.Set("created_at", httpadmin.Time(p.CreatedAt))
	o.Set("updated_at", httpadmin.Time(p.UpdatedAt))
	return o
}

// ---------------------------------------------------------------------------
// Pregnancy weeks

// ListPregnancyWeeks is GET /pregnancy-weeks: one slot per week 1…40 with the row id
// (null = no content yet), plus any stored week above 40.
func (h *Handlers) ListPregnancyWeeks(c fiber.Ctx) error {
	rows, err := h.q.ListPregnancyWeekIDs(c.Context())
	if err != nil {
		return err
	}
	ids := map[int32]uint64{}
	for _, r := range rows {
		ids[r.WeekNumber] = r.ID
	}
	slot := func(w int32) *jsonx.OrderedMap {
		var id any
		if v, ok := ids[w]; ok {
			id = v
		}
		return jsonx.Obj("week", w, "id", id)
	}
	items := []*jsonx.OrderedMap{}
	for w := int32(1); w <= pregnancyWeeks; w++ {
		items = append(items, slot(w))
	}
	for _, r := range rows {
		if r.WeekNumber < 1 || r.WeekNumber > pregnancyWeeks {
			items = append(items, slot(r.WeekNumber))
		}
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "fields", PregnancyWeekFields))
}

// ShowPregnancyWeek is GET /pregnancy-weeks/:id.
func (h *Handlers) ShowPregnancyWeek(c fiber.Ctx) error {
	w, err := find(c, "Pregnancy week", h.q.GetPregnancyWeek)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("pregnancy_week", pregnancyWeekJSON(&w)))
}

func (h *Handlers) validateWeek(c fiber.Ctx, exceptID uint64) (phpval.Map, error) {
	rules := validation.Rules{validation.F("week_number", "required|integer|min:1|max:42")}
	for _, f := range PregnancyWeekFields {
		rules = append(rules, form.Translatable(c, f, false)...)
	}
	return form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		v, ok := in.Get("week_number")
		if !ok || v == nil || !phpval.IsNumeric(v) {
			return nil
		}
		taken, err := h.q.PregnancyWeekTaken(c.Context(), store.PregnancyWeekTakenParams{
			WeekNumber: int32(phpval.ToFloat(v)), ExceptID: exceptID,
		})
		if err != nil {
			return err
		}
		if taken {
			add("week_number", form.Msg(c, "validation.unique", "week_number"))
		}
		return nil
	})
}

// weekFromData overlays the sent fields on cur (Eloquent update semantics).
func weekFromData(data phpval.Map, cur store.PregnancyWeeklyContent) store.PregnancyWeeklyContent {
	w := cur
	w.WeekNumber = form.Int32(data, "week_number", 0)
	for i, col := range weekColumns(&w) {
		*col = form.KeepJSON(data, PregnancyWeekFields[i], *col)
	}
	return w
}

// StorePregnancyWeek is POST /pregnancy-weeks {week_number, <field>{lang: text}…}.
func (h *Handlers) StorePregnancyWeek(c fiber.Ctx) error {
	data, err := h.validateWeek(c, 0)
	if err != nil {
		return err
	}
	w := weekFromData(data, store.PregnancyWeeklyContent{})
	res, err := h.q.CreatePregnancyWeek(c.Context(), store.CreatePregnancyWeekParams{
		WeekNumber: w.WeekNumber, FetalDevelopment: w.FetalDevelopment, MotherBodyChanges: w.MotherBodyChanges,
		DosAndDonts: w.DosAndDonts, CarePlan: w.CarePlan, BodyAdaptation: w.BodyAdaptation,
		EmotionalStatus: w.EmotionalStatus, KeyNutrition: w.KeyNutrition, PhysicalActivity: w.PhysicalActivity,
		TestsAndCheckups: w.TestsAndCheckups, Faq: w.Faq, Now: h.now(c),
	})
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	row, err := h.q.GetPregnancyWeek(c.Context(), id)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_week.create", "pregnancy_week", id, slog.Int("week", int(row.WeekNumber)))
	return httpadmin.Created(c, jsonx.Obj("pregnancy_week", pregnancyWeekJSON(&row)), "Pregnancy week saved.")
}

// UpdatePregnancyWeek is PUT /pregnancy-weeks/:id.
func (h *Handlers) UpdatePregnancyWeek(c fiber.Ctx) error {
	cur, err := find(c, "Pregnancy week", h.q.GetPregnancyWeek)
	if err != nil {
		return err
	}
	data, err := h.validateWeek(c, cur.ID)
	if err != nil {
		return err
	}
	w := weekFromData(data, cur)
	if err := h.q.UpdatePregnancyWeek(c.Context(), store.UpdatePregnancyWeekParams{
		WeekNumber: w.WeekNumber, FetalDevelopment: w.FetalDevelopment, MotherBodyChanges: w.MotherBodyChanges,
		DosAndDonts: w.DosAndDonts, CarePlan: w.CarePlan, BodyAdaptation: w.BodyAdaptation,
		EmotionalStatus: w.EmotionalStatus, KeyNutrition: w.KeyNutrition, PhysicalActivity: w.PhysicalActivity,
		TestsAndCheckups: w.TestsAndCheckups, Faq: w.Faq, Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	row, err := h.q.GetPregnancyWeek(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_week.update", "pregnancy_week", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("pregnancy_week", pregnancyWeekJSON(&row)), "Pregnancy week updated.")
}

// DestroyPregnancyWeek is DELETE /pregnancy-weeks/:id.
func (h *Handlers) DestroyPregnancyWeek(c fiber.Ctx) error {
	cur, err := find(c, "Pregnancy week", h.q.GetPregnancyWeek)
	if err != nil {
		return err
	}
	res, err := h.q.DeletePregnancyWeek(c.Context(), cur.ID)
	if err := deleted(res, err, "Pregnancy week"); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_week.delete", "pregnancy_week", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Pregnancy week deleted.")
}

// ---------------------------------------------------------------------------
// Phase contents

// ListPhaseContents is GET /phase-contents: one entry per content-backed sub-phase with
// the row id (null = no content yet), plus rows stored under other keys (legacy).
func (h *Handlers) ListPhaseContents(c fiber.Ctx) error {
	rows, err := h.q.ListPhaseContentIDs(c.Context())
	if err != nil {
		return err
	}
	ids := map[string]uint64{}
	for _, r := range rows {
		ids[r.Phase] = r.ID
	}
	loc := locale(c)
	items := []*jsonx.OrderedMap{}
	var known []string
	for _, o := range enums.CycleSubphaseOptions(loc) {
		known = append(known, o.Value)
		var id any
		if v, ok := ids[o.Value]; ok {
			id = v
		}
		items = append(items, jsonx.Obj("value", o.Value, "label", o.Label, "legacy", false, "id", id))
	}
	for _, r := range rows {
		if slices.Contains(known, r.Phase) {
			continue
		}
		label, ok := enums.CycleSubphaseLabelFor(r.Phase, loc)
		if !ok {
			label = r.Phase
		}
		items = append(items, jsonx.Obj("value", r.Phase, "label", label, "legacy", true, "id", r.ID))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "fields", PhaseContentFields,
		"phases", options(enums.CycleSubphaseOptions(loc))))
}

// ShowPhaseContent is GET /phase-contents/:id.
func (h *Handlers) ShowPhaseContent(c fiber.Ctx) error {
	p, err := find(c, "Phase content", h.q.GetPhaseContentByID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("phase_content", phaseContentJSON(&p)))
}

func (h *Handlers) validatePhase(c fiber.Ctx, exceptID uint64) (phpval.Map, error) {
	rules := validation.Rules{validation.F("phase", "required|string", validation.In(enums.CycleSubphaseValues()...))}
	for _, f := range PhaseContentFields {
		rules = append(rules, form.Translatable(c, f, false)...)
	}
	return form.Validate(c, rules, func(in phpval.Map, add form.Add) error {
		v, ok := in.Get("phase")
		s, isStr := v.(string)
		if !ok || !isStr {
			return nil
		}
		taken, err := h.q.PhaseContentTaken(c.Context(), store.PhaseContentTakenParams{Phase: s, ExceptID: exceptID})
		if err != nil {
			return err
		}
		if taken {
			add("phase", form.Msg(c, "validation.unique", "phase"))
		}
		return nil
	})
}

func phaseFromData(data phpval.Map, cur store.PhaseContent) store.PhaseContent {
	p := cur
	p.Phase = httpadmin.String(data, "phase")
	for i, col := range phaseColumns(&p) {
		*col = form.KeepJSON(data, PhaseContentFields[i], *col)
	}
	return p
}

// StorePhaseContent is POST /phase-contents {phase, <field>{lang: text}…}.
func (h *Handlers) StorePhaseContent(c fiber.Ctx) error {
	data, err := h.validatePhase(c, 0)
	if err != nil {
		return err
	}
	p := phaseFromData(data, store.PhaseContent{})
	res, err := h.q.CreatePhaseContent(c.Context(), store.CreatePhaseContentParams{
		Phase: p.Phase, SymptomPrediction: p.SymptomPrediction, VaginalDischarge: p.VaginalDischarge,
		Fertility: p.Fertility, HormonalChanges: p.HormonalChanges, SexTips: p.SexTips, Nutrition: p.Nutrition,
		Exercise: p.Exercise, SkinCare: p.SkinCare, Sleep: p.Sleep, Now: h.now(c),
	})
	if err != nil {
		return err
	}
	id, err := insertedID(res)
	if err != nil {
		return err
	}
	row, err := h.q.GetPhaseContentByID(c.Context(), id)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "phase_content.create", "phase_content", id, slog.String("phase", row.Phase))
	return httpadmin.Created(c, jsonx.Obj("phase_content", phaseContentJSON(&row)), "Phase content saved.")
}

// UpdatePhaseContent is PUT /phase-contents/:id.
func (h *Handlers) UpdatePhaseContent(c fiber.Ctx) error {
	cur, err := find(c, "Phase content", h.q.GetPhaseContentByID)
	if err != nil {
		return err
	}
	data, err := h.validatePhase(c, cur.ID)
	if err != nil {
		return err
	}
	p := phaseFromData(data, cur)
	if err := h.q.UpdatePhaseContent(c.Context(), store.UpdatePhaseContentParams{
		Phase: p.Phase, SymptomPrediction: p.SymptomPrediction, VaginalDischarge: p.VaginalDischarge,
		Fertility: p.Fertility, HormonalChanges: p.HormonalChanges, SexTips: p.SexTips, Nutrition: p.Nutrition,
		Exercise: p.Exercise, SkinCare: p.SkinCare, Sleep: p.Sleep, Now: h.now(c), ID: cur.ID,
	}); err != nil {
		return err
	}
	row, err := h.q.GetPhaseContentByID(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "phase_content.update", "phase_content", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("phase_content", phaseContentJSON(&row)), "Phase content updated.")
}

// DestroyPhaseContent is DELETE /phase-contents/:id.
func (h *Handlers) DestroyPhaseContent(c fiber.Ctx) error {
	cur, err := find(c, "Phase content", h.q.GetPhaseContentByID)
	if err != nil {
		return err
	}
	res, err := h.q.DeletePhaseContent(c.Context(), cur.ID)
	if err := deleted(res, err, "Phase content"); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "phase_content.delete", "phase_content", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Phase content deleted.")
}
