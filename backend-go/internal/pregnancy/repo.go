package pregnancy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/alerts"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// LoadProfile is $user->pregnancyProfile: nil (and no error) when the user has none.
func LoadProfile(ctx context.Context, q store.Querier, userID uint64) (*store.PregnancyProfile, error) {
	p, err := q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pregnancy: load profile: %w", err)
	}
	return &p, nil
}

// fill overlays attrs (in order) onto m; it reports whether any value changed the way
// Eloquent's isDirty() sees it (compared after the write conversions).
func fill(m *model, attrs *jsonx.OrderedMap) {
	for _, k := range attrs.Keys() {
		v, _ := attrs.Get(k)
		m.set(k, v)
	}
}

// saveProfile is PregnancyProfile::updateOrCreate(['user_id' => …], attrs) / $profile->update(attrs):
// a missing profile is inserted, an existing one is updated only when something changed
// (updated_at is bumped only then, as Eloquent's save() does).
func saveProfile(ctx context.Context, q store.Querier, userID uint64, existing *store.PregnancyProfile,
	attrs *jsonx.OrderedMap, now time.Time,
) error {
	ts := dbNow(now)
	if existing == nil {
		m := profileDefaults(userID)
		fill(m, attrs)
		m.set("created_at", ts).set("updated_at", ts)
		params, err := profileInsertParams(m)
		if err != nil {
			return err
		}
		if _, err := q.InsertProfile(ctx, params); err != nil {
			return fmt.Errorf("pregnancy: insert profile: %w", err)
		}
		return nil
	}
	before, err := profileUpdateParams(profileModel(existing))
	if err != nil {
		return err
	}
	m := profileModel(existing)
	fill(m, attrs)
	after, err := profileUpdateParams(m)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(before, after) {
		return nil
	}
	after.UpdatedAt = sql.NullTime{Time: ts, Valid: true}
	if err := q.UpdateProfile(ctx, after); err != nil {
		return fmt.Errorf("pregnancy: update profile: %w", err)
	}
	return nil
}

// upsert is Model::updateOrCreate($keys, $validated) for the log tables. It returns the
// model as Laravel holds it afterwards: the created model carries only the set attributes
// (+ timestamps + id); an updated one the full row with the new values.
type upsert struct {
	casts   casts
	find    func() (*model, error)                   // nil model when no row matches
	insert  func(m *model) (int64, error)            // → last insert id
	changed func(before, after *model) (bool, error) // isDirty
	update  func(m *model) error                     // writes m (updated_at already set)
	keys    *jsonx.OrderedMap                        // the where attributes
	attrs   phpval.Map                               // $validated
	now     time.Time
}

func (u upsert) run() (*model, error) {
	ts := dbNow(u.now)
	existing, err := u.find()
	if err != nil {
		return nil, err
	}
	if existing == nil {
		m := newModel(u.casts)
		fill(m, u.keys)
		fill(m, u.attrs)
		m.set("updated_at", ts).set("created_at", ts)
		id, err := u.insert(m)
		if err != nil {
			return nil, err
		}
		m.set("id", id)
		return m, nil
	}
	before := newModel(u.casts)
	fill(before, existing.attrs)
	fill(existing, u.attrs)
	dirty, err := u.changed(before, existing)
	if err != nil {
		return nil, err
	}
	if dirty {
		existing.set("updated_at", ts)
		if err := u.update(existing); err != nil {
			return nil, err
		}
	}
	return existing, nil
}

func noRows[T any](row T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// upsertSymptomLog is PregnancySymptomLog::updateOrCreate(user_id + log_date).
func upsertSymptomLog(ctx context.Context, q store.Querier, userID uint64, logDate civildate.Date,
	rawDate any, attrs phpval.Map, now time.Time,
) (*model, error) {
	return upsert{
		casts: symptomCasts,
		find: func() (*model, error) {
			r, err := noRows(q.GetSymptomLog(ctx, store.GetSymptomLogParams{UserID: userID, LogDate: logDate}))
			if r == nil || err != nil {
				return nil, err
			}
			return symptomModel(r), nil
		},
		insert: func(m *model) (int64, error) {
			return q.InsertSymptomLog(ctx, symptomInsertParams(m, userID))
		},
		changed: func(a, b *model) (bool, error) {
			pa, pb := symptomUpdateParams(a), symptomUpdateParams(b)
			pb.UpdatedAt = pa.UpdatedAt
			return !reflect.DeepEqual(pa, pb), nil
		},
		update: func(m *model) error { return q.UpdateSymptomLog(ctx, symptomUpdateParams(m)) },
		keys:   jsonx.Obj("user_id", int64(userID), "log_date", rawDate), //nolint:gosec // G115
		attrs:  attrs,
		now:    now,
	}.run()
}

// upsertWeeklyLog is PregnancyWeeklyLog::updateOrCreate(user_id + pregnancy_week).
func upsertWeeklyLog(ctx context.Context, q store.Querier, userID uint64, week int32, rawWeek any,
	attrs phpval.Map, now time.Time,
) (*model, error) {
	return upsert{
		casts: weeklyCasts,
		find: func() (*model, error) {
			r, err := noRows(q.GetWeeklyLog(ctx, store.GetWeeklyLogParams{UserID: userID, PregnancyWeek: week}))
			if r == nil || err != nil {
				return nil, err
			}
			return weeklyModel(r), nil
		},
		insert: func(m *model) (int64, error) {
			p, err := weeklyInsertParams(m, userID)
			if err != nil {
				return 0, err
			}
			return q.InsertWeeklyLog(ctx, p)
		},
		changed: func(a, b *model) (bool, error) {
			pa, err := weeklyUpdateParams(a)
			if err != nil {
				return false, err
			}
			pb, err := weeklyUpdateParams(b)
			if err != nil {
				return false, err
			}
			pb.UpdatedAt = pa.UpdatedAt
			return !reflect.DeepEqual(pa, pb), nil
		},
		update: func(m *model) error {
			p, err := weeklyUpdateParams(m)
			if err != nil {
				return err
			}
			return q.UpdateWeeklyLog(ctx, p)
		},
		keys:  jsonx.Obj("user_id", int64(userID), "pregnancy_week", rawWeek), //nolint:gosec // G115
		attrs: attrs,
		now:   now,
	}.run()
}

// upsertFetalMovement is PregnancyFetalMovement::updateOrCreate(user_id + log_date).
func upsertFetalMovement(ctx context.Context, q store.Querier, userID uint64, logDate civildate.Date,
	rawDate any, attrs phpval.Map, now time.Time,
) (*model, error) {
	return upsert{
		casts: fetalCasts,
		find: func() (*model, error) {
			r, err := noRows(q.GetFetalMovement(ctx, store.GetFetalMovementParams{UserID: userID, LogDate: logDate}))
			if r == nil || err != nil {
				return nil, err
			}
			return fetalModel(r), nil
		},
		insert: func(m *model) (int64, error) {
			p, err := fetalInsertParams(m, userID)
			if err != nil {
				return 0, err
			}
			return q.InsertFetalMovement(ctx, p)
		},
		changed: func(a, b *model) (bool, error) {
			pa, err := fetalUpdateParams(a)
			if err != nil {
				return false, err
			}
			pb, err := fetalUpdateParams(b)
			if err != nil {
				return false, err
			}
			pb.UpdatedAt = pa.UpdatedAt
			return !reflect.DeepEqual(pa, pb), nil
		},
		update: func(m *model) error {
			p, err := fetalUpdateParams(m)
			if err != nil {
				return err
			}
			return q.UpdateFetalMovement(ctx, p)
		},
		keys:  jsonx.Obj("user_id", int64(userID), "log_date", rawDate), //nolint:gosec // G115
		attrs: attrs,
		now:   now,
	}.run()
}

// createAlerts is PregnancyAlert::create() for each draft; the returned models carry only
// the attributes of the create array (+ timestamps + id), like Laravel's fresh models.
func createAlerts(ctx context.Context, q store.Querier, userID uint64, drafts []alerts.Draft, now time.Time) ([]*model, error) {
	ts := dbNow(now)
	out := make([]*model, 0, len(drafts))
	for _, d := range drafts {
		m := newModel(alertCasts).
			set("user_id", int64(userID)). //nolint:gosec // G115
			set("alert_level", string(d.Level)).
			set("alert_type", d.Type).
			set("title", d.Title).
			set("message", d.Message).
			set("pregnancy_week", int64(d.Week))
		if d.Trigger != nil {
			m.set("trigger_symptoms", d.Trigger)
		}
		if d.MedicalFlags != nil {
			m.set("medical_history_flags", d.MedicalFlags)
		}
		actions := make([]any, len(d.Actions))
		for i, a := range d.Actions {
			actions[i] = a
		}
		m.set("recommended_actions", actions).set("updated_at", ts).set("created_at", ts)
		id, err := q.InsertAlert(ctx, store.InsertAlertParams{
			UserID:              userID,
			AlertLevel:          string(d.Level),
			AlertType:           d.Type,
			Title:               d.Title,
			Message:             d.Message,
			PregnancyWeek:       wInt(int64(d.Week)),
			TriggerSymptoms:     wJSON(m.raw("trigger_symptoms")),
			MedicalHistoryFlags: wJSON(m.raw("medical_history_flags")),
			RecommendedActions:  wJSON(actions),
			CreatedAt:           sql.NullTime{Time: ts, Valid: true},
			UpdatedAt:           sql.NullTime{Time: ts, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("pregnancy: insert alert: %w", err)
		}
		m.set("id", id)
		out = append(out, m)
	}
	return out, nil
}
