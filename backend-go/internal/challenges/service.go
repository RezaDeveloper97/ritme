package challenges

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/plus"
	plusstore "github.com/ritme/backend-go/internal/plus/store"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Errors mapped to 404 / 409 / 422 by the handlers. A challenge the user cannot see is the same 404 as a missing one.
var (
	ErrNotFound    = errors.New("challenges: not found")
	ErrNotEligible = errors.New("challenges: not eligible")
	// ErrContraindicated is a refusal by the contraindications (always before the eligibility, SPEC §5.5).
	ErrContraindicated = errors.New("challenges: contraindicated")
	ErrNotStartable    = errors.New("challenges: cannot start now")
	ErrNotJoined       = errors.New("challenges: not joined")
	ErrBadDay          = errors.New("challenges: day out of range")
	ErrRunEnded        = errors.New("challenges: run ended")
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

// UserContext is what the rules look at: the facts (life_mode always; main_phase and the registry facts when a rule
// reads them) and the main phase of today and yesterday (the start rules).
type UserContext struct {
	Mode      string      // life mode
	Phase     string      // today's main phase; "" = not loaded or unavailable
	PrevPhase string      // yesterday's main phase (on_phase_start only); "" = not loaded
	Facts     rules.Facts // what eligibility and contraindications read
}

// Need is what Context computes.
type Need struct {
	Facts     []string // fact names the rules read
	Phase     bool     // today's main phase (a rule or the start rule reads it)
	PrevPhase bool     // yesterday's main phase too (on_phase_start)
}

// needOf is what the programs need.
func needOf(ps ...Program) Need {
	var n Need
	for _, p := range ps {
		n.Facts = append(n.Facts, p.Facts()...)
		n.Phase = n.Phase || p.NeedsPhase()
		n.PrevPhase = n.PrevPhase || p.StartRule == StartOnPhaseStart
	}
	if slices.Contains(n.Facts, FactMainPhase) {
		n.Phase = true
	}
	slices.Sort(n.Facts)
	n.Facts = slices.Compact(n.Facts)
	return n
}

// Context computes the user's rule facts and phases. life_mode is the healthlog life mode and main_phase the challenge
// phase below (both exactly as before T-M10-10, so legacy rules behave the same); every other fact comes from the
// message fact registry (T-M10-03). mode falls back to the life mode when the registry has no engine mode
// (CYCLE_ENGINE=v1 or no engine state): the six engine modes are the six life modes, and an absent mode would let a
// mode contraindication through.
func (s *Service) Context(ctx context.Context, userID uint64, today civildate.Date, need Need) (UserContext, error) {
	mode, err := healthlog.NewService(s.db).LifeMode(ctx, userID)
	if err != nil {
		return UserContext{}, err
	}
	uc := UserContext{Mode: mode, Facts: rules.Facts{}}
	var registry []string
	for _, n := range need.Facts {
		if n != FactLifeMode && n != FactMainPhase && facts.GroupOf(n) != "" {
			registry = append(registry, n)
		}
	}
	if len(registry) > 0 {
		src := messages.NewStoreSource(msgstore.New(s.db), pstore.New(s.db), userID, today).WithEngine(s.states)
		fs := facts.New(src.FactSources(), today, "")
		if err := fs.Need(ctx, registry...); err != nil {
			return UserContext{}, fmt.Errorf("challenges: facts: %w", err)
		}
		maps.Copy(uc.Facts, fs.Values())
	}
	uc.Facts[FactLifeMode] = mode
	if _, ok := uc.Facts[FactMode]; !ok && mode != "" {
		uc.Facts[FactMode] = mode
	}
	if need.Phase || need.PrevPhase {
		if uc.Phase, uc.PrevPhase, err = s.phases(ctx, userID, today, need.PrevPhase); err != nil {
			return UserContext{}, err
		}
		if uc.Phase != "" {
			uc.Facts[FactMainPhase] = uc.Phase
		}
	}
	return uc, nil
}

// phase is the cycle engine's main phase of today ("unknown" when the user has no usable cycle).
func (s *Service) phase(ctx context.Context, userID uint64, today civildate.Date) (string, error) {
	p, _, err := s.phases(ctx, userID, today, false)
	return p, err
}

// phases are the main phases of today and (withPrev) yesterday, both as seen today. With the v6 state provider
// (CYCLE_ENGINE ≠ v1, T-M9-36, D-136) they are the engine's, so a challenge gate and the home card agree; v1 keeps the
// legacy resolver.
func (s *Service) phases(ctx context.Context, userID uint64, today civildate.Date, withPrev bool) (cur, prev string, err error) {
	yesterday := today.AddDays(-1)
	if s.states != nil {
		es, err := s.states.ForUser(ctx, userID, today)
		if err != nil {
			return "", "", fmt.Errorf("challenges: engine state: %w", err)
		}
		d := engine.DayOf(today)
		cur = es.Resolve(d, d).MainPhase.String()
		if withPrev {
			prev = es.Resolve(engine.DayOf(yesterday), d).MainPhase.String()
		}
		return cur, prev, nil
	}
	from := today
	if withPrev {
		from = yesterday
	}
	sn, err := cycleservice.New(s.db, nil).Load(ctx, userID, from, today, today) // the one shared v1 loader
	if err != nil {
		return "", "", fmt.Errorf("challenges: cycle: %w", err)
	}
	profile := sn.EngineProfile()
	m := metrics.Calculate(sn.Histories, profile)
	cur = string(resolver.Resolve(sn.Histories, profile, today, today, m).MainPhase)
	if withPrev {
		prev = string(resolver.Resolve(sn.Histories, profile, yesterday, today, m).MainPhase)
	}
	return cur, prev, nil
}

// Item is one challenge of the user's list.
type Item struct {
	Program    Program
	Tasks      []store.ChallengeTask
	CanJoin    bool
	Enrollment *EnrollmentView
	// Contraindicated: the contraindications hold for the user today (a joined run is still listed, never pushed).
	Contraindicated bool
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

// List is the user's challenge list: every active v2 challenge the user is enrolled in or may join. A refused
// challenge (contraindicated or not eligible) is listed only when the user already joined it, and never joinable.
func (s *Service) List(ctx context.Context, userID uint64, today civildate.Date) ([]Item, error) {
	cat, err := s.catalog.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("challenges: list: %w", err)
	}
	if len(cat.Programs) == 0 { // nothing to list: no user read (the home card asks on every render)
		return nil, nil
	}
	enrolls, err := s.q.ListUserEnrollments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("challenges: enrollments: %w", err)
	}
	byChallenge := make(map[uint64]store.ChallengeEnrollment, len(enrolls))
	for _, e := range enrolls {
		byChallenge[e.ChallengeID] = e
	}
	ps := make([]Program, len(cat.Programs))
	for i, r := range cat.Programs {
		ps[i] = programOf(r)
	}
	uc, err := s.Context(ctx, userID, today, needOf(ps...))
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, p := range ps {
		e, joined := byChallenge[p.ID]
		reason := Decide(p.Eligibility, p.Contraindications, uc.Facts)
		if !joined && reason != "" {
			continue
		}
		tasks := cat.Tasks[p.ID]
		it := Item{Program: p, Tasks: tasks, CanJoin: !joined && startable(p, uc, today),
			Contraindicated: reason == ReasonContraindicated}
		if joined {
			if it.Enrollment, err = s.enrollmentView(ctx, e, p, tasks, today); err != nil {
				return nil, err
			}
		}
		out = append(out, it)
	}
	return out, nil
}

// startable applies the start rule. on_phase_start needs a start phase and today to be its first day: today's main
// phase is start_phase and yesterday's (as the engine sees it today) was not. An unknown rule is on_join, as before.
func startable(p Program, uc UserContext, today civildate.Date) bool {
	switch p.StartRule {
	case StartMenstrual:
		return uc.Phase == "menstrual"
	case StartOnPhaseStart:
		return p.StartPhase != "" && uc.Phase == p.StartPhase && uc.PrevPhase != "" && uc.PrevPhase != p.StartPhase
	case StartFixedDate:
		return !p.StartsOn.Valid || !today.Before(p.StartsOn.Date)
	}
	return true
}

// Join enrolls the user (idempotent: a second join returns the existing run, created = false). The contraindications
// are checked first (ErrContraindicated), then the eligibility (ErrNotEligible), then the start rule.
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
	uc, err := s.Context(ctx, userID, today, needOf(p))
	if err != nil {
		return Item{}, false, err
	}
	switch Decide(p.Eligibility, p.Contraindications, uc.Facts) {
	case ReasonContraindicated:
		return Item{}, false, ErrContraindicated
	case ReasonNotEligible:
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

// HomeCandidates are the daily_challenge candidates from the user's list (List: the legacy-exact context, the run
// views), in admin sort order, with the items by id. The home card uses them until it reads the feed.
func (s *Service) HomeCandidates(ctx context.Context, userID uint64, today civildate.Date) ([]feed.ChallengeCandidate, map[uint64]Item, error) {
	items, err := s.List(ctx, userID, today)
	if err != nil {
		return nil, nil, err
	}
	cands := make([]feed.ChallengeCandidate, 0, len(items))
	byID := make(map[uint64]Item, len(items))
	for _, it := range items {
		byID[it.Program.ID] = it
		c := feed.ChallengeCandidate{ID: it.Program.ID, Contraindicated: it.Contraindicated}
		if it.Enrollment != nil {
			c.Ongoing = it.Enrollment.Status == StatusActive // "expired" = the run ended without completion
		} else {
			c.Eligible = it.CanJoin
		}
		cands = append(cands, c)
	}
	return cands, byID, nil
}

// NeedFacts are the fact names the active programs' rules read (the feed adds them to its prefetch).
func (s *Service) NeedFacts(ctx context.Context) ([]string, error) {
	cat, err := s.catalog.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("challenges: catalog: %w", err)
	}
	ps := make([]Program, len(cat.Programs))
	for i, r := range cat.Programs {
		ps[i] = programOf(r)
	}
	return needOf(ps...).Facts, nil
}

// FeedCandidates are the daily_challenge candidates (T-M10-10), in admin sort order, decided on the facts the feed
// already built (f: the registry facts, which include NeedFacts) and the user's runs off the fact profile row: no
// query (the catalogue is the process snapshot). A start rule that reads the phase is judged on f's main_phase;
// on_phase_start also needs yesterday's phase, which f does not carry, so such a challenge is suggested only while
// the user is in its run already (never as a product candidate the join would refuse).
func (s *Service) FeedCandidates(ctx context.Context, today civildate.Date, f rules.Facts, runs []feed.ChallengeRun) ([]feed.ChallengeCandidate, error) {
	cat, err := s.catalog.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("challenges: catalog: %w", err)
	}
	if len(cat.Programs) == 0 {
		return nil, nil
	}
	byChallenge := runsByChallenge(runs)
	f = maps.Clone(f)
	if f == nil {
		f = rules.Facts{}
	}
	uc := UserContext{Facts: f}
	if m, ok := f[FactLifeMode].(string); ok {
		uc.Mode = m
		if _, has := f[FactMode]; !has && m != "" {
			f[FactMode] = m // as Context: an absent mode must not let a mode contraindication through
		}
	}
	if ph, ok := f[FactMainPhase].(string); ok {
		uc.Phase = ph
	}
	out := make([]feed.ChallengeCandidate, 0, len(cat.Programs))
	for _, r := range cat.Programs {
		p := programOf(r)
		reason := Decide(p.Eligibility, p.Contraindications, f)
		c := feed.ChallengeCandidate{ID: p.ID, Contraindicated: reason == ReasonContraindicated}
		if e, joined := byChallenge[p.ID]; joined {
			c.Ongoing = Ongoing(e, p, today)
		} else {
			c.Eligible = reason == "" && p.StartRule != StartOnPhaseStart && startable(p, uc, today)
		}
		out = append(out, c)
	}
	return out, nil
}

var _ feed.ChallengeSource = (*Service)(nil)

// FeedCard is the feed card of an active challenge (catalogue snapshot; the run's day from runs).
func (s *Service) FeedCard(ctx context.Context, id uint64, today civildate.Date, runs []feed.ChallengeRun) (feed.ChallengeCard, bool, error) {
	cat, err := s.catalog.Get(ctx)
	if err != nil {
		return feed.ChallengeCard{}, false, fmt.Errorf("challenges: catalog: %w", err)
	}
	for _, r := range cat.Programs {
		if r.ID != id {
			continue
		}
		p := programOf(r)
		card := feed.ChallengeCard{ID: p.ID, Title: p.Title, Category: p.Category, DurationDays: p.DurationDays}
		if p.Description.Valid {
			card.Description = p.Description.V
		}
		if e, ok := runsByChallenge(runs)[id]; ok && Ongoing(e, p, today) {
			card.Day = e.StartedOn.DiffDays(today) + 1
		}
		return card, true, nil
	}
	return feed.ChallengeCard{}, false, nil
}

// runsByChallenge indexes the runs as enrollment rows (a run with an unreadable start date is skipped).
func runsByChallenge(runs []feed.ChallengeRun) map[uint64]store.ChallengeEnrollment {
	out := make(map[uint64]store.ChallengeEnrollment, len(runs))
	for _, r := range runs {
		d, err := civildate.Parse(r.StartedOn)
		if err != nil {
			continue
		}
		out[r.ChallengeID] = store.ChallengeEnrollment{ChallengeID: r.ChallengeID, StartedOn: d, Status: r.Status}
	}
	return out
}

// Ongoing reports whether a run is in progress today: active (not completed) and inside its duration (a run past
// its last day is abandoned).
func Ongoing(e store.ChallengeEnrollment, p Program, today civildate.Date) bool {
	day := e.StartedOn.DiffDays(today) + 1
	return e.Status == StatusActive && day >= 1 && day <= p.DurationDays
}

// Result is the outcome of completing a task.
type Result struct {
	Item        Item
	Newly       bool // this call stored the completion (false = it was already done)
	JustFinally bool // this call completed the challenge and granted the reward
}

// CompleteTask marks a task of the user's run done on day (nil = today's day of the run). Idempotent. The run's row
// is locked for the whole call, so the completion check and the reward grant (a badge row and the free Plus days,
// UNIQUE per user and challenge) happen exactly once even under concurrent calls; the user's row is locked right
// after it, so two runs of one user finishing together queue their free periods instead of deadlocking.
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
	// The user row next, before any insert (T-M12-19b): the reward-grant insert's foreign-key check would take a
	// shared lock on it, and two runs of one user finishing together, both holding that while plus.GrantDays asks for
	// the exclusive lock, deadlock. Lock order: enrollment row, then the user (as in plus).
	if _, err := q.LockUser(ctx, userID); err != nil {
		return Result{}, fmt.Errorf("challenges: lock user: %w", err)
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
