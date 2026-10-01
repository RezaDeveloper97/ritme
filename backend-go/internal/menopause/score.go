package menopause

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Monthly score (nbl_Meno_Score): 11 questions rated 0–4 (catalog meno_score_items, meta {domain, max}), total /44
// and three domain subtotals, banded by catalog meno_score_bands (meta {min, max}). One questionnaire per Jalali
// month; filling it again replaces it.
const (
	DomainSomatic       = "somatic"
	DomainPsychological = "psychological"
	DomainUrogenital    = "urogenital"

	// DefaultItemMax is a question's top answer when its meta has no `max`.
	DefaultItemMax = 4
	// TrendMonths is the home's and the score screen's default chart length; MaxHistoryMonths caps ?months=.
	TrendMonths      = 6
	MaxHistoryMonths = 24
)

// Domains are the stored subtotal columns, in screen order.
var Domains = []string{DomainSomatic, DomainPsychological, DomainUrogenital}

// ErrScoreNotFound: no questionnaire for that month.
var ErrScoreNotFound = errors.New("menopause: score not found")

// Question is one meno_score_items item.
type Question struct {
	Code   string
	Domain string
	Max    int
}

// Band is one meno_score_bands item.
type Band struct {
	Item     catalog.Item
	Min, Max int
}

// Scale is the questionnaire as the catalog defines it now.
type Scale struct {
	Questions []Question
	Bands     []Band
}

// MaxTotal is the sum of the questions' maxima (44 for the seeded 11 × 4).
func (s Scale) MaxTotal() int {
	n := 0
	for _, q := range s.Questions {
		n += q.Max
	}
	return n
}

// DomainMax is the sum of the maxima of a domain's questions.
func (s Scale) DomainMax(domain string) int {
	n := 0
	for _, q := range s.Questions {
		if q.Domain == domain {
			n += q.Max
		}
	}
	return n
}

// Band is the band whose [min, max] holds total (nil when none does).
func (s Scale) Band(total int) *Band {
	for i, b := range s.Bands {
		if total >= b.Min && total <= b.Max {
			return &s.Bands[i]
		}
	}
	return nil
}

// Codes are the question codes in catalog order.
func (s Scale) Codes() []string {
	out := make([]string, len(s.Questions))
	for i, q := range s.Questions {
		out[i] = q.Code
	}
	return out
}

// Scale loads the questions and bands from the catalog.
func (s *Service) Scale(ctx context.Context) (Scale, error) {
	items, err := s.catalog.Items(ctx, GroupScoreItems)
	if err != nil {
		return Scale{}, fmt.Errorf("menopause: load score items: %w", err)
	}
	bands, err := s.catalog.Items(ctx, GroupScoreBands)
	if err != nil {
		return Scale{}, fmt.Errorf("menopause: load score bands: %w", err)
	}
	return NewScale(items, bands), nil
}

// NewScale reads the catalog items' meta (an item without a known domain counts towards the total only).
func NewScale(items, bands []catalog.Item) Scale {
	var sc Scale
	for _, it := range items {
		var meta struct {
			Domain string `json:"domain"`
			Max    *int   `json:"max"`
		}
		_ = json.Unmarshal(it.Meta, &meta)
		q := Question{Code: it.Code, Domain: meta.Domain, Max: DefaultItemMax}
		if meta.Max != nil && *meta.Max > 0 {
			q.Max = *meta.Max
		}
		sc.Questions = append(sc.Questions, q)
	}
	for _, it := range bands {
		var meta struct {
			Min *int `json:"min"`
			Max *int `json:"max"`
		}
		if json.Unmarshal(it.Meta, &meta) != nil || meta.Min == nil || meta.Max == nil {
			continue
		}
		sc.Bands = append(sc.Bands, Band{Item: it, Min: *meta.Min, Max: *meta.Max})
	}
	return sc
}

// Score is one month's questionnaire.
type Score struct {
	Month     civildate.Date // first day of the Jalali month
	Answers   map[string]int
	Total     int
	Subtotals map[string]int // by domain
	UpdatedAt time.Time
}

// Compute totals answers on the scale.
func (s Scale) Compute(month civildate.Date, answers map[string]int) Score {
	sc := Score{Month: month, Answers: answers, Subtotals: map[string]int{}}
	for _, q := range s.Questions {
		v, ok := answers[q.Code]
		if !ok {
			continue
		}
		sc.Total += v
		if slices.Contains(Domains, q.Domain) {
			sc.Subtotals[q.Domain] += v
		}
	}
	return sc
}

// MonthStart is the first day of the Jalali month that holds d.
func MonthStart(d civildate.Date) civildate.Date {
	jy, jm, _ := engine.ToJalali(d)
	return engine.FromJalali(jy, jm, 1)
}

// PrevMonth is the first day of the Jalali month before the one starting at m.
func PrevMonth(m civildate.Date) civildate.Date { return MonthStart(MonthStart(m).AddDays(-1)) }

func scoreOf(r store.MenopauseScore) Score {
	sc := Score{Month: r.Month, Answers: map[string]int{}, Total: int(r.Total), Subtotals: map[string]int{
		DomainSomatic: int(r.Somatic), DomainPsychological: int(r.Psychological), DomainUrogenital: int(r.Urogenital),
	}}
	_ = json.Unmarshal(r.Answers, &sc.Answers)
	if r.UpdatedAt.Valid {
		sc.UpdatedAt = r.UpdatedAt.Time
	}
	return sc
}

func u8(n int) uint8 { return uint8(min(max(n, 0), 255)) } //nolint:gosec // clamped

// SaveScore stores (or replaces) the month's questionnaire.
func (s *Service) SaveScore(ctx context.Context, userID uint64, sc Score, now time.Time) error {
	answers, err := json.Marshal(sc.Answers)
	if err != nil {
		return fmt.Errorf("menopause: encode answers: %w", err)
	}
	if err := s.q.UpsertMenopauseScore(ctx, store.UpsertMenopauseScoreParams{
		UserID: userID, Month: sc.Month, Answers: answers, Total: u8(sc.Total),
		Somatic: u8(sc.Subtotals[DomainSomatic]), Psychological: u8(sc.Subtotals[DomainPsychological]),
		Urogenital: u8(sc.Subtotals[DomainUrogenital]), Now: tehranNow(now),
	}); err != nil {
		return fmt.Errorf("menopause: save score: %w", err)
	}
	return nil
}

// Scores are the user's questionnaires from month `from` on, newest first.
func (s *Service) Scores(ctx context.Context, userID uint64, from civildate.Date) ([]Score, error) {
	rows, err := s.q.ListMenopauseScoresSince(ctx, store.ListMenopauseScoresSinceParams{UserID: userID, FromMonth: from})
	if err != nil {
		return nil, fmt.Errorf("menopause: list scores: %w", err)
	}
	out := make([]Score, len(rows))
	for i, r := range rows {
		out[i] = scoreOf(r)
	}
	return out, nil
}

// PreviousScore is the latest questionnaire before month (nil when none).
func (s *Service) PreviousScore(ctx context.Context, userID uint64, month civildate.Date) (*Score, error) {
	r, err := s.q.GetPreviousMenopauseScore(ctx, store.GetPreviousMenopauseScoreParams{UserID: userID, Month: month})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("menopause: load previous score: %w", err)
	}
	sc := scoreOf(r)
	return &sc, nil
}

// ScoreEntry is a questionnaire with its comparison to the one before it.
type ScoreEntry struct {
	Score
	Previous *Score // the latest questionnaire before this month (nil = none)
}

// Delta is Total − Previous.Total (nil without a previous questionnaire); negative = fewer symptoms.
func (e ScoreEntry) Delta() *int {
	if e.Previous == nil {
		return nil
	}
	d := e.Total - e.Previous.Total
	return &d
}

// TrendPoint is one month of the chart (Score nil = not filled).
type TrendPoint struct {
	Month civildate.Date
	Score *Score
}

// HRTEffect is the board's annotation «از وقتی هورمون‌درمانی را شروع کردی (مرداد)، امتیازت ۸ واحد کمتر شده»:
// the latest score against the last one before the HRT start month.
type HRTEffect struct {
	StartedOn civildate.Date
	Baseline  *Score // the latest questionnaire before the start month (nil = none)
	Latest    *Score // the latest questionnaire from the start month on (nil = none)
}

// Change is Latest − Baseline (nil when either is missing).
func (h HRTEffect) Change() *int {
	if h.Baseline == nil || h.Latest == nil {
		return nil
	}
	d := h.Latest.Total - h.Baseline.Total
	return &d
}

// History is GET /menopause/scores: the last `months` Jalali months (current included).
type History struct {
	Months  int
	Trend   []TrendPoint // oldest first
	Entries []ScoreEntry // newest first, filled months of the window only
	HRT     *HRTEffect
}

// Latest is the newest questionnaire of the window (nil when none).
func (h History) Latest() *ScoreEntry {
	if len(h.Entries) == 0 {
		return nil
	}
	return &h.Entries[0]
}

// History loads the window's questionnaires, their deltas and the HRT annotation.
func (s *Service) History(ctx context.Context, userID uint64, today civildate.Date, months int) (History, error) {
	cur := MonthStart(today)
	first := cur
	for range months - 1 {
		first = PrevMonth(first)
	}
	h := History{Months: months}
	all, err := s.Scores(ctx, userID, first)
	if err != nil {
		return History{}, err
	}
	before, err := s.PreviousScore(ctx, userID, first) // the oldest entry's delta
	if err != nil {
		return History{}, err
	}
	byMonth := map[civildate.Date]*Score{}
	for i := range all {
		byMonth[all[i].Month] = &all[i]
		e := ScoreEntry{Score: all[i]}
		if i+1 < len(all) {
			e.Previous = &all[i+1]
		} else {
			e.Previous = before
		}
		h.Entries = append(h.Entries, e)
	}
	for m := first; !m.After(cur); m = engine.JalaliMonthEnd(m).AddDays(1) {
		h.Trend = append(h.Trend, TrendPoint{Month: m, Score: byMonth[m]})
	}
	if h.HRT, err = s.hrtEffect(ctx, userID, today); err != nil {
		return History{}, err
	}
	return h, nil
}

// hrtEffect is the annotation for the earliest active HRT item with a start date (nil when there is none).
func (s *Service) hrtEffect(ctx context.Context, userID uint64, today civildate.Date) (*HRTEffect, error) {
	items, err := s.q.ListTreatmentItems(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("menopause: list treatment: %w", err)
	}
	var start *civildate.Date
	for _, it := range items {
		if it.Kind != KindHRT || !activeOn(it, today) || !it.StartedOn.Valid {
			continue
		}
		if d := it.StartedOn.Date; start == nil || d.Before(*start) {
			start = &d
		}
	}
	if start == nil {
		return nil, nil
	}
	h := &HRTEffect{StartedOn: *start}
	startMonth := MonthStart(*start)
	if h.Baseline, err = s.PreviousScore(ctx, userID, startMonth); err != nil {
		return nil, err
	}
	latest, err := s.PreviousScore(ctx, userID, engine.JalaliMonthEnd(today).AddDays(1)) // the newest, this month included
	if err != nil {
		return nil, err
	}
	if latest != nil && !latest.Month.Before(startMonth) {
		h.Latest = latest
	}
	return h, nil
}
