// Package okr is the OKR module of the admin panel (bloom B-N9-09, D-95; SPEC v6 §5.7, T-M10-30, D-179): objectives
// and key results with baseline, target, owner and period. A key result bound to a §5.6 metric key reads its value
// from the newest metrics snapshot (metrics.MetricValues), so progress updates by itself; a manual key result uses the
// value the admin typed. The evaluation also runs right after every snapshot computation (OnComputed) and is stored
// as the snapshot's `okr` summary. Aggregates only — nothing here touches a user row.
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
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Statuses of a key result / objective.
const (
	StatusOnTrack = "on_track"
	StatusAtRisk  = "at_risk"
	StatusDone    = "done"
	StatusNoData  = "no_data"
)

// paceSlackPoints: a key result is on track while its progress (in points) is at least the elapsed share of the period
// minus this many points (A-07).
const paceSlackPoints = 10

// pointsEps absorbs float error at the boundary (progress 10 at 20 % elapsed is on track).
const pointsEps = 1e-9

// Metric is one bindable live metric.
type Metric struct {
	Key  string
	Unit string // count | pct | toman | days
}

// Metrics is the catalogue of keys a key result can bind to, in display order: every §5.6 scalar metric
// (metrics.MetricKeys) plus the B-N9-09 names kept for existing key results.
var Metrics = []Metric{
	{"dau", "count"}, {"mau", "count"}, {"stickiness_pct", "pct"}, {"loggers", "count"}, {"loggers_15d", "pct"},
	{"signups_today", "count"}, {"activation_first_log_pct", "pct"}, {"activation_log_d7_pct", "pct"},
	{"retention_d7", "pct"}, {"retention_d30", "pct"},
	{"engine_mae", "days"}, {"accuracy_2d", "pct"}, {"engine_within_2_days_pct", "pct"}, {"engine_mean_error_days", "days"},
	{"low_confidence_share", "pct"}, {"low_data_quality_share", "pct"}, {"rui_resolved_7d", "pct"},
	{"plus_active", "count"}, {"trial_to_paid_pct", "pct"}, {"mrr_toman", "toman"}, {"ai_cost_share", "pct"},
	{"ocr_correction_rate", "pct"}, {"message_ctr", "pct"},
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

// StatusOf classifies a progress against the time elapsed (SPEC §5.7, A-07): done at 100 %, on track while
// progress ≥ elapsed% − 10 points, otherwise at risk. A period that has not started is on track.
func StatusOf(progress, elapsed float64) string {
	switch {
	case progress >= 1:
		return StatusDone
	case 100*progress+pointsEps >= 100*elapsed-paceSlackPoints:
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

// New builds the service; svc is the metrics service the bound key results read from. It also registers the OKR
// evaluation on svc, so every snapshot the job stores carries the summary of the objectives active that day.
func New(db *sql.DB, svc *metrics.Service) *Service {
	s := &Service{db: db, metrics: svc}
	if svc != nil && db != nil {
		svc.OnComputed(s.evaluateSnapshot)
	}
	return s
}

// evaluateSnapshot stores the OKR summary in a freshly computed payload (runs after the KPIs, before the upsert).
func (s *Service) evaluateSnapshot(ctx context.Context, now time.Time, p *metrics.Payload) error {
	objs, err := s.List(ctx, "")
	if err != nil {
		return fmt.Errorf("okr: evaluate: %w", err)
	}
	day := now.In(civildate.Tehran)
	if sum := Summarize(EvaluateWith(objs, Values(metrics.MetricValues(p)), day), day); sum != nil {
		p.OKR = sum
	}
	return nil
}

// Values are the resolved live metric values; a key is absent while it is not computable (no snapshot, suppressed).
type Values map[string]float64

// LiveValues resolves every catalogue key from the newest snapshot (one query; nothing is computed on the request).
func (s *Service) LiveValues(ctx context.Context) (Values, error) {
	sn, err := s.metrics.Latest(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return Values{}, nil
	}
	if err != nil {
		return nil, err
	}
	var p metrics.Payload
	if err := json.Unmarshal(sn.Payload, &p); err != nil {
		return nil, fmt.Errorf("okr: snapshot payload: %w", err)
	}
	return Values(metrics.MetricValues(&p)), nil
}

// Evaluate resolves live values (once) and computes progress and status for each objective.
func (s *Service) Evaluate(ctx context.Context, now time.Time, objs []Objective) ([]EvaluatedObjective, error) {
	live, err := s.LiveValues(ctx)
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
