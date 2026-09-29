package v2

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Sources are the dating sources of the Setup screen.
var Sources = []string{"lmp", "ultrasound", "manual"}

var previewFields = []string{"source", "lmp_date", "ultrasound_date", "ultrasound_weeks", "ultrasound_days", "manual_weeks", "manual_days"}

func previewRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("source", "required", validation.In(Sources...)),
		F("lmp_date", "required_if:source,lmp", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("ultrasound_date", "required_if:source,ultrasound", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("ultrasound_weeks", "required_if:source,ultrasound", "nullable", "integer", "min:0", "max:42"),
		F("ultrasound_days", "required_if:source,ultrasound", "nullable", "integer", "min:0", "max:6"),
		F("manual_weeks", "required_if:source,manual", "nullable", "integer", "min:0", "max:42"),
		F("manual_days", "required_if:source,manual", "nullable", "integer", "min:0", "max:6"),
	}
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// previewProfile validates the body into an unsaved profile (manual entry = today).
func previewProfile(body phpval.Map, locale string, now time.Time) (*store.PregnancyProfile, error) {
	data := phpval.NewMap()
	for _, k := range previewFields {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, previewRules(),
		validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return nil, failValidation(locale, v.ErrorBag())
	}
	get := func(k string) string {
		x, _ := data.Get(k)
		if x == nil {
			return ""
		}
		return phpval.ToString(x)
	}
	date := func(k string) (civildate.NullDate, error) {
		d, err := civildate.Parse(get(k))
		if err != nil {
			return civildate.NullDate{}, fmt.Errorf("pregnancy v2: %s: %w", k, err)
		}
		return civildate.NullDate{Date: d, Valid: true}, nil
	}
	integer := func(k string) sql.NullInt32 {
		n, _ := strconv.Atoi(get(k))
		return sql.NullInt32{Int32: int32(n), Valid: true} //nolint:gosec // validated 0..42
	}
	src := get("source")
	p := &store.PregnancyProfile{PregnancyMode: true, AgeSource: sql.NullString{String: src, Valid: true}}
	var err error
	switch src {
	case "lmp":
		p.LmpDate, err = date("lmp_date")
	case "ultrasound":
		p.UltrasoundDate, err = date("ultrasound_date")
		p.UltrasoundWeeks, p.UltrasoundDays = integer("ultrasound_weeks"), integer("ultrasound_days")
	case "manual":
		p.ManualEntryDate = civildate.NullDate{Date: civildate.InTehran(now), Valid: true}
		p.ManualWeeks, p.ManualDays = integer("manual_weeks"), integer("manual_days")
	}
	return p, err
}

// Preview is POST /pregnancy/v2/dating-preview: nothing is written. Two queries at most (the
// pregnancy_setup/result template rows).
func (s *Service) Preview(ctx context.Context, body phpval.Map, now time.Time, l Lang) (*jsonx.OrderedMap, error) {
	p, err := previewProfile(body, l.Locale, now)
	if err != nil {
		return nil, err
	}
	today := civildate.InTehran(now)
	d, ok := Resolve(p, today)
	if !ok {
		return nil, fmt.Errorf("pregnancy v2: preview: undated profile")
	}
	if d.TotalDays > MaxWeek*7+6 || d.TotalDays < 0 {
		return nil, failValidation(l.Locale, jsonx.Obj(previewDateField(d.Source),
			[]string{T("validation.dating_out_of_range", l.Locale)}))
	}
	rows, err := s.q.ListV2MessagePayloads(ctx, store.ListV2MessagePayloadsParams{
		MessageGroup: "pregnancy_setup", ItemKey: "result", Locales: l.codes(),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: setup copy: %w", err)
	}
	tpl := payload(rows, l)
	rFrom, rTo := d.BirthRange()
	var basis, rangeText any
	if s := str(tpl, "basis_"+d.Source); s != "" {
		basis = fill(s, "date", FullDate(d.BasisDate, l.Locale))
	}
	if s := str(tpl, "range"); s != "" {
		// Day + month only: the due date line above carries the year (design audit A9).
		rangeText = fill(s, "range_from", DayMonth(rFrom, l.Locale), "range_to", DayMonth(rTo, l.Locale))
	}
	resultCopy := jsonx.Obj()
	for _, k := range resultCopyKeys {
		v := str(tpl, k)
		resultCopy.Set(k, nullStr(v, v != ""))
	}
	return jsonx.Obj(
		"source", d.Source,
		"weeks", d.Weeks,
		"days", d.Days,
		"week", d.CurrentWeek(),
		"trimester", d.Trimester(),
		"age_label", ageLabel(d, l.Locale),
		"due_date", d.Due.String(),
		"due_date_label", FullDate(d.Due, l.Locale),
		"range", rangeJSON(rFrom, rTo),
		"range_label", rangeText,
		"uncertainty_days", d.Uncertainty,
		"confidence", confidenceJSON(d.Confidence, l.Locale),
		"basis", basis,
		"copy", resultCopy,
	), nil
}

// resultCopyKeys are the admin-edited pregnancy_setup/result texts the result step shows as they
// are (the basis and range templates are filled above); null when the row lacks one.
var resultCopyKeys = []string{"lead", "suffix", "due_label", "confidence", "primary", "secondary"}

func previewDateField(source string) string {
	switch source {
	case "lmp":
		return "lmp_date"
	case "ultrasound":
		return "ultrasound_weeks"
	}
	return "manual_weeks"
}

// ageLabel is «۸ هفته و ۳ روز».
func ageLabel(d Dating, locale string) string { return AgeLabel(d.Weeks, d.Days, locale) }

// AgeLabel is a gestational age «۸ هفته و ۳ روز» (or «۸ هفته» when days is 0).
func AgeLabel(weeks, days int, locale string) string {
	if days == 0 {
		return tr("age.weeks", locale, "weeks", num(weeks, locale))
	}
	return tr("age.weeks_days", locale, "weeks", num(weeks, locale), "days", num(days, locale))
}
