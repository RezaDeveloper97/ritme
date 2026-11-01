package safety

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/safety/store"
	vstore "github.com/ritme/backend-go/internal/vitals/store"
)

// Scenarios are every self-test scenario: the path scenarios, then the live configuration checks.
func Scenarios() []Scenario { return append(append([]Scenario{}, PathScenarios...), ConfigChecks...) }

// Result is one scenario's outcome.
type Result struct {
	Scenario string
	Kind     string
	Passed   bool
	Duration time.Duration
	Detail   Detail
}

// Run is one self-test run.
type Run struct {
	ID      string
	RanAt   time.Time
	Passed  int
	Total   int
	Failed  []string
	Results []Result
}

// RunScenarios runs list on e (a panicking scenario fails with «panic»).
func RunScenarios(ctx context.Context, e *Env, list []Scenario, now time.Time) Run {
	r := Run{RanAt: now, Total: len(list)}
	for _, s := range list {
		start := time.Now()
		d := safeRun(ctx, s, e)
		res := Result{Scenario: s.Name, Kind: s.Kind, Passed: d.Passed(), Duration: time.Since(start), Detail: d}
		r.Results = append(r.Results, res)
		if res.Passed {
			r.Passed++
		} else {
			r.Failed = append(r.Failed, s.Name)
		}
	}
	return r
}

func safeRun(ctx context.Context, s Scenario, e *Env) (d Detail) {
	defer func() {
		if p := recover(); p != nil {
			d = Detail{Problems: []string{fmt.Sprintf("panic: %v", p)}}
		}
	}()
	return s.Run(ctx, e)
}

var (
	selftestFailed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "safety_selftest_failed",
		Help: "Scenarios that failed in the last safety.nightly self-test run of this process (0 = all passed).",
	})
	selftestRuns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "safety_selftest_runs_total",
		Help: "safety.nightly self-test runs by result (ok, failed, skipped, error).",
	}, []string{"result"})
)

func init() { metrics.Registry().MustRegister(selftestFailed, selftestRuns) }

var (
	lastMu  sync.RWMutex
	lastRun *Run
)

// Last is the latest run of this process (nil before the first); the persisted runs are Service.Runs.
func Last() *Run {
	lastMu.RLock()
	defer lastMu.RUnlock()
	return lastRun
}

func record(r Run) {
	lastMu.Lock()
	lastRun = &r
	lastMu.Unlock()
}

// Runner runs the self-test against the live database and persists it (safety_scenario_runs).
type Runner struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewRunner wires the runner.
func NewRunner(sqlDB *sql.DB, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Runner{db: sqlDB, logger: logger}
}

// Nightly is one day's run: a no-op when the Tehran day of now already has a persisted run (the job's idempotency
// key), else Run. ran is false when skipped.
func (r *Runner) Nightly(ctx context.Context, now time.Time) (Run, bool, error) {
	day := civildate.InTehran(now)
	n, err := store.New(r.db).CountSafetyRunsBetween(ctx, store.CountSafetyRunsBetweenParams{
		FromAt: sql.NullTime{Time: day.TehranMidnight(), Valid: true},
		ToAt:   sql.NullTime{Time: day.AddDays(1).TehranMidnight(), Valid: true}})
	if err != nil {
		selftestRuns.WithLabelValues("error").Inc()
		return Run{}, false, fmt.Errorf("safety: today's run: %w", err)
	}
	if n > 0 {
		selftestRuns.WithLabelValues("skipped").Inc()
		return Run{}, false, nil
	}
	run, err := r.Run(ctx, now)
	return run, err == nil, err
}

// Run runs every scenario on today's live configuration (the clinical registry reloaded first), persists one row
// per scenario, sets safety_selftest_failed and logs one line per failure (scenario names and synthetic inputs only).
func (r *Runner) Run(ctx context.Context, now time.Time) (Run, error) {
	day := civildate.InTehran(now)
	_ = clinicalconfig.Refresh(ctx) // a failed reload keeps the previous snapshot (logged by the package)
	env := LiveEnv(ctx, store.New(r.db), mstore.New(r.db), vstore.New(r.db), PureEnv(day))
	run := RunScenarios(ctx, env, Scenarios(), now)
	run.ID = newULID(now)
	record(run)
	selftestFailed.Set(float64(len(run.Failed)))
	for _, res := range run.Results {
		if !res.Passed {
			r.logger.ErrorContext(ctx, "safety self-test scenario failed", slog.String("alert", "safety_selftest_failed"),
				slog.String("run_id", run.ID), slog.String("scenario", res.Scenario), slog.String("kind", res.Kind),
				slog.String("failures", strings.Join(res.Detail.Failures(), "; ")))
		}
	}
	if err := r.persist(ctx, run); err != nil {
		selftestRuns.WithLabelValues("error").Inc()
		return run, err
	}
	if len(run.Failed) > 0 {
		selftestRuns.WithLabelValues("failed").Inc()
	} else {
		selftestRuns.WithLabelValues("ok").Inc()
	}
	return run, nil
}

func (r *Runner) persist(ctx context.Context, run Run) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("safety: persist run: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := store.New(tx)
	at := sql.NullTime{Time: run.RanAt, Valid: true}
	for _, res := range run.Results {
		detail, err := json.Marshal(res.Detail)
		if err != nil {
			return fmt.Errorf("safety: detail: %w", err)
		}
		ms := uint32(min(max(res.Duration.Milliseconds(), 0), 1<<31)) //nolint:gosec // bounded
		if err := q.InsertSafetyScenarioRun(ctx, store.InsertSafetyScenarioRunParams{RunID: run.ID, Scenario: res.Scenario,
			Passed: res.Passed, Detail: db.NullRawJSON{V: detail, Valid: true}, DurationMs: ms, CreatedAt: at}); err != nil {
			return fmt.Errorf("safety: persist run: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("safety: persist run: %w", err)
	}
	return nil
}

// newULID is a ULID (48-bit ms time + 80 random bits, Crockford base32): run ids sort by time.
func newULID(t time.Time) string {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	var b [16]byte
	ms := uint64(t.UnixMilli())                        //nolint:gosec // after 1970
	binary.BigEndian.PutUint16(b[0:2], uint16(ms>>32)) //nolint:gosec // top 16 of 48 bits
	binary.BigEndian.PutUint32(b[2:6], uint32(ms))     //nolint:gosec // low 32 bits
	_, _ = rand.Read(b[6:])
	out := make([]byte, 26)
	// 128 bits → 26 chars of 5 bits (the first char holds the top 3 bits).
	hi, lo := binary.BigEndian.Uint64(b[0:8]), binary.BigEndian.Uint64(b[8:16])
	for i := 25; i >= 0; i-- {
		out[i] = alphabet[lo&31]
		lo = lo>>5 | hi<<59
		hi >>= 5
	}
	return string(out)
}

// Background job (T-M9-11 scheduler, docs/engine-v6/jobs.md): the nightly self-test at 03:00 Asia/Tehran on the
// critical queue, once per Tehran day.
const (
	JobNightly  = "safety.nightly"
	NightlyCron = "0 3 * * *"
)

// RegisterJobs puts safety.nightly on q.
func RegisterJobs(q *queue.Queue, r *Runner, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobNightly, Priority: queue.Critical, MaxTries: 3,
		Backoff: []time.Duration{5 * time.Minute, 15 * time.Minute}, Timeout: 5 * time.Minute,
	}, NightlyJob(r, now), NightlyCron, queue.Unique(23*time.Hour))
}

// NightlyJob is the safety.nightly handler. Idempotent on the Tehran day: a day that already has a persisted run is
// left as it is.
func NightlyJob(r *Runner, now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error {
		_, _, err := r.Nightly(ctx, now())
		return err
	}
}
