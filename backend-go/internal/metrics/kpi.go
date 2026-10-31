package metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// KPI is the document behind the admin «KPI و مانیتورها» screen (bloom B-N9-08, D-89; SPEC v6 §5.6, T-M10-30, D-179).
// It is read from the stored snapshots (the newest one and the one a period earlier, for the deltas) and never
// computed on the request. Aggregates only; any group under MinCohort is suppressed (null + suppressed).
type KPI struct {
	Day         *string           `json:"day"`
	GeneratedAt *string           `json:"generated_at"`
	EventsSince *string           `json:"events_since"` // first opt-in analytics event (the install step's source)
	Activity    *Activity         `json:"activity"`
	Funnel      Funnel            `json:"funnel"`
	Retention   Retention         `json:"retention"`
	Engine      *Engine           `json:"engine"`
	DataQuality *Distribution     `json:"data_quality"`
	RUI         *Rate             `json:"rui_resolved_7d"`
	Revenue     *Revenue          `json:"revenue"`
	AI          AIKPI             `json:"ai"`
	AICost      *AICost           `json:"ai_cost"`
	MessageCTR  *MessageCTR       `json:"message_ctr"`
	Family      any               `json:"family"` // null until T-M11-15
	Previous    *string           `json:"previous_day"`
	Deltas      map[string]Delta  `json:"deltas"`
	Notes       map[string]string `json:"notes"`
}

// Activity is the §5.6 activity block of the KPI document.
type Activity struct {
	DAU        Value `json:"dau"`
	MAU        Value `json:"mau"`
	Stickiness Value `json:"stickiness_pct"`
	Loggers    Value `json:"loggers"`
	Loggers15d Rate  `json:"loggers_15d"`
}

// Revenue is trial conversion and MRR (kept from the overview).
type Revenue struct {
	TrialToPaid Rate   `json:"trial_to_paid"`
	MRRToman    *int64 `json:"mrr_toman"`
}

// Delta is one metric now and a period earlier.
type Delta struct {
	Value *float64 `json:"value"`
	Prev  *float64 `json:"prev"`
	Delta *float64 `json:"delta"`
}

// FunnelStep is one activation step; Pct is null while the group is below MinCohort or not mature.
type FunnelStep struct {
	Key        string   `json:"key"`
	Source     string   `json:"source"` // server | opt_in_analytics
	Pct        *float64 `json:"pct"`
	Count      *int     `json:"count"`
	Of         *int     `json:"of"`
	Suppressed bool     `json:"suppressed"`
}

// Funnel is the activation funnel of the users who signed up in the last Days days.
type Funnel struct {
	Days  int          `json:"days"`
	Steps []FunnelStep `json:"steps"`
}

// CohortRow is one weekly signup cohort. Cells[i] is the share of the cohort active within ±1 day of day Offsets[i]
// after signup (null: not yet reached, or fewer than MinCohort users).
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

// Bucket is one class of a distribution.
type Bucket struct {
	Key        string   `json:"key"`
	Pct        *float64 `json:"pct"`
	Count      *int     `json:"count"`
	Suppressed bool     `json:"suppressed"`
}

// Distribution is a class distribution over the active users of one engine snapshot day.
type Distribution struct {
	Day        *string  `json:"day"`
	Of         *int     `json:"of"`
	Suppressed bool     `json:"suppressed"`
	Buckets    []Bucket `json:"buckets"`
}

// Distributions are the confidence and data_quality distributions (A-06).
type Distributions struct {
	Confidence  Distribution `json:"confidence"`
	DataQuality Distribution `json:"data_quality"`
}

// AICost is the AI API cost of the lab analysis and the assistant over Plus revenue (last AICostDays days).
type AICost struct {
	Days       int      `json:"days"`
	CostUSD    *float64 `json:"cost_usd"`
	FXRate     *float64 `json:"usd_irr_rate"`
	RevenueIRR *int64   `json:"revenue_irr"`
	SharePct   *float64 `json:"share_pct"`
	Suppressed bool     `json:"suppressed"`
	Note       string   `json:"note"`
}

// VersionCTR is the CTR of one message rule version.
type VersionCTR struct {
	RuleID     uint64   `json:"rule_id"`
	Version    uint32   `json:"version"`
	Shown      *int64   `json:"shown"`
	Clicked    *int64   `json:"clicked"`
	CTRPct     *float64 `json:"ctr_pct"`
	Suppressed bool     `json:"suppressed"`
}

// MessageCTR is clicked / shown per message version (T-M10-06 rollup) and overall, last MessageDays days.
type MessageCTR struct {
	Days     int          `json:"days"`
	Overall  Rate         `json:"overall"`
	Versions []VersionCTR `json:"versions"`
}

// KPIData is the KPI block stored in each snapshot (every variant the endpoint can ask for).
type KPIData struct {
	EventsSince   *string              `json:"events_since"`
	Funnels       map[string]Funnel    `json:"funnels"`   // by window: "30", "60", "90"
	Retention     map[string]Retention `json:"retention"` // by event: "any", "log_saved"
	Distributions Distributions        `json:"distributions"`
	RUI           Rate                 `json:"rui_resolved_7d"`
	AI            AIKPI                `json:"ai"`
	AICost        AICost               `json:"ai_cost"`
	MessageCTR    MessageCTR           `json:"message_ctr"`
	Family        any                  `json:"family"` // null until T-M11-15
}

// RetentionOffsets are the cohort columns.
var RetentionOffsets = []int{0, 1, 7, 14, 30, 60}

// FunnelDays are the accepted funnel windows.
var FunnelDays = []int{30, 60, 90}

// RetentionEvents are the accepted retention activities: any activity day, or a day with a data log.
var RetentionEvents = []string{"any", "log_saved"}

// CohortWeeks is how many weekly cohorts the table shows.
const CohortWeeks = 12

// Windows of the other KPIs.
const (
	AICostDays  = 30
	MessageDays = 30
	RUIDays     = 30 // episodes that started in the 30 days ending RUIWithin days ago
	RUIWithin   = 7
	// FXRateSetting is the plus_settings key of the admin-set USD → IRR rate of the AI cost share.
	FXRateSetting = "ai_cost_usd_irr_rate"
	// maxCTRVersions bounds the per-version CTR list (most shown first).
	maxCTRVersions = 50
)

// KPIOptions selects the funnel window and the retention activity («any» or «log_saved»).
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

func strp(s string) *string { return &s }

// KPI reads the document from the newest snapshot (query 1) and the one o.Days earlier (query 2, the deltas). Before
// the first snapshot every block is empty.
func (s *Service) KPI(ctx context.Context, _ time.Time, o KPIOptions) (*KPI, error) {
	if o.Days == 0 {
		o.Days = 90
	}
	if o.Event == "" {
		o.Event = "any"
	}
	k := &KPI{
		Funnel:    Funnel{Days: o.Days, Steps: []FunnelStep{}},
		Retention: Retention{Event: o.Event, Offsets: RetentionOffsets, Cohorts: []CohortRow{}},
		AI:        AIKPI{Days: 30},
		Deltas:    map[string]Delta{},
		Notes:     kpiNotes(),
	}
	sn, err := s.Latest(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return k, nil
	}
	if err != nil {
		return nil, fmt.Errorf("metrics: kpi snapshot: %w", err)
	}
	var p Payload
	if err := json.Unmarshal(sn.Payload, &p); err != nil {
		return nil, fmt.Errorf("metrics: kpi payload: %w", err)
	}
	k.Day = strp(sn.Day)
	k.GeneratedAt = strp(sn.ComputedAt.In(civildate.Tehran).Format(time.RFC3339))
	k.Activity = &Activity{DAU: p.DAU, MAU: p.MAU, Stickiness: p.Stickiness, Loggers: p.Loggers, Loggers15d: p.Loggers15d}
	eng := p.Engine
	k.Engine = &eng
	k.Revenue = &Revenue{TrialToPaid: p.TrialToPaid, MRRToman: p.MRRToman}
	if d := p.KPI; d != nil {
		k.EventsSince = d.EventsSince
		if f, ok := d.Funnels[strconv.Itoa(o.Days)]; ok {
			k.Funnel = f
		}
		if r, ok := d.Retention[o.Event]; ok {
			k.Retention = r
		}
		dq, rui, ac, ctr := d.Distributions.DataQuality, d.RUI, d.AICost, d.MessageCTR
		k.DataQuality, k.RUI, k.AICost, k.MessageCTR, k.AI = &dq, &rui, &ac, &ctr, d.AI
	}

	cur := MetricValues(&p)
	var prev map[string]float64
	day, _ := civildate.Parse(sn.Day)
	if ps, err := s.OnOrBefore(ctx, day.AddDays(-o.Days)); err == nil {
		var pp Payload
		if json.Unmarshal(ps.Payload, &pp) == nil {
			prev = MetricValues(&pp)
			k.Previous = strp(ps.Day)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("metrics: kpi previous snapshot: %w", err)
	}
	for _, key := range MetricKeys {
		d := Delta{}
		if v, ok := cur[key]; ok {
			d.Value = fp(v)
		}
		if v, ok := prev[key]; ok {
			d.Prev = fp(v)
		}
		if d.Value != nil && d.Prev != nil {
			d.Delta = fp(round1(*d.Value - *d.Prev))
		}
		k.Deltas[key] = d
	}
	return k, nil
}

func kpiNotes() map[string]string {
	return map[string]string{
		"source":       "Read from the hourly KPI snapshot (admin_metrics:snapshot); deltas compare with the snapshot one period earlier.",
		"suppression":  "Groups smaller than 10 users are hidden.",
		"funnel":       "Server facts of the users who signed up (verified the OTP) in the window; percentages are of those users. The install step comes from opt-in analytics only.",
		"retention":    "Share of the weekly signup cohort with activity (any, or a data log) within ±1 day of day n.",
		"engine":       "Engine prediction log: first prediction of cycles built from at least 2 real cycles; pregnancy, postpartum and hormonal contraception excluded.",
		"data_quality": "Distribution over the active users of the newest engine snapshot day.",
		"rui":          "requires_user_input episodes that started 7–36 days ago and were cleared by a user input within 7 days.",
		"ai_cost":      "Lab analysis + assistant API cost (USD × the admin-set rate) over the Plus revenue of the last 30 days.",
		"ocr":          "Share of lab values the user edited after the extraction (last 30 days).",
		"message_ctr":  "clicked / shown per message version, last 30 days (daily rollup).",
		"family":       "Family KPIs come with T-M11-15.",
	}
}

// MetricKeys are the §5.6 scalar metric keys (OKR catalogue and KPI deltas), in display order.
var MetricKeys = []string{
	"dau", "mau", "stickiness_pct", "loggers", "loggers_15d", "signups_today", "activation_first_log_pct",
	"activation_log_d7_pct", "retention_d7", "retention_d30", "engine_mae", "accuracy_2d", "engine_mean_error_days",
	"engine_within_2_days_pct", "low_confidence_share", "low_data_quality_share", "rui_resolved_7d", "plus_active",
	"trial_to_paid_pct", "mrr_toman", "ai_cost_share", "ocr_correction_rate", "message_ctr",
}

// MetricValues are the scalar §5.6 metrics of a stored payload; a key is absent while it is not computable
// (suppressed, no data, a v1 snapshot without it).
func MetricValues(p *Payload) map[string]float64 {
	v := map[string]float64{}
	put := func(k string, x *float64) {
		if x != nil {
			v[k] = *x
		}
	}
	put("dau", p.DAU.Value)
	put("mau", p.MAU.Value)
	put("stickiness_pct", p.Stickiness.Value)
	put("signups_today", p.SignupsToday.Value)
	put("plus_active", p.PlusActive.Value)
	put("trial_to_paid_pct", p.TrialToPaid.Pct)
	if p.MRRToman != nil {
		v["mrr_toman"] = float64(*p.MRRToman)
	}
	if p.Version >= PayloadVersion {
		put("loggers", p.Loggers.Value)
		put("loggers_15d", p.Loggers15d.Pct)
	}
	if !p.Engine.Suppressed {
		put("engine_within_2_days_pct", p.Engine.Within2DaysPct)
		put("engine_mean_error_days", p.Engine.MeanErrorDays)
		put("engine_mae", p.Engine.MeanErrorDays)
		put("accuracy_2d", p.Engine.Within2DaysPct)
	}
	d := p.KPI
	if d == nil {
		return v
	}
	if f, ok := d.Funnels["90"]; ok {
		for _, st := range f.Steps {
			switch st.Key {
			case "first_log":
				put("activation_first_log_pct", st.Pct)
			case "log_d7":
				put("activation_log_d7_pct", st.Pct)
			}
		}
	}
	if r, ok := d.Retention["log_saved"]; ok {
		for key, off := range map[string]int{"retention_d7": 7, "retention_d30": 30} {
			put(key, newestCell(r, off))
		}
	}
	put("low_confidence_share", bucketPct(d.Distributions.Confidence, "low"))
	if dq := d.Distributions.DataQuality; !dq.Suppressed && dq.Of != nil {
		poor, seen := 0.0, false
		for _, b := range dq.Buckets {
			if (b.Key == "poor" || b.Key == "insufficient") && b.Pct != nil {
				poor += *b.Pct
				seen = true
			}
		}
		if seen {
			v["low_data_quality_share"] = round1(poor)
		}
	}
	put("rui_resolved_7d", d.RUI.Pct)
	put("ai_cost_share", d.AICost.SharePct)
	put("ocr_correction_rate", d.AI.OCRCorrectionPct)
	put("message_ctr", d.MessageCTR.Overall.Pct)
	return v
}

// newestCell is the newest large-enough cohort's value at offset off.
func newestCell(r Retention, off int) *float64 {
	col := -1
	for i, o := range r.Offsets {
		if o == off {
			col = i
		}
	}
	for i := len(r.Cohorts) - 1; i >= 0 && col >= 0; i-- {
		c := r.Cohorts[i]
		if !c.Suppressed && col < len(c.Cells) && c.Cells[col] != nil {
			return c.Cells[col]
		}
	}
	return nil
}

func bucketPct(d Distribution, key string) *float64 {
	if d.Suppressed || d.Of == nil {
		return nil
	}
	for _, b := range d.Buckets {
		if b.Key == key {
			if b.Pct != nil {
				return b.Pct
			}
			return nil
		}
	}
	return fp(0) // no user in that class
}

// computeKPI builds the KPI block for now (Tehran): run by the snapshot job only.
func (s *Service) computeKPI(ctx context.Context, now time.Time) (*KPIData, error) {
	now = now.In(civildate.Tehran)
	today := parseDay(ymd(now))
	d := &KPIData{Funnels: map[string]Funnel{}, Retention: map[string]Retention{}}

	var first sql.NullString
	if err := s.db.QueryRowContext(ctx, "SELECT DATE_FORMAT(MIN(occurred_on), '%Y-%m-%d') FROM analytics_events").Scan(&first); err != nil {
		return nil, fmt.Errorf("metrics: kpi first event: %w", err)
	}
	if first.Valid {
		d.EventsSince = &first.String
	}
	for _, days := range FunnelDays {
		f, err := s.funnel(ctx, today, days)
		if err != nil {
			return nil, err
		}
		d.Funnels[strconv.Itoa(days)] = f
	}
	ret, err := s.retention(ctx, today)
	if err != nil {
		return nil, err
	}
	d.Retention = ret
	td := civildate.InTehran(now)
	if d.Distributions, err = s.distributions(ctx, td); err != nil {
		return nil, err
	}
	if d.RUI, err = s.rui(ctx, td); err != nil {
		return nil, err
	}
	if d.AI, err = s.aiKPI(ctx, now); err != nil {
		return nil, err
	}
	if d.AICost, err = s.aiCost(ctx, now); err != nil {
		return nil, err
	}
	if d.MessageCTR, err = s.messageCTR(ctx, td); err != nil {
		return nil, err
	}
	return d, nil
}

// Funnel step keys (SPEC §5.6: install → OTP → onboarding → first log → log on day 7 → log on day 30).
var funnelKeys = []string{"install", "otp_verified", "onboarding_done", "first_log", "log_d7", "log_d30"}

// FunnelFacts is one signed-up user's funnel facts (age = days since the signup day).
type FunnelFacts struct {
	Age                                           int
	OptInOpen, Onboarded, FirstLog, LogD7, LogD30 bool
}

// FunnelOf folds the facts into the steps. log_d7 / log_d30 (±1 day) count only users old enough (8 / 31 days);
// every percentage is of the users that step considers.
func FunnelOf(days int, facts []FunnelFacts) Funnel {
	type agg struct{ n, of int }
	a := make([]agg, len(funnelKeys))
	for _, f := range facts {
		hit := []bool{f.OptInOpen, true, f.Onboarded, f.FirstLog, f.LogD7, f.LogD30}
		for i := range funnelKeys {
			if (i == 4 && f.Age < 8) || (i == 5 && f.Age < 31) {
				continue // not mature yet: counted in neither numerator nor denominator
			}
			a[i].of++
			if hit[i] {
				a[i].n++
			}
		}
	}
	out := Funnel{Days: days, Steps: make([]FunnelStep, 0, len(funnelKeys))}
	for i, key := range funnelKeys {
		st := FunnelStep{Key: key, Source: "server"}
		if key == "install" {
			st.Source = "opt_in_analytics"
		}
		if a[i].of >= MinCohort {
			st.Pct, st.Count, st.Of = pct(a[i].n, a[i].of), ip(a[i].n), ip(a[i].of)
		} else {
			st.Suppressed = a[i].of > 0
		}
		out.Steps = append(out.Steps, st)
	}
	return out
}

func (s *Service) funnel(ctx context.Context, today time.Time, days int) (Funnel, error) {
	from := today.AddDate(0, 0, -(days - 1))
	rows, err := s.q.ListSignupFacts(ctx, store.ListSignupFactsParams{
		FromAt: sql.NullTime{Time: tehranMidnight(from), Valid: true}, ToAt: sql.NullTime{Time: tehranMidnight(today.AddDate(0, 0, 1)), Valid: true},
	})
	if err != nil {
		return Funnel{}, fmt.Errorf("metrics: funnel: %w", err)
	}
	td := civildate.FromTime(today)
	facts := make([]FunnelFacts, 0, len(rows))
	for _, r := range rows {
		facts = append(facts, FunnelFacts{Age: r.SignupDay.DiffDays(td), OptInOpen: r.OptInOpen != 0, Onboarded: r.Onboarded != 0,
			FirstLog: r.FirstLog != 0, LogD7: r.LoggedD7 != 0, LogD30: r.LoggedD30 != 0})
	}
	return FunnelOf(days, facts), nil
}

func tehranMidnight(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, civildate.Tehran)
}

// ActivityDay is one cohort member's activity day as an offset from the signup day.
type ActivityDay struct {
	UserID uint64
	Offset int
	Logged bool
}

// RetentionOf builds the weekly cohort table: signup = user → signup day, days = their activity offsets. A cell is the
// share of the members who reached day off+1 with activity (logged only for "log_saved") on a day in [off−1, off+1].
func RetentionOf(today time.Time, event string, signup map[uint64]time.Time, days []ActivityDay) Retention {
	r := Retention{Event: event, Offsets: RetentionOffsets, Cohorts: []CohortRow{}}
	active := map[uint64]map[int]bool{}
	for _, d := range days {
		if event == "log_saved" && !d.Logged {
			continue
		}
		if active[d.UserID] == nil {
			active[d.UserID] = map[int]bool{}
		}
		active[d.UserID][d.Offset] = true
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
				if int(today.Sub(signup[id]).Hours()/24) < off+1 {
					continue // this member has not reached the end of the ±1 window yet
				}
				of++
				a := active[id]
				if a[off-1] || a[off] || a[off+1] {
					n++
				}
			}
			if of >= MinCohort {
				row.Cells[i] = pct(n, of)
			}
		}
		r.Cohorts = append(r.Cohorts, row)
	}
	return r
}

func (s *Service) retention(ctx context.Context, today time.Time) (map[string]Retention, error) {
	oldest := weekStart(today).AddDate(0, 0, -7*(CohortWeeks-1))
	from := sql.NullTime{Time: tehranMidnight(oldest), Valid: true}
	users, err := s.q.ListSignupDays(ctx, store.ListSignupDaysParams{FromAt: from, ToAt: sql.NullTime{Time: tehranMidnight(today.AddDate(0, 0, 1)), Valid: true}})
	if err != nil {
		return nil, fmt.Errorf("metrics: cohorts: %w", err)
	}
	signup := make(map[uint64]time.Time, len(users))
	sd := make(map[uint64]civildate.Date, len(users))
	for _, u := range users {
		signup[u.ID] = parseDay(u.SignupDay.String())
		sd[u.ID] = u.SignupDay
	}
	rows, err := s.q.ListCohortActivityDays(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("metrics: cohort activity: %w", err)
	}
	days := make([]ActivityDay, 0, len(rows))
	for _, r := range rows {
		if d, ok := sd[r.UserID]; ok {
			days = append(days, ActivityDay{UserID: r.UserID, Offset: d.DiffDays(r.Day), Logged: r.Logged})
		}
	}
	out := map[string]Retention{}
	for _, ev := range RetentionEvents {
		out[ev] = RetentionOf(today, ev, signup, days)
	}
	return out, nil
}

// Spec orders of the distribution classes.
var (
	confidenceClasses  = []string{"high", "medium", "low", "unknown"}
	dataQualityClasses = []string{"good", "partial", "poor", "insufficient"}
)

func (s *Service) distributions(ctx context.Context, today civildate.Date) (Distributions, error) {
	out := Distributions{Confidence: Distribution{Buckets: []Bucket{}}, DataQuality: Distribution{Buckets: []Bucket{}}}
	day, err := s.q.LatestEngineSnapshotDay(ctx, today)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, fmt.Errorf("metrics: snapshot day: %w", err)
	}
	rows, err := s.q.CountEngineSnapshotClasses(ctx, day)
	if err != nil {
		return out, fmt.Errorf("metrics: snapshot classes: %w", err)
	}
	conf, dq := map[string]int{}, map[string]int{}
	total := 0
	for _, r := range rows {
		conf[r.Confidence] += int(r.Users)
		dq[r.DataQuality] += int(r.Users)
		total += int(r.Users)
	}
	ds := day.String()
	out.Confidence = DistributionOf(ds, total, confidenceClasses, conf)
	out.DataQuality = DistributionOf(ds, total, dataQualityClasses, dq)
	return out, nil
}

// DistributionOf is the distribution of counts over classes (the spec order, then any unknown class): hidden while the
// total is below MinCohort; a class under MinCohort keeps its share but hides its count.
func DistributionOf(day string, total int, classes []string, counts map[string]int) Distribution {
	d := Distribution{Day: &day, Buckets: []Bucket{}}
	if total < MinCohort {
		d.Suppressed = total > 0
		return d
	}
	d.Of = ip(total)
	keys := append([]string(nil), classes...)
	extra := []string{}
	for k := range counts {
		known := false
		for _, c := range classes {
			known = known || c == k
		}
		if !known {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range append(keys, extra...) {
		n := counts[k]
		b := Bucket{Key: k, Pct: pct(n, total)}
		if n >= MinCohort || n == 0 {
			b.Count = ip(n)
		} else {
			b.Suppressed = true
		}
		d.Buckets = append(d.Buckets, b)
	}
	return d
}

// ruiMask is the warning bits that require user input (§2.33).
func ruiMask() uint32 {
	var m uint32
	for w := engine.Warning(0); w < 32; w++ {
		if w.RequiresUserInput() {
			m |= 1 << w
		}
	}
	return m
}

// SnapshotPoint is one user's engine snapshot day (warnings and revision).
type SnapshotPoint struct {
	UserID   uint64
	Day      civildate.Date
	Warnings uint32
	Rev      uint64
}

// RUIOf counts the requires_user_input episodes that started in [from, to] (points in user, day order): an episode is
// a run of snapshot days with a user-input warning; it is resolved by the user when the first following day without
// one has a newer engine_rev (an engine-input write), and counts when that day is at most RUIWithin days after the
// start. Suppressed below MinCohort users.
func RUIOf(points []SnapshotPoint, mask uint32, from, to civildate.Date) Rate {
	n, of := 0, 0
	users := map[uint64]bool{}
	for i := 0; i < len(points); {
		p := points[i]
		prevRUI := i > 0 && points[i-1].UserID == p.UserID && points[i-1].Warnings&mask != 0
		if p.Warnings&mask == 0 || prevRUI {
			i++
			continue
		}
		start, lastRev := p.Day, p.Rev
		j := i + 1
		for j < len(points) && points[j].UserID == p.UserID && points[j].Warnings&mask != 0 {
			lastRev = points[j].Rev
			j++
		}
		if !start.Before(from) && !start.After(to) {
			of++
			users[p.UserID] = true
			if j < len(points) && points[j].UserID == p.UserID && points[j].Rev > lastRev &&
				start.DiffDays(points[j].Day) <= RUIWithin {
				n++
			}
		}
		i = j
	}
	if len(users) < MinCohort {
		return Rate{Of: ip(of), Suppressed: true}
	}
	return Rate{Pct: pct(n, of), Count: n, Of: ip(of)}
}

func (s *Service) rui(ctx context.Context, today civildate.Date) (Rate, error) {
	to := today.AddDays(-RUIWithin)
	from := to.AddDays(-(RUIDays - 1))
	mask := ruiMask()
	rows, err := s.q.ListUserInputSnapshots(ctx, store.ListUserInputSnapshotsParams{
		FromDay: from.AddDays(-14), ToDay: today, RuiFromDay: from, RuiToDay: to, RuiMask: mask,
	})
	if err != nil {
		return Rate{}, fmt.Errorf("metrics: rui: %w", err)
	}
	pts := make([]SnapshotPoint, 0, len(rows))
	for _, r := range rows {
		pts = append(pts, SnapshotPoint{UserID: r.UserID, Day: r.Day, Warnings: r.WarningsMask, Rev: r.EngineRev})
	}
	return RUIOf(pts, mask, from, to), nil
}

// AICostShare is cost (USD) × rate (IRR per USD) ÷ revenue (IRR), as a percentage.
func AICostShare(costUSD, rate float64, revenueIRR int64) *float64 {
	if rate <= 0 || revenueIRR <= 0 {
		return nil
	}
	return fp(round1(100 * costUSD * rate / float64(revenueIRR)))
}

func (s *Service) aiCost(ctx context.Context, now time.Time) (AICost, error) {
	a := AICost{Days: AICostDays, Note: "Set the USD → IRR rate as plus_settings `" + FXRateSetting + "` (IRR per USD)."}
	since := now.AddDate(0, 0, -AICostDays)
	cost, err := s.q.SumAICost(ctx, sql.NullTime{Time: since, Valid: true})
	if err != nil {
		return a, fmt.Errorf("metrics: ai cost: %w", err)
	}
	rev, err := s.q.SumPlusRevenue(ctx, store.SumPlusRevenueParams{FromAt: sql.NullTime{Time: since, Valid: true}, ToAt: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return a, fmt.Errorf("metrics: plus revenue: %w", err)
	}
	if v, err := s.q.GetKPISetting(ctx, FXRateSetting); err == nil {
		if r, perr := strconv.ParseFloat(strings.TrimSpace(v), 64); perr == nil && r > 0 {
			a.FXRate = fp(r)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return a, fmt.Errorf("metrics: fx rate: %w", err)
	}
	if cost.Users < MinCohort || rev.Payers < MinCohort {
		a.Suppressed = true
		return a, nil
	}
	usd := float64(cost.CostMicros) / 1e6
	a.CostUSD = fp(usd)
	a.RevenueIRR = &rev.RevenueRials
	if a.FXRate != nil {
		a.SharePct = AICostShare(usd, *a.FXRate, rev.RevenueRials)
	}
	return a, nil
}

func (s *Service) messageCTR(ctx context.Context, today civildate.Date) (MessageCTR, error) {
	m := MessageCTR{Days: MessageDays, Versions: []VersionCTR{}}
	rows, err := s.q.ListMessageVersionCTR(ctx, store.ListMessageVersionCTRParams{SinceDay: today.AddDays(-MessageDays), UntilDay: today.AddDays(-1)})
	if err != nil {
		return m, fmt.Errorf("metrics: message ctr: %w", err)
	}
	var shown, clicked int64
	for _, r := range rows {
		shown += r.Shown
		clicked += r.Clicked
		if len(m.Versions) >= maxCTRVersions {
			continue
		}
		v := VersionCTR{RuleID: r.RuleID, Version: r.Version}
		if r.Shown >= MinCohort {
			sh, cl := r.Shown, r.Clicked
			v.Shown, v.Clicked, v.CTRPct = &sh, &cl, pct(int(cl), int(sh))
		} else {
			v.Suppressed = true
		}
		m.Versions = append(m.Versions, v)
	}
	m.Overall = rate(int(clicked), int(shown))
	return m, nil
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
