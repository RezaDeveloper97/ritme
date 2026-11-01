// Package metrics is the aggregation job behind the admin overview dashboard (bloom B-N9-03, D-85) and the KPI screen
// (B-N9-08, T-M10-30, D-179): it computes the SPEC v6 §5.6 KPIs from the activity days, the engine snapshots and
// prediction log and the existing tables, and stores ONE JSON snapshot per Tehran day in `metric_snapshots`. The KPI
// and overview endpoints only read stored snapshots. Definitions: docs/engine-v6/kpi.md.
//
// Privacy: only aggregates leave this package — counts, rates and daily series. Any group smaller than MinCohort (k = 10)
// is suppressed (value null + suppressed true) before it is stored. A metric whose source does not exist is null with a
// note; nothing is faked.
package metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// MinCohort is the k-anonymity floor: smaller groups are suppressed.
const MinCohort = 10

// SeriesDays is the length of the daily series kept in each snapshot (the 90-day chart range).
const SeriesDays = 90

// Engine-health thresholds for the «healthy» badge (B-N9-11 will make them configurable).
const (
	healthyMeanError = 2.5
	healthyWithin2   = 0.70
)

// Value is a headline metric with its comparison value (null when not computable).
type Value struct {
	Value *float64 `json:"value"`
	Prev  *float64 `json:"prev"`
}

// Rate is a ratio over a counted group; Pct is null when the group is below MinCohort.
type Rate struct {
	Pct        *float64 `json:"pct"`
	Count      int      `json:"count"`
	Of         *int     `json:"of"`
	Suppressed bool     `json:"suppressed"`
}

// ModeCount is the users of one life mode (suppressed under MinCohort).
type ModeCount struct {
	Mode       string `json:"mode"`
	Count      *int   `json:"count"`
	Suppressed bool   `json:"suppressed"`
}

// Engine is the prediction-engine health (A-05): the error of the engine's own `first` prediction of each cycle built
// from ≥ 2 real cycles whose next real start fell in the last EngineDays days, pregnancy / postpartum / hormonal
// contraception cycles excluded (engine_prediction_log × engine_daily_snapshots).
type Engine struct {
	Status          *string       `json:"status"`
	MeanErrorDays   *float64      `json:"mean_error_days"`
	Within2DaysPct  *float64      `json:"within_2_days_pct"`
	Users           int           `json:"users"`
	Samples         int           `json:"samples"`
	Suppressed      bool          `json:"suppressed"`
	Confidence      *Distribution `json:"confidence"` // the active users' confidence on the newest snapshot day (A-06)
	ConfidenceNote  string        `json:"confidence_note"`
	MethodologyNote string        `json:"note"`
}

// Alert is a metric alert of the dashboard (T-M10-31, D-180): open ones in Payload.Alerts, the last resolved ones in
// Payload.AlertsDone. Only aggregates and rule/version ids — no user data.
type Alert struct {
	Key        string  `json:"key"`
	Severity   string  `json:"severity"`
	Text       string  `json:"text"`
	Subject    string  `json:"subject"`
	State      string  `json:"state"`
	Day        string  `json:"day"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	OpenedAt   string  `json:"opened_at"`
	ResolvedAt *string `json:"resolved_at"`
}

// Point is one day of the series (DAU = users with a session, Loggers = users with a data log; user_activity_days).
type Point struct {
	Date    string `json:"date"`
	DAU     int    `json:"dau"`
	Loggers int    `json:"loggers"`
	Signups int    `json:"signups"`
}

// Payload is the stored snapshot document.
type Payload struct {
	Version      int               `json:"version"`
	DAU          Value             `json:"dau"`
	MAU          Value             `json:"mau"`
	Stickiness   Value             `json:"stickiness_pct"`
	Loggers      Value             `json:"loggers"`
	Loggers15d   Rate              `json:"loggers_15d"`
	SignupsToday Value             `json:"signups_today"`
	PlusActive   Value             `json:"plus_active"`
	TrialToPaid  Rate              `json:"trial_to_paid"`
	MRRToman     *int64            `json:"mrr_toman"`
	Modes        []ModeCount       `json:"modes"`
	Engine       Engine            `json:"engine"`
	Alerts       []Alert           `json:"alerts"`          // open alerts, newest first
	AlertsDone   []Alert           `json:"resolved_alerts"` // the last AlertsResolvedKept resolved alerts
	OKR          any               `json:"okr"`
	Series       []Point           `json:"series"`
	KPI          *KPIData          `json:"kpi"`
	Notes        map[string]string `json:"notes"`
}

// PayloadVersion is the snapshot document version (2: §5.6 KPIs from activity days and engine snapshots, T-M10-30).
const PayloadVersion = 2

// Snapshot is a stored payload with its day and compute time.
type Snapshot struct {
	Day        string
	ComputedAt time.Time
	Payload    json.RawMessage
}

// AfterCompute runs on a freshly computed payload before it is stored (the OKR evaluation, T-M10-30).
type AfterCompute func(ctx context.Context, now time.Time, p *Payload) error

// Service computes and stores snapshots.
type Service struct {
	db     *sql.DB
	q      *store.Queries
	logger *slog.Logger

	mu    sync.Mutex
	after []AfterCompute
}

// New builds the service.
func New(db *sql.DB, logger *slog.Logger) *Service {
	s := &Service{db: db, logger: logger}
	if db != nil {
		s.q = store.New(db)
		s.OnComputed(s.evaluateAlerts) // T-M10-31: the alert rules run on every computed snapshot
	}
	return s
}

// OnComputed registers fn to run on every computed payload, in registration order, before Refresh stores it.
func (s *Service) OnComputed(fn AfterCompute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.after = append(s.after, fn)
}

func fp(v float64) *float64 { return &v }
func ip(v int) *int         { return &v }

func ymd(t time.Time) string { return t.Format("2006-01-02") }

func startOfDay(t time.Time) time.Time {
	t = t.In(civildate.Tehran)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, civildate.Tehran)
}

// Compute builds the payload for now (Tehran).
func (s *Service) Compute(ctx context.Context, now time.Time) (*Payload, error) {
	now = now.In(civildate.Tehran)
	today := startOfDay(now)
	p := &Payload{Version: PayloadVersion, Alerts: []Alert{}, AlertsDone: []Alert{}, Modes: []ModeCount{}, Notes: map[string]string{
		"activity":   "DAU/MAU count users with at least one signed-in request on their local day / in 30 days; loggers count users with at least one data log that day (user_activity_days).",
		"mrr":        "Active purchased subscriptions: invoice subtotal minus discount, per month, excluding VAT; admin grants excluded.",
		"confidence": "Confidence and data quality of the active users on the newest engine snapshot day.",
		"alerts":     "Open alerts of the enabled metric_alert_rules (newest first) and the last 10 resolved; evaluated right after each snapshot (D-180).",
		"okr":        "OKR summary of the objectives active today, evaluated right after the KPIs.",
	}}

	// Series (90 days): one grouped query per source.
	first := today.AddDate(0, 0, -(SeriesDays - 1))
	byDay := map[string]*Point{}
	for i := 0; i < SeriesDays; i++ {
		d := ymd(first.AddDate(0, 0, i))
		byDay[d] = &Point{Date: d}
	}
	act, err := s.q.CountActiveUsersByDay(ctx, store.CountActiveUsersByDayParams{FromDay: civildate.InTehran(first), ToDay: civildate.InTehran(today)})
	if err != nil {
		return nil, fmt.Errorf("metrics: activity series: %w", err)
	}
	for _, r := range act {
		if pt := byDay[r.Day.String()]; pt != nil {
			pt.DAU, pt.Loggers = int(r.SessionUsers), int(r.LoggedUsers)
		}
	}
	rows, err := s.db.QueryContext(ctx, "SELECT DATE_FORMAT(created_at, '%Y-%m-%d'), COUNT(*) FROM users WHERE created_at >= ? AND created_at < ? GROUP BY DATE(created_at)", first, today.AddDate(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("metrics: signup series: %w", err)
	}
	if err := scanDays(rows, func(d string, n int) {
		if pt := byDay[d]; pt != nil {
			pt.Signups = n
		}
	}); err != nil {
		return nil, err
	}
	p.Series = make([]Point, 0, SeriesDays)
	for i := 0; i < SeriesDays; i++ {
		p.Series = append(p.Series, *byDay[ymd(first.AddDate(0, 0, i))])
	}

	// Headlines: DAU = last full day; MAU = the 30 days ending yesterday (previous = the 30 days before).
	yesterday := today.AddDate(0, 0, -1)
	dau := byDay[ymd(yesterday)].DAU
	dauPrev := byDay[ymd(yesterday.AddDate(0, 0, -1))].DAU
	p.DAU = Value{Value: fp(float64(dau)), Prev: fp(float64(dauPrev))}
	yd := civildate.InTehran(yesterday)
	mau, err := s.sessionUsers(ctx, yd.AddDays(-29), yd)
	if err != nil {
		return nil, fmt.Errorf("metrics: mau: %w", err)
	}
	mauPrev, err := s.sessionUsers(ctx, yd.AddDays(-59), yd.AddDays(-30))
	if err != nil {
		return nil, fmt.Errorf("metrics: mau prev: %w", err)
	}
	p.MAU = Value{Value: fp(float64(mau)), Prev: fp(float64(mauPrev))}
	if mau > 0 {
		p.Stickiness.Value = fp(round1(100 * float64(dau) / float64(mau)))
	}
	if mauPrev > 0 {
		p.Stickiness.Prev = fp(round1(100 * float64(dauPrev) / float64(mauPrev)))
	}
	p.Loggers = Value{Value: fp(float64(byDay[ymd(yesterday)].Loggers)), Prev: fp(float64(byDay[ymd(yesterday.AddDate(0, 0, -1))].Loggers))}
	l15, err := s.q.CountLoggersAtLeast(ctx, store.CountLoggersAtLeastParams{FromDay: yd.AddDays(-29), ToDay: yd, MinDays: LoggerDays})
	if err != nil {
		return nil, fmt.Errorf("metrics: loggers 15d: %w", err)
	}
	p.Loggers15d = rate(int(l15), mau)
	p.SignupsToday = Value{Value: fp(float64(byDay[ymd(today)].Signups)), Prev: fp(float64(byDay[ymd(yesterday)].Signups))}

	// Plus.
	var plus int
	var mrrRials sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT user_id) FROM plus_subscriptions WHERE status = 'active' AND starts_at <= ? AND ends_at > ?", now, now).Scan(&plus); err != nil {
		return nil, fmt.Errorf("metrics: plus: %w", err)
	}
	p.PlusActive.Value = fp(float64(plus))
	if err := s.db.QueryRowContext(ctx, "SELECT SUM((i.subtotal_rials - i.discount_rials) / i.duration_months) FROM plus_subscriptions s JOIN plus_invoices i ON i.id = s.invoice_id "+
		"WHERE s.status = 'active' AND s.source = 'purchase' AND s.starts_at <= ? AND s.ends_at > ? AND i.duration_months > 0", now, now).Scan(&mrrRials); err != nil {
		return nil, fmt.Errorf("metrics: mrr: %w", err)
	}
	mrr := int64(math.Round(mrrRials.Float64 / 10)) // rials → toman
	p.MRRToman = &mrr

	// Trial → paid: trials that ended in the last 30 days and bought a plan after the trial began.
	var trials, converted int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*), COALESCE(SUM(EXISTS(SELECT 1 FROM plus_subscriptions s WHERE s.user_id = t.user_id AND s.source = 'purchase' AND s.created_at >= t.started_at)), 0) "+
		"FROM plus_trials t WHERE t.ends_at > ? AND t.ends_at <= ?", now.AddDate(0, 0, -30), now).Scan(&trials, &converted); err != nil {
		return nil, fmt.Errorf("metrics: trial: %w", err)
	}
	p.TrialToPaid = Rate{Count: converted, Of: ip(trials)}
	if trials >= MinCohort {
		p.TrialToPaid.Pct = fp(round1(100 * float64(converted) / float64(trials)))
	} else {
		p.TrialToPaid.Suppressed = true
		p.TrialToPaid.Count = 0
	}

	// Users by mode.
	rows, err = s.db.QueryContext(ctx, "SELECT COALESCE(lp.life_mode, 'cycle'), COUNT(*) FROM users u LEFT JOIN user_life_profiles lp ON lp.user_id = u.id WHERE u.blocked_at IS NULL GROUP BY 1 ORDER BY 2 DESC, 1")
	if err != nil {
		return nil, fmt.Errorf("metrics: modes: %w", err)
	}
	if err := scanDays(rows, func(m string, n int) { p.Modes = append(p.Modes, suppress(m, n)) }); err != nil {
		return nil, err
	}
	var comp int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT companion_user_id) FROM companions WHERE companion_user_id IS NOT NULL AND accepted_at IS NOT NULL AND revoked_at IS NULL").Scan(&comp); err != nil {
		return nil, fmt.Errorf("metrics: companions: %w", err)
	}
	p.Modes = append(p.Modes, suppress("companion", comp))

	eng, err := s.engineHealth(ctx, today)
	if err != nil {
		return nil, err
	}
	if p.KPI, err = s.computeKPI(ctx, now); err != nil {
		return nil, err
	}
	eng.Confidence = &p.KPI.Distributions.Confidence
	p.Engine = *eng
	return p, nil
}

// LoggerDays is the "loggers ≥ 15 days in the month" threshold.
const LoggerDays = 15

func (s *Service) sessionUsers(ctx context.Context, from, to civildate.Date) (int, error) {
	n, err := s.q.CountDistinctSessionUsers(ctx, store.CountDistinctSessionUsersParams{FromDay: from, ToDay: to})
	return int(n), err
}

// rate is n of `of` as a Rate, suppressed (pct and count hidden) while the group is below MinCohort.
func rate(n, of int) Rate {
	if of < MinCohort {
		return Rate{Of: ip(of), Suppressed: true}
	}
	return Rate{Pct: pct(n, of), Count: n, Of: ip(of)}
}

func suppress(mode string, n int) ModeCount {
	if n < MinCohort {
		return ModeCount{Mode: mode, Suppressed: true}
	}
	return ModeCount{Mode: mode, Count: ip(n)}
}

func scanDays(rows *sql.Rows, f func(key string, n int)) error {
	defer rows.Close()
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return err
		}
		f(k, n)
	}
	return rows.Err()
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// EngineDays is the MAE / ±2-day accuracy window (next real starts in the last EngineDays days).
const EngineDays = 180

// excludedMask is the warning bits whose cycles the MAE excludes (hormonal contraception; pregnancy and postpartum are
// excluded by the snapshot mode).
var excludedMask = uint32(1) << engine.WarnHormonalContraceptionActive

// engineHealth is the MAE and ±2-day accuracy of the engine's `first` predictions (A-05).
func (s *Service) engineHealth(ctx context.Context, today time.Time) (*Engine, error) {
	e := &Engine{
		ConfidenceNote:  "Distribution of the active users' prediction confidence on the newest engine snapshot day.",
		MethodologyNote: "Engine prediction log: the first prediction of each cycle built from at least 2 real cycles vs the next real start (last 180 days); pregnancy, postpartum and hormonal-contraception cycles excluded.",
	}
	td := civildate.InTehran(today)
	rows, err := s.q.ListResolvedFirstPredictions(ctx, store.ListResolvedFirstPredictionsParams{
		FromDay: civildate.NullDate{Date: td.AddDays(-(EngineDays - 1)), Valid: true}, ToDay: civildate.NullDate{Date: td, Valid: true},
		ExcludedMask: excludedMask,
	})
	if err != nil {
		return nil, fmt.Errorf("metrics: engine: %w", err)
	}
	m := MAEOf(rows)
	e.Users, e.Samples = m.Users, m.Samples
	if m.Users < MinCohort {
		e.Suppressed = true
		e.Users, e.Samples = 0, 0
		return e, nil
	}
	e.MeanErrorDays = fp(round1(m.Mean))
	e.Within2DaysPct = fp(round1(m.Within2Pct))
	st := "watch"
	if *e.MeanErrorDays <= healthyMeanError && *e.Within2DaysPct >= 100*healthyWithin2 {
		st = "healthy"
	}
	e.Status = &st
	return e, nil
}

// MAE is the aggregate of resolved predictions.
type MAE struct {
	Mean       float64
	Within2Pct float64
	Users      int
	Samples    int
}

// MAEOf folds the resolved predictions: mean |error| and the share with |error| ≤ 2 days.
func MAEOf(rows []store.ListResolvedFirstPredictionsRow) MAE {
	var m MAE
	users := map[uint64]bool{}
	var sum float64
	within := 0
	for _, r := range rows {
		if !r.ErrorDays.Valid {
			continue
		}
		d := math.Abs(float64(r.ErrorDays.Int16))
		sum += d
		if d <= 2 {
			within++
		}
		m.Samples++
		users[r.UserID] = true
	}
	m.Users = len(users)
	if m.Samples > 0 {
		m.Mean = sum / float64(m.Samples)
		m.Within2Pct = 100 * float64(within) / float64(m.Samples)
	}
	return m
}

// Refresh computes and upserts today's snapshot.
func (s *Service) Refresh(ctx context.Context, now time.Time) error {
	p, err := s.Compute(ctx, now)
	if err != nil {
		return err
	}
	s.mu.Lock()
	after := append([]AfterCompute(nil), s.after...)
	s.mu.Unlock()
	for _, fn := range after {
		if err := fn(ctx, now, p); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	now = now.In(civildate.Tehran)
	_, err = s.db.ExecContext(ctx, "INSERT INTO metric_snapshots (`day`, payload, computed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?) "+
		"ON DUPLICATE KEY UPDATE payload = VALUES(payload), computed_at = VALUES(computed_at), updated_at = VALUES(updated_at)",
		ymd(now), string(raw), now, now, now)
	return err
}

// Latest returns the newest snapshot (sql.ErrNoRows when none).
func (s *Service) Latest(ctx context.Context) (*Snapshot, error) {
	r, err := s.q.GetLatestMetricSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return &Snapshot{Day: r.Day.String(), ComputedAt: r.ComputedAt, Payload: json.RawMessage(r.Payload)}, nil
}

// OnOrBefore returns the newest snapshot on or before day (sql.ErrNoRows when none).
func (s *Service) OnOrBefore(ctx context.Context, day civildate.Date) (*Snapshot, error) {
	r, err := s.q.GetMetricSnapshotOnOrBefore(ctx, day)
	if err != nil {
		return nil, err
	}
	return &Snapshot{Day: r.Day.String(), ComputedAt: r.ComputedAt, Payload: json.RawMessage(r.Payload)}, nil
}

// Background job (T-M9-11, docs/engine-v6/jobs.md): the snapshot refresh hourly on the low queue, also at start-up.
// It recomputes today's row (Tehran) with every §5.6 KPI, then the OKR evaluation (OnComputed) on the same payload.
const (
	JobRefresh  = "admin_metrics:snapshot"
	RefreshCron = "15 * * * *"
)

// RegisterJobs puts the snapshot job on q.
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobRefresh, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{time.Minute, 5 * time.Minute}, Timeout: 10 * time.Minute,
	}, s.RefreshJob(now), RefreshCron, queue.Unique(55*time.Minute), queue.RunAtStart())
}

// RefreshJob is the admin_metrics:snapshot handler. Idempotent: one row per Tehran day, upserted.
func (s *Service) RefreshJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, _ []byte) error { return s.Refresh(ctx, now()) }
}
