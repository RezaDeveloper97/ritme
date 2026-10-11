// Package okr is the OKR module of the admin panel (bloom B-N9-09, D-95): objectives and key results with baseline,
// target, owner and period. A key result bound to a live metric key reads its value from the newest metrics snapshot
// (and the KPI cohorts) on every request, so progress updates by itself; a manual key result uses the value the admin
// typed. Aggregates only — nothing here touches a user row.
package okr

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ritme/backend-go/internal/metrics"
)

// Statuses of a key result / objective.
const (
	StatusOnTrack = "on_track"
	StatusAtRisk  = "at_risk"
	StatusDone    = "done"
	StatusNoData  = "no_data"
)

// paceFloor: a key result is on track while its progress is at least this share of the time elapsed in the period.
const paceFloor = 0.8

// Metric is one bindable live metric.
type Metric struct {
	Key  string
	Unit string // count | pct | toman | days
}

// Metrics is the catalogue of keys a key result can bind to, in display order.
var Metrics = []Metric{
	{"dau", "count"}, {"mau", "count"}, {"stickiness_pct", "pct"}, {"signups_today", "count"},
	{"plus_active", "count"}, {"trial_to_paid_pct", "pct"}, {"mrr_toman", "toman"},
	{"retention_d7", "pct"}, {"retention_d30", "pct"},
	{"engine_within_2_days_pct", "pct"}, {"engine_mean_error_days", "days"},
}

// Units a manual key result may use.
var Units = []string{"count", "pct", "toman", "days"}

// MetricUnit returns the unit of a catalogue key.
func MetricUnit(key string) (string, bool) {
	for _, m := range Metrics {
		if m.Key == key {
			return m.Unit, true
		}
	}
	return "", false
}

// Progress is the share (0..1) of the way from baseline to target; works for decreasing targets too.
func Progress(baseline, target, current float64) float64 {
	if target == baseline {
		if current >= target {
			return 1
		}
		return 0
	}
	return math.Max(0, math.Min(1, (current-baseline)/(target-baseline)))
}

// Elapsed is the share (0..1) of the period [start, end] (inclusive Tehran dates) that has passed at now.
func Elapsed(start, end, now time.Time) float64 {
	if !now.After(start) {
		return 0
	}
	total := end.Sub(start).Hours()/24 + 1
	if total <= 0 {
		return 1
	}
	return math.Min(1, (now.Sub(start).Hours()/24+1)/total)
}

// StatusOf classifies a progress against the time elapsed: done at 100 %, on track while at least 80 % of the
// expected pace, otherwise at risk. A period that has not started is on track.
func StatusOf(progress, elapsed float64) string {
	switch {
	case progress >= 1:
		return StatusDone
	case progress >= elapsed*paceFloor:
		return StatusOnTrack
	}
	return StatusAtRisk
}

// KeyResult is a stored key result.
type KeyResult struct {
	ID          uint64
	ObjectiveID uint64
	Title       string
	MetricKey   sql.NullString
	Unit        string
	Baseline    float64
	Target      float64
	ManualValue sql.NullFloat64
	Owner       sql.NullString
	SortOrder   int
}

// Objective is a stored objective with its key results.
type Objective struct {
	ID         uint64
	Title      string
	Owner      sql.NullString
	Period     string
	StartsOn   time.Time
	EndsOn     time.Time
	SortOrder  int
	KeyResults []KeyResult
}

// Evaluated is a key result with its live value, progress and status.
type Evaluated struct {
	KeyResult
	Current  *float64
	Progress *float64
	Status   string
}

// EvaluatedObjective is an objective with its evaluated key results and rolled-up progress.
type EvaluatedObjective struct {
	Objective
	KRs      []Evaluated
	Progress *float64
	Status   string
}

// Summary is the OKR block of the overview (counts over the objectives whose period contains today).
type Summary struct {
	Objectives int `json:"objectives"`
	KeyResults int `json:"key_results"`
	OnTrack    int `json:"on_track"`
	AtRisk     int `json:"at_risk"`
	Done       int `json:"done"`
	NoData     int `json:"no_data"`
}

// Service reads and writes OKRs and resolves live metrics.
type Service struct {
	db      *sql.DB
	metrics *metrics.Service
}

// New builds the service; svc is the metrics service the bound key results read from.
func New(db *sql.DB, svc *metrics.Service) *Service { return &Service{db: db, metrics: svc} }

// Values are the resolved live metric values; a key is absent while it is not computable (no snapshot, suppressed).
type Values map[string]float64

// LiveValues resolves every catalogue key from the newest snapshot; the retention keys read the KPI cohorts and are
// only computed when withRetention is set (it is an on-demand query).
func (s *Service) LiveValues(ctx context.Context, now time.Time, withRetention bool) (Values, error) {
	v := Values{}
	sn, err := s.metrics.Latest(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return v, nil
	}
	if err != nil {
		return nil, err
	}
	var p metrics.Payload
	if err := json.Unmarshal(sn.Payload, &p); err != nil {
		return nil, fmt.Errorf("okr: snapshot payload: %w", err)
	}
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
	if !p.Engine.Suppressed {
		put("engine_within_2_days_pct", p.Engine.Within2DaysPct)
		put("engine_mean_error_days", p.Engine.MeanErrorDays)
	}
	if withRetention {
		k, err := s.metrics.KPI(ctx, now, metrics.KPIOptions{})
		if err != nil {
			return nil, err
		}
		for key, off := range map[string]int{"retention_d7": 7, "retention_d30": 30} {
			col := -1
			for i, o := range metrics.RetentionOffsets {
				if o == off {
					col = i
				}
			}
			// Newest weekly cohort that is large enough and old enough to have a value.
			for i := len(k.Retention.Cohorts) - 1; i >= 0 && col >= 0; i-- {
				c := k.Retention.Cohorts[i]
				if !c.Suppressed && col < len(c.Cells) && c.Cells[col] != nil {
					v[key] = *c.Cells[col]
					break
				}
			}
		}
	}
	return v, nil
}

func isRetention(k sql.NullString) bool {
	return k.Valid && (k.String == "retention_d7" || k.String == "retention_d30")
}

// Evaluate resolves live values (once) and computes progress and status for each objective.
func (s *Service) Evaluate(ctx context.Context, now time.Time, objs []Objective) ([]EvaluatedObjective, error) {
	need := false
	for _, o := range objs {
		for _, kr := range o.KeyResults {
			need = need || isRetention(kr.MetricKey)
		}
	}
	live, err := s.LiveValues(ctx, now, need)
	if err != nil {
		return nil, err
	}
	return EvaluateWith(objs, live, now), nil
}

// EvaluateWith is Evaluate over already resolved values (pure; unit-tested).
func EvaluateWith(objs []Objective, live Values, now time.Time) []EvaluatedObjective {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	out := make([]EvaluatedObjective, 0, len(objs))
	for _, o := range objs {
		el := Elapsed(o.StartsOn, o.EndsOn, day)
		eo := EvaluatedObjective{Objective: o, KRs: make([]Evaluated, 0, len(o.KeyResults)), Status: StatusNoData}
		sum, n, done := 0.0, 0, 0
		for _, kr := range o.KeyResults {
			e := Evaluated{KeyResult: kr, Status: StatusNoData}
			if kr.MetricKey.Valid {
				if x, ok := live[kr.MetricKey.String]; ok {
					e.Current = &x
				}
			} else if kr.ManualValue.Valid {
				x := kr.ManualValue.Float64
				e.Current = &x
			}
			if e.Current != nil {
				p := Progress(kr.Baseline, kr.Target, *e.Current)
				e.Progress = &p
				e.Status = StatusOf(p, el)
				sum += p
				n++
				if e.Status == StatusDone {
					done++
				}
			}
			eo.KRs = append(eo.KRs, e)
		}
		if n > 0 {
			p := sum / float64(n)
			eo.Progress = &p
			if done == len(o.KeyResults) {
				eo.Status = StatusDone
			} else {
				eo.Status = StatusOf(p, el)
			}
		}
		out = append(out, eo)
	}
	return out
}

// Summarize counts key results of the objectives active on now (nil when none is active).
func Summarize(evs []EvaluatedObjective, now time.Time) *Summary {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var s Summary
	for _, o := range evs {
		if day.Before(o.StartsOn) || day.After(o.EndsOn) {
			continue
		}
		s.Objectives++
		for _, kr := range o.KRs {
			s.KeyResults++
			switch kr.Status {
			case StatusOnTrack:
				s.OnTrack++
			case StatusAtRisk:
				s.AtRisk++
			case StatusDone:
				s.Done++
			default:
				s.NoData++
			}
		}
	}
	if s.Objectives == 0 {
		return nil
	}
	return &s
}

// Summary is the overview block: the summary of the active objectives, or nil when there are none.
func (s *Service) Summary(ctx context.Context, now time.Time) (*Summary, error) {
	objs, err := s.List(ctx, "")
	if err != nil || len(objs) == 0 {
		return nil, err
	}
	evs, err := s.Evaluate(ctx, now, objs)
	if err != nil {
		return nil, err
	}
	return Summarize(evs, now), nil
}
