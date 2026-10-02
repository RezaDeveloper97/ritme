// Package postpartum is the postpartum mode API (bloom B-N5-01; artboards nbl_v15_Main, nbl_v15_Recovery,
// nbl_v15_MoodCheck): activation (from an active pregnancy or directly), the status since the birth, the recovery
// log and the EPDS mood checks with their safety path.
//
// Built on what exists, nothing duplicated:
//   - the mode is bloom's user_life_profiles.life_mode = postpartum (B-N2-01); activating from pregnancy closes the
//     pregnancy profile like /pregnancy/deactivate and records the birth here (source pregnancy = delivered; a loss is
//     CB-LOSS-01's own path and never lands here);
//   - the recovery log is log taxonomy v2 slots (healthlog.Service, B-N3-01) — see recovery.go;
//   - the week tips, alerts and safety texts are admin message_contents with an embedded fallback (guide).
//
// Health data: every read and write is scoped to the authenticated user; EPDS answers are never logged, never sent to
// analytics and never exposed to companions (no companion section reads postpartum data — a future grant must be
// explicit).
package postpartum

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum/store"
)

// Sources of a postpartum profile.
const (
	SourcePregnancy = "pregnancy" // activated from an active pregnancy: that pregnancy ended in a birth
	SourceDirect    = "direct"
)

// Delivery types.
var DeliveryTypes = []string{"vaginal", "cesarean"}

// Activation limits.
const (
	MaxBabyCount = 4
	// MaxBirthAgeDays is how far back a birth date may be (the mode covers the first year).
	MaxBirthAgeDays = 365
	// HistoryLimit caps GET /postpartum/epds.
	HistoryLimit = 52
)

// Service is the postpartum API's logic.
type Service struct {
	db   *sql.DB // nil in tests that pass a transaction
	q    *store.Queries
	logs *healthlog.Service
}

// NewService returns a Service on db (a *sql.DB, or a transaction).
func NewService(db store.DBTX) *Service {
	s := &Service{q: store.New(db), logs: healthlog.NewService(db)}
	if pool, ok := db.(*sql.DB); ok {
		s.db = pool
	}
	return s
}

func tehranNow(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// State is the user's mode and postpartum profile.
type State struct {
	Mode    enums.LifeMode
	Profile *store.PostpartumProfile // nil before the first activation
}

// Active reports whether the effective mode is postpartum.
func (s State) Active() bool { return s.Mode == enums.LifeModePostpartum }

// State loads the effective mode (enums.ResolveAccountMode: male → companion, active pregnancy wins, then the stored
// mode) and the profile.
func (s *Service) State(ctx context.Context, userID uint64) (State, error) {
	life, err := s.q.GetPostpartumLifeProfile(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return State{}, fmt.Errorf("postpartum: life profile: %w", err)
	}
	_, err = s.q.GetActivePregnancyProfile(ctx, userID)
	pregnant := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return State{}, fmt.Errorf("postpartum: pregnancy: %w", err)
	}
	goal, err := s.q.GetPostpartumUserGoal(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return State{}, fmt.Errorf("postpartum: user goal: %w", err)
	}
	st := State{Mode: enums.ResolveAccountMode(life.Gender.String, life.LifeMode.String, pregnant, goal)}
	p, err := s.q.GetPostpartumProfile(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return State{}, fmt.Errorf("postpartum: profile: %w", err)
	default:
		st.Profile = &p
	}
	return st, nil
}

// ActivateInput is a validated POST /postpartum/activate.
type ActivateInput struct {
	BirthDate    civildate.Date
	DeliveryType string // "" = not told
	BabyCount    int
}

// Activate switches the user to postpartum mode: an active pregnancy is closed as delivered (pregnancy_mode off, as
// /pregnancy/deactivate does), the birth is stored (a later activation overwrites it), the life mode becomes
// postpartum and user_goal / pregnancy_intention follow (non-TTC, no pregnant intention). One transaction.
func (s *Service) Activate(ctx context.Context, userID uint64, in ActivateInput, now time.Time) error {
	return s.inTx(ctx, func(q *store.Queries) error {
		ts := tehranNow(now)
		p := store.UpsertPostpartumProfileParams{
			UserID: userID, BirthDate: in.BirthDate, BabyCount: uint8(in.BabyCount), Source: SourceDirect, Now: ts, //nolint:gosec // G115: validated 1–MaxBabyCount
		}
		if in.DeliveryType != "" {
			p.DeliveryType = sql.NullString{String: in.DeliveryType, Valid: true}
		}
		pid, err := q.LockActivePregnancyProfile(ctx, userID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return fmt.Errorf("postpartum: lock pregnancy: %w", err)
		default:
			if err := q.ClosePregnancyAsDelivered(ctx, store.ClosePregnancyAsDeliveredParams{Now: ts, ID: pid, UserID: userID}); err != nil {
				return fmt.Errorf("postpartum: close pregnancy: %w", err)
			}
			p.Source = SourcePregnancy
			p.PregnancyProfileID = sql.NullInt64{Int64: int64(pid), Valid: true} //nolint:gosec // G115: auto-increment id
			p.PregnancyClosedAt = ts
		}
		if err := q.UpsertPostpartumProfile(ctx, p); err != nil {
			return fmt.Errorf("postpartum: save profile: %w", err)
		}
		if err := q.SetLifeModePostpartum(ctx, store.SetLifeModePostpartumParams{UserID: userID, Now: ts}); err != nil {
			return fmt.Errorf("postpartum: life mode: %w", err)
		}
		if err := q.SyncPostpartumLegacyGoal(ctx, store.SyncPostpartumLegacyGoalParams{Now: ts, UserID: userID}); err != nil {
			return fmt.Errorf("postpartum: legacy goal: %w", err)
		}
		return nil
	})
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	if s.db == nil {
		return fn(s.q)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postpartum: begin: %w", err)
	}
	if err := fn(store.New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postpartum: commit: %w", err)
	}
	return nil
}

// Recovery is the user's recovery log of a day.
func (s *Service) Recovery(ctx context.Context, userID uint64, date civildate.Date) (Recovery, error) {
	entries, err := s.logs.Day(ctx, userID, date)
	if err != nil {
		return Recovery{}, err
	}
	return RecoveryOf(entries), nil
}

// SaveRecovery writes the changed fields of a day through the taxonomy v2 storage (and its legacy write-back).
func (s *Service) SaveRecovery(ctx context.Context, userID uint64, date civildate.Date, in RecoveryInput, locale string, now time.Time) (Recovery, error) {
	changes := in.Changes()
	if len(changes) == 0 {
		return s.Recovery(ctx, userID, date)
	}
	day, err := s.logs.SaveDay(ctx, userID, date, changes, locale, now)
	if err != nil {
		return Recovery{}, err
	}
	return RecoveryOf(day), nil
}

func checkOf(id uint64, kind string, takenOn civildate.Date, total uint8, selfHarm sql.NullInt16, urgent bool) Check {
	c := Check{ID: id, Kind: kind, TakenOn: takenOn, Total: int(total), Urgent: urgent}
	if selfHarm.Valid {
		v := int(selfHarm.Int16)
		c.SelfHarm = &v
	}
	return c
}

// SaveCheck stores a scored check for today (a second check of the same kind on the same day replaces it) and
// returns it. The answers are stored, never logged.
func (s *Service) SaveCheck(ctx context.Context, userID uint64, r Result, today civildate.Date, now time.Time) (Check, error) {
	answers, err := json.Marshal(r.Answers)
	if err != nil {
		return Check{}, fmt.Errorf("postpartum: encode check: %w", err)
	}
	p := store.UpsertEpdsCheckParams{
		UserID: userID, Kind: r.Kind, TakenOn: today, Answers: answers, Total: uint8(r.Total), //nolint:gosec // G115: ≤ 30
		Urgent: r.Urgent, Now: tehranNow(now),
	}
	if r.SelfHarm != nil {
		p.SelfHarm = sql.NullInt16{Int16: int16(*r.SelfHarm), Valid: true} //nolint:gosec // G115: 0–3
	}
	if err := s.q.UpsertEpdsCheck(ctx, p); err != nil {
		return Check{}, fmt.Errorf("postpartum: save check: %w", err)
	}
	row, err := s.q.GetEpdsCheckOn(ctx, store.GetEpdsCheckOnParams{UserID: userID, Kind: r.Kind, TakenOn: today})
	if err != nil {
		return Check{}, fmt.Errorf("postpartum: reload check: %w", err)
	}
	return checkOf(row.ID, row.Kind, row.TakenOn, row.Total, row.SelfHarm, row.Urgent), nil
}

// History is the user's checks, newest first (totals only).
func (s *Service) History(ctx context.Context, userID uint64, limit int) ([]Check, error) {
	rows, err := s.q.ListEpdsChecks(ctx, store.ListEpdsChecksParams{UserID: userID, Limit: int32(limit)}) //nolint:gosec // G115: ≤ HistoryLimit
	if err != nil {
		return nil, fmt.Errorf("postpartum: history: %w", err)
	}
	out := make([]Check, 0, len(rows))
	for _, r := range rows {
		out = append(out, checkOf(r.ID, r.Kind, r.TakenOn, r.Total, r.SelfHarm, r.Urgent))
	}
	return out, nil
}

// LastChecks are the latest check of any kind and the latest full check (nil when none).
func (s *Service) LastChecks(ctx context.Context, userID uint64) (last, lastFull *Check, err error) {
	r, err := s.q.GetLastEpdsCheck(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, nil, fmt.Errorf("postpartum: last check: %w", err)
	default:
		c := checkOf(r.ID, r.Kind, r.TakenOn, r.Total, r.SelfHarm, r.Urgent)
		last = &c
	}
	f, err := s.q.GetLastEpdsCheckOfKind(ctx, store.GetLastEpdsCheckOfKindParams{UserID: userID, Kind: KindFull})
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, nil, fmt.Errorf("postpartum: last full check: %w", err)
	default:
		c := checkOf(f.ID, f.Kind, f.TakenOn, f.Total, f.SelfHarm, f.Urgent)
		lastFull = &c
	}
	return last, lastFull, nil
}

// Signals are the facts the postpartum message engine reads for a day.
type Signals struct {
	BirthDate    *civildate.Date
	DeliveryType string
	Recovery     Recovery
	Schedule     *Schedule // nil without a birth date
}

// SignalsOn loads the message engine's facts for userID on date.
func (s *Service) SignalsOn(ctx context.Context, userID uint64, date civildate.Date) (Signals, error) {
	var sig Signals
	p, err := s.q.GetPostpartumProfile(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return sig, fmt.Errorf("postpartum: profile: %w", err)
	default:
		b := p.BirthDate
		sig.BirthDate = &b
		sig.DeliveryType = p.DeliveryType.String
		last, lastFull, err := s.LastChecks(ctx, userID)
		if err != nil {
			return sig, err
		}
		sch := ScheduleOn(date, b, last, lastFull)
		sig.Schedule = &sch
	}
	rec, err := s.Recovery(ctx, userID, date)
	if err != nil {
		return sig, err
	}
	sig.Recovery = rec
	return sig, nil
}
