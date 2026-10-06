package menopause

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var itemFields = []string{
	"name", "dose", "schedule", "form", "started_on", "review_on", "stopped_on", "weekly_goal", "goal_unit", "remind",
}

func dateOf(data phpval.Map, key string) (civildate.Date, error) {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return civildate.Date{}, nil
	}
	return parseDate(v)
}

func strOf(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

// validateItem is POST /menopause/treatment/items (kind = "": taken from the body) or PUT …/{id} (kind = the item's;
// a full replace of the other fields). hrt / supplement need a schedule; lifestyle needs weekly_goal + goal_unit.
func validateItem(body phpval.Map, kind, locale string, now time.Time) (ItemInput, error) {
	F := validation.F
	data := phpval.NewMap()
	keys := itemFields
	rules := validation.Rules{}
	if kind == "" {
		keys = append([]string{"kind"}, itemFields...)
		rules = append(rules, F("kind", "required", "string", validation.In(Kinds...)))
		if v, ok := body.Get("kind"); ok {
			kind = phpval.ToString(v)
		}
	} else {
		data.Set("kind", kind)
	}
	schedule := []any{"nullable", "string", validation.In(Schedules...)}
	goal := []any{"nullable", "integer", "min:1", "max:" + strconv.Itoa(MaxGoalMinutes)}
	unit := []any{"nullable", "string", validation.In(GoalUnits...)}
	switch kind {
	case KindLifestyle:
		goal[0], unit[0] = "required", "required"
	case KindHRT, KindSupplement:
		schedule[0] = "required"
	}
	rules = append(rules,
		F("name", "required", "string", "max:"+strconv.Itoa(MaxNameLen)),
		F("dose", "nullable", "string", "max:"+strconv.Itoa(MaxDoseLen)),
		F("schedule", schedule...),
		F("form", "nullable", "string", validation.In(care.Forms...)),
		F("started_on", "nullable", "date_format:Y-m-d"),
		F("review_on", "nullable", "date_format:Y-m-d"),
		F("stopped_on", "nullable", "date_format:Y-m-d", "before_or_equal:today"),
		F("weekly_goal", goal...),
		F("goal_unit", unit...),
		F("remind", "nullable", "boolean"),
	)
	data, err := validate(data, body, keys, rules, locale, now)
	if err != nil {
		return ItemInput{}, err
	}
	in := ItemInput{
		Kind: kind, Name: strOf(data, "name"), Dose: strOf(data, "dose"), Remind: true,
	}
	if in.Name == "" {
		return ItemInput{}, fieldFail(locale, "name", "name_blank")
	}
	if in.StartedOn, err = dateOf(data, "started_on"); err != nil {
		return ItemInput{}, err
	}
	if in.ReviewOn, err = dateOf(data, "review_on"); err != nil {
		return ItemInput{}, err
	}
	if in.StoppedOn, err = dateOf(data, "stopped_on"); err != nil {
		return ItemInput{}, err
	}
	if !in.StartedOn.IsZero() && !in.StoppedOn.IsZero() && in.StoppedOn.Before(in.StartedOn) {
		return ItemInput{}, fieldFail(locale, "stopped_on", "stopped_before_start")
	}
	if !in.StartedOn.IsZero() && !in.ReviewOn.IsZero() && in.ReviewOn.Before(in.StartedOn) {
		return ItemInput{}, fieldFail(locale, "review_on", "review_before_start")
	}
	if kind == KindLifestyle {
		in.GoalUnit = strOf(data, "goal_unit")
		v, _ := data.Get("weekly_goal")
		in.WeeklyGoal = intOf(v)
		if in.GoalUnit == UnitSessions && in.WeeklyGoal > MaxGoalSessions {
			return ItemInput{}, fieldFail(locale, "weekly_goal", "goal_too_many_sessions")
		}
		return in, nil
	}
	in.Schedule = strOf(data, "schedule")
	in.Form = strOf(data, "form")
	if v, ok := data.Get("remind"); ok && v != nil {
		in.Remind = boolOf(v)
	}
	return in, nil
}

// parseLogDay validates the {date} segment of an intake or side-effect route: Y-m-d, not in the future, at most
// IntakeBackfillDays back (422 on the `date` field).
func parseLogDay(raw, locale string, now time.Time) (civildate.Date, error) {
	body := phpval.NewMap()
	body.Set("date", raw)
	if _, err := validate(phpval.NewMap(), body, []string{"date"}, validation.Rules{
		validation.F("date", "required", "date_format:Y-m-d", "before_or_equal:today"),
	}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	d, err := civildate.Parse(raw)
	if err != nil {
		return civildate.Date{}, fieldFail(locale, "date", "date_future")
	}
	if d.Before(civildate.InTehran(now).AddDays(-IntakeBackfillDays)) {
		return civildate.Date{}, fieldFail(locale, "date", "date_too_old")
	}
	return d, nil
}

// validateIntake is PUT …/intakes/{date} {amount?}: lifestyle minutes need the minutes (1–MaxIntakeMinutes);
// sessions may send a count (1–MaxIntakeSessions, default one session); other kinds ignore amount.
func validateIntake(body phpval.Map, it ItemKind, locale string, now time.Time) (int, error) {
	if it.Kind != KindLifestyle {
		return 0, nil
	}
	maxAmount, need := MaxIntakeSessions, "nullable"
	if it.Unit == UnitMinutes {
		maxAmount, need = MaxIntakeMinutes, "required"
	}
	data, err := validate(phpval.NewMap(), body, []string{"amount"}, validation.Rules{
		validation.F("amount", need, "integer", "min:1", "max:"+strconv.Itoa(maxAmount)),
	}, locale, now)
	if err != nil {
		return 0, err
	}
	if v, ok := data.Get("amount"); ok && v != nil {
		return intOf(v), nil
	}
	return 0, nil
}

// ItemKind is what intake validation needs of an item.
type ItemKind struct{ Kind, Unit string }

// validateSideEffects is PUT …/side-effects/{date} {codes: [...], treatment_item_id?}: codes from SideEffectCodes
// (empty clears the day).
func validateSideEffects(body phpval.Map, locale string, now time.Time) (SideEffectsInput, error) {
	F := validation.F
	data, err := validate(phpval.NewMap(), body, []string{"codes", "treatment_item_id"}, validation.Rules{
		F("codes", "present", "array", "max:"+strconv.Itoa(len(SideEffectCodes))),
		F("codes.*", "required", "string", validation.In(SideEffectCodes...)),
		F("treatment_item_id", "nullable", "integer", "min:1"),
	}, locale, now)
	if err != nil {
		return SideEffectsInput{}, err
	}
	v, _ := data.Get("codes")
	in := SideEffectsInput{Codes: listOf(v, SideEffectCodes)}
	if id, ok := data.Get("treatment_item_id"); ok && id != nil {
		in.ItemID = uint64(max(intOf(id), 0)) //nolint:gosec // min:1
	}
	return in, nil
}

// validateReportMonths is ?months= of GET /menopause/report (ReportMonths, default DefaultReportMonths).
func validateReportMonths(query phpval.Map, locale string, now time.Time) (int, error) {
	allowed := make([]string, len(ReportMonths))
	for i, m := range ReportMonths {
		allowed[i] = strconv.Itoa(m)
	}
	data, err := validate(phpval.NewMap(), query, []string{"months"}, validation.Rules{
		validation.F("months", "nullable", "integer", validation.In(allowed...)),
	}, locale, now)
	if err != nil {
		return 0, err
	}
	if v, ok := data.Get("months"); ok && v != nil && slices.Contains(allowed, phpval.ToString(v)) {
		return intOf(v), nil
	}
	return DefaultReportMonths, nil
}
