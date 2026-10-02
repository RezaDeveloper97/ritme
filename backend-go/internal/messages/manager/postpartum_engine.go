package manager

import (
	"context"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// The postpartum engine (bloom B-N5-01, deviations.md D-54): Laravel has no engine for the postpartum mode (its
// MessageResult::empty()); Go serves the week's tip since the birth (admin message_contents postpartum_week_tip with
// the embedded fallback of internal/postpartum/guide) and, as the override, the most pressing alert of the day —
// heavy bleeding, large clots, then a due mood check-in. No correlations, patterns or supplements.

// PostpartumState is what the postpartum engine reads about the user on the asked day.
type PostpartumState struct {
	BirthDate *civildate.Date // nil = no birth stored (mode chosen without one)
	// Alerts are guide.AlertGroup keys raised by the day's recovery log (heavy_bleeding, large_clots).
	Alerts []string
	// CheckinDue is the due EPDS check ("short" | "full" | "").
	CheckinDue string
}

// PostpartumSource is an optional Source extension: the postpartum state on date (nil = none).
type PostpartumSource interface {
	Postpartum(ctx context.Context, date civildate.Date) (*PostpartumState, error)
}

// postpartumCall is the number the bleeding alerts offer (Iran emergency).
const postpartumCall = "115"

func (m *Manager) postpartum(ctx context.Context, date civildate.Date) (*Result, error) {
	var st PostpartumState
	if ps, ok := m.src.(PostpartumSource); ok {
		s, err := ps.Postpartum(ctx, date)
		if err != nil {
			return nil, err
		}
		if s != nil {
			st = *s
		}
	}
	profile, err := m.src.Profile(ctx)
	if err != nil {
		return nil, err
	}
	res := &Result{Mode: enums.MessageModePostpartum, Date: date, UserGoal: string(enums.UserGoalNonTtc), SubscriptionType: "free"}
	if profile != nil && profile.SubscriptionType != "" {
		res.SubscriptionType = profile.SubscriptionType
	}
	payloads, _ := m.content.(guide.Payloader)
	cp := guide.NewCopy(payloads, m.locale, "")

	tipKey := guide.LateTipKey
	var week, days, since, weeks any
	phase := any(nil)
	if st.BirthDate != nil {
		s := guide.StatusOn(*st.BirthDate, date)
		tipKey = guide.TipKey(s.Week)
		week, days, since, weeks, phase = s.Week, s.Days, s.DaysSinceBirth, s.Weeks, s.Phase
		res.ContextInfo = jsonx.Obj(
			"birth_date", st.BirthDate.String(),
			"days_since_birth", since,
			"weeks", weeks,
			"days", days,
			"week", week,
			"phase", phase,
		)
	}
	tip, err := cp.Text(ctx, guide.TipGroup, tipKey)
	if err != nil {
		return nil, err
	}
	primary := jsonx.Obj(
		"week", week,
		"day", days,
		"days_since_birth", since,
		"phase", phase,
		"short_message", tip["title"],
		"long_message", tip["body"],
		"action_suggestion", "",
		"dos", emptyList(),
		"donts", emptyList(),
		"week_tip", jsonx.Obj("week", week, "key", tipKey, "title", tip["title"], "body", tip["body"]),
	)
	over, err := postpartumOverride(ctx, cp, st)
	if err != nil {
		return nil, err
	}
	if over != nil {
		for _, k := range over.Keys() {
			v, _ := over.Get(k)
			primary.Set(k, v)
		}
		primary.Set("has_override", true)
	} else {
		primary.Set("has_override", false)
	}
	res.PrimaryMessage = primary
	return res, nil
}

// postpartumOverride is the day's most pressing alert (nil when none).
func postpartumOverride(ctx context.Context, cp *guide.Copy, st PostpartumState) (*jsonx.OrderedMap, error) {
	key, level := "", ""
	var call any
	for _, k := range []string{guide.AlertHeavyBleeding, guide.AlertLargeClots} {
		for _, a := range st.Alerts {
			if a == k && key == "" {
				key, level, call = k, "warning", postpartumCall
			}
		}
	}
	if key == "" {
		switch st.CheckinDue {
		case "full":
			key, level = guide.AlertCheckinFull, "info"
		case "short":
			key, level = guide.AlertCheckinShort, "info"
		default:
			return nil, nil
		}
	}
	t, err := cp.Text(ctx, guide.AlertGroup, key)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"override_type", key,
		"alert_level", level,
		"short_message", t["title"],
		"long_message", t["body"],
		"action_suggestion", t["action"],
		"call_number", call,
	), nil
}
