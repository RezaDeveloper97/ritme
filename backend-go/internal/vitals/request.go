package vitals

import (
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

// dateTimeFormats are the accepted measurement times (Tehran wall-clock), like /children/{id}/feeds.
const dateTimeFormats = "date_format:Y-m-d H:i:s,Y-m-d H:i"

// MaxListDays caps GET /vitals/readings?from&to.
const MaxListDays = 366

// DefaultListDays is GET /vitals/readings without from.
const DefaultListDays = 30

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func pick(body phpval.Map, keys []string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func intOf(data phpval.Map, key string) int {
	s := str(data, key)
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return int(phpval.ToFloat(s))
}

func between(lo, hi int) string { return "between:" + strconv.Itoa(lo) + "," + strconv.Itoa(hi) }

var readingKeys = []string{
	"type", "measured_at", "systolic", "diastolic", "pulse", "arm", "position", "value", "unit", "context", "method",
	"bpm", "note",
}

// ValidateReading is the body of POST /vitals/readings and PUT /vitals/readings/{id}:
//
//	{type: bp, systolic, diastolic, pulse?, arm?, position?, measured_at?, note?}
//	{type: glucose, value, unit: mg_dl|mmol_l, context, method?, measured_at?, note?}
//	{type: hr, bpm, context, measured_at?, note?}
//
// measured_at (Y-m-d H:i[:s], Tehran) defaults to now; not in the future, not more than two years back.
func ValidateReading(body phpval.Map, locale string, now time.Time) (Input, error) {
	data := pick(body, readingKeys)
	typ := str(data, "type")
	rules := validation.Rules{
		validation.F("type", "required", validation.In(Types...)),
		validation.F("measured_at", "nullable", dateTimeFormats),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxNoteLen)),
	}
	switch typ {
	case TypeBP:
		rules = append(rules,
			validation.F("systolic", "required", "integer", between(MinSystolic, MaxSystolic)),
			validation.F("diastolic", "required", "integer", between(MinDiastolic, MaxDiastolic)),
			validation.F("pulse", "nullable", "integer", between(MinPulse, MaxPulse)),
			validation.F("arm", "nullable", validation.In(Arms...)),
			validation.F("position", "nullable", validation.In(Positions...)),
		)
	case TypeGlucose:
		rules = append(rules,
			validation.F("value", "required", "numeric"),
			validation.F("unit", "required", validation.In(GlucoseUnits...)),
			validation.F("context", "required", validation.In(GlucoseContexts...)),
			validation.F("method", "nullable", validation.In(GlucoseMethods...)),
		)
	case TypeHR:
		rules = append(rules,
			validation.F("bpm", "required", "integer", between(MinPulse, MaxPulse)),
			validation.F("context", "required", validation.In(HRContexts...)),
		)
	}
	if err := validate(data, rules, locale, now); err != nil {
		return Input{}, err
	}
	in := Input{Type: typ, MeasuredAt: now, Note: str(data, "note")}
	if s := str(data, "measured_at"); s != "" {
		for _, layout := range []string{time.DateTime, "2006-01-02 15:04"} {
			if t, err := time.ParseInLocation(layout, s, civildate.Tehran); err == nil {
				in.MeasuredAt = t
				break
			}
		}
		if in.MeasuredAt.After(now.Add(time.Minute)) {
			return Input{}, fieldFail(locale, "measured_at", "future")
		}
		if in.MeasuredAt.Before(now.AddDate(0, 0, -MaxBackfillDays)) {
			return Input{}, fieldFail(locale, "measured_at", "too_old")
		}
	}
	switch typ {
	case TypeBP:
		in.Systolic, in.Diastolic, in.Pulse = intOf(data, "systolic"), intOf(data, "diastolic"), intOf(data, "pulse")
		in.Arm, in.Position = str(data, "arm"), str(data, "position")
		if in.Systolic <= in.Diastolic {
			return Input{}, fieldFail(locale, "systolic", "systolic_gt_diastolic")
		}
	case TypeGlucose:
		in.Unit, in.Context, in.Method = str(data, "unit"), str(data, "context"), str(data, "method")
		v := phpval.ToFloat(str(data, "value"))
		if in.Unit == UnitMmolL && (v < MinMmolL || v > MaxMmolL) {
			return Input{}, fieldFail(locale, "value", "glucose_range_mmol_l")
		}
		if in.Unit == UnitMgDl && (v < MinMgDl || v > MaxMgDl) {
			return Input{}, fieldFail(locale, "value", "glucose_range_mg_dl")
		}
		in.MgDl = MgDlFrom(v, in.Unit)
	case TypeHR:
		in.Pulse, in.Context = intOf(data, "bpm"), str(data, "context")
	}
	return in, nil
}

// ListQuery is GET /vitals/readings?type&from&to.
type ListQuery struct {
	Type     string
	From, To civildate.Date
}

// ValidateList checks the list query (default: the last 30 days of every type; ≤ 366 days).
func ValidateList(q phpval.Map, locale string, now time.Time) (ListQuery, error) {
	data := pick(q, []string{"type", "from", "to"})
	if err := validate(data, validation.Rules{
		validation.F("type", "nullable", validation.In(Types...)),
		validation.F("from", "nullable", "date_format:Y-m-d"),
		validation.F("to", "nullable", "date_format:Y-m-d"),
	}, locale, now); err != nil {
		return ListQuery{}, err
	}
	today := civildate.InTehran(now)
	out := ListQuery{Type: str(data, "type"), To: today}
	if s := str(data, "to"); s != "" {
		out.To = civildate.MustParse(s)
	}
	out.From = out.To.AddDays(1 - DefaultListDays)
	if s := str(data, "from"); s != "" {
		out.From = civildate.MustParse(s)
	}
	if out.To.Before(out.From) {
		return ListQuery{}, fieldFail(locale, "to", "to_before_from")
	}
	if out.From.DiffDays(out.To)+1 > MaxListDays {
		return ListQuery{}, fieldFail(locale, "from", "range_too_long")
	}
	return out, nil
}

// ReportQuery is GET /vitals/reports/{type}?range&filter.
type ReportQuery struct{ Range, Filter string }

// ValidateReport checks the report query (range default per type; glucose filter all|<context>).
func ValidateReport(typ string, q phpval.Map, locale string, now time.Time) (ReportQuery, error) {
	data := pick(q, []string{"range", "filter"})
	rules := validation.Rules{validation.F("range", "nullable", validation.In(RangeKeys...))}
	if typ == TypeGlucose {
		rules = append(rules, validation.F("filter", "nullable", validation.In(append([]string{GlucoseFilterAll}, GlucoseContexts...)...)))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return ReportQuery{}, err
	}
	out := ReportQuery{Range: str(data, "range"), Filter: str(data, "filter")}
	if out.Range == "" {
		out.Range = DefaultRange[typ]
	}
	if typ == TypeGlucose && out.Filter == "" {
		out.Filter = GlucoseFilterAll
	}
	return out, nil
}

// elems are the values of a decoded JSON list (a []any or a PHP-style Map).
func elems(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case phpval.Map:
		out := make([]any, 0, x.Len())
		for _, k := range x.Keys() {
			e, _ := x.Get(k)
			out = append(out, e)
		}
		return out
	}
	return nil
}

// allSlots is every slot of every type (the In list of items.*.slot).
func allSlots() []string {
	var out []string
	for _, t := range Types {
		for _, s := range PlanSlots[t] {
			if !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	}
	return out
}

// ValidatePlan is PUT /vitals/plan {items: [{type, slot, days: [0..6], remind_at: "HH:MM"|null}]} (≤ 8 items, one per
// type + slot; days 0 = Saturday; an empty list clears the plan).
func ValidatePlan(body phpval.Map, locale string, now time.Time) ([]PlanItem, error) {
	data := pick(body, []string{"items"})
	if err := validate(data, validation.Rules{
		validation.F("items", "present", "array", "max:"+strconv.Itoa(MaxPlanItems)),
		validation.F("items.*", "required", "array"),
		validation.F("items.*.type", "required", validation.In(Types...)),
		validation.F("items.*.slot", "required", validation.In(allSlots()...)),
		validation.F("items.*.days", "required", "array", "min:1", "max:7"),
		validation.F("items.*.days.*", "required", "integer", between(0, 6)),
		validation.F("items.*.remind_at", "nullable", "date_format:H:i"),
	}, locale, now); err != nil {
		return nil, err
	}
	raw, _ := data.Get("items")
	list := elems(raw)
	items := make([]PlanItem, 0, len(list))
	seen := map[string]bool{}
	for i, v := range list {
		m, _ := v.(phpval.Map)
		if m == nil {
			m = phpval.NewMap()
		}
		it := PlanItem{Type: str(m, "type"), Slot: str(m, "slot"), RemindAt: str(m, "remind_at")}
		field := "items." + strconv.Itoa(i) + ".slot"
		if !validSlot(it.Type, it.Slot) {
			return nil, fieldFail(locale, field, "slot_mismatch")
		}
		if seen[it.Type+"|"+it.Slot] {
			return nil, fieldFail(locale, field, "duplicate_slot")
		}
		seen[it.Type+"|"+it.Slot] = true
		dv, _ := m.Get("days")
		for _, d := range elems(dv) {
			it.Days |= 1 << (int(phpval.ToFloat(phpval.ToString(d))) % 7)
		}
		items = append(items, it)
	}
	return items, nil
}
