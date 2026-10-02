package postpartum

import (
	"context"
	"math"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/postpartum/guide"
	"github.com/ritme/backend-go/internal/postpartum/store"
)

// National lines of the urgent safety path (existing app copy uses the same: 115 emergency, 123 social emergency,
// 1480 counselling line «صدای مشاور»). The labels are admin copy (guide.SafetyGroup / urgent).
const (
	EmergencyNumber       = "115"
	SocialEmergencyNumber = "123"
	CounselNumber         = "1480"
)

// Alert levels.
const (
	LevelUrgent   = "urgent"
	LevelWarning  = "warning"
	LevelInfo     = "info"
	LevelAdvice   = "advice"
	LevelFollowUp = "follow_up"
)

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func listOrEmpty(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}

func labelOr(key, locale string) any { return strOrNil(line(key, locale)) }

// ProfileJSON is the stored birth.
func ProfileJSON(p *store.PostpartumProfile, locale string) any {
	if p == nil {
		return nil
	}
	dt := p.DeliveryType.String
	var closed any
	if p.PregnancyClosedAt.Valid {
		closed = jsonx.DateTime(p.PregnancyClosedAt.Time)
	}
	return jsonx.Obj(
		"birth_date", p.BirthDate.String(),
		"delivery_type", strOrNil(dt),
		"delivery_type_label", labelOr("delivery_types."+dt, locale),
		"baby_count", int(p.BabyCount),
		"source", p.Source,
		"pregnancy_closed_at", closed,
		"updated_at", jsonx.DateTime(p.UpdatedAt.Time),
	)
}

// StatusJSON is the time since the birth.
func StatusJSON(s guide.Status, locale string) *jsonx.OrderedMap {
	return jsonx.Obj(
		"days_since_birth", s.DaysSinceBirth,
		"weeks", s.Weeks,
		"days", s.Days,
		"week", s.Week,
		"phase", s.Phase,
		"phase_label", labelOr("phases."+s.Phase, locale),
		"puerperium_days", guide.PuerperiumDays,
		"puerperium_days_left", s.PuerperiumDaysLeft,
		"progress", jsonx.Float(math.Round(s.Progress*100)/100),
	)
}

// CheckJSON is one stored check (totals only, never the answers). birth (nil = unknown) adds the week since the
// birth the check was taken in.
func CheckJSON(c Check, birth *civildate.Date, locale string) *jsonx.OrderedMap {
	var week any
	if birth != nil {
		week = guide.StatusOn(*birth, c.TakenOn).Week
	}
	var selfHarm any
	if c.SelfHarm != nil {
		selfHarm = *c.SelfHarm > 0
	}
	band := c.Band()
	return jsonx.Obj(
		"id", c.ID,
		"kind", c.Kind,
		"taken_on", c.TakenOn.String(),
		"week", week,
		"total", c.Total,
		"max", c.Max(),
		"band", band,
		"band_label", labelOr("epds.bands."+band, locale),
		"urgent", c.Urgent,
		"self_harm", selfHarm,
	)
}

// ScheduleJSON is the check-in schedule with the latest check.
func ScheduleJSON(s Schedule, last *Check, birth *civildate.Date, locale string) *jsonx.OrderedMap {
	var lastJSON any
	if last != nil {
		lastJSON = CheckJSON(*last, birth, locale)
	}
	return jsonx.Obj(
		"due", strOrNil(s.Due),
		"next_due_on", s.NextDueOn.String(),
		"last", lastJSON,
	)
}

// RecoveryJSON is one day of the recovery log.
func RecoveryJSON(date civildate.Date, r Recovery, alerts []any) *jsonx.OrderedMap {
	var feeds, sleep, breasts any
	if r.FeedsCount != nil {
		feeds = *r.FeedsCount
	}
	if r.SleepHours != nil {
		sleep = jsonx.Float(*r.SleepHours)
	}
	if r.Breasts != nil {
		breasts = r.Breasts
	}
	return jsonx.Obj(
		"date", date.String(),
		"lochia_amount", strOrNil(r.LochiaAmount),
		"lochia_color", strOrNil(r.LochiaColor),
		"pain_level", strOrNil(r.PainLevel),
		"pain_locations", listOrEmpty(r.PainLocations),
		"breasts", breasts,
		"feeds_count", feeds,
		"sleep_hours", sleep,
		"alerts", alerts,
	)
}

// alertJSON is one alert from guide.AlertGroup.
func alertJSON(ctx context.Context, cp *guide.Copy, key, level string, action any) (*jsonx.OrderedMap, error) {
	t, err := cp.Text(ctx, guide.AlertGroup, key)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"key", key,
		"level", level,
		"title", t["title"],
		"body", t["body"],
		"action_label", strOrNil(t["action"]),
		"action", action,
	), nil
}

func callAction(number string) *jsonx.OrderedMap {
	return jsonx.Obj("type", "call", "number", number)
}

// RecoveryAlerts are the day's recovery alerts (heavy bleeding, large clots) with a call action.
func RecoveryAlerts(ctx context.Context, cp *guide.Copy, r Recovery) ([]any, error) {
	out := []any{}
	for _, k := range r.Alerts() {
		a, err := alertJSON(ctx, cp, k, LevelWarning, callAction(EmergencyNumber))
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// CheckinAlert is the «check-in due» prompt (nil when nothing is due).
func CheckinAlert(ctx context.Context, cp *guide.Copy, s Schedule) (*jsonx.OrderedMap, error) {
	key := ""
	switch s.Due {
	case KindShort:
		key = guide.AlertCheckinShort
	case KindFull:
		key = guide.AlertCheckinFull
	default:
		return nil, nil
	}
	return alertJSON(ctx, cp, key, LevelInfo, jsonx.Obj("type", "open_check", "kind", s.Due))
}

// SafetyJSON is the result message of a scored check (nil for a low result): urgent with the call actions (EPDS item
// 10 > 0 or total ≥ 13), advice (full 10–12), or a follow-up to the full check (short ≥ cutoff).
func SafetyJSON(ctx context.Context, cp *guide.Copy, r Result) (any, error) {
	switch {
	case r.Urgent:
		t, err := cp.Text(ctx, guide.SafetyGroup, guide.SafetyUrgent)
		if err != nil {
			return nil, err
		}
		return jsonx.Obj(
			"level", LevelUrgent,
			"reasons", r.Reasons,
			"title", t["title"],
			"body", t["body"],
			"actions", []*jsonx.OrderedMap{
				jsonx.Obj("type", "call", "number", EmergencyNumber, "label", t["emergency_label"]),
				jsonx.Obj("type", "call", "number", SocialEmergencyNumber, "label", t["social_label"]),
				jsonx.Obj("type", "call", "number", CounselNumber, "label", t["counsel_label"]),
			},
		), nil
	case r.Band == BandPossible:
		t, err := cp.Text(ctx, guide.SafetyGroup, guide.SafetyAdvice)
		if err != nil {
			return nil, err
		}
		return jsonx.Obj("level", LevelAdvice, "reasons", []string{}, "title", t["title"], "body", t["body"],
			"actions", []any{}), nil
	case r.FollowUp == KindFull:
		t, err := cp.Text(ctx, guide.SafetyGroup, guide.SafetyShortElevated)
		if err != nil {
			return nil, err
		}
		return jsonx.Obj("level", LevelFollowUp, "reasons", []string{}, "title", t["title"], "body", t["body"],
			"actions", []*jsonx.OrderedMap{jsonx.Obj("type", "open_check", "kind", KindFull, "label", t["action"])}), nil
	}
	return nil, nil
}

// QuestionsJSON is a check's questionnaire in locale: every item with its options and their scores, in
// questionnaire order (the client sends the score of the chosen option).
func QuestionsJSON(ctx context.Context, cp *guide.Copy, kind, locale string) (*jsonx.OrderedMap, error) {
	disc, err := cp.Text(ctx, guide.SafetyGroup, guide.SafetyDisclaimer)
	if err != nil {
		return nil, err
	}
	items := []*jsonx.OrderedMap{}
	for _, it := range ItemsOf(kind) {
		labels := optionLabels(it.Code, locale)
		opts := make([]*jsonx.OrderedMap, 0, len(it.Scores))
		for i, sc := range it.Scores {
			label := ""
			if i < len(labels) {
				label = labels[i]
			}
			opts = append(opts, jsonx.Obj("score", sc, "label", label))
		}
		items = append(items, jsonx.Obj(
			"code", it.Code,
			"number", itemNumber(it.Code),
			"text", line("epds.items."+it.Code+".text", locale),
			"options", opts,
		))
	}
	return jsonx.Obj(
		"kind", kind,
		"intro", line("epds.intro", locale),
		"disclaimer", disc["body"],
		"max", len(ItemsOf(kind))*ItemMax,
		"items", items,
	), nil
}

func itemNumber(code string) int {
	for i, it := range Items {
		if it.Code == code {
			return i + 1
		}
	}
	return 0
}
