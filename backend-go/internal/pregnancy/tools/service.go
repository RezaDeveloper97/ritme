// Package tools is the live pregnancy tools of the log sheet (bloom B-N5-03; artboards nbl_Log_Kick,
// nbl_Log_Contraction): the kick counter and the contraction timer.
//
// Built on what exists, nothing duplicated:
//   - kick counts are sessions (pregnancy_kick_sessions) that write their day total into the existing daily fetal
//     movement log (pregnancy_fetal_movements, GET|POST /pregnancy/fetal-movement) when they end, so the log sheet's
//     «حرکات جنین» tile, the pregnancy analysis and the alert engine keep reading one table;
//   - the 5-1-1 rule is the pregnancy alert engine's rule contractions_511 (messages/pregnancyalerts): thresholds,
//     call / hospital copy and actions are its admin-editable message_contents row; a hit is a v2 alert in the alert
//     centre, deduplicated per session. The maths is package labor.
//
// Both tools need an active, dated pregnancy to start; history stays readable. Every query is scoped by user id;
// nothing is logged.
package tools

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// Kick counter rules (nbl_Log_Kick: «۱۰ حرکت», «اگر در ۲ ساعت کمتر از ۱۰ حرکت حس کردی …»).
const (
	KickTarget        = 10
	KickWindowMinutes = 120
	MaxKicks          = 200 // per session
	HistoryLimit      = 20
)

// Rule is the pregnancy alert rule of the contraction timer.
const Rule = "contractions_511"

// Errors.
var (
	ErrNotActive          = errors.New("pregnancy tools: pregnancy not active")
	ErrNotFound           = errors.New("pregnancy tools: session not found")
	ErrSessionActive      = errors.New("pregnancy tools: a session is already running")
	ErrSessionEnded       = errors.New("pregnancy tools: session ended")
	ErrKickLimit          = errors.New("pregnancy tools: kick limit")
	ErrContractionRunning = errors.New("pregnancy tools: a contraction is in progress")
	ErrNoContraction      = errors.New("pregnancy tools: no contraction in progress")
)

// Alerts is the pregnancy alert engine (messages/pregnancyalerts.Engine).
type Alerts interface {
	Evaluate(ctx context.Context, userID uint64, now time.Time, l v2.Lang) ([]*jsonx.OrderedMap, error)
	RuleParams(ctx context.Context, rule string, l v2.Lang) (labor.Params, error)
}

// Service is the kick counter and contraction timer logic.
type Service struct {
	db     *sql.DB // nil when built on a transaction (tests)
	q      *store.Queries
	alerts Alerts // nil = no alert rules
}

// NewService returns a Service on db (a *sql.DB or a transaction); alerts may be nil.
func NewService(db store.DBTX, alerts Alerts) *Service {
	s := &Service{q: store.New(db), alerts: alerts}
	if pool, ok := db.(*sql.DB); ok {
		s.db = pool
	}
	return s
}

func dbTime(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: dbTime(t), Valid: true} }

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	if s.db == nil {
		return fn(s.q)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pregnancy tools: begin: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pregnancy tools: commit: %w", err)
	}
	return nil
}

// dating is the user's active, dated pregnancy on day (ErrNotActive otherwise).
func dating(ctx context.Context, q *store.Queries, userID uint64, day civildate.Date) (v2.Dating, error) {
	p, err := q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return v2.Dating{}, ErrNotActive
	}
	if err != nil {
		return v2.Dating{}, fmt.Errorf("pregnancy tools: profile: %w", err)
	}
	if !p.PregnancyMode {
		return v2.Dating{}, ErrNotActive
	}
	d, ok := v2.Resolve(&p, day)
	if !ok {
		return v2.Dating{}, ErrNotActive
	}
	return d, nil
}

// ---------------------------------------------------------------- kick counter

// KickOverview is GET /pregnancy/kick-sessions.
type KickOverview struct {
	Active  *store.PregnancyKickSession
	Today   int // kicks counted today (ended and running sessions started today)
	Count   int // sessions started today
	History []store.PregnancyKickSession
}

// Kicks loads the running session, today's total and the recent history.
func (s *Service) Kicks(ctx context.Context, userID uint64, now time.Time) (KickOverview, error) {
	out := KickOverview{}
	a, err := s.q.GetActiveKickSession(ctx, userID)
	switch {
	case err == nil:
		out.Active = &a
	case !errors.Is(err, sql.ErrNoRows):
		return out, fmt.Errorf("pregnancy tools: active kicks: %w", err)
	}
	today := civildate.InTehran(now)
	day, err := s.q.ListKickSessionsBetween(ctx, store.ListKickSessionsBetweenParams{
		UserID: userID, DateFrom: today.TehranMidnight(), DateTo: today.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return out, fmt.Errorf("pregnancy tools: day kicks: %w", err)
	}
	for _, k := range day {
		out.Today += int(k.Kicks)
		out.Count++
	}
	if out.Active != nil && civildate.InTehran(out.Active.StartedAt) == today {
		out.Today += int(out.Active.Kicks)
		out.Count++
	}
	if out.History, err = s.q.ListKickSessions(ctx, store.ListKickSessionsParams{UserID: userID, Limit: HistoryLimit}); err != nil {
		return out, fmt.Errorf("pregnancy tools: kick history: %w", err)
	}
	return out, nil
}

// KickSession loads one of the user's sessions.
func (s *Service) KickSession(ctx context.Context, userID, id uint64) (store.PregnancyKickSession, error) {
	k, err := s.q.GetKickSession(ctx, store.GetKickSessionParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return k, ErrNotFound
	}
	if err != nil {
		return k, fmt.Errorf("pregnancy tools: kick session: %w", err)
	}
	return k, nil
}

// StartKicks opens a kick count (ErrSessionActive while another runs, ErrNotActive outside pregnancy).
func (s *Service) StartKicks(ctx context.Context, userID uint64, now time.Time) (store.PregnancyKickSession, error) {
	d, err := dating(ctx, s.q, userID, civildate.InTehran(now))
	if err != nil {
		return store.PregnancyKickSession{}, err
	}
	id, err := s.q.InsertKickSession(ctx, store.InsertKickSessionParams{
		UserID: userID, StartedAt: dbTime(now), Now: nullTime(now),
		PregnancyWeek: sql.NullInt16{Int16: int16(d.CurrentWeek()), Valid: true}, //nolint:gosec // 1..42
	})
	if isDuplicate(err) {
		return store.PregnancyKickSession{}, ErrSessionActive
	}
	if err != nil {
		return store.PregnancyKickSession{}, fmt.Errorf("pregnancy tools: start kicks: %w", err)
	}
	return s.KickSession(ctx, userID, uint64(id)) //nolint:gosec // auto-increment id
}

// running loads a session that must still run (ErrNotFound / ErrSessionEnded).
func (s *Service) runningKicks(ctx context.Context, userID, id uint64) (store.PregnancyKickSession, error) {
	k, err := s.KickSession(ctx, userID, id)
	if err != nil {
		return k, err
	}
	if !k.ActiveLock.Valid {
		return k, ErrSessionEnded
	}
	return k, nil
}

// Kick counts one movement on a running session.
func (s *Service) Kick(ctx context.Context, userID, id uint64, now time.Time) (store.PregnancyKickSession, error) {
	n, err := s.q.AddKick(ctx, store.AddKickParams{
		Target: KickTarget, At: nullTime(now), Now: nullTime(now), ID: id, UserID: userID, MaxKicks: MaxKicks,
	})
	if err != nil {
		return store.PregnancyKickSession{}, fmt.Errorf("pregnancy tools: kick: %w", err)
	}
	k, err := s.runningKicks(ctx, userID, id)
	if err != nil {
		return k, err
	}
	if n == 0 {
		return k, ErrKickLimit
	}
	return k, nil
}

// UndoKick takes the last movement back (a session at 0 stays at 0).
func (s *Service) UndoKick(ctx context.Context, userID, id uint64, now time.Time) (store.PregnancyKickSession, error) {
	if _, err := s.q.UndoKick(ctx, store.UndoKickParams{Target: KickTarget, Now: nullTime(now), ID: id, UserID: userID}); err != nil {
		return store.PregnancyKickSession{}, fmt.Errorf("pregnancy tools: undo kick: %w", err)
	}
	return s.runningKicks(ctx, userID, id)
}

// StopKicks ends a running session and writes the day total into the fetal movement log.
func (s *Service) StopKicks(ctx context.Context, userID, id uint64, now time.Time) (store.PregnancyKickSession, error) {
	var out store.PregnancyKickSession
	err := s.inTx(ctx, func(q *store.Queries) error {
		n, err := q.StopKickSession(ctx, store.StopKickSessionParams{EndedAt: nullTime(now), Now: nullTime(now), ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("pregnancy tools: stop kicks: %w", err)
		}
		k, err := q.GetKickSession(ctx, store.GetKickSessionParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("pregnancy tools: kick session: %w", err)
		}
		if n == 0 {
			return ErrSessionEnded
		}
		out = k
		return writeFetalDay(ctx, q, userID, civildate.InTehran(k.StartedAt), now)
	})
	return out, err
}

// DeleteKicks removes a session (running or ended); an ended one's day total is rewritten.
func (s *Service) DeleteKicks(ctx context.Context, userID, id uint64, now time.Time) error {
	return s.inTx(ctx, func(q *store.Queries) error {
		k, err := q.GetKickSession(ctx, store.GetKickSessionParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("pregnancy tools: kick session: %w", err)
		}
		if _, err := q.DeleteKickSession(ctx, store.DeleteKickSessionParams{ID: id, UserID: userID}); err != nil {
			return fmt.Errorf("pregnancy tools: delete kicks: %w", err)
		}
		if k.ActiveLock.Valid {
			return nil
		}
		return writeFetalDay(ctx, q, userID, civildate.InTehran(k.StartedAt), now)
	})
}

// FetalDay is what the ended kick sessions of one day say for the daily fetal movement log.
type FetalDay struct {
	Kicks       int
	First, Last time.Time // first session start, last kick (zero when Kicks == 0)
	Normal      bool      // a session reached KickTarget within KickWindowMinutes
}

// SummarizeDay folds the ended sessions of a day.
func SummarizeDay(sessions []store.PregnancyKickSession) FetalDay {
	var d FetalDay
	for _, k := range sessions {
		if k.Kicks == 0 {
			continue
		}
		d.Kicks += int(k.Kicks)
		if d.First.IsZero() || k.StartedAt.Before(d.First) {
			d.First = k.StartedAt
		}
		last := k.StartedAt
		if k.LastKickAt.Valid {
			last = k.LastKickAt.Time
		}
		if last.After(d.Last) {
			d.Last = last
		}
		if k.TenthKickAt.Valid && k.TenthKickAt.Time.Sub(k.StartedAt) <= KickWindowMinutes*time.Minute {
			d.Normal = true
		}
	}
	return d
}

// writeFetalDay writes the day's kick total into pregnancy_fetal_movements: a new row gets status normal (10 within
// the window) or felt; an existing row keeps its status (not_felt_yet moves to felt / normal), week and notes. A day
// whose sessions counted nothing is left as it is (a manual log stays). The first counted kick also marks the
// profile's first felt movement.
func writeFetalDay(ctx context.Context, q *store.Queries, userID uint64, day civildate.Date, now time.Time) error {
	sessions, err := q.ListKickSessionsBetween(ctx, store.ListKickSessionsBetweenParams{
		UserID: userID, DateFrom: day.TehranMidnight(), DateTo: day.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return fmt.Errorf("pregnancy tools: day sessions: %w", err)
	}
	sum := SummarizeDay(sessions)
	if sum.Kicks == 0 {
		return nil
	}
	status := "felt"
	if sum.Normal {
		status = "normal"
	}
	count := sql.NullInt32{Int32: int32(sum.Kicks), Valid: true} //nolint:gosec // ≤ sessions × MaxKicks
	first := sql.NullString{String: sum.First.In(civildate.Tehran).Format("15:04"), Valid: true}
	last := sql.NullString{String: sum.Last.In(civildate.Tehran).Format("15:04"), Valid: true}
	ts := nullTime(now)
	row, err := q.GetFetalMovement(ctx, store.GetFetalMovementParams{UserID: userID, LogDate: day})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		week := int32(0)
		if d, derr := dating(ctx, q, userID, day); derr == nil {
			week = int32(d.CurrentWeek()) //nolint:gosec // 1..42
		} else if !errors.Is(derr, ErrNotActive) {
			return derr
		}
		for _, k := range sessions {
			if week == 0 && k.PregnancyWeek.Valid {
				week = int32(k.PregnancyWeek.Int16)
			}
		}
		if week == 0 {
			return nil // no week to log against
		}
		if _, err := q.InsertFetalMovement(ctx, store.InsertFetalMovementParams{
			UserID: userID, LogDate: day, PregnancyWeek: week, MovementStatus: status, MovementCount: count,
			FirstMovementTime: first, LastMovementTime: last, CreatedAt: ts, UpdatedAt: ts,
		}); err != nil {
			return fmt.Errorf("pregnancy tools: fetal insert: %w", err)
		}
	case err != nil:
		return fmt.Errorf("pregnancy tools: fetal day: %w", err)
	default:
		if row.MovementStatus != "not_felt_yet" {
			status = row.MovementStatus
		}
		if err := q.UpdateFetalMovement(ctx, store.UpdateFetalMovementParams{
			PregnancyWeek: row.PregnancyWeek, MovementStatus: status, MovementCount: count,
			FirstMovementTime: first, LastMovementTime: last, Notes: row.Notes, UpdatedAt: ts, ID: row.ID,
		}); err != nil {
			return fmt.Errorf("pregnancy tools: fetal update: %w", err)
		}
	}
	if err := q.MarkFetalMovementFelt(ctx, store.MarkFetalMovementFeltParams{Day: civildate.NullDate{Date: day, Valid: true}, Now: ts, UserID: userID}); err != nil {
		return fmt.Errorf("pregnancy tools: felt: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------- contraction timer

// Timing is a contraction session with its contractions (oldest first).
type Timing struct {
	Session      store.PregnancyContractionSession
	Contractions []store.PregnancyContraction
}

// Labor converts the contractions for the maths.
func (t Timing) Labor() []labor.Contraction {
	out := make([]labor.Contraction, 0, len(t.Contractions))
	for _, c := range t.Contractions {
		out = append(out, toLabor(c))
	}
	return out
}

func toLabor(c store.PregnancyContraction) labor.Contraction {
	lc := labor.Contraction{Start: c.StartedAt}
	if c.EndedAt.Valid {
		lc.End = c.EndedAt.Time
	}
	return lc
}

// Params are the live 5-1-1 thresholds (labor defaults without an alert engine).
func (s *Service) Params(ctx context.Context, l v2.Lang) (labor.Params, error) {
	if s.alerts == nil {
		return labor.DefaultParams(), nil
	}
	return s.alerts.RuleParams(ctx, Rule, l)
}

func (s *Service) timing(ctx context.Context, q *store.Queries, userID uint64, sess store.PregnancyContractionSession) (Timing, error) {
	cs, err := q.ListSessionContractions(ctx, store.ListSessionContractionsParams{SessionID: sess.ID, UserID: userID})
	if err != nil {
		return Timing{}, fmt.Errorf("pregnancy tools: contractions: %w", err)
	}
	return Timing{Session: sess, Contractions: cs}, nil
}

// ActiveTiming is the running session (nil when none).
func (s *Service) ActiveTiming(ctx context.Context, userID uint64) (*Timing, error) {
	sess, err := s.q.GetActiveContractionSession(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pregnancy tools: active timing: %w", err)
	}
	t, err := s.timing(ctx, s.q, userID, sess)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TimingByID loads one of the user's sessions.
func (s *Service) TimingByID(ctx context.Context, userID, id uint64) (Timing, error) {
	sess, err := s.q.GetContractionSession(ctx, store.GetContractionSessionParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Timing{}, ErrNotFound
	}
	if err != nil {
		return Timing{}, fmt.Errorf("pregnancy tools: timing: %w", err)
	}
	return s.timing(ctx, s.q, userID, sess)
}

// History is the user's ended sessions with their contractions, newest first.
func (s *Service) History(ctx context.Context, userID uint64) ([]Timing, error) {
	list, err := s.q.ListContractionSessions(ctx, store.ListContractionSessionsParams{UserID: userID, Limit: HistoryLimit})
	if err != nil {
		return nil, fmt.Errorf("pregnancy tools: timing history: %w", err)
	}
	out := make([]Timing, 0, len(list))
	if len(list) == 0 {
		return out, nil
	}
	cs, err := s.q.ListContractionsSince(ctx, store.ListContractionsSinceParams{UserID: userID, Since: list[len(list)-1].StartedAt})
	if err != nil {
		return nil, fmt.Errorf("pregnancy tools: history contractions: %w", err)
	}
	by := map[uint64][]store.PregnancyContraction{}
	for _, c := range cs {
		by[c.SessionID] = append(by[c.SessionID], c)
	}
	for _, sess := range list {
		out = append(out, Timing{Session: sess, Contractions: by[sess.ID]})
	}
	return out, nil
}

// StartContraction starts a contraction, opening a session when none runs (ErrContractionRunning while one is in
// progress, ErrNotActive outside pregnancy).
func (s *Service) StartContraction(ctx context.Context, userID uint64, now time.Time) (Timing, error) {
	if _, err := dating(ctx, s.q, userID, civildate.InTehran(now)); err != nil {
		return Timing{}, err
	}
	var out Timing
	err := s.inTx(ctx, func(q *store.Queries) error {
		sess, err := q.GetActiveContractionSession(ctx, userID)
		if errors.Is(err, sql.ErrNoRows) {
			id, ierr := q.InsertContractionSession(ctx, store.InsertContractionSessionParams{
				UserID: userID, StartedAt: dbTime(now), Now: nullTime(now),
			})
			if isDuplicate(ierr) {
				return ErrContractionRunning // a concurrent first tap opened it
			}
			if ierr != nil {
				return fmt.Errorf("pregnancy tools: open session: %w", ierr)
			}
			sess, err = q.GetContractionSession(ctx, store.GetContractionSessionParams{ID: uint64(id), UserID: userID}) //nolint:gosec // auto-increment id
		}
		if err != nil {
			return fmt.Errorf("pregnancy tools: session: %w", err)
		}
		if _, err := q.InsertContraction(ctx, store.InsertContractionParams{
			SessionID: sess.ID, UserID: userID, StartedAt: dbTime(now), Now: nullTime(now),
		}); err != nil {
			if isDuplicate(err) {
				return ErrContractionRunning
			}
			return fmt.Errorf("pregnancy tools: start contraction: %w", err)
		}
		out, err = s.timing(ctx, q, userID, sess)
		return err
	})
	return out, err
}

// StopContraction ends the contraction in progress, records the first 5-1-1 hit on the session and runs the alert
// rules; it returns the session and the alerts created.
func (s *Service) StopContraction(ctx context.Context, userID uint64, now time.Time, l v2.Lang) (Timing, []*jsonx.OrderedMap, error) {
	sess, err := s.q.GetActiveContractionSession(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Timing{}, nil, ErrNoContraction
	}
	if err != nil {
		return Timing{}, nil, fmt.Errorf("pregnancy tools: session: %w", err)
	}
	n, err := s.q.StopContraction(ctx, store.StopContractionParams{EndedAt: nullTime(now), Now: nullTime(now), SessionID: sess.ID, UserID: userID})
	if err != nil {
		return Timing{}, nil, fmt.Errorf("pregnancy tools: stop contraction: %w", err)
	}
	if n == 0 {
		return Timing{}, nil, ErrNoContraction
	}
	t, err := s.timing(ctx, s.q, userID, sess)
	if err != nil {
		return Timing{}, nil, err
	}
	p, err := s.Params(ctx, l)
	if err != nil {
		return Timing{}, nil, err
	}
	if r := labor.FiveOneOne(t.Labor(), p); r.Met && !t.Session.AlertAt.Valid {
		if err := s.q.SetContractionAlertAt(ctx, store.SetContractionAlertAtParams{
			AlertAt: nullTime(r.MetAt), Now: nullTime(now), ID: sess.ID, UserID: userID,
		}); err != nil {
			return Timing{}, nil, fmt.Errorf("pregnancy tools: alert at: %w", err)
		}
		t.Session.AlertAt = nullTime(r.MetAt)
	}
	alerts := []*jsonx.OrderedMap{}
	if s.alerts != nil {
		raised, err := s.alerts.Evaluate(ctx, userID, now, l)
		if err != nil {
			return Timing{}, nil, err
		}
		alerts = append(alerts, raised...)
	}
	return t, alerts, nil
}

// FinishTiming ends a running session («توقف و ذخیره»); a contraction still in progress ends now.
func (s *Service) FinishTiming(ctx context.Context, userID, id uint64, now time.Time) (Timing, error) {
	var out Timing
	err := s.inTx(ctx, func(q *store.Queries) error {
		sess, err := q.GetContractionSession(ctx, store.GetContractionSessionParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("pregnancy tools: timing: %w", err)
		}
		if _, err := q.StopContraction(ctx, store.StopContractionParams{EndedAt: nullTime(now), Now: nullTime(now), SessionID: id, UserID: userID}); err != nil {
			return fmt.Errorf("pregnancy tools: stop contraction: %w", err)
		}
		n, err := q.FinishContractionSession(ctx, store.FinishContractionSessionParams{EndedAt: nullTime(now), Now: nullTime(now), ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("pregnancy tools: finish timing: %w", err)
		}
		if n == 0 {
			return ErrSessionEnded
		}
		sess.EndedAt, sess.ActiveLock = nullTime(now), sql.NullInt16{}
		out, err = s.timing(ctx, q, userID, sess)
		return err
	})
	return out, err
}

// DeleteTiming removes a session with its contractions.
func (s *Service) DeleteTiming(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteContractionSession(ctx, store.DeleteContractionSessionParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("pregnancy tools: delete timing: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
