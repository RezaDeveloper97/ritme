package metrics

// Automatic dashboard alerts (T-M10-31, D-180; SPEC v6 §5.1). After every computed snapshot the enabled
// `metric_alert_rules` (migration 00105 seed, thresholds editable in the admin) are evaluated: an alert opens when its
// condition holds and resolves when it no longer does. At most one alert is open per (rule, subject) — the subject is
// "r<rule id>v<version>" for the per-version message CTR and '' otherwise.
//
// Rules (metric_key → source):
//   - ocr_correction_rate  gt           edited / extracted+edited lab values of uploaded sheets, last window_days days.
//   - message_ctr          drop_vs_mean yesterday's CTR of a rule version against the mean of the daily CTRs of the
//     window_days days before, only for versions shown at least min_volume times yesterday.
//   - rui_users            rising_days  users with a requires_user_input warning rose day over day for MORE than
//     threshold consecutive days (engine snapshots, ending yesterday).
//   - engine_mae           gt           mean |error| in days of the engine's first predictions resolved in window_days days.
//   - safety_selftest_failed gt         failed scenarios of the newest nightly self-test run (T-M10-24).
//
// A rule whose data is missing or under the k = 10 floor is «unknown»: its alerts keep their state. No user id, phone
// number or health value is stored or logged here — only aggregates, rule ids and versions.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Rule keys of the seeded rules.
const (
	RuleOCR      = "ocr_correction_rate"
	RuleCTR      = "message_ctr"
	RuleRUI      = "rui_users"
	RuleMAE      = "engine_mae"
	RuleSafety   = "safety_selftest_failed"
	cmpGT        = "gt"
	cmpDrop      = "drop_vs_mean"
	cmpRising    = "rising_days"
	stateOpen    = "open"
	stateResolve = "resolved"
)

// AlertsResolvedKept is how many resolved alerts the dashboard payload lists.
const AlertsResolvedKept = 10

// Exceeds is the gt comparator: strictly above the threshold (equal does not alert).
func Exceeds(value, threshold float64) bool { return value > threshold }

// DropVsMean is how far cur fell below the mean of prior, as a fraction of that mean (0.3 = 30 % lower). ok is false
// without a positive mean.
func DropVsMean(cur float64, prior []float64) (drop float64, ok bool) {
	if len(prior) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, v := range prior {
		sum += v
	}
	mean := sum / float64(len(prior))
	if mean <= 0 {
		return 0, false
	}
	return (mean - cur) / mean, true
}

// RisingStreak is the number of consecutive days, ending at the last point, on which the value was higher than the day
// before. The rising_days rule alerts when it is MORE than the threshold.
func RisingStreak(series []float64) int {
	n := 0
	for i := len(series) - 1; i > 0 && series[i] > series[i-1]; i-- {
		n++
	}
	return n
}

// observation is one evaluated subject of a rule.
type observation struct {
	subject string
	value   float64
	breach  bool
}

// evaluateAlerts is the AfterCompute hook: it reconciles the alerts with the rules and fills p.Alerts / p.AlertsDone.
// A failure is logged and leaves the alert lists empty; it never fails the snapshot.
func (s *Service) evaluateAlerts(ctx context.Context, now time.Time, p *Payload) error {
	if err := s.EvaluateAlerts(ctx, now); err != nil {
		s.logger.ErrorContext(ctx, "metric alerts: evaluation failed", slog.String("err", err.Error()))
	}
	open, done, err := s.AlertLists(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "metric alerts: listing failed", slog.String("err", err.Error()))
		return nil
	}
	p.Alerts, p.AlertsDone = open, done
	return nil
}

// EvaluateAlerts runs every rule once for now (Tehran day). Idempotent: the same data opens no second alert.
func (s *Service) EvaluateAlerts(ctx context.Context, now time.Time) error {
	rules, err := s.q.ListMetricAlertRules(ctx)
	if err != nil {
		return fmt.Errorf("metrics: alert rules: %w", err)
	}
	today := civildate.InTehran(now)
	var firstErr error
	for _, r := range rules {
		if !r.Enabled {
			if _, err := s.reconcile(ctx, r, today, now, nil); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		obs, known, err := s.observe(ctx, r, today, now)
		if err != nil {
			s.logger.ErrorContext(ctx, "metric alerts: rule failed", slog.String("rule", r.MetricKey), slog.String("err", err.Error()))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !known {
			continue
		}
		if _, err := s.reconcile(ctx, r, today, now, obs); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// reconcile opens the breaching observations and resolves the open subjects that no longer breach.
func (s *Service) reconcile(ctx context.Context, r store.MetricAlertRule, today civildate.Date, now time.Time, obs []observation) (int, error) {
	changed := 0
	breaching := map[string]bool{}
	if r.Enabled {
		for _, o := range obs {
			if !o.breach {
				continue
			}
			breaching[o.subject] = true
			if err := s.q.OpenMetricAlert(ctx, store.OpenMetricAlertParams{
				RuleID: r.ID, Subject: o.subject, Day: today, Value: dec(o.value), Threshold: r.Threshold,
				Now: sql.NullTime{Time: now, Valid: true},
			}); err != nil {
				return changed, fmt.Errorf("metrics: open alert: %w", err)
			}
			changed++
		}
	}
	open, err := s.q.ListOpenMetricAlertSubjects(ctx, r.ID)
	if err != nil {
		return changed, fmt.Errorf("metrics: open subjects: %w", err)
	}
	for _, sub := range open {
		if breaching[sub] {
			continue
		}
		if _, err := s.q.ResolveMetricAlert(ctx, store.ResolveMetricAlertParams{RuleID: r.ID, Subject: sub, Now: sql.NullTime{Time: now, Valid: true}}); err != nil {
			return changed, fmt.Errorf("metrics: resolve alert: %w", err)
		}
		changed++
	}
	return changed, nil
}

func dec(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }

// observe evaluates one rule against live data. known=false: no usable data (keep the alerts as they are).
func (s *Service) observe(ctx context.Context, r store.MetricAlertRule, today civildate.Date, now time.Time) ([]observation, bool, error) {
	thr, err := strconv.ParseFloat(r.Threshold, 64)
	if err != nil {
		return nil, false, fmt.Errorf("metrics: rule %s threshold: %w", r.MetricKey, err)
	}
	window := int(r.WindowDays)
	if window < 1 {
		window = 1
	}
	switch {
	case r.MetricKey == RuleOCR && r.Comparator == cmpGT:
		since := now.In(civildate.Tehran).AddDate(0, 0, -window)
		row, err := s.q.CountOCRMarkersSince(ctx, sql.NullTime{Time: since, Valid: true})
		if err != nil {
			return nil, false, fmt.Errorf("metrics: ocr alert: %w", err)
		}
		if row.Users < MinCohort || row.Total == 0 {
			return nil, false, nil
		}
		v := float64(row.Edited) / float64(row.Total)
		return []observation{{value: v, breach: Exceeds(v, thr)}}, true, nil

	case r.MetricKey == RuleMAE && r.Comparator == cmpGT:
		rows, err := s.q.ListResolvedFirstPredictions(ctx, store.ListResolvedFirstPredictionsParams{
			FromDay:      civildate.NullDate{Date: today.AddDays(-(window - 1)), Valid: true},
			ToDay:        civildate.NullDate{Date: today, Valid: true},
			ExcludedMask: excludedMask,
		})
		if err != nil {
			return nil, false, fmt.Errorf("metrics: mae alert: %w", err)
		}
		m := MAEOf(rows)
		if m.Users < MinCohort || m.Samples == 0 {
			return nil, false, nil
		}
		return []observation{{value: m.Mean, breach: Exceeds(m.Mean, thr)}}, true, nil

	case r.MetricKey == RuleSafety && r.Comparator == cmpGT:
		row, err := s.q.GetLatestSafetyRunFailures(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("metrics: safety alert: %w", err)
		}
		v := float64(row.Failed)
		return []observation{{value: v, breach: Exceeds(v, thr)}}, true, nil

	case r.MetricKey == RuleRUI && r.Comparator == cmpRising:
		return s.observeRUI(ctx, today, thr)

	case r.MetricKey == RuleCTR && r.Comparator == cmpDrop:
		return s.observeCTR(ctx, r, today, window, thr)
	}
	s.logger.WarnContext(ctx, "metric alerts: unknown rule skipped", slog.String("rule", r.MetricKey), slog.String("comparator", r.Comparator))
	return nil, false, nil
}

// observeRUI: the daily count of snapshots with a user-input warning over the ceil(threshold)+2 days ending yesterday
// (enough points for threshold+1 consecutive rises).
func (s *Service) observeRUI(ctx context.Context, today civildate.Date, thr float64) ([]observation, bool, error) {
	n := int(math.Ceil(thr)) + 2
	last := today.AddDays(-1)
	first := last.AddDays(-(n - 1))
	rows, err := s.q.CountRUIUsersByDay(ctx, store.CountRUIUsersByDayParams{RuiMask: ruiMask(), FromDay: first, ToDay: last})
	if err != nil {
		return nil, false, fmt.Errorf("metrics: rui alert: %w", err)
	}
	byDay := map[string]store.CountRUIUsersByDayRow{}
	for _, r := range rows {
		byDay[r.Day.String()] = r
	}
	if byDay[last.String()].Total == 0 { // the newest snapshot day is missing: nothing to judge
		return nil, false, nil
	}
	series := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		series = append(series, float64(byDay[first.AddDays(i).String()].Rui))
	}
	streak := float64(RisingStreak(series))
	return []observation{{value: streak, breach: Exceeds(streak, thr)}}, true, nil
}

// observeCTR: one observation per rule version shown at least min_volume times yesterday.
func (s *Service) observeCTR(ctx context.Context, r store.MetricAlertRule, today civildate.Date, window int, thr float64) ([]observation, bool, error) {
	yesterday := today.AddDays(-1)
	rows, err := s.q.ListMessageVersionCTRByDay(ctx, store.ListMessageVersionCTRByDayParams{SinceDay: yesterday.AddDays(-window), UntilDay: yesterday})
	if err != nil {
		return nil, false, fmt.Errorf("metrics: ctr alert: %w", err)
	}
	type key struct {
		rule uint64
		ver  uint32
	}
	type day struct{ shown, clicked int64 }
	by := map[key]map[string]day{}
	order := []key{}
	for _, row := range rows {
		k := key{row.RuleID, row.Version}
		if by[k] == nil {
			by[k] = map[string]day{}
			order = append(order, k)
		}
		by[k][row.Day.String()] = day{row.Shown, row.Clicked}
	}
	minVol := int64(0)
	if r.MinVolume.Valid {
		minVol = int64(r.MinVolume.Int32)
	}
	var out []observation
	for _, k := range order {
		cur, ok := by[k][yesterday.String()]
		if !ok || cur.shown == 0 || cur.shown < minVol {
			continue
		}
		var prior []float64
		for i := window; i >= 1; i-- {
			if d, ok := by[k][yesterday.AddDays(-i).String()]; ok && d.shown > 0 {
				prior = append(prior, float64(d.clicked)/float64(d.shown))
			}
		}
		drop, ok := DropVsMean(float64(cur.clicked)/float64(cur.shown), prior)
		if !ok {
			continue
		}
		out = append(out, observation{subject: fmt.Sprintf("r%dv%d", k.rule, k.ver), value: drop, breach: Exceeds(drop, thr)})
	}
	return out, true, nil
}

// AlertLists returns the open alerts (newest first) and the last AlertsResolvedKept resolved ones.
func (s *Service) AlertLists(ctx context.Context) (open, done []Alert, err error) {
	open, done = []Alert{}, []Alert{}
	o, err := s.q.ListOpenMetricAlerts(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, a := range o {
		open = append(open, buildAlert(a.MetricKey, a.Subject, a.Day, a.Value, a.Threshold, a.OpenedAt, sql.NullTime{}))
	}
	d, err := s.q.ListResolvedMetricAlerts(ctx, AlertsResolvedKept)
	if err != nil {
		return nil, nil, err
	}
	for _, a := range d {
		done = append(done, buildAlert(a.MetricKey, a.Subject, a.Day, a.Value, a.Threshold, a.OpenedAt, a.ResolvedAt))
	}
	return open, done, nil
}

func buildAlert(metricKey, subject string, day civildate.Date, value, threshold string, opened, resolved sql.NullTime) Alert {
	v, _ := strconv.ParseFloat(value, 64)
	t, _ := strconv.ParseFloat(threshold, 64)
	a := Alert{Key: metricKey, Severity: "warning", Subject: subject, State: stateOpen, Day: day.String(), Value: v, Threshold: t}
	if resolved.Valid {
		a.State = stateResolve
		ts := resolved.Time.In(civildate.Tehran).Format(time.RFC3339)
		a.ResolvedAt = &ts
	}
	if opened.Valid {
		a.OpenedAt = opened.Time.In(civildate.Tehran).Format(time.RFC3339)
	}
	switch metricKey {
	case RuleOCR:
		a.Text = fmt.Sprintf("OCR correction rate %.2f%% is above %.2f%%", 100*v, 100*t)
	case RuleCTR:
		a.Text = fmt.Sprintf("Message CTR (%s) fell %.1f%% below its %s mean, limit %.1f%%", subject, 100*v, "7-day", 100*t)
	case RuleRUI:
		a.Text = fmt.Sprintf("Users needing input have risen for %.0f days in a row, limit %.0f", v, t)
	case RuleMAE:
		a.Text = fmt.Sprintf("Engine mean error %.2f days is above %.2f", v, t)
	case RuleSafety:
		a.Severity = "critical"
		a.Text = fmt.Sprintf("Safety self-test: %.0f scenario(s) failed", v)
	default:
		a.Text = fmt.Sprintf("%s: %.4g (limit %.4g)", metricKey, v, t)
	}
	return a
}
