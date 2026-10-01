package contraception

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/contraception/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Errors of the pill log.
var (
	ErrNoPillMethod = errors.New("contraception: no pill method")
	ErrBreakDay     = errors.New("contraception: break day")
	ErrBeforePack   = errors.New("contraception: before the pack start")
)

// Conn is the database the service writes through in one transaction (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service reads and writes the user's method, pill log and method reminders.
type Service struct {
	conn Conn
}

// NewService returns a Service on conn.
func NewService(conn Conn) *Service { return &Service{conn: conn} }

// PillReminder is the B-N1-09 pill reminder: the `pill` notification category and its time (minutes, Tehran).
type PillReminder struct {
	Enabled bool
	Minute  int
}

// LinkedReminder is a care reminder the method created, as it is now (the user may have edited it).
type LinkedReminder struct {
	Kind string
	Row  store.Reminder
}

// Overview is GET /contraception: the switch, the method (nil when none) and what hangs off it.
type Overview struct {
	Today     civildate.Date
	Tracking  bool
	Method    *Method
	Reminder  PillReminder
	Logs      Logs
	Reminders []LinkedReminder
}

func tehranNow(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// Overview loads the contraception screen as of today.
func (s *Service) Overview(ctx context.Context, userID uint64, today civildate.Date) (Overview, error) {
	q := store.New(s.conn)
	o := Overview{Today: today, Logs: NewLogs(civildate.Date{}), Reminders: []LinkedReminder{}}
	var err error
	if o.Tracking, err = q.GetTrackContraception(ctx, userID); err != nil {
		return Overview{}, fmt.Errorf("contraception: load switch: %w", err)
	}
	prefs, err := notifications.Load(ctx, profilestore.New(s.conn), userID)
	if err != nil {
		return Overview{}, err
	}
	minute, _ := prefs.TimeOf(notifications.Pill)
	o.Reminder = PillReminder{Enabled: prefs.Enabled(notifications.Pill), Minute: minute}

	row, err := q.GetMethod(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return o, nil
	}
	if err != nil {
		return Overview{}, fmt.Errorf("contraception: load method: %w", err)
	}
	m := methodFromRow(row)
	o.Method = &m
	if row.CreatedAt.Valid {
		o.Logs.Since = civildate.InTehran(row.CreatedAt.Time)
	}
	if pack, ok := m.Pack(); ok {
		from := pack.PackStart(m.PackStartedOn, today)
		if lookback := today.AddDays(-StreakWindow); lookback.Before(from) {
			from = lookback
		}
		if from.Before(m.PackStartedOn) {
			from = m.PackStartedOn
		}
		logs, err := q.ListPillLogs(ctx, store.ListPillLogsParams{
			UserID: userID, FromDate: from, ToDate: pack.PackStart(m.PackStartedOn, today).AddDays(pack.Length - 1),
		})
		if err != nil {
			return Overview{}, fmt.Errorf("contraception: load pill logs: %w", err)
		}
		for _, l := range logs {
			o.Logs.Days[l.LogDate] = l.Status
		}
	}
	links, err := q.ListReminderLinks(ctx, userID)
	if err != nil {
		return Overview{}, fmt.Errorf("contraception: load reminders: %w", err)
	}
	for _, l := range links {
		o.Reminders = append(o.Reminders, LinkedReminder{Kind: l.Kind, Row: l.Reminder})
	}
	return o, nil
}

// Input is a validated PUT /contraception/method.
type Input struct {
	Method           string
	PackType         string
	PackStartedOn    civildate.Date
	PacksLeft        *int
	ReminderMinute   *int  // nil = keep the B-N1-09 time
	ReminderEnabled  *bool // nil = on
	InsertedOn       civildate.Date
	IUDLifetimeYears int
	FollowupDone     bool
	InjectedOn       civildate.Date
	ReplaceOn        civildate.Date
}

// methodFrom keeps only the fields of the chosen method; a pack count is dated to the pack in use today.
func methodFrom(in Input, today civildate.Date) Method {
	m := Method{Method: in.Method}
	switch {
	case IsPill(in.Method):
		m.PackStartedOn = in.PackStartedOn
		if in.Method == MethodCombinedPill {
			m.PackType = in.PackType
		}
		if pack, ok := m.Pack(); ok && in.PacksLeft != nil {
			n := *in.PacksLeft
			m.PacksLeft = &n
			m.PacksCountedOn = pack.PackStart(m.PackStartedOn, today)
		}
	case IsIUD(in.Method):
		m.InsertedOn = in.InsertedOn
		m.IUDLifetimeYears = in.IUDLifetimeYears
		m.FollowupDone = in.FollowupDone
	case in.Method == MethodInjection:
		m.InjectedOn = in.InjectedOn
	case in.Method == MethodImplant:
		m.InsertedOn = in.InsertedOn
		m.ReplaceOn = in.ReplaceOn
	}
	return m
}

// SaveMethod stores the method and, in the same transaction, switches «track contraception» on, sets the one pill
// reminder (on with its time for a pill method, off otherwise) and syncs the method's care reminders.
func (s *Service) SaveMethod(ctx context.Context, userID uint64, in Input, now time.Time, locale string) error {
	today := civildate.InTehran(now)
	m := methodFrom(in, today)
	return s.inTx(ctx, func(q *store.Queries, pq *profilestore.Queries) error {
		if err := q.UpsertMethod(ctx, m.params(userID, tehranNow(now))); err != nil {
			return fmt.Errorf("contraception: save method: %w", err)
		}
		if err := q.SetTrackContraception(ctx, store.SetTrackContraceptionParams{
			UserID: userID, TrackContraception: true, Now: tehranNow(now),
		}); err != nil {
			return fmt.Errorf("contraception: switch on: %w", err)
		}
		on := IsPill(m.Method) && (in.ReminderEnabled == nil || *in.ReminderEnabled)
		if err := setPillReminder(ctx, pq, userID, on, in.ReminderMinute, now); err != nil {
			return err
		}
		return syncReminders(ctx, q, userID, Plan(m, today), now, locale)
	})
}

// Stop is DELETE /contraception/method: the method and its care reminders go, the switch and the pill reminder
// are turned off. The pill log stays (the user's own history).
func (s *Service) Stop(ctx context.Context, userID uint64, now time.Time) error {
	return s.inTx(ctx, func(q *store.Queries, pq *profilestore.Queries) error {
		if err := q.DeleteMethod(ctx, userID); err != nil {
			return fmt.Errorf("contraception: delete method: %w", err)
		}
		if err := q.SetTrackContraception(ctx, store.SetTrackContraceptionParams{
			UserID: userID, TrackContraception: false, Now: tehranNow(now),
		}); err != nil {
			return fmt.Errorf("contraception: switch off: %w", err)
		}
		if err := setPillReminder(ctx, pq, userID, false, nil, now); err != nil {
			return err
		}
		return syncReminders(ctx, q, userID, nil, now, "")
	})
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries, pq *profilestore.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("contraception: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(store.New(tx), profilestore.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("contraception: commit: %w", err)
	}
	return nil
}

// setPillReminder writes the B-N1-09 pill category (and its time when minute is set) — the one pill schedule the
// cycle-settings screen edits too. Quiet hours and neutral copy are kept.
func setPillReminder(ctx context.Context, pq *profilestore.Queries, userID uint64, on bool, minute *int, now time.Time) error {
	cur, err := notifications.Load(ctx, pq, userID)
	if err != nil {
		return err
	}
	if cur.Enabled(notifications.Pill) == on && minute == nil {
		return nil
	}
	cur.Categories[notifications.Pill] = on
	if minute != nil {
		cur.Schedule.Times[notifications.Pill] = *minute
	}
	if err := pq.UpsertReminderPreferences(ctx, profilestore.UpsertReminderPreferencesParams{
		UserID:            userID,
		Categories:        db.NullRawJSON{V: cur.CategoriesJSON(), Valid: true},
		Schedule:          db.NullRawJSON{V: cur.ScheduleJSON(), Valid: true},
		QuietHoursEnabled: cur.QuietEnabled,
		QuietStart:        notifications.DBClock(cur.QuietStart),
		QuietEnd:          notifications.DBClock(cur.QuietEnd),
		NeutralCopy:       cur.NeutralCopy,
		Now:               tehranNow(now),
	}); err != nil {
		return fmt.Errorf("contraception: save pill reminder: %w", err)
	}
	return nil
}

// syncReminders makes the linked care reminders match specs: a kind no longer wanted is deleted, a kept one gets
// its new date (the user's switch, notes and appointment details stay), a new one is created and linked.
func syncReminders(ctx context.Context, q *store.Queries, userID uint64, specs []Spec, now time.Time, locale string) error {
	links, err := q.ListReminderLinks(ctx, userID)
	if err != nil {
		return fmt.Errorf("contraception: load reminders: %w", err)
	}
	want := make(map[string]Spec, len(specs))
	for _, s := range specs {
		want[s.Kind] = s
	}
	have := map[string]bool{}
	at := tehranNow(now)
	for _, l := range links {
		s, ok := want[l.Kind]
		if !ok {
			if err := q.DeleteReminder(ctx, store.DeleteReminderParams{ID: l.Reminder.ID, UserID: userID}); err != nil {
				return fmt.Errorf("contraception: delete reminder: %w", err)
			}
			continue
		}
		have[l.Kind] = true
		c := s.Columns()
		if err := q.UpdateReminderSchedule(ctx, store.UpdateReminderScheduleParams{
			Title: reminderTitle(s.Kind, locale, l.Reminder.Title), ScheduledAt: nullTime(c.ScheduledAt),
			Recurrence: c.Recurrence, RecurrenceTime: nullString(c.RecurrenceTime), StartsOn: nullDatePtr(c.StartsOn),
			Now: at, ID: l.Reminder.ID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("contraception: update reminder: %w", err)
		}
	}
	for _, s := range specs {
		if have[s.Kind] {
			continue
		}
		c := s.Columns()
		params := store.InsertReminderParams{
			UserID: userID, Type: s.Type, Title: reminderTitle(s.Kind, locale, ""), ScheduledAt: nullTime(c.ScheduledAt),
			Recurrence: c.Recurrence, RecurrenceTime: nullString(c.RecurrenceTime), StartsOn: nullDatePtr(c.StartsOn),
			Now: at,
		}
		if c.Meta != nil {
			params.Meta = db.NullRawJSON{V: c.Meta, Valid: true}
		}
		id, err := q.InsertReminder(ctx, params)
		if err != nil {
			return fmt.Errorf("contraception: insert reminder: %w", err)
		}
		if err := q.InsertReminderLink(ctx, store.InsertReminderLinkParams{
			UserID: userID, Kind: s.Kind, ReminderID: uint64(id), Now: at, //nolint:gosec // G115: auto-increment id
		}); err != nil {
			return fmt.Errorf("contraception: link reminder: %w", err)
		}
	}
	return nil
}

// reminderTitle is the kind's title in locale; without a locale (stop) the current title is kept.
func reminderTitle(kind, locale, current string) string {
	if locale == "" {
		return current
	}
	return T("reminders."+kind, locale)
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullDatePtr(d *civildate.Date) civildate.NullDate {
	if d == nil {
		return civildate.NullDate{}
	}
	return civildate.NullDate{Date: *d, Valid: true}
}

// LogPill records date as taken / missed. The method must be a pill whose schedule has begun on date and date
// must not be a break day.
func (s *Service) LogPill(ctx context.Context, userID uint64, date civildate.Date, status string, now time.Time) error {
	q := store.New(s.conn)
	pack, start, err := s.pillPack(ctx, q, userID)
	if err != nil {
		return err
	}
	if date.Before(start) {
		return ErrBeforePack
	}
	if pack.DayKind(start, date) == KindBreak {
		return ErrBreakDay
	}
	if err := q.UpsertPillLog(ctx, store.UpsertPillLogParams{
		UserID: userID, LogDate: date, Status: status,
		LoggedAt: now.In(civildate.Tehran).Truncate(time.Second), Now: tehranNow(now),
	}); err != nil {
		return fmt.Errorf("contraception: save pill: %w", err)
	}
	return nil
}

// UnlogPill removes date's log (undo); a day without a log is a no-op.
func (s *Service) UnlogPill(ctx context.Context, userID uint64, date civildate.Date) error {
	if _, err := store.New(s.conn).DeletePillLog(ctx, store.DeletePillLogParams{UserID: userID, LogDate: date}); err != nil {
		return fmt.Errorf("contraception: delete pill: %w", err)
	}
	return nil
}

func (s *Service) pillPack(ctx context.Context, q *store.Queries, userID uint64) (Pack, civildate.Date, error) {
	row, err := q.GetMethod(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Pack{}, civildate.Date{}, ErrNoPillMethod
	}
	if err != nil {
		return Pack{}, civildate.Date{}, fmt.Errorf("contraception: load method: %w", err)
	}
	m := methodFromRow(row)
	pack, ok := m.Pack()
	if !ok {
		return Pack{}, civildate.Date{}, ErrNoPillMethod
	}
	return pack, m.PackStartedOn, nil
}
