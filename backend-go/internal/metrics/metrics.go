// Package metrics is the aggregation job behind the admin overview dashboard (bloom B-N9-03, D-85): it computes
// product KPIs from existing tables and stores ONE JSON snapshot per Tehran day in `metric_snapshots`.
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
	"sort"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// MinCohort is the k-anonymity floor: smaller groups are suppressed.
const MinCohort = 10

// Interval is how often Loop recomputes the snapshot (hourly; the first run is at start-up).
const Interval = time.Hour

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

// Engine is the prediction-engine health: how well the median of the previous (up to 3) cycle lengths predicted each
// confirmed period start in the last 180 days.
type Engine struct {
	Status          *string  `json:"status"`
	MeanErrorDays   *float64 `json:"mean_error_days"`
	Within2DaysPct  *float64 `json:"within_2_days_pct"`
	Users           int      `json:"users"`
	Samples         int      `json:"samples"`
	Suppressed      bool     `json:"suppressed"`
	Confidence      any      `json:"confidence"` // null: confidence is not stored anywhere
	ConfidenceNote  string   `json:"confidence_note"`
	MethodologyNote string   `json:"note"`
}

// Alert is an active metric alert (B-N9-04+ fills the list; none exist yet).
type Alert struct {
	Key      string `json:"key"`
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

// Point is one day of the series.
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
	SignupsToday Value             `json:"signups_today"`
	PlusActive   Value             `json:"plus_active"`
	TrialToPaid  Rate              `json:"trial_to_paid"`
	MRRToman     *int64            `json:"mrr_toman"`
	Modes        []ModeCount       `json:"modes"`
	Engine       Engine            `json:"engine"`
	Alerts       []Alert           `json:"alerts"`
	OKR          any               `json:"okr"`
	Series       []Point           `json:"series"`
	Notes        map[string]string `json:"notes"`
}

// Snapshot is a stored payload with its day and compute time.
type Snapshot struct {
	Day        string
	ComputedAt time.Time
	Payload    json.RawMessage
}

// Service computes and stores snapshots.
type Service struct {
	db     *sql.DB
	logger *slog.Logger
}

// New builds the service.
func New(db *sql.DB, logger *slog.Logger) *Service { return &Service{db: db, logger: logger} }

func fp(v float64) *float64 { return &v }
func ip(v int) *int         { return &v }

func ymd(t time.Time) string { return t.Format("2006-01-02") }

func startOfDay(t time.Time) time.Time {
	t = t.In(civildate.Tehran)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, civildate.Tehran)
}

// activeUsers counts distinct users active on a day in [from, to] (inclusive Tehran dates).
// Activity = a token issued / refreshed (oauth_access_tokens.updated_at), a daily health log entry (log_date) or a
// first-party analytics event (analytics_events, B-N9-08 — only users who opted in to anonymous stats). The first two
// stay as the fallback for everyone else, so users who stay signed in without logging are still undercounted.
const activeUnion = "SELECT user_id AS uid, DATE(updated_at) AS d FROM oauth_access_tokens WHERE user_id IS NOT NULL AND updated_at >= ? AND updated_at < ? " +
	"UNION SELECT user_id, log_date FROM daily_health_logs WHERE log_date >= ? AND log_date <= ? " +
	"UNION SELECT user_id, occurred_on FROM analytics_events WHERE occurred_on >= ? AND occurred_on <= ?"

func (s *Service) activeUsers(ctx context.Context, from, to time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT uid) FROM ("+activeUnion+") a",
		from, to.AddDate(0, 0, 1), ymd(from), ymd(to), ymd(from), ymd(to)).Scan(&n)
	return n, err
}

// Compute builds the payload for now (Tehran).
func (s *Service) Compute(ctx context.Context, now time.Time) (*Payload, error) {
	now = now.In(civildate.Tehran)
	today := startOfDay(now)
	p := &Payload{Version: 1, Alerts: []Alert{}, Modes: []ModeCount{}, Notes: map[string]string{
		"activity":   "DAU/MAU count users with a sign-in token refresh, a daily health log or a product analytics event (opt-in users only) that day, so they undercount.",
		"mrr":        "Active purchased subscriptions: invoice subtotal minus discount, per month, excluding VAT; admin grants excluded.",
		"confidence": "Prediction confidence is not stored per user, so the distribution is not available.",
		"alerts":     "No alert source exists yet.",
		"okr":        "OKR summary is provided by B-N9-09.",
	}}

	// Series (90 days): one grouped query per source.
	first := today.AddDate(0, 0, -(SeriesDays - 1))
	byDay := map[string]*Point{}
	for i := 0; i < SeriesDays; i++ {
		d := ymd(first.AddDate(0, 0, i))
		byDay[d] = &Point{Date: d}
	}
	rows, err := s.db.QueryContext(ctx, "SELECT DATE_FORMAT(d, '%Y-%m-%d'), COUNT(*) FROM ("+activeUnion+") a GROUP BY d",
		first, today.AddDate(0, 0, 1), ymd(first), ymd(today), ymd(first), ymd(today))
	if err != nil {
		return nil, fmt.Errorf("metrics: dau series: %w", err)
	}
	if err := scanDays(rows, func(d string, n int) {
		if pt := byDay[d]; pt != nil {
			pt.DAU = n
		}
	}); err != nil {
		return nil, err
	}
	rows, err = s.db.QueryContext(ctx, "SELECT DATE_FORMAT(log_date, '%Y-%m-%d'), COUNT(DISTINCT user_id) FROM daily_health_logs WHERE log_date >= ? AND log_date <= ? GROUP BY log_date", ymd(first), ymd(today))
	if err != nil {
		return nil, fmt.Errorf("metrics: logger series: %w", err)
	}
	if err := scanDays(rows, func(d string, n int) {
		if pt := byDay[d]; pt != nil {
			pt.Loggers = n
		}
	}); err != nil {
		return nil, err
	}
	rows, err = s.db.QueryContext(ctx, "SELECT DATE_FORMAT(created_at, '%Y-%m-%d'), COUNT(*) FROM users WHERE created_at >= ? AND created_at < ? GROUP BY DATE(created_at)", first, today.AddDate(0, 0, 1))
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
	mau, err := s.activeUsers(ctx, yesterday.AddDate(0, 0, -29), yesterday)
	if err != nil {
		return nil, fmt.Errorf("metrics: mau: %w", err)
	}
	mauPrev, err := s.activeUsers(ctx, yesterday.AddDate(0, 0, -59), yesterday.AddDate(0, 0, -30))
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

	eng, err := s.engineHealth(ctx, now)
	if err != nil {
		return nil, err
	}
	p.Engine = *eng
	return p, nil
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

// engineHealth replays the stored period history: for each confirmed cycle in the last 180 days that had at least two
// earlier valid cycles, the prediction is the median of the previous (up to 3) cycle lengths and the error is the
// absolute difference to the real length = the error of the predicted next period start. Aggregated across users.
func (s *Service) engineHealth(ctx context.Context, now time.Time) (*Engine, error) {
	e := &Engine{
		ConfidenceNote:  "Prediction confidence is not stored per user, so the distribution is not available.",
		MethodologyNote: "Backtest on confirmed period history: median of the previous 1-3 cycle lengths vs the next real length (last 180 days).",
	}
	cut := startOfDay(now).AddDate(0, 0, -180)
	rows, err := s.db.QueryContext(ctx, "SELECT user_id, DATE_FORMAT(period_start_date, '%Y-%m-%d'), cycle_length FROM cycle_histories WHERE is_estimated = 0 AND cycle_length BETWEEN 15 AND 90 AND period_start_date >= ? ORDER BY user_id, period_start_date",
		ymd(cut.AddDate(0, 0, -400)))
	if err != nil {
		return nil, fmt.Errorf("metrics: engine: %w", err)
	}
	defer rows.Close()
	var cur uint64
	var prior []int
	users := map[uint64]bool{}
	var sum float64
	var within, n int
	for rows.Next() {
		var uid uint64
		var start string
		var l int
		if err := rows.Scan(&uid, &start, &l); err != nil {
			return nil, err
		}
		if uid != cur {
			cur, prior = uid, nil
		}
		if len(prior) >= 2 && start >= ymd(cut) {
			k := prior
			if len(k) > 3 {
				k = k[len(k)-3:]
			}
			d := math.Abs(float64(l) - median(k))
			sum += d
			n++
			if d <= 2 {
				within++
			}
			users[uid] = true
		}
		prior = append(prior, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	e.Users, e.Samples = len(users), n
	if len(users) < MinCohort {
		e.Suppressed = true
		e.Users, e.Samples = 0, 0
		return e, nil
	}
	e.MeanErrorDays = fp(round1(sum / float64(n)))
	e.Within2DaysPct = fp(round1(100 * float64(within) / float64(n)))
	st := "watch"
	if *e.MeanErrorDays <= healthyMeanError && *e.Within2DaysPct >= 100*healthyWithin2 {
		st = "healthy"
	}
	e.Status = &st
	return e, nil
}

func median(v []int) float64 {
	c := append([]int(nil), v...)
	sort.Ints(c)
	if len(c)%2 == 1 {
		return float64(c[len(c)/2])
	}
	return float64(c[len(c)/2-1]+c[len(c)/2]) / 2
}

// Refresh computes and upserts today's snapshot.
func (s *Service) Refresh(ctx context.Context, now time.Time) error {
	p, err := s.Compute(ctx, now)
	if err != nil {
		return err
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
	var sn Snapshot
	var raw string
	var day time.Time
	err := s.db.QueryRowContext(ctx, "SELECT `day`, payload, computed_at FROM metric_snapshots ORDER BY `day` DESC LIMIT 1").Scan(&day, &raw, &sn.ComputedAt)
	if err != nil {
		return nil, err
	}
	sn.Day, sn.Payload = ymd(day), json.RawMessage(raw)
	return &sn, nil
}

// Loop refreshes the snapshot at start-up and then every Interval until ctx ends.
func (s *Service) Loop(ctx context.Context, now func() time.Time) {
	t := time.NewTicker(Interval)
	defer t.Stop()
	for {
		if err := s.Refresh(ctx, now()); err != nil && ctx.Err() == nil {
			s.logger.ErrorContext(ctx, "metrics: refresh failed", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
