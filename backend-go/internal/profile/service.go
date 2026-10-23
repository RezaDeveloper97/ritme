package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/auth"
	companionstore "github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/files"
	filestore "github.com/ritme/backend-go/internal/files/store"
	"github.com/ritme/backend-go/internal/notify"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/profile/store"
)

// profileFields is the $request->only([...]) list of ProfileController::store.
var profileFields = []string{
	"birthday", "weight", "height", "period_duration", "cycle_duration", "last_period_start",
	"user_goal", "subscription_type", "pregnancy_intention", "chronic_conditions",
}

// cycleFields is hasCycleFieldsChanged's list.
var cycleFields = []string{"birthday", "period_duration", "cycle_duration", "last_period_start"}

// DBError marks a failure of a database statement (Laravel's QueryException).
type DBError struct{ Err error }

func (e *DBError) Error() string { return e.Err.Error() }
func (e *DBError) Unwrap() error { return e.Err }

func dbErr(what string, err error) error {
	return &DBError{Err: fmt.Errorf("profile: %s: %w", what, err)}
}

// Service holds the write side of ProfileController.
type Service struct {
	DB          *sql.DB
	Q           *store.Queries
	Telegram    *notify.Telegram
	StoragePath string // STORAGE_PATH: support-report screenshots are removed with the account
	Logger      *slog.Logger

	tx *sql.Tx // set on a Service bound to a transaction by EngineTx (Q runs on it); nil otherwise
}

// SaveResult is what POST /profile answers with.
type SaveResult struct {
	User    auth.User
	Profile store.UserProfile
}

// Save is the body of ProfileController::store after validation. input is $request->all()
// (validated); now is the request clock (Carbon::now()). The profile write runs in one transaction that bumps the
// engine revision (EngineTx, T-M9-30); on a Service already bound to a transaction it joins that one.
func (s *Service) Save(ctx context.Context, u *auth.User, input phpval.Map, now time.Time) (*SaveResult, error) {
	if err := s.updateName(ctx, u, input, now); err != nil {
		return nil, err
	}
	var res *SaveResult
	err := s.EngineTx(ctx, u.ID, func(t *Service) error {
		r, err := t.saveProfile(ctx, u, input, now)
		res = r
		return err
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// EngineTx runs fn on a Service bound to one transaction and bumps userID's engine revision before the commit
// (T-M9-30: user_profiles, user_life_profiles and cycle_preferences are engine inputs), so one write request moves
// the revision by exactly 1 however many of those rows it writes. On a Service that is already bound to a
// transaction fn runs on it and the transaction's owner bumps. A transaction InnoDB rolled back as a deadlock (1213:
// racing first profile inserts on the UNIQUE user_id, T-M9-23 / D-127) runs again, up to firstSaveAttempts times,
// so fn must be safe to re-run.
func (s *Service) EngineTx(ctx context.Context, userID uint64, fn func(t *Service) error) error {
	if s.tx != nil {
		return fn(s)
	}
	for attempt := 1; ; attempt++ {
		err := s.engineTxOnce(ctx, userID, fn)
		if err == nil || attempt >= firstSaveAttempts || !isDeadlock(err) {
			return err
		}
	}
}

func (s *Service) engineTxOnce(ctx context.Context, userID uint64, fn func(t *Service) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return dbErr("begin", err)
	}
	defer func() { _ = tx.Rollback() }()
	// The per-user lock first (D-131), as every other engine-input writer takes it: concurrent saves of one user
	// (double tap, racing first inserts) queue here instead of deadlocking on the profile row and the revision row.
	if _, err := filestore.New(tx).LockOwner(ctx, userID); err != nil {
		return dbErr("lock user", err)
	}
	t := *s
	t.Q, t.tx = s.Q.WithTx(tx), tx
	if err := fn(&t); err != nil {
		return err
	}
	if err := revision.Bump(ctx, tx, userID); err != nil {
		return dbErr("bump revision", err)
	}
	if err := tx.Commit(); err != nil {
		return dbErr("commit", err)
	}
	return nil
}

// saveProfile is Save's profile write on s (bound to the transaction).
func (s *Service) saveProfile(ctx context.Context, u *auth.User, input phpval.Map, now time.Time) (*SaveResult, error) {
	current, err := s.Q.GetProfileByUserID(ctx, u.ID)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, dbErr("load profile", err)
	}
	var orig *store.UserProfile
	if exists {
		orig = &current
	}

	// $request->only([...]): present keys, null values included.
	data := map[string]any{}
	for _, f := range profileFields {
		if v, ok := input.Get(f); ok {
			data[f] = v
		}
	}

	// user_goal follows the stated intention unless the caller set it (only "trying" is TTC).
	if pi, has := data["pregnancy_intention"]; has {
		if _, hasGoal := data["user_goal"]; !hasGoal {
			if pi == string(enums.PregnancyIntentionTrying) {
				data["user_goal"] = string(enums.UserGoalTtc)
			} else {
				data["user_goal"] = string(enums.UserGoalNonTtc)
			}
		}
	}

	// Period tracking is off in pregnancy mode. D-24 (T-M2-34): unlike Laravel, a missing
	// last_period_start is NOT defaulted to today — that fabricated a confirmed period the cycle
	// engine treated as real. The cycle history stays empty until the user logs a period.
	intention := data["pregnancy_intention"]
	if intention == nil && orig != nil && orig.PregnancyIntention.Valid {
		intention = orig.PregnancyIntention.String
	}
	isPregnant := intention == string(enums.PregnancyIntentionPregnant)

	changed := cycleFieldsChanged(orig, data)
	declared := declaredLMPChanged(orig, data)

	id, err := s.writeOnce(ctx, u.ID, orig, data, now)
	if err != nil {
		return nil, err
	}

	fresh, err := s.Q.GetProfileByID(ctx, id)
	if err != nil {
		return nil, dbErr("reload profile", err)
	}
	// T-M9-27 (I-01, D-123): the declared LMP is the engine's onboarding anchor, stored in its own column — no
	// cycle_histories row any more (the old seed stored an assumed end as real, §2.2/§2.40, and fed the medians,
	// §2.9). last_period_start keeps its legacy meaning (latest actual start) for the legacy readers. Same scope as
	// the old seed: not in pregnancy mode (period tracking is off there).
	if !isPregnant && declared && fresh.OnboardingLastPeriodStart != fresh.LastPeriodStart {
		if err := s.Q.SetOnboardingLastPeriodStart(ctx, store.SetOnboardingLastPeriodStartParams{
			UserID: u.ID, OnboardingLastPeriodStart: fresh.LastPeriodStart, Now: dbTime(now),
		}); err != nil {
			return nil, dbErr("set declared last period", err)
		}
		fresh.OnboardingLastPeriodStart = fresh.LastPeriodStart
	}
	if changed && fresh.LastPeriodStart.Valid {
		if err := MarkRecalculated(ctx, s.Q, fresh.ID, now); err != nil {
			return nil, err
		}
		if fresh, err = s.Q.GetProfileByID(ctx, id); err != nil {
			return nil, dbErr("reload profile", err)
		}
	}

	user, err := s.Q.GetUser(ctx, u.ID)
	if err != nil {
		return nil, dbErr("reload user", err)
	}
	return &SaveResult{User: auth.User(user), Profile: fresh}, nil
}

// DeleteAccount deletes the user's refresh and access tokens, then the user (cascading to
// every user-owned table), in one transaction (D-25).
func (s *Service) DeleteAccount(ctx context.Context, userID uint64) (err error) {
	shots, err := s.Q.ListUserSupportScreenshots(ctx, userID)
	if err != nil {
		return fmt.Errorf("profile: delete account: screenshots: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("profile: delete account: begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	q := s.Q.WithTx(tx)
	uid := sql.NullInt64{Int64: int64(userID), Valid: true} //nolint:gosec // G115: ids fit int64
	if err = q.DeleteUserRefreshTokens(ctx, uid); err != nil {
		return fmt.Errorf("profile: delete refresh tokens: %w", err)
	}
	if err = q.DeleteUserAccessTokens(ctx, uid); err != nil {
		return fmt.Errorf("profile: delete access tokens: %w", err)
	}
	// CB-LOSS-01: the pregnancy notices her companions got live in their inboxes (no FK to her): remove them.
	if err = companionstore.New(tx).DeletePregnancyNoticesForOwner(ctx, userID); err != nil {
		return fmt.Errorf("profile: delete companion notices: %w", err)
	}
	if _, err = q.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("profile: delete user: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("profile: delete account: commit: %w", err)
	}
	RemoveSupportFiles(s.StoragePath, shots, s.Logger)
	// B-N6-06 / CB-CORE-05: the encrypted files of every purpose, lab sheets included (their rows went with the user).
	if err := files.RemoveUser(s.StoragePath, userID); err != nil && s.Logger != nil {
		s.Logger.ErrorContext(ctx, "profile: delete account: files", slog.String("error", err.Error()))
	}
	return nil
}

// MarkRecalculated is UserProfile::markRecalculated(): calculation_version + 1 (one atomic
// UPDATE), status completed, started/completed/updated_at = now.
func MarkRecalculated(ctx context.Context, q interface {
	MarkProfileRecalculated(context.Context, store.MarkProfileRecalculatedParams) error
}, profileID uint64, now time.Time) error {
	if err := q.MarkProfileRecalculated(ctx, store.MarkProfileRecalculatedParams{Now: dbTime(now), ID: profileID}); err != nil {
		return dbErr("mark recalculated", err)
	}
	return nil
}

// updateName is `$user->update(['name' => $request->name])` + the Telegram notice.
func (s *Service) updateName(ctx context.Context, u *auth.User, input phpval.Map, now time.Time) error {
	v, has := input.Get("name")
	if !has {
		return nil
	}
	var prev any
	if u.Name.Valid {
		prev = u.Name.String
	}
	if !attributeDirty(prev, v, "") {
		return nil
	}
	name := sql.NullString{}
	if v != nil {
		name = sql.NullString{String: phpval.ToString(v), Valid: true}
	}
	if err := s.Q.UpdateUserName(ctx, store.UpdateUserNameParams{Name: name, UpdatedAt: dbTime(now), ID: u.ID}); err != nil {
		return dbErr("update name", err)
	}
	if prev != v {
		s.Telegram.SendAsync(NameChangedNotice(u.ID, prev, v), nil)
	}
	return nil
}

// NameChangedNotice is ProfileController::notifyNameChanged's message.
func NameChangedNotice(userID uint64, previous, current any) string {
	show := func(v any) string {
		if v == nil {
			return "—"
		}
		return notify.Escape(phpval.ToString(v))
	}
	return strings.Join([]string{
		"✏️ <b>تغییر نام کاربر</b>",
		fmt.Sprintf("شناسه: <code>%d</code>", userID),
		"قبلی: " + show(previous),
		"جدید: " + show(current),
	}, "\n")
}

// firstSaveAttempts bounds the runs of a transaction that hit an InnoDB deadlock (T-M9-23).
const firstSaveAttempts = 5

// isDeadlock is MySQL 1213 (ER_LOCK_DEADLOCK): InnoDB rolled the transaction back.
func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1213
}

// writeOnce is $profile->fill($data)->save() on s (bound to Save's transaction): a new profile is inserted (DB
// defaults, then the attributes — a rejected value rolls the whole save back, as a failed INSERT leaves no row); an
// existing one gets an UPDATE of its dirty attributes only, and none when nothing changed.
//
// T-M9-23 (D-127): user_id is UNIQUE and InsertProfile is an upsert, so concurrent first saves all land on one row.
// Racing inserts on the same unique key can still deadlock in InnoDB (1213); EngineTx runs the loser's save again,
// which re-reads the profile.
func (s *Service) writeOnce(ctx context.Context, userID uint64, orig *store.UserProfile, data map[string]any, now time.Time) (uint64, error) {
	params, dirty := updateParams(orig, data)
	params.UpdatedAt = dbTime(now)
	if orig != nil {
		if dirty {
			params.ID = orig.ID
			if err := s.Q.UpdateProfileAttributes(ctx, params); err != nil {
				return 0, dbErr("update profile", err)
			}
		}
		return orig.ID, nil
	}

	q := s.Q
	res, err := q.InsertProfile(ctx, store.InsertProfileParams{UserID: userID, CreatedAt: dbTime(now), UpdatedAt: dbTime(now)})
	if err != nil {
		return 0, dbErr("insert profile", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, dbErr("insert profile", err)
	}
	params.ID = uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	if dirty {
		if err := q.UpdateProfileAttributes(ctx, params); err != nil {
			return 0, dbErr("insert profile", err)
		}
	}
	return params.ID, nil
}

// updateParams builds the conditional UPDATE: set_x is true for every dirty attribute
// (all present ones for a new model), bound as the PDO value Laravel would send.
func updateParams(orig *store.UserProfile, data map[string]any) (store.UpdateProfileAttributesParams, bool) {
	var p store.UpdateProfileAttributesParams
	anyDirty := false
	for _, f := range profileFields {
		v, has := data[f]
		if !has {
			continue
		}
		set := orig == nil || attributeDirty(originalAttribute(orig, f), v, model.UserProfileCasts[f])
		anyDirty = anyDirty || set
		bound := bindValue(f, v)
		switch f {
		case "birthday":
			p.SetBirthday, p.Birthday = set, bound
		case "weight":
			p.SetWeight, p.Weight = set, bound
		case "height":
			p.SetHeight, p.Height = set, bound
		case "period_duration":
			p.SetPeriodDuration, p.PeriodDuration = set, bound
		case "cycle_duration":
			p.SetCycleDuration, p.CycleDuration = set, bound
		case "last_period_start":
			p.SetLastPeriodStart, p.LastPeriodStart = set, bound
		case "user_goal":
			p.SetUserGoal, p.UserGoal = set, bound
		case "subscription_type":
			p.SetSubscriptionType, p.SubscriptionType = set, bound
		case "pregnancy_intention":
			p.SetPregnancyIntention, p.PregnancyIntention = set, bound
		case "chronic_conditions":
			p.SetChronicConditions, p.ChronicConditions = set, bound
		}
	}
	for _, f := range []*any{&p.SetBirthday, &p.SetWeight, &p.SetHeight, &p.SetPeriodDuration,
		&p.SetCycleDuration, &p.SetLastPeriodStart, &p.SetUserGoal, &p.SetSubscriptionType,
		&p.SetPregnancyIntention, &p.SetChronicConditions} {
		if *f == nil {
			*f = false
		}
	}
	return p, anyDirty
}

// bindValue is the value PDO receives: the array cast json_encodes, floats go as PHP's
// (string) cast, everything else as is (MariaDB converts strings for DATE/DECIMAL columns).
func bindValue(field string, v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case bool:
		if x {
			return int64(1)
		}
		return int64(0)
	case float64:
		return phpround.String(x)
	}
	if field == "chronic_conditions" {
		b, err := jsonx.Marshal(phpval.Packed(v), 0)
		if err != nil {
			return nil
		}
		return string(b)
	}
	if phpval.IsArray(v) { // cannot pass validation; keep PDO's "Array" conversion
		return "Array"
	}
	return v
}

// originalAttribute is the raw original attribute (as PDO returned it), nil for NULL.
func originalAttribute(p *store.UserProfile, field string) any {
	switch field {
	case "birthday":
		return nullDate(p.Birthday)
	case "weight":
		if p.Weight.Valid {
			return p.Weight.String
		}
	case "height":
		if p.Height.Valid {
			return int64(p.Height.Int16)
		}
	case "period_duration":
		if p.PeriodDuration.Valid {
			return int64(p.PeriodDuration.Int16)
		}
	case "cycle_duration":
		if p.CycleDuration.Valid {
			return int64(p.CycleDuration.Int16)
		}
	case "last_period_start":
		return nullDate(p.LastPeriodStart)
	case "user_goal":
		return p.UserGoal
	case "subscription_type":
		return p.SubscriptionType
	case "pregnancy_intention":
		if p.PregnancyIntention.Valid {
			return p.PregnancyIntention.String
		}
	case "chronic_conditions":
		if p.ChronicConditions.Valid {
			return string(p.ChronicConditions.V)
		}
	}
	return nil
}

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

// attributeDirty is !Model::originalIsEquivalent for an attribute with the given cast.
func attributeDirty(original, value any, cast model.Cast) bool {
	if original == value {
		return false
	}
	if value == nil {
		return true
	}
	if original == nil && cast != model.CastArray {
		return true
	}
	switch cast {
	case model.CastDateYMD:
		a, okA := parseDateTime(value)
		b, okB := parseDateTime(original)
		return !okA || !okB || !a.Equal(b)
	case model.CastFloat:
		d := phpval.ToFloat(value) - phpval.ToFloat(original)
		return d >= 4*2.220446049250313e-16 || d <= -4*2.220446049250313e-16
	case model.CastInteger:
		return int64(phpval.ToFloat(value)) != int64(phpval.ToFloat(original))
	case model.CastArray:
		var o []byte
		if s, ok := original.(string); ok {
			o, _ = jsonx.Marshal(model.DecodeJSONColumn([]byte(s)), 0)
		} else {
			o = []byte("null")
		}
		n, _ := jsonx.Marshal(phpval.Packed(value), 0)
		return string(o) != string(n)
	}
	// Uncast: numeric strings that print the same are equal.
	if phpval.IsNumeric(value) && phpval.IsNumeric(original) {
		return phpval.ToString(value) != phpval.ToString(original)
	}
	return true
}

// parseDateTime is fromDateTime() for comparison: a Y-m-d string is midnight, anything else
// goes through the lenient Carbon::parse port.
func parseDateTime(v any) (time.Time, bool) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, false
	}
	if d, err := civildate.Parse(s); err == nil {
		return d.TehranMidnight(), true
	}
	t, err := civildate.ParseLenient(s, time.Now(), civildate.Tehran)
	return t, err == nil
}

// cycleFieldsChanged is hasCycleFieldsChanged: a present non-null value loosely (!=)
// different from the original cast value (dates as Y-m-d).
func cycleFieldsChanged(orig *store.UserProfile, data map[string]any) bool {
	for _, f := range cycleFields {
		v := data[f]
		if v == nil {
			continue
		}
		var old any
		if orig != nil {
			old = originalAttribute(orig, f)
		}
		if !phpval.LooseEqual(old, v) {
			return true
		}
	}
	return false
}

// declaredLMPChanged: the request sets last_period_start to a new value (a first value included) — the user
// declared her last period (onboarding cycle step or the profile form). Re-sending the stored value is no declaration.
func declaredLMPChanged(orig *store.UserProfile, data map[string]any) bool {
	v, has := data["last_period_start"]
	if !has {
		return false
	}
	if orig == nil {
		return v != nil
	}
	return attributeDirty(originalAttribute(orig, "last_period_start"), v, model.UserProfileCasts["last_period_start"])
}

// dbTime is a Carbon value as Eloquent writes it: 'Y-m-d H:i:s' in Asia/Tehran.
func dbTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}
