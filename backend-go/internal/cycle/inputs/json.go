package inputs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The L2 snapshot format of engine.Input (T-M9-25 scope §5, read by T-M9-31). A stable JSON object with a version
// field "v": dates are ISO YYYY-MM-DD (null = none / open end), the mode and period sources are their spellings,
// BBT is integer hundredths of °C. Empty lists are omitted and decode to nil, so a loaded Input round-trips exactly.
// Bump Version whenever the shape or a meaning changes: a snapshot of another version fails to decode
// (ErrVersion), which forces a reload from the database.

// Version is the snapshot format version written by Encode.
const Version = 2 // 2: hormonal_from (T-M9-48d)

// ErrVersion is returned by Decode for a snapshot of another (or no) version.
var ErrVersion = errors.New("inputs: unknown snapshot version")

type wireInput struct {
	V              int          `json:"v"`
	Periods        []wirePeriod `json:"periods,omitempty"`
	Skips          []string     `json:"skips,omitempty"`
	Declared       wireDeclared `json:"declared"`
	BirthDate      *string      `json:"birth_date"`
	TeenBandOnly   bool         `json:"teen_band_only"`
	Mode           string       `json:"mode"`
	Hormonal       bool         `json:"hormonal"`
	HormonalFrom   *string      `json:"hormonal_from"`
	DeliveryDate   *string      `json:"delivery_date"`
	Breastfeeding  bool         `json:"breastfeeding"`
	LH             []wireTest   `json:"lh,omitempty"`
	BBT            []wireBBT    `json:"bbt,omitempty"`
	PregnancyTests []wireTest   `json:"pregnancy_tests,omitempty"`
}

type wirePeriod struct {
	Start  string  `json:"start"`
	End    *string `json:"end"`
	Source string  `json:"source"`
}

type wireDeclared struct {
	LastPeriodStart *string `json:"last_period_start"`
	CycleLength     *int    `json:"cycle_length"`
	PeriodLength    *int    `json:"period_length"`
	Manual          bool    `json:"manual"`
}

type wireTest struct {
	Date     string `json:"date"`
	Positive bool   `json:"positive"`
}

type wireBBT struct {
	Date  string `json:"date"`
	Centi int32  `json:"centi"`
}

// Encode writes in as a versioned snapshot.
func Encode(in engine.Input) ([]byte, error) {
	w := wireInput{
		V: Version, TeenBandOnly: in.TeenBandOnly, Mode: in.Mode.String(), Hormonal: in.Hormonal,
		Breastfeeding: in.Breastfeeding,
		BirthDate:     optISO(in.BirthDate), DeliveryDate: optISO(in.DeliveryDate),
		HormonalFrom: optISO(in.HormonalFrom),
		Declared: wireDeclared{
			LastPeriodStart: optISO(in.Declared.LastPeriodStart), CycleLength: optInt(in.Declared.CycleLength),
			PeriodLength: optInt(in.Declared.PeriodLength), Manual: in.Declared.Manual,
		},
	}
	for _, p := range in.Periods {
		w.Periods = append(w.Periods, wirePeriod{Start: p.Start.String(), End: dayISO(p.End), Source: p.Source.String()})
	}
	for _, d := range in.Skips {
		w.Skips = append(w.Skips, d.String())
	}
	for _, t := range in.LH {
		w.LH = append(w.LH, wireTest{Date: t.Date.String(), Positive: t.Positive})
	}
	for _, r := range in.BBT {
		w.BBT = append(w.BBT, wireBBT{Date: r.Date.String(), Centi: r.Centi})
	}
	for _, t := range in.PregnancyTests {
		w.PregnancyTests = append(w.PregnancyTests, wireTest{Date: t.Date.String(), Positive: t.Positive})
	}
	return json.Marshal(w)
}

// Decode reads a snapshot written by Encode. Another version → ErrVersion; a malformed date, mode or source → an
// error (the caller reloads).
func Decode(data []byte) (engine.Input, error) {
	var w wireInput
	if err := json.Unmarshal(data, &w); err != nil {
		return engine.Input{}, fmt.Errorf("inputs: decode snapshot: %w", err)
	}
	if w.V != Version {
		return engine.Input{}, fmt.Errorf("%w %d", ErrVersion, w.V)
	}
	d := decoder{}
	in := engine.Input{
		TeenBandOnly: w.TeenBandOnly, Hormonal: w.Hormonal, Breastfeeding: w.Breastfeeding,
		BirthDate: d.opt(w.BirthDate), DeliveryDate: d.opt(w.DeliveryDate), HormonalFrom: d.opt(w.HormonalFrom),
		Declared: engine.Declared{
			LastPeriodStart: d.opt(w.Declared.LastPeriodStart), Manual: w.Declared.Manual,
			CycleLength: intOpt(w.Declared.CycleLength), PeriodLength: intOpt(w.Declared.PeriodLength),
		},
	}
	mode, ok := engine.ParseMode(w.Mode)
	if !ok {
		return engine.Input{}, fmt.Errorf("inputs: decode snapshot: mode %q", w.Mode)
	}
	in.Mode = mode
	for _, p := range w.Periods {
		src, ok := parseSource(p.Source)
		if !ok {
			return engine.Input{}, fmt.Errorf("inputs: decode snapshot: period source %q", p.Source)
		}
		end := engine.NoDay
		if p.End != nil {
			end = d.day(*p.End)
		}
		in.Periods = append(in.Periods, engine.Period{Start: d.day(p.Start), End: end, Source: src})
	}
	for _, s := range w.Skips {
		in.Skips = append(in.Skips, d.day(s))
	}
	for _, t := range w.LH {
		in.LH = append(in.LH, engine.LHTest{Date: d.day(t.Date), Positive: t.Positive})
	}
	for _, r := range w.BBT {
		in.BBT = append(in.BBT, engine.BBTReading{Date: d.day(r.Date), Centi: r.Centi})
	}
	for _, t := range w.PregnancyTests {
		in.PregnancyTests = append(in.PregnancyTests, engine.PregnancyTest{Date: d.day(t.Date), Positive: t.Positive})
	}
	if d.err != nil {
		return engine.Input{}, d.err
	}
	return in, nil
}

// decoder parses ISO days and keeps the first error.
type decoder struct{ err error }

func (d *decoder) day(s string) engine.Day {
	c, err := civildate.Parse(s)
	day := engine.DayOf(c)
	if err != nil || day == engine.NoDay {
		if d.err == nil {
			d.err = fmt.Errorf("inputs: decode snapshot: date %q", s)
		}
		return engine.NoDay
	}
	return day
}

func (d *decoder) opt(s *string) engine.Opt[engine.Day] {
	if s == nil {
		return engine.None[engine.Day]()
	}
	return engine.Some(d.day(*s))
}

func parseSource(s string) (engine.PeriodSource, bool) {
	for _, src := range []engine.PeriodSource{engine.PeriodUserLogged, engine.PeriodBleedingLog} {
		if src.String() == s {
			return src, true
		}
	}
	return engine.PeriodUserLogged, false
}

// dayISO is the ISO date of d, nil for NoDay.
func dayISO(d engine.Day) *string {
	if d == engine.NoDay {
		return nil
	}
	s := d.String()
	return &s
}

// optISO is the ISO date of o, nil when absent (Some(NoDay) is absent too).
func optISO(o engine.Opt[engine.Day]) *string {
	d, ok := o.Get()
	if !ok {
		return nil
	}
	return dayISO(d)
}

func optInt(o engine.Opt[int]) *int {
	v, ok := o.Get()
	if !ok {
		return nil
	}
	return &v
}

func intOpt(p *int) engine.Opt[int] {
	if p == nil {
		return engine.None[int]()
	}
	return engine.Some(*p)
}
