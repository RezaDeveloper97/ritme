package metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// KPI is the document behind the admin «KPI و مانیتورها» screen (bloom B-N9-08, D-89), computed on demand from the
// first-party analytics events and the existing tables. Aggregates only; any group under MinCohort is suppressed
// (null + suppressed) — never a per-user value.
type KPI struct {
	GeneratedAt string            `json:"generated_at"`
	EventsSince *string           `json:"events_since"`
	Funnel      Funnel            `json:"funnel"`
	Retention   Retention         `json:"retention"`
	Engine      *Engine           `json:"engine"`
	DataQuality any               `json:"data_quality"`
	AI          AIKPI             `json:"ai"`
	Notes       map[string]string `json:"notes"`
}

// FunnelStep is one activation step; Pct is null while the group is below MinCohort or not mature.
type FunnelStep struct {
	Key        string   `json:"key"`
	Pct        *float64 `json:"pct"`
	Count      *int     `json:"count"`
	Of         *int     `json:"of"`
	Suppressed bool     `json:"suppressed"`
}

// Funnel is the activation funnel of the users who signed up in the last Days days (after the first event).
type Funnel struct {
	Days  int          `json:"days"`
	Steps []FunnelStep `json:"steps"`
}

// CohortRow is one weekly signup cohort. Cells[i] is the share of the cohort active on day Offsets[i] after signup
// (null: not yet reached, or fewer than MinCohort users).
type CohortRow struct {
	WeekStart  string     `json:"week_start"`
	Size       *int       `json:"size"`
	Suppressed bool       `json:"suppressed"`
	Cells      []*float64 `json:"cells"`
}

// Retention is the D0–D60 weekly cohort table.
type Retention struct {
	Event   string      `json:"event"`
	Offsets []int       `json:"offsets"`
	Cohorts []CohortRow `json:"cohorts"`
}

// AIKPI are the 30-day AI / assistant indicators; null = no source or suppressed.
type AIKPI struct {
	Days                int      `json:"days"`
	LabsPerDay          *float64 `json:"labs_per_day"`
	OCRCorrectionPct    *float64 `json:"ocr_correction_pct"`
	CostPerAnalysisUSD  *float64 `json:"cost_per_analysis_usd"`
	AssistantMsgsPerDay *float64 `json:"assistant_msgs_per_day"`
	ReferralPct         *float64 `json:"referral_pct"`
	Suppressed          bool     `json:"suppressed"`
}

// RetentionOffsets are the cohort columns.
var RetentionOffsets = []int{0, 1, 7, 14, 30, 60}

// FunnelDays are the accepted funnel windows.
var FunnelDays = []int{30, 60, 90}

// CohortWeeks is how many weekly cohorts the table shows.
const CohortWeeks = 12

// KPIOptions selects the funnel window and the retention activity («any» event or «log_saved»).
type KPIOptions struct {
	Days  int
	Event string
}

func parseDay(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, time.UTC)
	return t
}

// weekStart is the Saturday on or before d (the Iranian week start).
func weekStart(d time.Time) time.Time {
	back := (int(d.Weekday()) + 1) % 7 // Saturday(6) → 0, Sunday(0) → 1 … Friday(5) → 6
	return d.AddDate(0, 0, -back)
}

func pct(n, of int) *float64 { return fp(round1(100 * float64(n) / float64(of))) }

// KPI computes the document for now (Tehran).
func (s *Service) KPI(ctx context.Context, now time.Time, o KPIOptions) (*KPI, error) {
	now = now.In(civildate.Tehran)
	if o.Days == 0 {
		o.Days = 90
	}
	if o.Event == "" {
		o.Event = "any"
	}
	today := parseDay(ymd(now))
	k := &KPI{GeneratedAt: now.Format(time.RFC3339), Notes: map[string]string{
		"events":       "Counts start with the first product analytics event; earlier signups are not in the funnel or the cohorts. Only users who opted in to anonymous statistics are recorded.",
		"suppression":  "Groups smaller than 10 users are hidden.",
		"data_quality": "The data-quality class is not stored per user, so its distribution is not available.",
		"ocr":          "Share of lab values the user edited after the extraction (last 30 days).",
		"cost":         "Average provider cost of a successful lab analysis call, USD.",
	}}

	var first sql.NullString
	if err := s.db.QueryRowContext(ctx, "SELECT DATE_FORMAT(MIN(occurred_on), '%Y-%m-%d') FROM analytics_events").Scan(&first); err != nil {
		return nil, fmt.Errorf("metrics: kpi first event: %w", err)
	}
	if first.Valid {
		k.EventsSince = &first.String
	}
	var err error
	if k.Funnel, err = s.funnel(ctx, today, first, o.Days); err != nil {
		return nil, err
	}
	if k.Retention, err = s.retention(ctx, today, first, o.Event); err != nil {
		return nil, err
	}
	if sn, err := s.Latest(ctx); err == nil {
		var p struct {
			Engine *Engine `json:"engine"`
		}
		if json.Unmarshal(sn.Payload, &p) == nil {
			k.Engine = p.Engine
		}
	}
	if k.AI, err = s.aiKPI(ctx, now); err != nil {
		return nil, err
	}
	return k, nil
}

var funnelKeys = []string{"signup", "app_open", "onboarding_done", "first_log", "log_d7", "log_d30"}

func (s *Service) funnel(ctx context.Context, today time.Time, first sql.NullString, days int) (Funnel, error) {
	f := Funnel{Days: days, Steps: make([]FunnelStep, 0, len(funnelKeys))}
	type agg struct{ n, of int }
	a := make([]agg, len(funnelKeys))
	if first.Valid {
		from := today.AddDate(0, 0, -(days - 1))
		if fd := parseDay(first.String); fd.After(from) {
			from = fd
		}
		rows, err := s.db.QueryContext(ctx, "SELECT DATE_FORMAT(DATE(u.created_at), '%Y-%m-%d'), "+
			"EXISTS(SELECT 1 FROM analytics_events e WHERE e.user_id = u.id AND e.name = 'app_open'), "+
			"EXISTS(SELECT 1 FROM analytics_events e WHERE e.user_id = u.id AND e.name = 'onboarding_done'), "+
			"EXISTS(SELECT 1 FROM analytics_events e WHERE e.user_id = u.id AND e.name = 'log_saved'), "+
			"EXISTS(SELECT 1 FROM analytics_events e WHERE e.user_id = u.id AND e.name = 'log_saved' AND e.occurred_on >= DATE_ADD(DATE(u.created_at), INTERVAL 7 DAY)), "+
			"EXISTS(SELECT 1 FROM analytics_events e WHERE e.user_id = u.id AND e.name = 'log_saved' AND e.occurred_on >= DATE_ADD(DATE(u.created_at), INTERVAL 30 DAY)) "+
			"FROM users u WHERE DATE(u.created_at) >= ? AND DATE(u.created_at) <= ?", ymd(from), ymd(today))
		if err != nil {
			return f, fmt.Errorf("metrics: funnel: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var cd string
			var fl [5]bool
			if err := rows.Scan(&cd, &fl[0], &fl[1], &fl[2], &fl[3], &fl[4]); err != nil {
				return f, err
			}
			age := int(today.Sub(parseDay(cd)).Hours() / 24)
			for i := range funnelKeys {
				if (i == 4 && age < 7) || (i == 5 && age < 30) {
					continue // not mature yet: counted in neither numerator nor denominator
				}
				a[i].of++
				if i == 0 || fl[i-1] { //nolint:gosec // i is 1..5 over a 5-element array
					a[i].n++
				}
			}
		}
		if err := rows.Err(); err != nil {
			return f, err
		}
	}
	for i, key := range funnelKeys {
		st := FunnelStep{Key: key}
		switch {
		case a[i].of >= MinCohort:
			st.Pct, st.Count, st.Of = pct(a[i].n, a[i].of), ip(a[i].n), ip(a[i].of)
		default:
			st.Suppressed = a[i].of > 0
		}
		f.Steps = append(f.Steps, st)
	}
	return f, nil
}

func (s *Service) retention(ctx context.Context, today time.Time, first sql.NullString, event string) (Retention, error) {
	r := Retention{Event: event, Offsets: RetentionOffsets, Cohorts: []CohortRow{}}
	if !first.Valid {
		return r, nil
	}
	cur := weekStart(today)
	oldest := cur.AddDate(0, 0, -7*(CohortWeeks-1))
	if fw := weekStart(parseDay(first.String)); fw.After(oldest) {
		oldest = fw
	}
	// Signups from the first event's week on: users who signed up earlier in that week may have no event for their day 0.
	rows, err := s.db.QueryContext(ctx, "SELECT id, DATE_FORMAT(DATE(created_at), '%Y-%m-%d') FROM users WHERE DATE(created_at) >= ? AND DATE(created_at) <= ?", ymd(oldest), ymd(today))
	if err != nil {
		return r, fmt.Errorf("metrics: cohorts: %w", err)
	}
	defer rows.Close()
	signup := map[uint64]time.Time{}
	for rows.Next() {
		var id uint64
		var cd string
		if err := rows.Scan(&id, &cd); err != nil {
			return r, err
		}
		signup[id] = parseDay(cd)
	}
	if err := rows.Err(); err != nil {
		return r, err
	}
	q := "SELECT DISTINCT e.user_id, DATE_FORMAT(e.occurred_on, '%Y-%m-%d') FROM analytics_events e JOIN users u ON u.id = e.user_id WHERE DATE(u.created_at) >= ?"
	args := []any{ymd(oldest)}
	if event != "any" {
		q += " AND e.name = ?"
		args = append(args, event)
	}
	erows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return r, fmt.Errorf("metrics: cohort events: %w", err)
	}
	defer erows.Close()
	active := map[uint64]map[int]bool{} // user → offsets (from RetentionOffsets) with activity
	want := map[int]bool{}
	for _, o := range RetentionOffsets {
		want[o] = true
	}
	for erows.Next() {
		var id uint64
		var d string
		if err := erows.Scan(&id, &d); err != nil {
			return r, err
		}
		cd, ok := signup[id]
		if !ok {
			continue
		}
		off := int(parseDay(d).Sub(cd).Hours() / 24)
		if want[off] {
			if active[id] == nil {
				active[id] = map[int]bool{}
			}
			active[id][off] = true
		}
	}
	if err := erows.Err(); err != nil {
		return r, err
	}

	byWeek := map[time.Time][]uint64{}
	for id, cd := range signup {
		w := weekStart(cd)
		byWeek[w] = append(byWeek[w], id)
	}
	weeks := make([]time.Time, 0, len(byWeek))
	for w := range byWeek {
		weeks = append(weeks, w)
	}
	sort.Slice(weeks, func(i, j int) bool { return weeks[i].Before(weeks[j]) })
	for _, w := range weeks {
		ids := byWeek[w]
		row := CohortRow{WeekStart: ymd(w), Cells: make([]*float64, len(RetentionOffsets))}
		if len(ids) < MinCohort {
			row.Suppressed = true
			r.Cohorts = append(r.Cohorts, row)
			continue
		}
		row.Size = ip(len(ids))
		for i, off := range RetentionOffsets {
			of, n := 0, 0
			for _, id := range ids {
				if int(today.Sub(signup[id]).Hours()/24) < off {
					continue // this member has not reached day `off` yet
				}
				of++
				if active[id][off] {
					n++
				}
			}
			if of >= MinCohort {
				row.Cells[i] = pct(n, of)
			}
		}
		r.Cohorts = append(r.Cohorts, row)
	}
	return r, nil
}

func (s *Service) aiKPI(ctx context.Context, now time.Time) (AIKPI, error) {
	const days = 30
	a := AIKPI{Days: days}
	since := now.AddDate(0, 0, -days)
	var labs, labUsers int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(DISTINCT user_id) FROM lab_reports WHERE source = 'upload' AND created_at >= ?", since).Scan(&labs, &labUsers); err != nil {
		return a, fmt.Errorf("metrics: ai labs: %w", err)
	}
	if labUsers >= MinCohort {
		a.LabsPerDay = fp(round1(float64(labs) / days))
	}
	var edited, total, markerUsers int
	if err := s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(m.source = 'edited'), 0), COUNT(*), COUNT(DISTINCT r.user_id) FROM lab_markers m JOIN lab_reports r ON r.id = m.lab_id "+
		"WHERE r.source = 'upload' AND m.source IN ('extracted', 'edited') AND r.created_at >= ?", since).Scan(&edited, &total, &markerUsers); err != nil {
		return a, fmt.Errorf("metrics: ai ocr: %w", err)
	}
	if markerUsers >= MinCohort && total > 0 {
		a.OCRCorrectionPct = pct(edited, total)
	}
	var cost sql.NullFloat64
	var calls, costUsers int
	if err := s.db.QueryRowContext(ctx, "SELECT SUM(cost_micros), COUNT(*), COUNT(DISTINCT user_id) FROM ai_usage_logs WHERE feature = 'lab_analysis' AND ok = 1 AND created_at >= ?", since).Scan(&cost, &calls, &costUsers); err != nil {
		return a, fmt.Errorf("metrics: ai cost: %w", err)
	}
	if costUsers >= MinCohort && calls > 0 {
		a.CostPerAnalysisUSD = fp(cost.Float64 / 1e6 / float64(calls))
	}
	var msgs, msgUsers int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(DISTINCT c.user_id) FROM assistant_messages m JOIN assistant_conversations c ON c.id = m.conversation_id WHERE m.role = 'user' AND m.created_at >= ?", since).Scan(&msgs, &msgUsers); err != nil {
		return a, fmt.Errorf("metrics: ai msgs: %w", err)
	}
	if msgUsers >= MinCohort {
		a.AssistantMsgsPerDay = fp(round1(float64(msgs) / days))
	}
	var convs, handoffs int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT c.id), COUNT(DISTINCT h.conversation_id) FROM assistant_conversations c LEFT JOIN assistant_handoffs h ON h.conversation_id = c.id WHERE c.created_at >= ?", since).Scan(&convs, &handoffs); err != nil {
		return a, fmt.Errorf("metrics: ai referral: %w", err)
	}
	if convs >= MinCohort {
		a.ReferralPct = pct(handoffs, convs)
	}
	a.Suppressed = a.LabsPerDay == nil || a.AssistantMsgsPerDay == nil
	return a, nil
}
