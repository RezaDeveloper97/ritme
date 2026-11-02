package engine

import (
	"encoding/json"
	"testing"
)

// fuzzInput decodes fuzz bytes into an Input: unsorted, overlapping, duplicated and open periods, skips, BBT, LH
// and pregnancy tests, declared values, every mode — with dates near each other (overlaps, duplicates) or absurd
// (anywhere within ±100 years of today). Each record is a tag byte and its fields; missing bytes read as zero.
type fuzzInput struct {
	b     []byte
	today Day
	prev  Day
}

func (f *fuzzInput) u8() byte {
	if len(f.b) == 0 {
		return 0
	}
	c := f.b[0]
	f.b = f.b[1:]
	return c
}

// date is NoDay (1 in 16), an absurd date within ±100 years of today (1 in 8), else a date near the previous one.
func (f *fuzzInput) date() Day {
	c := f.u8()
	switch {
	case c%16 == 0:
		return NoDay
	case c%8 == 1:
		off := int(f.u8())<<16 | int(f.u8())<<8 | int(f.u8())
		f.prev = f.today + Day(off%73051-36525)
	default:
		f.prev += Day(int8(f.u8()))
	}
	return f.prev
}

func (f *fuzzInput) input(flags uint16) Input {
	in := none()
	in.Mode = Mode(flags % uint16(modeCount))
	in.Hormonal = flags&(1<<3) != 0
	if flags&(1<<7) != 0 { // a method start date (T-M9-48d), up to ~3 years back
		in.HormonalFrom = Some(f.today - Day(int(flags>>8)*4))
	}
	in.Breastfeeding = flags&(1<<4) != 0
	in.TeenBandOnly = flags&(1<<5) != 0
	in.Declared.Manual = flags&(1<<6) != 0
	f.prev = f.today
	for records := 0; len(f.b) > 0 && records < 400; records++ {
		switch f.u8() % 9 {
		case 0, 1, 2:
			start := f.date()
			end := start
			if c := f.u8(); c%5 == 0 {
				end = NoDay
			} else if start != NoDay {
				end = start + Day(int8(c)) // ends before the start are malformed on purpose
			}
			in.Periods = append(in.Periods, Period{Start: start, End: end, Source: PeriodSource(f.u8() % 2)})
		case 3:
			in.Skips = append(in.Skips, f.date())
		case 4:
			in.BBT = append(in.BBT, BBTReading{Date: f.date(), Centi: 3500 + int32(f.u8())*2})
		case 5:
			in.LH = append(in.LH, LHTest{Date: f.date(), Positive: f.u8()%2 == 0})
		case 6:
			in.PregnancyTests = append(in.PregnancyTests, PregnancyTest{Date: f.date(), Positive: f.u8()%2 == 0})
		case 7:
			in.Declared.CycleLength = Some(int(f.u8()))
			in.Declared.PeriodLength = Some(int(f.u8() % 16))
		default:
			switch f.u8() % 3 {
			case 0:
				in.Declared.LastPeriodStart = Some(f.date())
			case 1:
				in.BirthDate = Some(f.date())
			default:
				in.DeliveryDate = Some(f.date())
			}
		}
	}
	return in
}

// FuzzPrepare: Prepare, Resolve, Range, FutureCycles, TodayUI and the §2.35 encoder never panic on any input, and
// every invariant of invariants_test.go holds (state, every status of a window around today and around the first
// recorded dates, future_cycles); the today payload is valid JSON that decodes back to the same status. The seed
// corpus is testdata/fuzz/FuzzPrepare plus the seeds below.
func FuzzPrepare(f *testing.F) {
	f.Add([]byte{0, 2, 0, 4, 0, 28, 4, 0, 3, 2, 0, 28, 4, 0}, int16(0), uint16(0))
	f.Add([]byte{0, 2, 0, 0, 0, 2, 0, 0, 1, 2, 3, 0, 0}, int16(40), uint16(1<<3))      // duplicate + overlap
	f.Add([]byte{0, 9, 1, 2, 3, 0, 0, 2, 200, 5, 0, 3, 2, 28}, int16(-300), uint16(2)) // absurd date, open
	f.Add([]byte{8, 0, 2, 200, 7, 28, 5, 8, 1, 2, 180, 0}, int16(10), uint16(1<<4|3))  // declared + birth
	f.Add([]byte{4, 2, 0, 1, 4, 2, 1, 1, 4, 2, 1, 2, 5, 2, 12, 0, 6, 2, 30, 0}, int16(5), uint16(1))
	f.Add([]byte{}, int16(0), uint16(2))
	p := DefaultParams()
	f.Fuzz(func(t *testing.T, data []byte, todayOff int16, flags uint16) {
		today := day("2026-10-21") + Day(todayOff)
		fi := fuzzInput{b: data, today: today}
		in := fi.input(flags)
		s := Prepare(in, p)
		if v := stateViolations(s, in); len(v) > 0 {
			t.Fatalf("state: %v", v)
		}

		check := func(st *Status) {
			if v := violations(s, st, today); len(v) > 0 {
				t.Fatalf("day %s (today %s): %v\n%s", st.Date, today, v, AppendJSON(nil, st, nil))
			}
		}
		window := s.Range(today-60, today+200, today)
		for i := range window {
			check(&window[i])
		}
		if len(window) > 0 {
			if again := s.Resolve(window[0].Date, today); string(AppendJSON(nil, &again, nil)) !=
				string(AppendJSON(nil, &window[0], nil)) {
				t.Fatalf("Range ≠ Resolve on %s", window[0].Date)
			}
		}
		// Far from today: around the first records (absurd dates included).
		for i, r := range in.Periods {
			if i == 4 || r.Start == NoDay {
				break
			}
			st := s.Resolve(r.Start, today)
			check(&st)
			for _, d := range []Day{r.Start - 1, r.Start + 40} {
				st := s.Resolve(d, today)
				check(&st)
			}
		}
		if v := futureViolations(s, s.FutureCycles(today), today); len(v) > 0 {
			t.Fatalf("future_cycles (today %s): %v", today, v)
		}

		st, ui := s.Today(today)
		b := AppendJSON(nil, &st, &ui)
		if !json.Valid(b) {
			t.Fatalf("invalid JSON: %s", b)
		}
		back, backUI, err := DecodeJSON(b)
		if err != nil {
			t.Fatalf("decode: %v\n%s", err, b)
		}
		if string(AppendJSON(nil, &back, &ui)) != string(b) || backUI != Some(ui) {
			t.Fatalf("round trip changed the payload\n%s", b)
		}
	})
}
