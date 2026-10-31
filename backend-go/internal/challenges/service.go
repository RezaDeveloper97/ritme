package challenges

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/plus"
	plusstore "github.com/ritme/backend-go/internal/plus/store"
)

// Errors mapped to 404 / 409 / 422 by the handlers. A challenge the user cannot see is the same 404 as a missing one.
var (
	ErrNotFound     = errors.New("challenges: not found")
	ErrNotEligible  = errors.New("challenges: not eligible")
	ErrNotStartable = errors.New("challenges: cannot start now")
	ErrNotJoined    = errors.New("challenges: not joined")
	ErrBadDay       = errors.New("challenges: day out of range")
	ErrRunEnded     = errors.New("challenges: run ended")
)

// DB is the handle the service reads through and opens transactions on (*sql.DB).
type DB interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service is the user side of challenges v2. Every query is scoped by the user id.
type Service struct {
	db      DB
	q       *store.Queries
	catalog *configcache.Cache[catalog]
	states  state.UserStates // nil = CYCLE_ENGINE=v1
}

// WithEngine gates phase eligibility on the v6 engine state (T-M9-36, D-136). nil keeps v1.
func (s *Service) WithEngine(st state.UserStates) *Service {
	s.states = st
	return s
}

// Tables are what the catalog snapshot reads; the admin challenge writers publish them (configcache.Changed).
var Tables = []string{"challenges", "challenge_programs", "challenge_tasks"}

// catalog is the active v2 programs with their tasks (admin config, T-M9-13: one load per process until an admin
// write or the 60 s TTL). Read-only once loaded.
type catalog struct {
	Programs []store.ListActiveProgramsRow
	Tasks    map[uint64][]store.ChallengeTask
}

// NewService wires the service on db.
func NewService(db DB) *Service {
	s := &Service{db: db, q: store.New(db)}
	s.catalog = configcache.New("challenges:catalog", Tables, s.loadCatalog)
	return s
}

func (s *Service) loadCatalog(ctx context.Context) (catalog, error) {
	rows, err := s.q.ListActivePrograms(ctx)
	if err != nil {
		return catalog{}, err
	}
	c := catalog{Programs: rows, Tasks: make(map[uint64][]store.ChallengeTask, len(rows))}
	for _, r := range rows {
		if c.Tasks[r.ID], err = s.q.ListTasks(ctx, r.ID); err != nil {
			return catalog{}, err
		}
	}
	return c, nil
}

// UserContext is what eligibility looks at: the life mode and (lazily) the cycle main phase.
type UserContext struct {
	Mode  string
	Phase string // "" = not loaded or unavailable
}

// Context reads the user's life mode; withPhase also runs the cycle engine for the main phase.
func (s *Service) Context(ctx context.Context, userID uint64, today civildate.Date, withPhase bool) (UserContext, error) {
	mode, err := healthlog.NewService(s.db).LifeMode(ctx, userID)
	if err != nil {
		return UserContext{}, err
	}
	uc := UserContext{Mode: mode}
	if withPhase {
		if uc.Phase, err = s.phase(ctx, userID, today); err != nil {
			return UserContext{}, err
		}
	}
	return uc, nil
}

// phase is the cycle engine's main phase of today ("unknown" when the user has no usable cycle). With the v6 state
// provider (CYCLE_ENGINE ≠ v1, T-M9-36, D-136) the phase is the engine's, so a challenge gate and the home card
// agree; v1 keeps the legacy resolver.
func (s *Service) phase(ctx context.Context, userID uint64, today civildate.Date) (string, error) {
	if s.states != nil {
		es, err := s.states.ForUser(ctx, userID, today)
		if err != nil {
			return "", fmt.Errorf("challenges: engine state: %w", err)
		}
		d := engine.DayOf(today)
		return es.Resolve(d, d).MainPhase.String(), nil
	}
	sn, err := cycleservice.New(s.db, nil).Load(ctx, userID, today, today, today) // the one shared v1 loader
	if err != nil {
		return "", fmt.Errorf("challenges: cycle: %w", err)
	}
	profile := sn.EngineProfile()
	st := resolver.Resolve(sn.Histories, profile, today, today, metrics.Calculate(sn.Histories, profile))
	return string(st.MainPhase), nil
}

// Item is one challenge of the user's list.
type Item struct {
	Program    Program
	Tasks      []store.ChallengeTask
	CanJoin    bool
	Enrollment *EnrollmentView
}

// EnrollmentView is the user's run of a challenge.
type EnrollmentView struct {
	Status        string // active | completed | expired (a run past its last day without completion)
	StartedOn     civildate.Date
	Day           int // 1-based day of the run today, capped at the duration
	CompletedAt   sql.NullTime
	DoneDays      int
	RequiredDays  int
	DoneToday     map[uint64]bool // task id → done today
	RewardGranted bool
}

func (s *Service) enrollmentView(ctx context.Context, e store.ChallengeEnrollment, p Program, tasks []store.ChallengeTask, today civildate.Date) (*EnrollmentView, error) {
	rows, err := s.q.ListEnrollmentCompletions(ctx, e.ID)
	if err != nil {
		return nil, fmt.Errorf("challenges: completions: %w", err)
	}
	done := make(map[[2]uint64]bool, len(rows))
	for _, r := range rows {
		done[[2]uint64{r.TaskID, uint64(r.DayNumber)}] = true
	}
	day := e.StartedOn.DiffDays(today) + 1
	status := e.Status
	if status == StatusActive && day > p.DurationDays {
		status = "expired"
	}
	day = max(1, min(day, p.DurationDays))
	v := &EnrollmentView{Status: status, StartedOn: e.StartedOn, Day: day, CompletedAt: e.CompletedAt,
		DoneToday: map[uint64]bool{}, RewardGranted: e.Status == StatusCompleted}
	v.DoneDays, v.RequiredDays = Progress(p, tasks, done)
	for _, t := range DayTasks(tasks, day) {
		v.DoneToday[t.ID] = done[[2]uint64{t.ID, uint64(day)}] //nolint:gosec // G115: day ≥ 1
	}
	return v, nil
}

// List is the user's challenge list: every active v2 challenge the user is enrolled in or may join.
func (s *Service) List(ctx context.Context, userID uint64, today civildate.Date) ([]Item, error) {
	cat, err := s.catalog.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("challenges: list: %w", err)
	}
	rows := cat.Programs
	enrolls, err := s.q.ListUserEnrollments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("challenges: enrollments: %w", err)
	}
	byChallenge := make(map[uint64]store.ChallengeEnrollment, len(enrolls))
	for _, e := range enrolls {
		byChallenge[e.ChallengeID] = e
	}
	needPhase := false
	for _, r := range rows {
		p := programOf(r)
		if p.Eligibility.NeedsPhase() || p.StartRule == StartMenstrual {
			needPhase = true
		}
	}
	uc, err := s.Context(ctx, userID, today, needPhase)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, r := range rows {
		p := programOf(r)
		e, joined := byChallenge[p.ID]
		eligible := p.Eligibility.Allows(uc.Mode, uc.Phase)
		if !joined && !eligible {
			continue
		}
		tasks := cat.Tasks[p.ID]
		it := Item{Program: p, Tasks: tasks, CanJoin: !joined && startable(p, uc, today)}
		if joined {
			if it.Enrollment, err = s.enrollmentView(ctx, e, p, tasks, today); err != nil {
				return nil, err
			}
		}
		out = append(out, it)
	}
	return out, nil
}

// startable applies the start rule.
func startable(p Program, uc UserContext, today civildate.Date) bool {
	switch p.StartRule {
	case StartMenstrual:
		return uc.Phase == "menstrual"
	case StartFixedDate:
		return !p.StartsOn.Valid || !today.Before(p.StartsOn.Date)
	}
	return true
}

// Join enrolls the user (idempotent: a second join returns the existing run, created = false).
func (s *Service) Join(ctx context.Context, userID, challengeID uint64, today civildate.Date, now time.Time) (Item, bool, error) {
	row, err := s.q.GetActiveProgram(ctx, challengeID)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, false, ErrNotFound
	} else if err != nil {
		return Item{}, false, fmt.Errorf("challenges: program: %w", err)
	}
	p := programOf(store.ListActiveProgramsRow(row))
	tasks, err := s.q.ListTasks(ctx, p.ID)
	if err != nil {
		return Item{}, false, fmt.Errorf("challenges: tasks: %w", err)
	}
	if e, err := s.q.GetUserEnrollment(ctx, store.GetUserEnrollmentParams{UserID: userID, ChallengeID: challengeID}); err == nil {
		v, err := s.enrollmentView(ctx, e, p, tasks, today)
		return Item{Program: p, Tasks: tasks, Enrollment: v}, false, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Item{}, false, fmt.Errorf("challenges: enrollment: %w", err)
	}
	uc, err := s.Context(ctx, userID, today, p.Eligibility.NeedsPhase() || p.StartRule == StartMenstrual)
	if err != nil {
		return Item{}, false, err
	}
	if !p.Eligibility.Allows(uc.Mode, uc.Phase) {
		return Item{}, false, ErrNotEligible
	}
	if !startable(p, uc, today) {
		return Item{}, false, ErrNotStartable
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	if _, err := s.q.InsertEnrollment(ctx, store.InsertEnrollmentParams{
		UserID: userID, ChallengeID: challengeID, StartedOn: today, CreatedAt: stamp, UpdatedAt: stamp,
	}); err != nil && !isDuplicate(err) { // a concurrent join: the row exists, answer it
		return Item{}, false, fmt.Errorf("challenges: enroll: %w", err)
	}
	e, err := s.q.GetUserEnrollment(ctx, store.GetUserEnrollmentParams{UserID: userID, ChallengeID: challengeID})
	if err != nil {
		return Item{}, false, fmt.Errorf("challenges: enrollment: %w", err)
	}
	events.Goal(ctx, userID, "challenge_joined") // T-M10-06, M-17: server-matched message goal
	v, err := s.enrollmentView(ctx, e, p, tasks, today)
	return Item{Program: p, Tasks: tasks, Enrollment: v}, true, err
}

// Result is the outcome of completing a task.
type Result struct {
	Item        Item
	Newly       bool // this call stored the completion (false = it was already done)
	JustFinally bool // this call completed the challenge and granted the reward
}

// CompleteTask marks a task of the user's run done on day (nil = today's day of the run). Idempotent. The run's row
// is locked for the whole call, so the completion check and the reward grant (a badge row and the free Plus days,
// UNIQUE per user and challenge) happen exactly once even under concurrent calls.
func (s *Service) CompleteTask(ctx context.Context, userID, challengeID, taskID uint64, day *int, today civildate.Date, now time.Time) (Result, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Result{}, fmt.Errorf("challenges: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)

	e, err := q.LockUserEnrollment(ctx, store.LockUserEnrollmentParams{UserID: userID, ChallengeID: challengeID})
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotJoined
	} else if err != nil {
		return Result{}, fmt.Errorf("challenges: lock: %w", err)
	}
	prow, err := q.GetProgram(ctx, challengeID)
	if err != nil {
		return Result{}, fmt.Errorf("challenges: program: %w", err)
	}
	ch, err := q.GetActiveProgram(ctx, challengeID)
	p := Program{}
	switch {
	case err == nil:
		p = programOf(store.ListActiveProgramsRow(ch))
	case errors.Is(err, sql.ErrNoRows): // deactivated meanwhile: a started run may still be finished
		p = Program{ID: challengeID, DurationDays: int(prow.DurationDays), CompletionMinDays: int(prow.CompletionMinDays.Int16),
			RewardBadgeCode: prow.RewardBadgeCode.String, RewardBadgeTitle: prow.RewardBadgeTitle, RewardPlusDays: int(prow.RewardPlusDays)}
	default:
		return Result{}, fmt.Errorf("challenges: program: %w", err)
	}
	current := e.StartedOn.DiffDays(today) + 1
	if current > p.DurationDays {
		return Result{}, ErrRunEnded
	}
	d := current
	if day != nil {
		d = *day
	}
	if d < 1 || d > current {
		return Result{}, ErrBadDay
	}
	task, err := q.GetTask(ctx, store.GetTaskParams{ID: taskID, ChallengeID: challengeID})
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	} else if err != nil {
		return Result{}, fmt.Errorf("challenges: task: %w", err)
	}
	if d < int(task.DayFrom) || d > int(task.DayTo) {
		return Result{}, ErrBadDay
	}
	n, err := q.InsertTaskCompletion(ctx, store.InsertTaskCompletionParams{
		EnrollmentID: e.ID, TaskID: taskID, DayNumber: uint16(d), CompletedAt: sql.NullTime{Time: now, Valid: true}, //nolint:gosec // G115: 1…365
	})
	if err != nil {
		return Result{}, fmt.Errorf("challenges: complete task: %w", err)
	}

	tasks, err := q.ListTasks(ctx, challengeID)
	if err != nil {
		return Result{}, fmt.Errorf("challenges: tasks: %w", err)
	}
	finished := false
	if e.Status == StatusActive {
		rows, err := q.ListEnrollmentCompletions(ctx, e.ID)
		if err != nil {
			return Result{}, fmt.Errorf("challenges: completions: %w", err)
		}
		done := make(map[[2]uint64]bool, len(rows))
		for _, r := range rows {
			done[[2]uint64{r.TaskID, uint64(r.DayNumber)}] = true
		}
		if doneDays, required := Progress(p, tasks, done); required > 0 && doneDays >= required {
			if err := s.finish(ctx, tx, q, e, p, now); err != nil {
				return Result{}, err
			}
			finished = true
			e.Status = StatusCompleted
			e.CompletedAt = sql.NullTime{Time: now, Valid: true}
		}
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("challenges: commit: %w", err)
	}
	v, err := s.enrollmentView(ctx, e, p, tasks, today)
	return Result{Item: Item{Program: p, Tasks: tasks, Enrollment: v}, Newly: n > 0, JustFinally: finished}, err
}

// finish completes the run and grants the reward, inside the caller's locked transaction.
func (s *Service) finish(ctx context.Context, tx *sql.Tx, q *store.Queries, e store.ChallengeEnrollment, p Program, now time.Time) error {
	stamp := sql.NullTime{Time: now, Valid: true}
	if err := q.CompleteEnrollment(ctx, store.CompleteEnrollmentParams{CompletedAt: stamp, UpdatedAt: stamp, ID: e.ID}); err != nil {
		return fmt.Errorf("challenges: complete run: %w", err)
	}
	if p.RewardBadgeCode == "" && !p.RewardBadgeTitle.Valid && p.RewardPlusDays == 0 {
		return nil
	}
	gid, err := q.InsertRewardGrant(ctx, store.InsertRewardGrantParams{
		UserID: e.UserID, ChallengeID: e.ChallengeID, EnrollmentID: e.ID,
		BadgeCode: sql.NullString{String: p.RewardBadgeCode, Valid: p.RewardBadgeCode != ""}, BadgeTitle: p.RewardBadgeTitle,
		PlusDays: uint16(p.RewardPlusDays), GrantedAt: stamp, //nolint:gosec // G115: ≤ 365
	})
	if err != nil {
		if isDuplicate(err) {
			return nil // already granted: never twice
		}
		return fmt.Errorf("challenges: grant: %w", err)
	}
	if p.RewardPlusDays > 0 {
		sub, err := plus.GrantDays(ctx, plusstore.New(tx), e.UserID, p.RewardPlusDays, plus.SourceChallenge, now)
		if err != nil {
			return err
		}
		if err := q.SetGrantSubscription(ctx, store.SetGrantSubscriptionParams{
			SubscriptionID: sql.NullInt64{Int64: sub, Valid: true}, ID: uint64(gid), //nolint:gosec // G115: ids are positive
		}); err != nil {
			return fmt.Errorf("challenges: grant subscription: %w", err)
		}
	}
	return nil
}

// Rewards lists the badges the user earned, newest first.
func (s *Service) Rewards(ctx context.Context, userID uint64) ([]store.ListUserRewardsRow, error) {
	rows, err := s.q.ListUserRewards(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("challenges: rewards: %w", err)
	}
	return rows, nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
