package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// DecodeJSON is the inverse of AppendJSON, for tests and the v1↔v6 shadow diff (T-M9-32): it reads one §2.35 object
// (with limits, future_k and an optional today_ui) back into a Status and its TodayUI (none when the key is absent).
// Unknown keys, unknown enum spellings and malformed dates are errors; is_predicted is derived from
// resolution_source and must agree with it. It is not on a hot path and uses encoding/json.
func DecodeJSON(data []byte) (Status, Opt[TodayUI], error) {
	var w wireStatus
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return Status{}, None[TodayUI](), fmt.Errorf("engine: decode status: %w", err)
	}
	var d decoder
	st := d.status(&w)
	ui := None[TodayUI]()
	if w.TodayUI != nil {
		ui = Some(d.todayUI(w.TodayUI))
	}
	if d.err == nil && w.IsPredicted != st.IsPredicted() {
		d.err = errors.New("engine: decode status: is_predicted disagrees with resolution_source")
	}
	if d.err != nil {
		return Status{}, None[TodayUI](), d.err
	}
	return st, ui, nil
}

type wireStatus struct {
	Date            *string `json:"date"`
	CycleDay        *int    `json:"cycle_day"`
	MainPhase       string  `json:"main_phase"`
	Subphase        string  `json:"subphase"`
	FertilityLevel  string  `json:"fertility_level"`
	DaysToOvulation *int    `json:"days_to_ovulation"`
	DaysToPeriod    *int    `json:"days_to_period"`
	DaysLate        *int    `json:"days_late"`
	Anchors         struct {
		CurrentPeriodStart          *string `json:"current_period_start"`
		CurrentPeriodStartSource    string  `json:"current_period_start_source"`
		CurrentPeriodEnd            *string `json:"current_period_end"`
		CurrentPeriodEndSource      string  `json:"current_period_end_source"`
		CurrentPeriodEndIsConfirmed bool    `json:"current_period_end_is_confirmed"`
		PredictedNextPeriodStart    *string `json:"predicted_next_period_start"`
		EstimatedOvulationDate      *string `json:"estimated_ovulation_date"`
		OvulationSource             string  `json:"ovulation_source"`
	} `json:"anchors"`
	Metrics struct {
		EffectiveCycleLength  int    `json:"effective_cycle_length"`
		EffectivePeriodLength int    `json:"effective_period_length"`
		CycleVariability      *int   `json:"cycle_variability"`
		CycleRegularity       string `json:"cycle_regularity"`
	} `json:"metrics"`
	ClinicalFlags struct {
		Frequency *string `json:"frequency"`
		Duration  *string `json:"duration"`
		Absent    *bool   `json:"absent"`
	} `json:"clinical_flags"`
	FutureCycles         []wireFutureCycle `json:"future_cycles"`
	SuggestPregnancyTest bool              `json:"suggest_pregnancy_test"`
	Confidence           string            `json:"confidence"`
	ConfidenceReasons    []string          `json:"confidence_reasons"`
	DataQuality          string            `json:"data_quality"`
	ResolutionSource     string            `json:"resolution_source"`
	IsPredicted          bool              `json:"is_predicted"`
	RequiresUserInput    bool              `json:"requires_user_input"`
	Warnings             []string          `json:"warnings"`
	Limits               struct {
		FertilitySuppressed bool    `json:"fertility_suppressed"`
		SuppressionReason   *string `json:"suppression_reason"`
		EngineSuspended     bool    `json:"engine_suspended"`
		PromptPregnancyMode bool    `json:"prompt_pregnancy_mode"`
	} `json:"limits"`
	FutureK int          `json:"future_k"`
	TodayUI *wireTodayUI `json:"today_ui"`
}

type wireFutureCycle struct {
	K               int     `json:"k"`
	PeriodStart     *string `json:"period_start"`
	PeriodEnd       *string `json:"period_end"`
	FertileStart    *string `json:"fertile_start"`
	FertileEnd      *string `json:"fertile_end"`
	Ovulation       *string `json:"ovulation"`
	OvulationSource string  `json:"ovulation_source"`
	Confidence      string  `json:"confidence"`
}

type wireRange struct {
	Start *string `json:"start"`
	End   *string `json:"end"`
}

type wireTodayUI struct {
	RingAction              string  `json:"ring_action"`
	Banner                  *string `json:"banner"`
	ShowPregnancyTestButton bool    `json:"show_pregnancy_test_button"`
	PredictionCard          *struct {
		Period    wireRange  `json:"period"`
		PMS       wireRange  `json:"pms"`
		Fertile   *wireRange `json:"fertile"`
		Ovulation *struct {
			Date   *string `json:"date"`
			Source string  `json:"source"`
		} `json:"ovulation"`
		CycleLength     int    `json:"cycle_length"`
		CycleRegularity string `json:"cycle_regularity"`
	} `json:"prediction_card"`
}

// decoder keeps the first error, so the field mapping reads straight through.
type decoder struct{ err error }

func (d *decoder) fail(format string, args ...any) {
	if d.err == nil {
		d.err = fmt.Errorf("engine: decode status: "+format, args...)
	}
}

func (d *decoder) status(w *wireStatus) Status {
	st := Status{
		Date:           d.day(w.Date),
		CycleDay:       optInt(w.CycleDay),
		MainPhase:      enum[MainPhase](d, "main_phase", w.MainPhase),
		Subphase:       enum[Subphase](d, "subphase", w.Subphase),
		FertilityLevel: enum[FertilityLevel](d, "fertility_level", w.FertilityLevel),

		DaysToOvulation: optInt(w.DaysToOvulation),
		DaysToPeriod:    optInt(w.DaysToPeriod),
		DaysLate:        optInt(w.DaysLate),

		SuggestPregnancyTest: w.SuggestPregnancyTest,
		Confidence:           enum[Confidence](d, "confidence", w.Confidence),
		DataQuality:          enum[DataQuality](d, "data_quality", w.DataQuality),
		ResolutionSource:     enum[ResolutionSource](d, "resolution_source", w.ResolutionSource),
		RequiresUserInput:    w.RequiresUserInput,
		FutureK:              w.FutureK,
	}
	a := &w.Anchors
	st.Anchors = Anchors{
		CurrentPeriodStart:          d.day(a.CurrentPeriodStart),
		CurrentPeriodStartSource:    enum[StartSource](d, "current_period_start_source", a.CurrentPeriodStartSource),
		CurrentPeriodEnd:            d.day(a.CurrentPeriodEnd),
		CurrentPeriodEndSource:      enum[EndSource](d, "current_period_end_source", a.CurrentPeriodEndSource),
		CurrentPeriodEndIsConfirmed: a.CurrentPeriodEndIsConfirmed,
		PredictedNextPeriodStart:    d.day(a.PredictedNextPeriodStart),
		EstimatedOvulationDate:      d.day(a.EstimatedOvulationDate),
		OvulationSource:             enum[OvulationSource](d, "ovulation_source", a.OvulationSource),
	}
	st.Metrics = Metrics{
		EffectiveCycleLength:  w.Metrics.EffectiveCycleLength,
		EffectivePeriodLength: w.Metrics.EffectivePeriodLength,
		CycleVariability:      optInt(w.Metrics.CycleVariability),
		CycleRegularity:       enum[CycleRegularity](d, "cycle_regularity", w.Metrics.CycleRegularity),
	}
	if v := w.ClinicalFlags.Frequency; v != nil {
		st.ClinicalFlags.Frequency = Some(enum[Frequency](d, "frequency", *v))
	}
	if v := w.ClinicalFlags.Duration; v != nil {
		st.ClinicalFlags.Duration = Some(enum[Duration](d, "duration", *v))
	}
	if v := w.ClinicalFlags.Absent; v != nil {
		st.ClinicalFlags.Absent = Some(*v)
	}
	if w.FutureCycles != nil {
		st.FutureCycles = make([]FutureCycle, len(w.FutureCycles))
		for i, c := range w.FutureCycles {
			st.FutureCycles[i] = FutureCycle{
				K:               c.K,
				PeriodStart:     d.day(c.PeriodStart),
				PeriodEnd:       d.day(c.PeriodEnd),
				FertileStart:    d.day(c.FertileStart),
				FertileEnd:      d.day(c.FertileEnd),
				Ovulation:       d.day(c.Ovulation),
				OvulationSource: enum[OvulationSource](d, "future_cycles.ovulation_source", c.OvulationSource),
				Confidence:      enum[Confidence](d, "future_cycles.confidence", c.Confidence),
			}
		}
	}
	for _, s := range w.ConfidenceReasons {
		st.ConfidenceReasons.Add(enum[Reason](d, "confidence_reasons", s))
	}
	for _, s := range w.Warnings {
		st.Warnings.Add(enum[Warning](d, "warnings", s))
	}
	l := &w.Limits
	st.Limits = Limits{
		FertilitySuppressed: l.FertilitySuppressed,
		EngineSuspended:     l.EngineSuspended,
		PromptPregnancyMode: l.PromptPregnancyMode,
	}
	if l.SuppressionReason != nil {
		st.Limits.SuppressionReason = enum[SuppressionReason](d, "suppression_reason", *l.SuppressionReason)
	}
	return st
}

func (d *decoder) todayUI(w *wireTodayUI) TodayUI {
	ui := TodayUI{
		RingAction:              enum[RingAction](d, "ring_action", w.RingAction),
		ShowPregnancyTestButton: w.ShowPregnancyTestButton,
	}
	if w.Banner != nil {
		ui.Banner = enum[Banner](d, "banner", *w.Banner)
	}
	c := w.PredictionCard
	if c == nil {
		return ui
	}
	card := PredictionCard{
		Period:          d.dayRange(c.Period),
		PMS:             d.dayRange(c.PMS),
		CycleLength:     c.CycleLength,
		CycleRegularity: enum[CycleRegularity](d, "prediction_card.cycle_regularity", c.CycleRegularity),
	}
	if c.Fertile != nil {
		card.Fertile = Some(d.dayRange(*c.Fertile))
	}
	if o := c.Ovulation; o != nil {
		card.Ovulation = Some(OvulationMark{Date: d.day(o.Date),
			Source: enum[OvulationSource](d, "prediction_card.ovulation.source", o.Source)})
	}
	ui.PredictionCard = Some(card)
	return ui
}

func (d *decoder) dayRange(r wireRange) DayRange {
	return DayRange{Start: d.day(r.Start), End: d.day(r.End)}
}

// day reads an ISO date (YYYY-MM-DD, strict); null is NoDay.
func (d *decoder) day(s *string) Day {
	if s == nil {
		return NoDay
	}
	v, ok := parseISO(*s)
	if !ok {
		d.fail("bad date %q", *s)
	}
	return v
}

func parseISO(s string) (Day, bool) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return NoDay, false
	}
	num := func(p string) (int, bool) {
		n := 0
		for i := range len(p) {
			c := p[i]
			if c < '0' || c > '9' {
				return 0, false
			}
			n = n*10 + int(c-'0')
		}
		return n, true
	}
	y, ok1 := num(s[0:4])
	m, ok2 := num(s[5:7])
	dd, ok3 := num(s[8:10])
	if !ok1 || !ok2 || !ok3 {
		return NoDay, false
	}
	return MakeDay(y, time.Month(m), dd)
}

func optInt(p *int) Opt[int] {
	if p == nil {
		return None[int]()
	}
	return Some(*p)
}

// enum reads a spelling of E; "" (the null member's String) is never a valid spelling in JSON.
func enum[E interface {
	~uint8
	String() string
}](d *decoder, field, s string) E {
	if s != "" {
		for i := range 256 {
			if e := E(uint8(i)); e.String() == s { //nolint:gosec // i < 256
				return e
			}
		}
	}
	d.fail("unknown %s %q", field, s)
	return 0
}
