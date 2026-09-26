package pregnancy

import (
	"context"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/alerts"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Bridge for the v2 day log (T-M7-03): the v2 API writes the v1 symptom and weekly logs
// through the same upsert + alert rules as POST /pregnancy/symptoms and /pregnancy/weekly,
// so v1 readers and the v1 alert side effects stay identical.

// RaisedAlert is one pregnancy_alerts row created by a save.
type RaisedAlert struct {
	ID        int64
	Level     string // v1 alert_level: info | warning | emergency
	Type      string
	Title     string
	Message   string
	Week      int
	Actions   []string
	CreatedAt time.Time
}

func raise(ctx context.Context, q store.Querier, userID uint64, locale string, now time.Time,
	rules func(alerts.Context) []alerts.Draft,
) ([]RaisedAlert, error) {
	p, err := LoadProfile(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	cl := calc.New(p, locale, civildate.InTehran(now))
	drafts := rules(alerts.Context{Locale: locale, Week: cl.CurrentWeek(), Profile: cl.Profile()})
	created, err := createAlerts(ctx, q, userID, drafts, now)
	if err != nil {
		return nil, err
	}
	out := make([]RaisedAlert, len(created))
	for i, m := range created {
		d := drafts[i]
		id, _ := m.raw("id").(int64)
		out[i] = RaisedAlert{
			ID: id, Level: string(d.Level), Type: d.Type, Title: d.Title, Message: d.Message,
			Week: d.Week, Actions: d.Actions, CreatedAt: dbNow(now),
		}
	}
	return out, nil
}

// SaveSymptomDay is PregnancySymptomLog::updateOrCreate(user_id + log_date, attrs) followed by
// the v1 symptom alert rules. attrs holds has_<symptom> / <symptom>_severity columns; keys
// that are absent keep their stored value.
func SaveSymptomDay(ctx context.Context, q store.Querier, userID uint64, date civildate.Date,
	attrs phpval.Map, locale string, now time.Time,
) ([]RaisedAlert, error) {
	log, err := upsertSymptomLog(ctx, q, userID, date, date.String(), attrs, now)
	if err != nil {
		return nil, err
	}
	return raise(ctx, q, userID, locale, now, func(c alerts.Context) []alerts.Draft { return alerts.ForSymptomLog(c, log) })
}

// SaveWeeklyWeight is PregnancyWeeklyLog::updateOrCreate(user_id + pregnancy_week,
// {log_date, weight}) followed by the v1 weekly alert rules. weight nil clears it.
func SaveWeeklyWeight(ctx context.Context, q store.Querier, userID uint64, week int, date civildate.Date,
	weight any, locale string, now time.Time,
) ([]RaisedAlert, error) {
	attrs := phpval.NewMap()
	attrs.Set("log_date", date.String())
	attrs.Set("pregnancy_week", int64(week))
	attrs.Set("weight", weight)
	log, err := upsertWeeklyLog(ctx, q, userID, int32(week), int64(week), attrs, now) //nolint:gosec // G115: 1..42
	if err != nil {
		return nil, err
	}
	return raise(ctx, q, userID, locale, now, func(c alerts.Context) []alerts.Draft { return alerts.ForWeeklyLog(c, log) })
}
