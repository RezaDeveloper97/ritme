package pregnancy

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Bridge for the v2 day log (T-M7-03): the v2 API writes the v1 symptom and weekly logs through the same upsert as
// POST /pregnancy/symptoms and /pregnancy/weekly, so v1 readers stay identical. The alerts of every save come from the
// pregnancy alert rules of the message engine (T-M10-08, D-165: the alerts inbox; the v1 rules are retired).

// RaisedAlert is one alert a save raised (an alerts inbox row in the v1 alert shape).
type RaisedAlert struct {
	ID        int64
	Level     string // v1 alert_level: info | warning | emergency
	Type      string // the v2 rule key (vomiting_streak, critical_symptom, bp_high, …)
	Title     string
	Message   string
	Week      int
	Actions   []string
	CreatedAt time.Time
	// Trigger is the evidence in the v1 terms (trigger_symptoms; nil = not set).
	Trigger *jsonx.OrderedMap
}

// raiseAlerts evaluates the user's pregnancy alert rules after a save (pregnancyalerts → the alerts inbox, the same
// evaluation as inbox.EvaluateNow over the preg.* rules) and returns the alerts the save raised or re-raised: v2 alert
// maps, newest rule first. Best-effort for the save (D-21: the log is committed): a failed evaluation is logged and
// answers no alerts; the next read of the alerts evaluates again. Not pregnant → none.
func raiseAlerts(ctx context.Context, q store.Querier, userID uint64, locale, defaultLocale string, now time.Time) []RaisedAlert {
	if defaultLocale == "" {
		defaultLocale = locale
	}
	list, err := pregnancyalerts.New(q).EvaluateV1(ctx, q, userID, now, v2.Lang{Locale: locale, Default: defaultLocale})
	switch {
	case errors.Is(err, pregnancyalerts.ErrNotActive):
		return nil
	case err != nil:
		slog.WarnContext(ctx, "pregnancy: alert evaluation failed after a log save",
			slog.Uint64("user_id", userID), slog.String("error", err.Error()))
		return nil
	}
	out := make([]RaisedAlert, 0, len(list))
	for _, a := range list {
		out = append(out, raisedOf(a, now))
	}
	return out
}

// raisedOf reads one v2 alert map (pregnancyalerts' alert shape).
func raisedOf(a *jsonx.OrderedMap, now time.Time) RaisedAlert {
	str := func(k string) string { v, _ := a.Get(k); s, _ := v.(string); return s }
	r := RaisedAlert{Level: str("alert_level"), Type: str("rule_key"), Title: str("title"), Message: str("advice"),
		CreatedAt: dbNow(now)}
	if v, _ := a.Get("id"); v != nil {
		if id, ok := v.(uint64); ok {
			r.ID = int64(id) //nolint:gosec // G115: auto-increment id
		}
	}
	if v, _ := a.Get("pregnancy_week"); v != nil {
		r.Week, _ = v.(int)
	}
	if v, _ := a.Get("created_at"); v != nil {
		if s, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				r.CreatedAt = dbNow(t)
			}
		}
	}
	if v, _ := a.Get("trigger_symptoms"); v != nil {
		r.Trigger, _ = v.(*jsonx.OrderedMap)
	}
	if v, _ := a.Get("actions"); v != nil {
		list, _ := v.([]any)
		for _, x := range list {
			if m, ok := x.(*jsonx.OrderedMap); ok {
				if l, _ := m.Get("label"); l != nil {
					s, _ := l.(string)
					r.Actions = append(r.Actions, s)
				}
			}
		}
	}
	return r
}

// v1Alerts are the raised alerts as v1 alert models (the `alerts` array of POST /pregnancy/symptoms, /weekly and
// /fetal-movement: the keys PregnancyAlert::create() returned; ids are message_inbox ids, D-165). flags is the
// medical_history_flags of a symptom save (nil = not part of the alert, as for the weekly and fetal-movement saves).
func v1Alerts(userID uint64, raised []RaisedAlert, flags *jsonx.OrderedMap) []*model {
	out := make([]*model, 0, len(raised))
	for _, r := range raised {
		actions := make([]any, len(r.Actions))
		for i, a := range r.Actions {
			actions[i] = a
		}
		m := newModel(alertCasts).
			set("user_id", int64(userID)). //nolint:gosec // G115
			set("alert_level", r.Level).
			set("alert_type", r.Type).
			set("title", r.Title).
			set("message", r.Message).
			set("pregnancy_week", int64(r.Week))
		if r.Trigger != nil {
			m.set("trigger_symptoms", r.Trigger)
		}
		if flags != nil {
			m.set("medical_history_flags", flags)
		}
		out = append(out, m.set("recommended_actions", actions).
			set("updated_at", r.CreatedAt).
			set("created_at", r.CreatedAt).
			set("id", r.ID))
	}
	return out
}

// SaveSymptomDay is PregnancySymptomLog::updateOrCreate(user_id + log_date, attrs). It raises no alerts itself: the
// day log evaluates the pregnancy alert rules once for the whole save, in its transaction. attrs holds has_<symptom> / <symptom>_severity columns; keys
// that are absent keep their stored value.
func SaveSymptomDay(ctx context.Context, q store.Querier, userID uint64, date civildate.Date,
	attrs phpval.Map, locale string, now time.Time,
) ([]RaisedAlert, error) {
	if _, err := upsertSymptomLog(ctx, q, userID, date, date.String(), attrs, now); err != nil {
		return nil, err
	}
	return nil, nil
}

// SaveWeeklyWeight is PregnancyWeeklyLog::updateOrCreate(user_id + pregnancy_week,
// {log_date, weight}) (no alerts: the day log evaluates the
// pregnancy alert rules once for the whole save, in its transaction). weight nil clears it.
func SaveWeeklyWeight(ctx context.Context, q store.Querier, userID uint64, week int, date civildate.Date,
	weight any, locale string, now time.Time,
) ([]RaisedAlert, error) {
	attrs := phpval.NewMap()
	attrs.Set("log_date", date.String())
	attrs.Set("pregnancy_week", int64(week))
	attrs.Set("weight", weight)
	if _, err := upsertWeeklyLog(ctx, q, userID, int32(week), int64(week), attrs, now); err != nil { //nolint:gosec // G115: 1..42
		return nil, err
	}
	return nil, nil
}
