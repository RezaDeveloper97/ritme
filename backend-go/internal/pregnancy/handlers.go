// Package pregnancy is the pregnancy mode of the API: the 26 /api/v1/pregnancy routes
// (PregnancyProfileController, PregnancySymptomController, PregnancyWeeklyController,
// PregnancyAlertController), the Eloquent-shaped model JSON and the persistence of logs and
// alerts. The engine lives in calc (PregnancyCalculationService), alerts
// (PregnancyAlertService rules) and content (weekly content).
//
// Reusable API: LoadProfile, ProfileJSON, AlertJSON, calc.New.
package pregnancy

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/alerts"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/content"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Handlers serves the pregnancy routes. Every handler runs behind auth RequireUser and the
// locale middleware.
type Handlers struct {
	q         store.Querier
	afterSave AfterLogSave
}

// AfterLogSave runs after every symptom / weekly / fetal-movement save (the v2 alert rules of
// messages/pregnancyalerts, T-M7-04). It never changes the v1 response.
type AfterLogSave func(ctx context.Context, userID uint64, locale, defaultLocale string, now time.Time) error

// NewHandlers wires the handlers.
func NewHandlers(q store.Querier) *Handlers { return &Handlers{q: q} }

// SetAfterLogSave installs the log-save hook.
func (h *Handlers) SetAfterLogSave(f AfterLogSave) { h.afterSave = f }

func (h *Handlers) afterLogSave(r req) error {
	if h.afterSave == nil {
		return nil
	}
	return h.afterSave(r.c.Context(), r.userID, r.locale, i18n.LanguagesOf(r.c).DefaultCode(), r.now)
}

// req is the per-request context the controllers use.
type req struct {
	c      fiber.Ctx
	userID uint64
	locale string // resolveLocale($request)
	now    time.Time
}

func (h *Handlers) request(c fiber.Ctx) req {
	id, _ := auth.CurrentUserID(c)
	return req{
		c:      c,
		userID: id,
		locale: i18n.ResolveLocale(c, ""),
		now:    clock.FromContext(c.Context(), clock.Real{}).Now(),
	}
}

func (r req) today() civildate.Date { return civildate.InTehran(r.now) }

func (r req) tr(fa, en string) string {
	if r.locale == "fa" {
		return fa
	}
	return en
}

// validate runs a FormRequest: nil map + the framework 422 error on failure.
func (r req) validate(rules validation.Rules, messages []string) (phpval.Map, error) {
	vd := validation.Make(lang.Default(), i18n.Locale(r.c), validation.Input(r.c), rules,
		validation.Now(r.now), validation.Messages(messages...))
	if vd.Fails() {
		return nil, vd.Errors()
	}
	return vd.Validated(), nil
}

func (h *Handlers) calculator(r req) (*calc.Calculator, error) {
	p, err := LoadProfile(r.c.Context(), h.q, r.userID)
	if err != nil {
		return nil, err
	}
	return calc.New(p, r.locale, r.today()), nil
}

func (r req) profileNotFound() error {
	return httpx.Fail(fiber.StatusNotFound, r.tr("پروفایل بارداری یافت نشد", "Pregnancy profile not found"))
}

// ---------------- PregnancyProfileController ----------------

// Activate is POST /pregnancy/activate.
func (h *Handlers) Activate(c fiber.Ctx) error {
	r := h.request(c)
	ctx := c.Context()
	p, err := LoadProfile(ctx, h.q, r.userID)
	if err != nil {
		return err
	}
	if err := saveProfile(ctx, h.q, r.userID, p, jsonx.Obj("pregnancy_mode", true, "cycle_mode", false), r.now); err != nil {
		return err
	}
	// A freshly created model has no onboarding_completed attribute (!null → true).
	onboarded := p != nil && p.OnboardingCompleted
	return httpx.OK(c, jsonx.Obj(
		"pregnancy_mode", true,
		"cycle_mode", false,
		"onboarding_required", !onboarded,
	), r.tr("حالت بارداری فعال شد", "Pregnancy mode activated"))
}

// Deactivate is POST /pregnancy/deactivate.
func (h *Handlers) Deactivate(c fiber.Ctx) error {
	r := h.request(c)
	p, err := LoadProfile(c.Context(), h.q, r.userID)
	if err != nil {
		return err
	}
	if p != nil {
		if err := saveProfile(c.Context(), h.q, r.userID, p, jsonx.Obj("pregnancy_mode", false, "cycle_mode", true), r.now); err != nil {
			return err
		}
	}
	return httpx.OK(c, jsonx.Obj("pregnancy_mode", false, "cycle_mode", true),
		r.tr("حالت بارداری غیرفعال شد", "Pregnancy mode deactivated"))
}

// confidenceFor maps an age source to [confidence_level, uncertainty_days].
func confidenceFor(source string) (enums.ConfidenceLevel, int, bool) {
	switch enums.PregnancyAgeSource(source) {
	case enums.PregnancyAgeSourceUltrasound:
		return enums.ConfidenceLevelHigh, 1, true
	case enums.PregnancyAgeSourceLmp:
		return enums.ConfidenceLevelMedium, 3, true
	case enums.PregnancyAgeSourceManual:
		return enums.ConfidenceLevelLow, 5, true
	}
	return enums.ConfidenceLevelLow, 5, false
}

// Onboarding is POST /pregnancy/onboarding (201).
func (h *Handlers) Onboarding(c fiber.Ctx) error {
	r := h.request(c)
	ctx := c.Context()
	validated, err := r.validate(storeProfileRules(), storeProfileMessages)
	if err != nil {
		return err
	}
	source := phpval.ToString(get(validated, "age_source"))
	conf, uncertainty, _ := confidenceFor(source)
	if source == string(enums.PregnancyAgeSourceManual) {
		validated.Set("manual_entry_date", r.today().String())
	}
	rh, hasRh := validated.Get("rh_factor")
	validated.Set("pregnancy_mode", true).
		Set("cycle_mode", false).
		Set("confidence_level", string(conf)).
		Set("uncertainty_days", int64(uncertainty)).
		Set("rh_negative_care_flag", hasRh && rh == string(enums.RhFactorNegative)).
		Set("onboarding_completed", true).
		Set("onboarding_completed_at", dbNow(r.now))

	existing, err := LoadProfile(ctx, h.q, r.userID)
	if err != nil {
		return err
	}
	if err := saveProfile(ctx, h.q, r.userID, existing, validated, r.now); err != nil {
		return err
	}
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	ga, edd := cl.GestationalAge(), cl.EDD()
	if edd != nil {
		if err := saveProfile(ctx, h.q, r.userID, cl.Profile(), jsonx.Obj("estimated_due_date", edd.Date), r.now); err != nil {
			return err
		}
	}
	p, err := LoadProfile(ctx, h.q, r.userID)
	if err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj(
		"profile", ProfileJSON(p),
		"gestational_age", ga.JSON(),
		"estimated_due_date", edd.JSON(),
	), r.tr("آنبوردینگ بارداری تکمیل شد", "Pregnancy onboarding completed"))
}

// Show is GET /pregnancy/profile.
func (h *Handlers) Show(c fiber.Ctx) error {
	r := h.request(c)
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	if cl.Profile() == nil {
		return r.profileNotFound()
	}
	return httpx.OK(c, jsonx.Obj("profile", ProfileJSON(cl.Profile()), "status", cl.Status().JSON()))
}

// Update is PUT /pregnancy/profile.
func (h *Handlers) Update(c fiber.Ctx) error {
	r := h.request(c)
	ctx := c.Context()
	p, err := LoadProfile(ctx, h.q, r.userID)
	if err != nil {
		return err
	}
	if p == nil {
		return r.profileNotFound()
	}
	validated, err := r.validate(updateProfileRules(), updateProfileMessages)
	if err != nil {
		return err
	}
	if src := get(validated, "age_source"); src != nil {
		if conf, unc, ok := confidenceFor(phpval.ToString(src)); ok {
			validated.Set("confidence_level", string(conf)).Set("uncertainty_days", int64(unc))
		} else {
			validated.Set("confidence_level", rawString(p.ConfidenceLevel)).Set("uncertainty_days", int64(p.UncertaintyDays))
		}
		if src == string(enums.PregnancyAgeSourceManual) {
			validated.Set("manual_entry_date", r.today().String())
		}
	}
	if rh := get(validated, "rh_factor"); rh != nil {
		validated.Set("rh_negative_care_flag", rh == string(enums.RhFactorNegative))
	}
	if err := saveProfile(ctx, h.q, r.userID, p, validated, r.now); err != nil {
		return err
	}
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	if edd := cl.EDD(); edd != nil {
		if err := saveProfile(ctx, h.q, r.userID, cl.Profile(), jsonx.Obj("estimated_due_date", edd.Date), r.now); err != nil {
			return err
		}
	}
	fresh, err := LoadProfile(ctx, h.q, r.userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("profile", ProfileJSON(fresh), "status", cl.Status().JSON()),
		r.tr("پروفایل با موفقیت بروزرسانی شد", "Profile updated successfully"))
}

// Confirm is POST /pregnancy/confirm.
func (h *Handlers) Confirm(c fiber.Ctx) error {
	r := h.request(c)
	p, err := LoadProfile(c.Context(), h.q, r.userID)
	if err != nil {
		return err
	}
	if p == nil {
		return r.profileNotFound()
	}
	if err := saveProfile(c.Context(), h.q, r.userID, p, jsonx.Obj("is_locked", true), r.now); err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("is_locked", true), r.tr("پروفایل تأیید و قفل شد", "Profile confirmed and locked"))
}

// Status is GET /pregnancy/status.
func (h *Handlers) Status(c fiber.Ctx) error {
	r := h.request(c)
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	return httpx.OK(c, cl.Status().JSON())
}

type labeled interface {
	~string
	Label(string) string
}

type described interface {
	labeled
	Description(string) string
}

func options[E labeled](cases []E, locale string) []any {
	out := make([]any, len(cases))
	for i, e := range cases {
		out[i] = jsonx.Obj("value", string(e), "label", e.Label(locale))
	}
	return out
}

func describedOptions[E described](cases []E, locale string) []any {
	out := make([]any, len(cases))
	for i, e := range cases {
		out[i] = jsonx.Obj("value", string(e), "label", e.Label(locale), "description", e.Description(locale))
	}
	return out
}

// Enums is GET /pregnancy/enums.
func (h *Handlers) Enums(c fiber.Ctx) error {
	l := h.request(c).locale
	return httpx.OK(c, jsonx.Obj(
		"age_sources", describedOptions(enums.PregnancyAgeSourceCases(), l),
		"confidence_levels", describedOptions(enums.ConfidenceLevelCases(), l),
		"blood_types", options(enums.BloodTypeCases(), l),
		"rh_factors", options(enums.RhFactorCases(), l),
		"pre_existing_conditions", options(enums.PreExistingConditionCases(), l),
		"alert_levels", describedOptions(enums.AlertLevelCases(), l),
	))
}

// ---------------- PregnancySymptomController ----------------

// SymptomEnums is GET /pregnancy/symptoms/enums.
func (h *Handlers) SymptomEnums(c fiber.Ctx) error {
	return httpx.OK(c, jsonx.Obj("severity", options(enums.SymptomSeverityCases(), h.request(c).locale)))
}

// rangeFilter reads ?from / ?to like `$request->has(k)` + where('log_date', op, $request->get(k)):
// a present-but-null value becomes whereNotNull (no filter on a NOT NULL column).
func rangeFilter(c fiber.Ctx, key string) (int64, string) {
	q := validation.Query(c)
	v, ok := q.Get(key)
	if !ok || v == nil {
		return 0, ""
	}
	return 1, phpval.ToString(v)
}

// SymptomIndex is GET /pregnancy/symptoms.
func (h *Handlers) SymptomIndex(c fiber.Ctx) error {
	r := h.request(c)
	hasFrom, from := rangeFilter(c, "from")
	hasTo, to := rangeFilter(c, "to")
	rows, err := h.q.ListSymptomLogs(c.Context(), store.ListSymptomLogsParams{
		UserID: r.userID, HasFrom: hasFrom, FromValue: from, HasTo: hasTo, ToValue: to,
	})
	if err != nil {
		return err
	}
	logs := make([]any, len(rows))
	for i := range rows {
		logs[i] = symptomModel(&rows[i]).JSON()
	}
	return httpx.OK(c, jsonx.Obj("logs", logs, "count", len(logs)))
}

// SymptomStore is POST /pregnancy/symptoms (201, upsert by date, alerts).
func (h *Handlers) SymptomStore(c fiber.Ctx) error {
	r := h.request(c)
	ctx := c.Context()
	validated, err := r.validate(symptomRules(), symptomMessages)
	if err != nil {
		return err
	}
	rawDate := get(validated, "log_date")
	date, ok := toDate(rawDate)
	if !ok {
		return httpx.ServerError()
	}
	log, err := upsertSymptomLog(ctx, h.q, r.userID, date, rawDate, validated, r.now)
	if err != nil {
		return err
	}
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	drafts := alerts.ForSymptomLog(alerts.Context{Locale: r.locale, Week: cl.CurrentWeek(), Profile: cl.Profile()}, log)
	created, err := createAlerts(ctx, h.q, r.userID, drafts, r.now)
	if err != nil {
		return err
	}
	if err := h.afterLogSave(r); err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("log", log, "alerts", jsonx.List(created)),
		r.tr("علائم با موفقیت ذخیره شد", "Symptom log saved successfully"))
}

// dateParam is Carbon::parse($date) of a route segment (422 when unparseable).
func (r req) dateParam(raw string) (civildate.Date, error) {
	t, err := civildate.ParseLenient(raw, r.now, civildate.Tehran)
	if err != nil {
		return civildate.Date{}, httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid date format. Use YYYY-MM-DD.")
	}
	return civildate.FromTime(t), nil
}

func (r req) symptomNotFound() error {
	return httpx.Fail(fiber.StatusNotFound, r.tr("گزارش علائم یافت نشد", "Symptom log not found"))
}

// SymptomShow is GET /pregnancy/symptoms/{date}.
func (h *Handlers) SymptomShow(c fiber.Ctx) error {
	r := h.request(c)
	date, err := r.dateParam(c.Params("date"))
	if err != nil {
		return err
	}
	row, err := noRows(h.q.GetSymptomLog(c.Context(), store.GetSymptomLogParams{UserID: r.userID, LogDate: date}))
	if err != nil {
		return err
	}
	if row == nil {
		return r.symptomNotFound()
	}
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("log", symptomModel(row).JSON(), "pregnancy_week", cl.CurrentWeek()))
}

// SymptomDestroy is DELETE /pregnancy/symptoms/{date}.
func (h *Handlers) SymptomDestroy(c fiber.Ctx) error {
	r := h.request(c)
	date, err := r.dateParam(c.Params("date"))
	if err != nil {
		return err
	}
	n, err := h.q.DeleteSymptomLog(c.Context(), store.DeleteSymptomLogParams{UserID: r.userID, LogDate: date})
	if err != nil {
		return err
	}
	if n == 0 {
		return r.symptomNotFound()
	}
	return httpx.Message(c, r.tr("گزارش علائم حذف شد", "Symptom log deleted successfully"))
}

// ---------------- PregnancyWeeklyController ----------------

// WeeklyEnums is GET /pregnancy/weekly/enums.
func (h *Handlers) WeeklyEnums(c fiber.Ctx) error {
	l := h.request(c).locale
	return httpx.OK(c, jsonx.Obj(
		"swelling_locations", options(enums.SwellingLocationCases(), l),
		"overall_mood", options(enums.MentalHealthStatusCases(), l),
		"severity", options(enums.SymptomSeverityCases(), l),
		"fetal_movement_status", options(enums.FetalMovementStatusCases(), l),
	))
}

// WeeklyIndex is GET /pregnancy/weekly.
func (h *Handlers) WeeklyIndex(c fiber.Ctx) error {
	r := h.request(c)
	rows, err := h.q.ListWeeklyLogs(c.Context(), r.userID)
	if err != nil {
		return err
	}
	logs := make([]any, len(rows))
	for i := range rows {
		logs[i] = weeklyModel(&rows[i]).JSON()
	}
	return httpx.OK(c, jsonx.Obj("logs", logs, "count", len(logs)))
}

// WeeklyStore is POST /pregnancy/weekly (201, upsert by week, alerts).
func (h *Handlers) WeeklyStore(c fiber.Ctx) error {
	r := h.request(c)
	ctx := c.Context()
	validated, err := r.validate(weeklyRules(), weeklyMessages)
	if err != nil {
		return err
	}
	rawWeek := get(validated, "pregnancy_week")
	log, err := upsertWeeklyLog(ctx, h.q, r.userID, wInt(rawWeek).Int32, rawWeek, validated, r.now)
	if err != nil {
		return err
	}
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	drafts := alerts.ForWeeklyLog(alerts.Context{Locale: r.locale, Week: cl.CurrentWeek(), Profile: cl.Profile()}, log)
	created, err := createAlerts(ctx, h.q, r.userID, drafts, r.now)
	if err != nil {
		return err
	}
	if err := h.afterLogSave(r); err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("log", log, "alerts", jsonx.List(created)),
		r.tr("گزارش هفتگی با موفقیت ذخیره شد", "Weekly log saved successfully"))
}

// intParam is an `int $x` route parameter: numeric strings are coerced like PHP; anything
// else is a 404 (D-02; Laravel answers 500 with a TypeError).
func intParam(c fiber.Ctx, name string) (int, error) {
	raw := c.Params(name)
	if !phpval.IsNumericString(raw) {
		return 0, httpx.NotFound()
	}
	return int(phpval.ToFloat(strings.TrimSpace(raw))), nil
}

// WeeklyShow is GET /pregnancy/weekly/{week}.
func (h *Handlers) WeeklyShow(c fiber.Ctx) error {
	r := h.request(c)
	week, err := intParam(c, "week")
	if err != nil {
		return err
	}
	if week < 1 || week > 42 {
		return httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid week number. Must be between 1 and 42.")
	}
	row, err := noRows(h.q.GetWeeklyLog(c.Context(), store.GetWeeklyLogParams{UserID: r.userID, PregnancyWeek: int32(week)}))
	if err != nil {
		return err
	}
	if row == nil {
		return httpx.Fail(fiber.StatusNotFound, r.tr("گزارش هفتگی یافت نشد", "Weekly log not found"))
	}
	return httpx.OK(c, jsonx.Obj("log", weeklyModel(row).JSON()))
}

// FetalIndex is GET /pregnancy/fetal-movement.
func (h *Handlers) FetalIndex(c fiber.Ctx) error {
	r := h.request(c)
	hasFrom, from := rangeFilter(c, "from")
	hasTo, to := rangeFilter(c, "to")
	rows, err := h.q.ListFetalMovements(c.Context(), store.ListFetalMovementsParams{
		UserID: r.userID, HasFrom: hasFrom, FromValue: from, HasTo: hasTo, ToValue: to,
	})
	if err != nil {
		return err
	}
	logs := make([]any, len(rows))
	for i := range rows {
		logs[i] = fetalModel(&rows[i]).JSON()
	}
	return httpx.OK(c, jsonx.Obj("logs", logs, "count", len(logs)))
}

// feltStatuses mark the first felt movement on the profile.
var feltStatuses = []string{"felt", "normal", "increased"}

// FetalStore is POST /pregnancy/fetal-movement (201, upsert by date, alerts).
func (h *Handlers) FetalStore(c fiber.Ctx) error {
	r := h.request(c)
	ctx := c.Context()
	validated, err := r.validate(fetalRules(), fetalMessages)
	if err != nil {
		return err
	}
	rawDate := get(validated, "log_date")
	date, ok := toDate(rawDate)
	if !ok {
		return httpx.ServerError()
	}
	log, err := upsertFetalMovement(ctx, h.q, r.userID, date, rawDate, validated, r.now)
	if err != nil {
		return err
	}
	status := get(validated, "movement_status")
	for _, felt := range feltStatuses {
		if !phpval.LooseEqual(status, felt) {
			continue
		}
		p, err := LoadProfile(ctx, h.q, r.userID)
		if err != nil {
			return err
		}
		if p != nil && !p.FetalMovementFelt {
			first := rawDate
			if p.FirstFetalMovementDate.Valid {
				first = p.FirstFetalMovementDate.Date
			}
			if err := saveProfile(ctx, h.q, r.userID, p,
				jsonx.Obj("fetal_movement_felt", true, "first_fetal_movement_date", first), r.now); err != nil {
				return err
			}
		}
		break
	}
	cl, err := h.calculator(r)
	if err != nil {
		return err
	}
	drafts := alerts.ForFetalMovement(alerts.Context{Locale: r.locale, Week: cl.CurrentWeek(), Profile: cl.Profile()}, log)
	created, err := createAlerts(ctx, h.q, r.userID, drafts, r.now)
	if err != nil {
		return err
	}
	if err := h.afterLogSave(r); err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("log", log, "alerts", jsonx.List(created)),
		r.tr("حرکات جنین ثبت شد", "Fetal movement logged successfully"))
}

// Content is GET /pregnancy/content/{week}: locale clamped to fa/en (default en).
func (h *Handlers) Content(c fiber.Ctx) error {
	r := h.request(c)
	locale := content.Locale(r.locale)
	week, err := intParam(c, "week")
	if err != nil {
		return err
	}
	if week < content.MinWeek || week > content.MaxWeek {
		return httpx.Fail(fiber.StatusUnprocessableEntity, "Invalid week number. Must be between 1 and 40.")
	}
	row, err := noRows(h.q.GetWeeklyContent(c.Context(), int32(week)))
	if err != nil {
		return err
	}
	if row == nil {
		msg := "Content not found for this week"
		if locale == "fa" {
			msg = "محتوای این هفته یافت نشد"
		}
		return httpx.Fail(fiber.StatusNotFound, msg)
	}
	return httpx.OK(c, jsonx.Obj("week", week, "content", content.Localize(*row, locale)))
}

// ---------------- PregnancyAlertController ----------------

func alertCounts(cnt store.CountActiveAlertsRow, emergency int64) *jsonx.OrderedMap {
	return jsonx.Obj(
		"total", cnt.Total,
		"unread", cnt.Unread,
		"emergency", emergency,
		"warning", cnt.Warning,
		"info", cnt.Info,
	)
}

// AlertIndex is GET /pregnancy/alerts (?level, ?unread_only).
func (h *Handlers) AlertIndex(c fiber.Ctx) error {
	r := h.request(c)
	q := validation.Query(c)
	level := ""
	if v, ok := q.Get("level"); ok {
		for _, l := range enums.AlertLevelValues() {
			if phpval.LooseEqual(v, l) {
				level = l
				break
			}
		}
	}
	unread := int64(0)
	if v, ok := q.Get("unread_only"); ok && filterBool(v) {
		unread = 1
	}
	rows, err := h.q.ListActiveAlerts(c.Context(), store.ListActiveAlertsParams{UserID: r.userID, Level: level, UnreadOnly: unread})
	if err != nil {
		return err
	}
	cnt, err := h.q.CountActiveAlerts(c.Context(), r.userID)
	if err != nil {
		return err
	}
	list := make([]any, len(rows))
	for i := range rows {
		list[i] = AlertJSON(&rows[i])
	}
	return httpx.OK(c, jsonx.Obj("alerts", list, "counts", alertCounts(cnt, cnt.Emergency)))
}

// filterBool is $request->boolean(): FILTER_VALIDATE_BOOLEAN ("1", "true", "on", "yes").
func filterBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	switch strings.ToLower(strings.TrimSpace(phpval.ToString(v))) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

func (h *Handlers) findAlert(c fiber.Ctx, r req) (*store.PregnancyAlert, error) {
	id, err := intParam(c, "id")
	if err != nil {
		return nil, err
	}
	row, err := noRows(h.q.GetAlert(c.Context(), store.GetAlertParams{UserID: r.userID, ID: uint64(max(id, 0))}))
	if err != nil {
		return nil, err
	}
	if row == nil || id <= 0 {
		return nil, httpx.Fail(fiber.StatusNotFound, r.tr("هشدار یافت نشد", "Alert not found"))
	}
	return row, nil
}

// AlertShow is GET /pregnancy/alerts/{id}.
func (h *Handlers) AlertShow(c fiber.Ctx) error {
	r := h.request(c)
	a, err := h.findAlert(c, r)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("alert", AlertJSON(a)))
}

// AlertRead is POST /pregnancy/alerts/{id}/read.
func (h *Handlers) AlertRead(c fiber.Ctx) error {
	r := h.request(c)
	a, err := h.findAlert(c, r)
	if err != nil {
		return err
	}
	ts := sqlTime(dbNow(r.now))
	if err := h.q.MarkAlertRead(c.Context(), store.MarkAlertReadParams{ReadAt: ts, UpdatedAt: ts, ID: a.ID}); err != nil {
		return err
	}
	return httpx.Message(c, r.tr("هشدار خوانده شد", "Alert marked as read"))
}

// AlertReadAll is POST /pregnancy/alerts/read-all.
func (h *Handlers) AlertReadAll(c fiber.Ctx) error {
	r := h.request(c)
	ts := sqlTime(dbNow(r.now))
	n, err := h.q.MarkAllAlertsRead(c.Context(), store.MarkAllAlertsReadParams{ReadAt: ts, UpdatedAt: ts, UserID: r.userID})
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("count", n), r.tr("همه هشدارها خوانده شدند", "All alerts marked as read"))
}

// AlertDismiss is POST /pregnancy/alerts/{id}/dismiss.
func (h *Handlers) AlertDismiss(c fiber.Ctx) error {
	r := h.request(c)
	a, err := h.findAlert(c, r)
	if err != nil {
		return err
	}
	ts := sqlTime(dbNow(r.now))
	if err := h.q.DismissAlert(c.Context(), store.DismissAlertParams{DismissedAt: ts, UpdatedAt: ts, ID: a.ID}); err != nil {
		return err
	}
	return httpx.Message(c, r.tr("هشدار رد شد", "Alert dismissed"))
}

// AlertSummary is GET /pregnancy/alerts/summary.
func (h *Handlers) AlertSummary(c fiber.Ctx) error {
	r := h.request(c)
	cnt, err := h.q.CountActiveAlerts(c.Context(), r.userID)
	if err != nil {
		return err
	}
	var latest any
	if cnt.UnreadEmergency > 0 {
		row, err := noRows(h.q.LatestUnreadEmergencyAlert(c.Context(), r.userID))
		if err != nil {
			return err
		}
		latest = AlertJSON(row)
	}
	return httpx.OK(c, jsonx.Obj(
		"has_emergency", cnt.UnreadEmergency > 0,
		"has_unread", cnt.Unread > 0,
		"counts", alertCounts(cnt, cnt.UnreadEmergency),
		"latest_emergency", latest,
	))
}

func get(m phpval.Map, key string) any {
	v, _ := m.Get(key)
	return v
}
