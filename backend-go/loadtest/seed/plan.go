package main

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// Mode is the synthetic persona of a load-test user.
type Mode string

const (
	ModeCycle      Mode = "cycle"
	ModeTTC        Mode = "ttc"
	ModePregnant   Mode = "pregnancy"
	ModePostpartum Mode = "postpartum"
	ModePill       Mode = "pill" // cycle user on the combined pill
)

// MobilePrefix marks every synthetic user (users.mobile = prefix + 7 digits, 11 chars). Reset deletes by it.
const MobilePrefix = "0988"

// Months of history: 6..36, spread evenly over the users.
const (
	MinMonths = 6
	MaxMonths = 36
)

// ModeOf is deterministic from the index: per 100 users 15 TTC, 5 pregnant, 4 postpartum, 5 on the pill, 71 cycle.
func ModeOf(i int) Mode {
	switch b := i % 100; {
	case b < 15:
		return ModeTTC
	case b < 20:
		return ModePregnant
	case b < 24:
		return ModePostpartum
	case b < 29:
		return ModePill
	}
	return ModeCycle
}

// MonthsOf is the history length of user i, 6..36 months.
func MonthsOf(i int) int { return MinMonths + (i*7)%(MaxMonths-MinMonths+1) }

// Mobile is the synthetic mobile of user i.
func Mobile(i int) string { return fmt.Sprintf("%s%07d", MobilePrefix, i+1) }

// Day is a civil date as a UTC midnight, so AddDate never meets a DST jump.
type Day = time.Time

// Period is one cycle_histories row.
type Period struct {
	Start       Day
	End         *Day // nil: still open (or unknown)
	CycleLength *int // days to the next start; nil for the latest
	BleedLength *int
}

// DayLog is one daily_health_logs row.
type DayLog struct {
	Date      Day
	Bleeding  string // "" none
	Energy    string
	Headache  string
	Weight    float64
	BBT       float64 // 0 none
	HasWeight bool
}

// Fertility is one fertility_logs row.
type Fertility struct {
	Date   Day
	LH     string
	Mucus  string
	BBTAt  string
	HasBBT bool
}

// User is everything seeded for one synthetic user.
type User struct {
	Index       int
	Mobile      string
	Mode        Mode
	Months      int
	CycleLen    int
	PeriodLen   int
	LastStart   Day
	Periods     []Period // newest first
	Logs        []DayLog
	Fertility   []Fertility
	LMP         *Day // pregnancy
	DueDate     *Day
	BirthDate   *Day // postpartum
	PillStarted *Day
	PillLogs    []Day
}

func date(t Day, d int) Day { return t.AddDate(0, 0, d) }
func ptr[T any](v T) *T     { return &v }

// Build is the deterministic plan of user i: the same (i, today) always yields the same rows.
func Build(i int, today Day) User {
	r := rand.New(rand.NewPCG(uint64(i)+1, 0x9e3779b97f4a7c15)) //nolint:gosec // G404: synthetic data, determinism wanted
	u := User{Index: i, Mobile: Mobile(i), Mode: ModeOf(i), Months: MonthsOf(i)}
	u.CycleLen = 24 + r.IntN(10)
	u.PeriodLen = 3 + r.IntN(5)

	switch u.Mode {
	case ModePregnant:
		weeks := 4 + i%30
		u.LMP = ptr(date(today, -7*weeks))
		u.DueDate = ptr(date(*u.LMP, 280))
		u.LastStart = *u.LMP
	case ModePostpartum:
		u.BirthDate = ptr(date(today, -(10 + i%150)))
		u.LastStart = date(*u.BirthDate, -280)
	default:
		u.LastStart = date(today, -(i % u.CycleLen))
	}

	horizon := date(today, -int(float64(u.Months)*30.4))
	start := u.LastStart
	for n := 0; !start.Before(horizon); n++ {
		p := Period{Start: start, BleedLength: ptr(u.PeriodLen)}
		if end := date(start, u.PeriodLen-1); !end.After(today) {
			p.End = &end
		} else {
			p.BleedLength = nil
		}
		u.Periods = append(u.Periods, p)
		start = date(start, -(u.CycleLen + r.IntN(5) - 2))
	}
	for k := 0; k+1 < len(u.Periods); k++ {
		l := int(u.Periods[k].Start.Sub(u.Periods[k+1].Start).Hours() / 24)
		u.Periods[k+1].CycleLength = ptr(l)
	}

	u.buildLogs(r, today, horizon)
	if u.Mode == ModePill {
		u.PillStarted = ptr(date(today, -10))
		for d := 60; d >= 0; d-- {
			u.PillLogs = append(u.PillLogs, date(today, -d))
		}
	}
	return u
}

func (u *User) buildLogs(r *rand.Rand, today, horizon Day) {
	bleed := map[Day]bool{}
	ovul := map[Day]bool{}
	for _, p := range u.Periods {
		for d := 0; d < u.PeriodLen; d++ {
			bleed[date(p.Start, d)] = true
		}
		ovul[date(p.Start, u.CycleLen-14)] = true
	}
	bbtFrom := date(today, -180)
	energies := []string{"low", "medium", "high"}
	for d := horizon; !d.After(today); d = date(d, 1) {
		isBleed := bleed[d]
		ttcWindow := u.Mode == ModeTTC && !d.Before(bbtFrom)
		if !isBleed && !ttcWindow && r.IntN(4) != 0 {
			continue
		}
		l := DayLog{Date: d, Energy: energies[r.IntN(len(energies))]}
		if isBleed {
			l.Bleeding = []string{"low", "medium", "high"}[r.IntN(3)]
		}
		if r.IntN(5) == 0 {
			l.Headache = "low"
		}
		if r.IntN(7) == 0 {
			l.HasWeight, l.Weight = true, 55+float64(r.IntN(300))/10
		}
		if ttcWindow {
			// biphasic curve: ~36.4 before ovulation, ~36.8 after; one day in seven not measured
			if r.IntN(7) != 0 {
				l.BBT = 36.3 + float64(r.IntN(20))/100
				if lo := lastOvulation(ovul, d); !lo.IsZero() && d.After(lo) {
					l.BBT += 0.4
				}
			}
			f := Fertility{Date: d, Mucus: []string{"dry", "sticky", "creamy", "egg_white"}[r.IntN(4)], LH: "negative"}
			switch int(d.Sub(lastOrNextOvulation(ovul, d)).Hours() / 24) {
			case -1, 0:
				f.LH = "positive"
			case -2:
				f.LH = "faint"
			}
			if l.BBT > 0 {
				f.BBTAt, f.HasBBT = "07:00:00", true
			}
			if r.IntN(2) == 0 || f.LH != "negative" {
				u.Fertility = append(u.Fertility, f)
			}
		}
		u.Logs = append(u.Logs, l)
	}
}

// lastOvulation is the latest ovulation day on or before d (zero when none).
func lastOvulation(ov map[Day]bool, d Day) Day {
	for k := 0; k < 40; k++ {
		if x := date(d, -k); ov[x] {
			return x
		}
	}
	return Day{}
}

// lastOrNextOvulation is the ovulation day nearest to d within the next 3 days, else the last one.
func lastOrNextOvulation(ov map[Day]bool, d Day) Day {
	for k := 0; k <= 3; k++ {
		if x := date(d, k); ov[x] {
			return x
		}
	}
	return lastOvulation(ov, d)
}
