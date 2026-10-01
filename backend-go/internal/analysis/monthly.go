package analysis

import (
	"fmt"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Calendars of the monthly report's `ym`.
const (
	CalendarGregorian = "gregorian"
	CalendarJalali    = "jalali"
)

const (
	// monthTopSymptoms: «علائم پرتکرار» lists 3.
	monthTopSymptoms = 3
	// frequentSymptomDays: a symptom logged on at least this many days of the month is «پرتکرار».
	frequentSymptomDays = 2
	// littleDataDays: fewer logged days than this reads «داده کمی ثبت شده».
	littleDataDays = 7
	// calmShare: symptoms on at most this share of the logged days reads «ماه آرامی بود».
	calmShare = 0.3
	// sleepNightsForCorrelation: fewer logged nights than this suggests logging sleep.
	sleepNightsForCorrelation = 10
	// fullMonthDays: at least this many logged days reads a well-logged month.
	fullMonthDays = 15
)

// Month is a calendar month [From, To] (To clipped to the request day for the running month).
type Month struct {
	Key, Calendar string
	From, To, End civildate.Date
}

// MonthMetrics are one month's figures (nil = no data).
type MonthMetrics struct {
	DaysLogged          int
	CycleLength         *int
	Sleep, Weight       *float64
	Systolic, Diastolic *float64
	GoodMoodPct         *int
}

// MonthlyReport is GET /analysis/monthly/{ym}.
type MonthlyReport struct {
	Month         Month
	This, Prev    MonthMetrics
	Regularity    string
	Top           []SymptomCount
	FrequentCount int
	SymptomDays   int
	SleepNights   int
	CyclesCounted int
	Headline      Phrase
	Summary       []Phrase
	Suggestion    Phrase
	PDFLocked     bool
}

// MonthOf builds the month of a calendar key; prev is the month before.
func MonthOf(year, month int, calendar string, today civildate.Date) (cur, prev Month) {
	build := func(y, m int) Month {
		var from, end civildate.Date
		if calendar == CalendarJalali {
			from = fromJalali(y, m, 1)
			end = jalaliMonthEnd(from)
		} else {
			from = civildate.New(y, timeMonth(m), 1)
			end = civildate.New(y, timeMonth(m+1), 1).AddDays(-1)
		}
		to := end
		if to.After(today) {
			to = today
		}
		return Month{Key: monthKey(y, m), Calendar: calendar, From: from, To: to, End: end}
	}
	py, pm := year, month-1
	if pm == 0 {
		py, pm = year-1, 12
	}
	return build(year, month), build(py, pm)
}

// lastCompletedBy is the last completed cycle that ended by to and started or ended inside [from, to],
// and the completed cycle before it.
func lastCompletedBy(cycles []Cycle, today, from, to civildate.Date) (cur, prev *Cycle) {
	for i := range cycles {
		c := cycles[i]
		if c.Current {
			continue
		}
		end := c.End(today)
		if end.After(to) || (c.Start.Before(from) && end.Before(from)) {
			continue
		}
		cur = &cycles[i]
		prev = nil
		if i > 0 && !cycles[i-1].Current {
			prev = &cycles[i-1]
		}
	}
	return cur, prev
}

func (in *Input) monthMetrics(m Month, cycles []Cycle) MonthMetrics {
	out := MonthMetrics{}
	var sleep []float64
	good, moods := 0, 0
	for d, day := range in.Days {
		if d.Before(m.From) || d.After(m.To) {
			continue
		}
		out.DaysLogged++
		if h, ok := day.SleepHours(); ok {
			sleep = append(sleep, h)
		}
		if g, ok := day.GoodMood(); ok {
			moods++
			if g {
				good++
			}
		}
	}
	out.Sleep = roundPtr(meanPtr(sleep), 1)
	if moods > 0 {
		out.GoodMoodPct = intPtr(int(round(float64(good)*100/float64(moods), 0)))
	}
	if c, _ := lastCompletedBy(cycles, in.Today, m.From, m.To); c != nil {
		out.CycleLength = intPtr(c.Length)
	}
	values := in.weights()
	for d := m.To; !d.Before(m.From); d = d.AddDays(-1) {
		if avg, ok := MovingAverage7(values, d); ok {
			v := round(avg, 1)
			out.Weight = &v
			break
		}
	}
	v := in.vitals(m.From, m.To)
	out.Systolic, out.Diastolic = v.Systolic, v.Diastolic
	return out
}

// BuildMonthly computes the monthly report; in.Days must cover prev.From − 6 … cur.To.
func BuildMonthly(in *Input, cur, prev Month, pdfEntitled bool) MonthlyReport {
	cycles := in.cycles()
	out := MonthlyReport{Month: cur, PDFLocked: !pdfEntitled, Top: []SymptomCount{}}
	out.This = in.monthMetrics(cur, cycles)
	out.Prev = in.monthMetrics(prev, cycles)
	if c, p := lastCompletedBy(cycles, in.Today, cur.From, cur.To); c != nil && p != nil {
		out.Prev.CycleLength = intPtr(p.Length)
	} else {
		out.Prev.CycleLength = nil
	}

	// Regularity as of the month's end (the 6-month window ending then).
	asOf := *in
	asOf.Range = NewRange(Range6M, cur.To)
	cr := BuildCycle(&asOf)
	out.Regularity = cr.Regularity
	out.CyclesCounted = cr.BasedOn

	month := *in
	month.Range = Range{Key: "month", From: cur.From, To: cur.To}
	counts, _, symptomDays := month.symptomCounts()
	out.SymptomDays = symptomDays
	for _, c := range counts {
		if c.Days >= frequentSymptomDays {
			out.FrequentCount++
		}
	}
	if len(counts) > monthTopSymptoms {
		counts = counts[:monthTopSymptoms]
	}
	if counts != nil {
		out.Top = counts
	}
	for d, day := range in.Days {
		if _, ok := day.SleepHours(); ok && !d.Before(cur.From) && !d.After(cur.To) {
			out.SleepNights++
		}
	}
	out.Headline, out.Summary, out.Suggestion = out.copy()
	return out
}

// copy picks the headline, the summary sentence parts and the next-month suggestion.
func (r MonthlyReport) copy() (Phrase, []Phrase, Phrase) {
	headline := Phrase{Key: "monthly.headline.busy"}
	switch {
	case r.This.DaysLogged < littleDataDays:
		headline = Phrase{Key: "monthly.headline.little_data"}
	case float64(r.SymptomDays) <= calmShare*float64(r.This.DaysLogged):
		headline = Phrase{Key: "monthly.headline.calm"}
	}

	summary := []Phrase{{Key: "monthly.summary.days", Args: []Arg{Num("days", r.This.DaysLogged)}}}
	if l := r.This.CycleLength; l != nil {
		key := "monthly.summary.cycle"
		switch r.Regularity {
		case RegularityRegular:
			key = "monthly.summary.cycle_regular"
		case RegularityIrregular:
			key = "monthly.summary.cycle_irregular"
		}
		summary = append(summary, Phrase{Key: key, Args: []Arg{Num("length", *l)}})
	}
	if s := r.This.Sleep; s != nil {
		summary = append(summary, Phrase{Key: "monthly.summary.sleep", Args: []Arg{Num("hours", *s)}})
	}
	if r.FrequentCount > 0 {
		summary = append(summary, Phrase{Key: "monthly.summary.symptoms", Args: []Arg{Num("count", r.FrequentCount)}})
	}

	needed := max(0, MinPatternCycles-r.CyclesCounted)
	var suggestion Phrase
	switch {
	case r.SleepNights < sleepNightsForCorrelation:
		suggestion = Phrase{Key: "monthly.suggestion.log_sleep"}
	case needed > 0 && len(r.Top) > 0:
		suggestion = Phrase{Key: "monthly.suggestion.sleep_symptom_needs_cycles", Args: []Arg{
			Symptom("symptom", r.Top[0].Key), Num("needed", needed)}}
	case needed > 0:
		suggestion = Phrase{Key: "monthly.suggestion.more_cycles", Args: []Arg{Num("needed", needed)}}
	case r.This.DaysLogged < fullMonthDays:
		suggestion = Phrase{Key: "monthly.suggestion.log_more"}
	default:
		suggestion = Phrase{Key: "monthly.suggestion.keep_going"}
	}
	return headline, summary, suggestion
}

func diffInt(a, b *int) any {
	if a == nil || b == nil {
		return nil
	}
	return *a - *b
}

func diffFloat(a, b *float64, places int) any {
	if a == nil || b == nil {
		return nil
	}
	return round(*a-*b, places)
}

func metric(key string, value, delta any, unit string) *jsonx.OrderedMap {
	var u any
	if unit != "" {
		u = unit
	}
	return jsonx.Obj("key", key, "value", value, "delta", delta, "unit", u)
}

// JSON is the `data` of GET /analysis/monthly/{ym}.
func (r MonthlyReport) JSON(c *Copy) *jsonx.OrderedMap {
	t, p := r.This, r.Prev
	var bp, bpDelta any
	if t.Systolic != nil {
		bp = jsonx.Obj("systolic", int(*t.Systolic), "diastolic", int(*t.Diastolic))
		if p.Systolic != nil {
			bpDelta = jsonx.Obj("systolic", int(*t.Systolic-*p.Systolic), "diastolic", int(*t.Diastolic-*p.Diastolic))
		}
	}
	summaryParts := make([]*jsonx.OrderedMap, 0, len(r.Summary))
	texts := make([]string, 0, len(r.Summary))
	for _, ph := range r.Summary {
		summaryParts = append(summaryParts, ph.JSON(c))
		if s := c.Render(ph); s != "" {
			texts = append(texts, s)
		}
	}
	summaryText := ""
	if len(texts) > 0 {
		summaryText = c.Render(Phrase{Key: "monthly.summary.sentence"})
		if summaryText == "" {
			summaryText = "{list}."
		}
		summaryText = replaceList(summaryText, c.List(texts))
	}
	return jsonx.Obj(
		"month", jsonx.Obj(
			"key", r.Month.Key, "calendar", r.Month.Calendar, "from", r.Month.From.String(), "to", r.Month.To.String(),
			"complete", r.Month.To == r.Month.End,
		),
		"headline", r.Headline.JSON(c),
		"summary", jsonx.Obj("parts", summaryParts, "text", summaryText),
		"metrics", []*jsonx.OrderedMap{
			metric("cycle_length", ptrOrNil(t.CycleLength), diffInt(t.CycleLength, p.CycleLength), "days"),
			metric("days_logged", t.DaysLogged, t.DaysLogged-p.DaysLogged, "days"),
			metric("sleep", ptrOrNil(t.Sleep), diffFloat(t.Sleep, p.Sleep, 1), "hours"),
			metric("weight", ptrOrNil(t.Weight), diffFloat(t.Weight, p.Weight, 1), "kg"),
			metric("blood_pressure", bp, bpDelta, "mmhg"),
			metric("good_mood", ptrOrNil(t.GoodMoodPct), diffInt(t.GoodMoodPct, p.GoodMoodPct), "percent"),
		},
		"top_symptoms", topJSON(r.Top, c),
		"suggestion", r.Suggestion.JSON(c),
		"pdf", jsonx.Obj("plus", true, "locked", r.PDFLocked),
	)
}

func fromJalali(y, m, d int) civildate.Date { return engine.FromJalali(y, m, d) }

func jalaliMonthEnd(d civildate.Date) civildate.Date { return engine.JalaliMonthEnd(d) }

func timeMonth(m int) time.Month { return time.Month(m) }

func monthKey(y, m int) string { return fmt.Sprintf("%04d-%02d", y, m) }

func replaceList(tpl, list string) string { return strings.ReplaceAll(tpl, "{list}", list) }
