package ivf

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// dateTimeFormats are the accepted datetime formats (Tehran wall-clock), like /care/appointments.
const dateTimeFormats = "date_format:Y-m-d H:i:s,Y-m-d H:i"

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// fieldFail is a 422 with one message on field.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{fieldMessage(field, key, locale)}))
}

// pick copies the keys of body that are in keys (anything else is ignored).
func pick(body phpval.Map, keys []string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

// validate runs rules over data with the IVF attribute names and messages.
func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	future := T("validation.date_future", locale)
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now),
		validation.Attributes(attributes(locale)...),
		validation.Messages("started_on.before_or_equal", future, "date.before_or_equal", future))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func present(data phpval.Map, key string) (any, bool) { return data.Get(key) }

// str is a present, non-null value as a trimmed string ("" when absent or null).
func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func intOf(v any) int {
	if n, err := strconv.Atoi(phpval.ToString(v)); err == nil {
		return n
	}
	return int(phpval.ToFloat(v))
}

func dateOf(data phpval.Map, key string) (civildate.Date, error) {
	s := str(data, key)
	if s == "" {
		return civildate.Date{}, nil
	}
	d, err := civildate.Parse(s)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("ivf: %s: %w", key, err)
	}
	return d, nil
}

func dateTimeOf(data phpval.Map, key string) (time.Time, error) {
	s := str(data, key)
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{time.DateTime, "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, s, civildate.Tehran); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("ivf: %s %q: unparseable", key, s)
}

var startFields = []string{"number", "protocol", "stage", "started_on", "stim_started_on", "notify_companion"}

// validateStart is POST /ivf/cycles: started_on defaults to today, stage to prep.
func validateStart(body phpval.Map, locale string, now time.Time) (StartInput, error) {
	F, in := validation.F, validation.In
	data := pick(body, startFields)
	if err := validate(data, validation.Rules{
		F("number", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxCycleNumber)),
		F("protocol", "nullable", "string", "max:32"),
		F("stage", "nullable", in(Stages...)),
		F("started_on", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("stim_started_on", "nullable", "date_format:Y-m-d"),
		F("notify_companion", "sometimes", "boolean"),
	}, locale, now); err != nil {
		return StartInput{}, err
	}
	out := StartInput{Protocol: str(data, "protocol"), Stage: str(data, "stage"), StartedOn: civildate.InTehran(now)}
	if out.Stage == "" {
		out.Stage = StagePrep
	}
	if v, ok := present(data, "number"); ok && v != nil {
		out.Number = intOf(v)
	}
	if d, err := dateOf(data, "started_on"); err != nil {
		return out, err
	} else if !d.IsZero() {
		out.StartedOn = d
	}
	var err error
	if out.StimStartedOn, err = dateOf(data, "stim_started_on"); err != nil {
		return out, err
	}
	if v, ok := present(data, "notify_companion"); ok {
		out.NotifyCompanion = phpval.Truthy(v)
	}
	return out, nil
}

var patchFields = []string{
	"number", "protocol", "stage", "started_on", "stim_started_on", "retrieval_at", "transfer_at", "beta_on",
	"next_scan_at", "notify_companion",
}

// validatePatch is PUT /ivf/cycles/current: only the keys sent change; null clears an optional field.
func validatePatch(body phpval.Map, locale string, now time.Time) (CyclePatch, error) {
	F, in := validation.F, validation.In
	data := pick(body, patchFields)
	if err := validate(data, validation.Rules{
		F("number", "sometimes", "required", "integer", "min:1", "max:"+strconv.Itoa(MaxCycleNumber)),
		F("protocol", "sometimes", "nullable", "string", "max:32"),
		F("stage", "sometimes", "required", in(Stages...)),
		F("started_on", "sometimes", "required", "date_format:Y-m-d", "before_or_equal:today"),
		F("stim_started_on", "sometimes", "nullable", "date_format:Y-m-d"),
		F("retrieval_at", "sometimes", "nullable", dateTimeFormats),
		F("transfer_at", "sometimes", "nullable", dateTimeFormats),
		F("beta_on", "sometimes", "nullable", "date_format:Y-m-d"),
		F("next_scan_at", "sometimes", "nullable", dateTimeFormats),
		F("notify_companion", "sometimes", "boolean"),
	}, locale, now); err != nil {
		return CyclePatch{}, err
	}
	var p CyclePatch
	if v, ok := present(data, "number"); ok {
		n := intOf(v)
		p.Number = &n
	}
	if _, ok := present(data, "protocol"); ok {
		s := str(data, "protocol")
		p.Protocol = &s
	}
	if _, ok := present(data, "stage"); ok {
		s := str(data, "stage")
		p.Stage = &s
	}
	if _, ok := present(data, "started_on"); ok {
		d, err := dateOf(data, "started_on")
		if err != nil {
			return p, err
		}
		p.StartedOn = &d
	}
	for key, dst := range map[string]*OptDate{"stim_started_on": &p.StimStartedOn, "beta_on": &p.BetaOn} {
		if _, ok := present(data, key); ok {
			d, err := dateOf(data, key)
			if err != nil {
				return p, err
			}
			*dst = OptDate{Set: true, V: d}
		}
	}
	for key, dst := range map[string]*OptTime{"retrieval_at": &p.RetrievalAt, "transfer_at": &p.TransferAt, "next_scan_at": &p.NextScanAt} {
		if _, ok := present(data, key); ok {
			t, err := dateTimeOf(data, key)
			if err != nil {
				return p, err
			}
			*dst = OptTime{Set: true, V: t}
		}
	}
	if v, ok := present(data, "notify_companion"); ok {
		b := phpval.Truthy(v)
		p.NotifyCompanion = &b
	}
	return p, nil
}

// OutcomeInput is a validated POST /ivf/cycles/current/outcome.
type OutcomeInput struct {
	Result string
	Date   civildate.Date
}

func validateOutcome(body phpval.Map, locale string, now time.Time) (OutcomeInput, error) {
	F := validation.F
	data := pick(body, []string{"result", "date"})
	if err := validate(data, validation.Rules{
		F("result", "required", validation.In(Outcomes...)),
		F("date", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
	}, locale, now); err != nil {
		return OutcomeInput{}, err
	}
	out := OutcomeInput{Result: str(data, "result"), Date: civildate.InTehran(now)}
	if d, err := dateOf(data, "date"); err != nil {
		return out, err
	} else if !d.IsZero() {
		out.Date = d
	}
	return out, nil
}

var medFields = []string{
	"name", "role", "route", "dose", "unit", "times", "trigger_at", "starts_on", "ends_on", "notes", "stock_units",
	"stock_unit", "doses_per_unit",
}

// validateMed is POST /ivf/meds and PUT /ivf/meds/{id} (a full replace). The trigger is one dose at trigger_at
// (its day is the start and the end); every other medicine needs 1–4 daily times, starts today by default and is
// ongoing without ends_on.
func validateMed(body phpval.Map, locale string, now time.Time, subtitleLocale string) (MedInput, error) {
	F, in := validation.F, validation.In
	data := pick(body, medFields)
	for _, k := range []string{"dose", "unit"} { // numbers are taken as their string form (like /care)
		if v, ok := data.Get(k); ok && v != nil {
			if _, isStr := v.(string); !isStr && phpval.IsNumeric(v) {
				data.Set(k, phpval.ToString(v))
			}
		}
	}
	if err := validate(data, validation.Rules{
		F("name", "required", "string", "max:100"),
		F("role", "required", in(Roles...)),
		F("route", "required", in(Routes...)),
		F("dose", "nullable", "string", "max:50"),
		F("unit", "nullable", "string", "max:50"),
		F("times", "nullable", "array", "max:"+strconv.Itoa(MaxTimes)),
		F("times.*", "required", "date_format:H:i"),
		F("trigger_at", "required_if:role,"+RoleTrigger, "nullable", dateTimeFormats),
		F("starts_on", "nullable", "date_format:Y-m-d"),
		F("ends_on", "nullable", "date_format:Y-m-d", "after_or_equal:starts_on"),
		F("notes", "nullable", "string", "max:2000"),
		F("stock_units", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxStockUnits)),
		F("stock_unit", "nullable", in(StockUnits...)),
		F("doses_per_unit", "nullable", "integer", "min:1", "max:"+strconv.Itoa(MaxDosesPerUnit)),
	}, locale, now); err != nil {
		return MedInput{}, err
	}
	out := MedInput{
		Name: str(data, "name"), Role: str(data, "role"), Route: str(data, "route"), DosesPerUnit: 1,
		SubtitleLocale: subtitleLocale,
	}
	for key, dst := range map[string]**string{"dose": &out.Dose, "unit": &out.Unit, "notes": &out.Notes} {
		if v, ok := data.Get(key); ok && v != nil {
			s := phpval.ToString(v)
			*dst = &s
		}
	}
	if v, ok := present(data, "stock_units"); ok && v != nil {
		n := intOf(v)
		out.StockUnits = &n
		out.StockUnit = str(data, "stock_unit")
		if out.StockUnit == "" {
			out.StockUnit = "other"
		}
	}
	if v, ok := present(data, "doses_per_unit"); ok && v != nil {
		out.DosesPerUnit = intOf(v)
	}
	if out.Role == RoleTrigger {
		at, err := dateTimeOf(data, "trigger_at")
		if err != nil {
			return out, err
		}
		out.TriggerAt, out.Times = at, []string{at.Format("15:04")}
		out.StartsOn, out.EndsOn = civildate.InTehran(at), civildate.InTehran(at)
		return out, nil
	}
	times, _ := data.Get("times")
	_, tv := phpval.Entries(times)
	for _, t := range tv {
		if s := phpval.ToString(t); !slices.Contains(out.Times, s) {
			out.Times = append(out.Times, s)
		}
	}
	if len(out.Times) == 0 {
		return out, fieldFail(locale, "times", "times_required")
	}
	slices.Sort(out.Times)
	var err error
	if out.StartsOn, err = dateOf(data, "starts_on"); err != nil {
		return out, err
	}
	if out.StartsOn.IsZero() {
		out.StartsOn = civildate.InTehran(now)
	}
	if out.EndsOn, err = dateOf(data, "ends_on"); err != nil {
		return out, err
	}
	if !out.EndsOn.IsZero() && out.EndsOn.Before(out.StartsOn) {
		return out, failValidation(locale, jsonx.Obj("ends_on", []string{T("validation.ends_before_start", locale)}))
	}
	return out, nil
}

// DoseInput is a validated POST/DELETE /ivf/meds/{id}/doses.
type DoseInput struct {
	Date civildate.Date
	Slot string
	Site string
}

// validateDose: logging defaults date to today and refuses the future; undo needs the date.
func validateDose(body phpval.Map, locale string, now time.Time, logging bool) (DoseInput, error) {
	F := validation.F
	data := pick(body, []string{"date", "slot", "site"})
	date := []any{"required", "date_format:Y-m-d"}
	if logging {
		date = []any{"nullable", "date_format:Y-m-d", "before_or_equal:today"}
	}
	if err := validate(data, validation.Rules{
		F("date", date...),
		F("slot", "required", "date_format:H:i"),
		F("site", "nullable", "string", "max:32"),
	}, locale, now); err != nil {
		return DoseInput{}, err
	}
	out := DoseInput{Date: civildate.InTehran(now), Slot: str(data, "slot")}
	if logging {
		out.Site = str(data, "site")
	}
	if d, err := dateOf(data, "date"); err != nil {
		return out, err
	} else if !d.IsZero() {
		out.Date = d
	}
	return out, nil
}

// parseDate validates a {date} path segment: a well-formed date, not in the future when past is set.
func parseDate(raw, locale string, now time.Time, past bool) (civildate.Date, error) {
	data := phpval.NewMap()
	data.Set("date", raw)
	rule := []any{"required", "date_format:Y-m-d"}
	if past {
		rule = append(rule, "before_or_equal:today")
	}
	if err := validate(data, validation.Rules{validation.F("date", rule...)}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("ivf: date: %w", err)
	}
	return d, nil
}

// decimalOf formats a validated numeric field to places decimals ("" when absent or null).
func decimalOf(data phpval.Map, key string, places int) string {
	v, ok := data.Get(key)
	if !ok || v == nil || phpval.ToString(v) == "" {
		return ""
	}
	return strconv.FormatFloat(phpval.ToFloat(v), 'f', places, 64)
}

// validateScan is PUT /ivf/scans/{date}: follicle counts per ovary ({right|left: {lt_10, 10_14, 15_17, 18_plus}},
// missing bins = 0), endometrium (mm, 1 decimal), E2 (2 decimals, unit pg_ml by default) and notes.
func validateScan(body phpval.Map, locale string, now time.Time) (ScanInput, error) {
	F := validation.F
	data := pick(body, []string{"right", "left", "endometrium_mm", "e2", "e2_unit", "notes"})
	rules := validation.Rules{
		F("right", "nullable", "array"),
		F("left", "nullable", "array"),
		F("endometrium_mm", "nullable", "numeric", "min:0", "max:40"),
		F("e2", "nullable", "numeric", "min:0", "max:999999"),
		F("e2_unit", "nullable", validation.In(E2Units...)),
		F("notes", "nullable", "string", "max:500"),
	}
	for _, side := range []string{"right", "left"} {
		for _, b := range Bins {
			rules = append(rules, F(side+"."+b, "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxFollicles)))
		}
	}
	if err := validate(data, rules, locale, now); err != nil {
		return ScanInput{}, err
	}
	out := ScanInput{
		EndometriumMM: decimalOf(data, "endometrium_mm", 1), E2: decimalOf(data, "e2", 2),
		E2Unit: str(data, "e2_unit"), Notes: str(data, "notes"),
	}
	if out.E2 != "" && out.E2Unit == "" {
		out.E2Unit = E2Units[0]
	}
	if out.E2 == "" {
		out.E2Unit = ""
	}
	for side, dst := range map[string]*Ovary{"right": &out.Right, "left": &out.Left} {
		for i, b := range Bins {
			if v, ok := phpval.Get(data, side+"."+b); ok && v != nil {
				dst[i] = intOf(v)
			}
		}
	}
	return out, nil
}

// validateMood is PUT /ivf/tww/{date} {mood}: a chip, or null to clear.
func validateMood(body phpval.Map, locale string, now time.Time) (string, error) {
	data := pick(body, []string{"mood"})
	if err := validate(data, validation.Rules{
		validation.F("mood", "present", "nullable", validation.In(Moods...)),
	}, locale, now); err != nil {
		return "", err
	}
	return str(data, "mood"), nil
}
