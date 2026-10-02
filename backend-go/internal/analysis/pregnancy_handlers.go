package analysis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	healthlogstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/internal/pregnancy/v2/calendar"
)

// PregnancyNamespace holds the pregnancy analysis copy (resources/translations/<code>/analysis-pregnancy.json,
// a byte-identical copy of frontend/messages/<code>/analysis-pregnancy.json). The server reads only its
// symptom_names (labels of the v1 pregnancy symptoms the taxonomy has no item for).
const PregnancyNamespace = "analysis-pregnancy"

// PregnancyHandlers serve GET /api/v1/analysis/pregnancy. The report reads only the authenticated user's
// own rows (no id parameter) and logs nothing.
type PregnancyHandlers struct {
	preg      store.Querier
	profiles  healthlogstore.Querier
	logs      *healthlog.Service
	plus      *plus.Service
	bundles   *i18n.TranslationStore
	languages *i18n.Registry
	clock     clock.Clock
}

// NewPregnancyHandlers wires the handlers; base is the fallback clock (clock.Middleware's request clock wins).
func NewPregnancyHandlers(preg store.Querier, profiles healthlogstore.Querier, logs *healthlog.Service,
	plusSvc *plus.Service, bundles *i18n.TranslationStore, languages *i18n.Registry, base clock.Clock,
) *PregnancyHandlers {
	return &PregnancyHandlers{preg: preg, profiles: profiles, logs: logs, plus: plusSvc, bundles: bundles,
		languages: languages, clock: base}
}

// v1SymptomKeys maps the v1 pregnancy symptom columns (and the v2 extras) onto symptom keys: the taxonomy
// item when there is one, else pregnancy.symptoms.<code> (labelled from PregnancyNamespace). Bleeding,
// spotting, fluid leakage and severe sudden pain are alert symptoms, not a pattern: left out.
var v1SymptomKeys = map[string]string{
	"nausea":               "symptoms.digestive.nausea",
	"vomiting":             "pregnancy.symptoms.vomiting",
	"fatigue":              "symptoms.general.fatigue",
	"headache":             "pain.location.head",
	"dizziness":            "symptoms.general.dizziness",
	"breast_pain":          "symptoms.general.breast_tenderness",
	"lower_abdominal_pain": "pain.location.abdomen",
	"cramping":             "pregnancy.symptoms.cramping",
	"back_pain":            "pain.location.back",
	"pelvic_pressure":      "pregnancy.symptoms.pelvic_pressure",
	"heartburn":            "symptoms.digestive.heartburn",
	"constipation":         "symptoms.digestive.constipation",
	"swelling":             "symptoms.general.swelling",
	"shortness_of_breath":  "pregnancy.symptoms.shortness_of_breath",
}

func has(b sql.NullBool, sev sql.NullString) bool {
	return (b.Valid && b.Bool) || (sev.Valid && sev.String != "" && sev.String != "none")
}

func v1Symptoms(r *store.PregnancySymptomLog) []string {
	pairs := []struct {
		code string
		b    sql.NullBool
		s    sql.NullString
	}{
		{"nausea", r.HasNausea, r.NauseaSeverity}, {"vomiting", r.HasVomiting, r.VomitingSeverity},
		{"fatigue", r.HasFatigue, r.FatigueSeverity}, {"headache", r.HasHeadache, r.HeadacheSeverity},
		{"dizziness", r.HasDizziness, r.DizzinessSeverity}, {"breast_pain", r.HasBreastPain, r.BreastPainSeverity},
		{"lower_abdominal_pain", r.HasLowerAbdominalPain, r.LowerAbdominalPainSeverity},
		{"cramping", r.HasCramping, r.CrampingSeverity}, {"back_pain", r.HasBackPain, r.BackPainSeverity},
		{"pelvic_pressure", r.HasPelvicPressure, r.PelvicPressureSeverity},
	}
	var out []string
	for _, p := range pairs {
		if has(p.b, p.s) {
			out = append(out, p.code)
		}
	}
	return out
}

func decimal(s sql.NullString) (float64, bool) {
	if !s.Valid {
		return 0, false
	}
	f, err := strconv.ParseFloat(s.String, 64)
	return f, err == nil && f > 0
}

// clockMinutes parses "HH:MM[:SS]" into minutes after midnight.
func clockMinutes(s sql.NullString) (int, bool) {
	if !s.Valid || len(s.String) < 5 {
		return 0, false
	}
	h, err1 := strconv.Atoi(s.String[:2])
	m, err2 := strconv.Atoi(s.String[3:5])
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return h*60 + m, true
}

func (in *PregnancyInput) addSymptom(d civildate.Date, key string) {
	if in.Symptoms[d] == nil {
		in.Symptoms[d] = map[string]bool{}
	}
	in.Symptoms[d][key] = true
}

// PregnancyLabeler names a symptom key: the pregnancy namespace's symptom_names (own), then the analysis
// overrides and the taxonomy label (Copy.SymptomLabel).
func PregnancyLabeler(own any, cp *Copy) func(string) string {
	return func(key string) string {
		if s, ok := getString(own, "symptom_names."+key); ok {
			return s
		}
		return cp.SymptomLabel(key)
	}
}

func (h *PregnancyHandlers) labeler(c fiber.Ctx) func(string) string {
	locale, def := i18n.Locale(c), h.languages.DefaultCode(c.Context())
	return PregnancyLabeler(h.bundles.NamespaceMessages(locale, PregnancyNamespace, def),
		NewCopy(h.bundles.NamespaceMessages(locale, Namespace, def),
			h.bundles.NamespaceMessages(locale, healthlog.TaxonomyNamespace, def)))
}

// Pregnancy is GET /analysis/pregnancy: 409 pregnancy_not_active when pregnancy mode is off or undated.
func (h *PregnancyHandlers) Pregnancy(c fiber.Ctx) error {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	ctx := c.Context()
	now := clock.FromContext(c, h.clock).Now()
	today := civildate.InTehran(now)
	p, err := h.preg.GetProfileByUser(ctx, uid)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("analysis pregnancy: profile: %w", err)
	}
	var d v2.Dating
	if err == nil && p.PregnancyMode {
		d, ok = v2.Resolve(&p, today)
	} else {
		ok = false
	}
	if !ok {
		return httpx.Fail(fiber.StatusConflict, v2.T("messages.not_active", i18n.Locale(c)), "error_code", "pregnancy_not_active")
	}
	ent, err := h.plus.Entitlement(ctx, uid, plus.DeepAnalysis, now)
	if err != nil {
		return err
	}
	in := &PregnancyInput{
		Today: today, Start: d.Start(), Due: d.Due, DeepAnalysis: ent.Allowed, Label: h.labeler(c),
		Weights: map[civildate.Date]float64{}, BP: map[civildate.Date]BPReading{},
		Symptoms: map[civildate.Date]map[string]bool{}, Kicks: map[civildate.Date]KickSession{},
	}
	if err := h.loadPregnancy(ctx, uid, in); err != nil {
		return err
	}
	return httpx.OK(c, BuildPregnancy(in))
}

func (h *PregnancyHandlers) loadPregnancy(ctx context.Context, uid uint64, in *PregnancyInput) error {
	from := in.Start.AddDays(-BaselineLookbackDays)
	if prof, err := h.profiles.GetUserProfile(ctx, uid); err == nil {
		if prof.Height.Valid {
			in.HeightCM = int(prof.Height.Int16)
		}
		if w, ok := decimal(prof.Weight); ok {
			in.ProfileWeight = &w
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("analysis pregnancy: user profile: %w", err)
	}

	// Weekly logs (v1 / v2 day log): weight, blood pressure, glucose, swelling, shortness of breath.
	weekly, err := h.preg.ListWeeklyLogs(ctx, uid)
	if err != nil {
		return fmt.Errorf("analysis pregnancy: weekly logs: %w", err)
	}
	for i := range weekly {
		w := &weekly[i]
		if w.LogDate.Before(from) || w.LogDate.After(in.Today) {
			continue
		}
		if v, ok := decimal(w.Weight); ok {
			in.Weights[w.LogDate] = v
		}
		if w.SystolicPressure.Valid && w.DiastolicPressure.Valid {
			in.BP[w.LogDate] = BPReading{Systolic: float64(w.SystolicPressure.Int32), Diastolic: float64(w.DiastolicPressure.Int32)}
		}
		if v, ok := decimal(w.FastingBloodSugar); ok {
			in.Glucose = append(in.Glucose, GlucoseReading{Date: w.LogDate, Slot: GlucoseFasting, Value: v})
		}
		// The v1 «post-meal» sugar is checked against 140 by the v1 alert rules: the 1-hour target.
		if v, ok := decimal(w.PostMealBloodSugar); ok {
			in.Glucose = append(in.Glucose, GlucoseReading{Date: w.LogDate, Slot: GlucoseOneHour, Value: v})
		}
		if w.HasSwelling.Valid && w.HasSwelling.Bool {
			in.addSymptom(w.LogDate, v1SymptomKeys["swelling"])
		}
		if w.HasShortnessOfBreath.Valid && w.HasShortnessOfBreath.Bool {
			in.addSymptom(w.LogDate, v1SymptomKeys["shortness_of_breath"])
		}
	}

	// Taxonomy v2 day logs (the log sheet): they win over a weekly-log value of the same day.
	rows, err := h.logs.Range(ctx, uid, from, in.Today)
	if err != nil {
		return err
	}
	dayRows := make([]DayEntries, len(rows))
	for i, r := range rows {
		dayRows[i] = DayEntries{Date: r.Date, Entries: r.Entries}
	}
	for date, day := range BuildDays(dayRows) {
		if day.Weight != nil && *day.Weight > 0 {
			in.Weights[date] = *day.Weight
		}
		if day.Systolic != nil && day.Diastolic != nil {
			in.BP[date] = BPReading{Systolic: *day.Systolic, Diastolic: *day.Diastolic}
		}
		for k := range day.Symptoms {
			if !isMoodKey(k) && !date.Before(in.Start) {
				in.addSymptom(date, k)
			}
		}
	}

	// v1 symptom logs + v2 extras (heartburn, constipation).
	syms, err := h.preg.ListV2SymptomLogsRange(ctx, store.ListV2SymptomLogsRangeParams{UserID: uid, DateFrom: in.Start, DateTo: in.Today})
	if err != nil {
		return fmt.Errorf("analysis pregnancy: symptoms: %w", err)
	}
	for i := range syms {
		for _, code := range v1Symptoms(&syms[i]) {
			in.addSymptom(syms[i].LogDate, v1SymptomKeys[code])
		}
	}
	extras, err := h.preg.ListDailyExtrasRange(ctx, store.ListDailyExtrasRangeParams{UserID: uid, DateFrom: in.Start, DateTo: in.Today})
	if err != nil {
		return fmt.Errorf("analysis pregnancy: extras: %w", err)
	}
	for _, x := range extras {
		if has(sql.NullBool{}, x.HeartburnSeverity) {
			in.addSymptom(x.LogDate, v1SymptomKeys["heartburn"])
		}
		if has(sql.NullBool{}, x.ConstipationSeverity) {
			in.addSymptom(x.LogDate, v1SymptomKeys["constipation"])
		}
	}

	// Fetal movements of the last week.
	kicks, err := h.preg.ListFetalMovements(ctx, store.ListFetalMovementsParams{
		UserID: uid, HasFrom: 1, FromValue: in.Today.AddDays(-(KicksChartDays - 1)).String(), HasTo: 1, ToValue: in.Today.String(),
	})
	if err != nil {
		return fmt.Errorf("analysis pregnancy: fetal movements: %w", err)
	}
	for _, k := range kicks {
		if !k.MovementCount.Valid {
			continue
		}
		s := KickSession{Count: int(k.MovementCount.Int32)}
		a, okA := clockMinutes(k.FirstMovementTime)
		b, okB := clockMinutes(k.LastMovementTime)
		if okA && okB {
			m := b - a
			if m < 0 {
				m += 24 * 60
			}
			s.Minutes = &m
		}
		in.Kicks[k.LogDate] = s
	}

	// Visits: the M3 appointments (the pregnancy calendar's source).
	appts, err := h.preg.ListV2CalendarAppointments(ctx, uid)
	if err != nil {
		return fmt.Errorf("analysis pregnancy: appointments: %w", err)
	}
	for _, v := range calendar.VisitsOf(appts) {
		if v.Date.Before(in.Start) {
			continue // an appointment from before this pregnancy
		}
		in.Visits = append(in.Visits, PregnancyVisit{
			Date: v.Date, Title: v.Title, Done: v.Stage == "done" || v.Stage == "result",
			ResultNote: strings.TrimSpace(v.ResultNote),
		})
	}
	return nil
}
