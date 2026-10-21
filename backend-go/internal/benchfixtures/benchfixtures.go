// Package benchfixtures builds deterministic engine inputs for the benchmarks and the query-budget tests
// (T-M9-08): period histories of H = 6 / 24 / 60 / 120 periods, up to 365 days of daily logs with BBT and LH,
// and one user per life-stage mode.
//
// Everything is a pure function of its arguments: a fixed request day (Today) and a seeded PCG source, so two
// runs (and two machines) benchmark exactly the same data. Nothing here touches a database; the query-budget
// tests turn the same values into rows.
//
// The package imports only leaf packages (model, store, taxonomy, civildate, enums), so any engine package's
// external test package can import it without an import cycle.
package benchfixtures

import (
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"strconv"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Today is the fixed request day of every fixture (Tehran calendar day).
var Today = civildate.New(2026, 10, 21)

// Birthday is the fixture users' birthday (age 31 on Today).
var Birthday = civildate.New(1995, 3, 10)

// Sizes are the history sizes the benchmarks sweep (periods).
var Sizes = []int{6, 24, 60, 120}

// LogDays is the length of the daily-log fixture (days ending Today).
const LogDays = 365

// currentCycleDay is the cycle day of Today for a cycle user: the newest period started 9 days ago.
const currentCycleDay = 10

// seed fixes the PCG stream; change it and every baseline number moves.
const seed1, seed2 = 0x5249544d45, 0x4d39303038 // "RITME", "M9008"

// Period is one logged period, oldest-to-newest order is up to the caller.
type Period struct {
	Start civildate.Date
	// Bleeding is the logged bleed length (days); the period ends on Start + Bleeding − 1.
	Bleeding int
	// Length is the cycle length from this start to the next one (0 for the current cycle).
	Length int
}

// End is the period's last day.
func (p Period) End() civildate.Date { return p.Start.AddDays(p.Bleeding - 1) }

// Periods returns h periods, newest first: the newest started currentCycleDay−1 days before Today, earlier
// cycles are 26–32 days long (one in nine 35–38, an outlier), bleeds 4–6 days. Periods(n) is a prefix of
// Periods(m) for n < m.
func Periods(h int) []Period {
	rng := rand.New(rand.NewPCG(seed1, seed2)) //nolint:gosec // deterministic fixture, not crypto
	out := make([]Period, 0, h)
	start := Today.AddDays(-(currentCycleDay - 1))
	for range h {
		out = append(out, Period{Start: start, Bleeding: 4 + rng.IntN(3)})
		length := 26 + rng.IntN(7)
		if rng.IntN(9) == 0 {
			length = 35 + rng.IntN(4)
		}
		start = start.AddDays(-length)
	}
	// Length is the distance to the next (newer) start.
	for i := 1; i < len(out); i++ {
		out[i].Length = out[i].Start.DiffDays(out[i-1].Start)
	}
	return out
}

// Histories are Periods(h) as engine rows (newest first, confirmed, closed, user_logged).
func Histories(h int) []model.History {
	ps := Periods(h)
	out := make([]model.History, len(ps))
	for i, p := range ps {
		bleed := p.Bleeding
		out[i] = model.History{
			ID: int64(len(ps) - i), PeriodStart: p.Start, PeriodEnd: p.End(), IsConfirmed: true,
			Source: string(enums.DataSourceUserLogged), BleedingLength: &bleed,
		}
	}
	return out
}

// HistoryRows are Periods(h) as cycle_histories rows of userID (newest first), the shape the cycle service
// loads and hashes into its cache key.
func HistoryRows(userID uint64, h int) []store.CycleHistory {
	ps := Periods(h)
	out := make([]store.CycleHistory, len(ps))
	for i, p := range ps {
		out[i] = store.CycleHistory{
			ID: uint64(len(ps) - i), UserID: userID, PeriodStartDate: p.Start, //nolint:gosec // small
			PeriodEndDate:  civildate.NullDate{Date: p.End(), Valid: true},
			BleedingLength: sql.NullInt32{Int32: int32(p.Bleeding), Valid: true}, //nolint:gosec // 4..6
			IsConfirmed:    true, Source: string(enums.DataSourceUserLogged),
			CreatedAt: sql.NullTime{Time: p.Start.TehranMidnight(), Valid: true},
			UpdatedAt: sql.NullTime{Time: p.Start.TehranMidnight(), Valid: true},
		}
	}
	return out
}

// Profile is the engine profile of a cycle user whose newest period is Periods(1)[0].
func Profile() *model.Profile {
	return &model.Profile{
		LastPeriodStart: Periods(1)[0].Start, CycleDuration: model.Int(28), PeriodDuration: model.Int(5),
		Birthday: Birthday, Goal: string(enums.UserGoalNonTtc),
	}
}

// ProfileRow is Profile as a user_profiles row of userID.
func ProfileRow(userID uint64) *store.UserProfile {
	return &store.UserProfile{
		ID: userID, UserID: userID,
		Birthday:           civildate.NullDate{Date: Birthday, Valid: true},
		Weight:             sql.NullString{String: "61.00", Valid: true},
		Height:             sql.NullInt16{Int16: 165, Valid: true},
		PeriodDuration:     sql.NullInt16{Int16: 5, Valid: true},
		CycleDuration:      sql.NullInt16{Int16: 28, Valid: true},
		LastPeriodStart:    civildate.NullDate{Date: Periods(1)[0].Start, Valid: true},
		UserGoal:           string(enums.UserGoalNonTtc),
		SubscriptionType:   "free",
		CalculationStatus:  string(enums.CalculationStatusCompleted),
		CalculationVersion: 7,
	}
}

// Day is one fixture day: the log-sheet entries (health_log_entries), the BBT reading (hundredths °C, 0 = not
// measured), the LH test ("" = none; negative | faint | positive) and the cervical mucus ("" = none).
type Day struct {
	Date     civildate.Date
	CycleDay int
	Entries  []taxonomy.Entry
	BBT      int
	LH       string
	Mucus    string
}

// BBTString is the decimal(4,2) column text of the reading ("36.48"); "" when not measured.
func (d Day) BBTString() string {
	if d.BBT == 0 {
		return ""
	}
	return strconv.Itoa(d.BBT/100) + "." + pad2(d.BBT%100)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

var (
	flows  = []string{"heavy", "heavy", "medium", "light", "light", "light"}
	cramps = []string{"severe", "moderate", "mild"}
)

// Days returns the n days ending Today, oldest first, laid over the Periods history: flow and cramps on bleed
// days, a biphasic BBT curve (low before ovulation = length − 14, high from the day after; one day in seven
// unmeasured), LH tests around ovulation (faint two days before, positive the day before), egg-white mucus near
// ovulation, sleep, mood, energy and a daily weight.
func Days(n int) []Day {
	// Enough periods to cover n days (cycles are ≥ 26 days).
	ps := Periods(n/26 + 2)
	rng := rand.New(rand.NewPCG(seed2, seed1)) //nolint:gosec // deterministic fixture, not crypto
	out := make([]Day, 0, n)
	first := Today.AddDays(-(n - 1))
	pi := len(ps) - 1
	for d := first; !d.After(Today); d = d.AddDays(1) {
		for pi > 0 && !d.Before(ps[pi-1].Start) {
			pi--
		}
		p := ps[pi]
		cd := p.Start.DiffDays(d) + 1
		length := p.Length
		if length == 0 {
			length = 28
		}
		ov := length - 14
		day := Day{Date: d, CycleDay: cd}
		idx := first.DiffDays(d)
		if cd >= 1 && cd <= p.Bleeding {
			day.Entries = append(day.Entries, code("bleeding", "flow", "", flows[cd-1]))
			if cd <= len(cramps) {
				day.Entries = append(day.Entries, code("pain", "location", "abdomen", cramps[cd-1]))
			}
		}
		if cd >= length-3 && cd <= length-1 {
			day.Entries = append(day.Entries, code("symptoms", "digestive", "bloating", "mild"))
		}
		sleep := "6_9"
		if idx%4 == 0 {
			sleep = "3_6"
		}
		day.Entries = append(day.Entries, code("sleep", "duration", "", sleep))
		switch {
		case cd >= 6 && cd <= ov:
			day.Entries = append(day.Entries, code("mood", "moods", "happy", taxonomy.Yes))
		case cd > length-5:
			day.Entries = append(day.Entries, code("mood", "moods", "irritable", taxonomy.Yes))
		case idx%3 == 0:
			day.Entries = append(day.Entries, code("mood", "moods", "calm", taxonomy.Yes))
		}
		energy := "medium"
		if cd >= 6 && cd <= ov {
			energy = "high"
		}
		day.Entries = append(day.Entries, code("appetite_energy", "energy", "", energy),
			num("measurements", "weight", 60.0-0.004*float64(idx)+float64(rng.IntN(5))/10))
		if rng.IntN(7) != 0 {
			base := 3640
			if cd > ov {
				base = 3685
			}
			day.BBT = base + rng.IntN(9) - 4
		}
		switch cd - ov {
		case -4, -3, 1:
			day.LH = "negative"
		case -2:
			day.LH = "faint"
		case -1:
			day.LH = "positive"
		}
		if cd >= ov-2 && cd <= ov {
			day.Mucus = "egg_white"
		} else if cd > p.Bleeding && cd < ov-2 {
			day.Mucus = "sticky"
		}
		out = append(out, day)
	}
	return out
}

func code(cat, param, item, value string) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Item: item, Code: sql.NullString{String: value, Valid: true}}
}

func num(cat, param string, v float64) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Num: sql.NullString{String: strconv.FormatFloat(v, 'f', 2, 64), Valid: true}}
}

// DailyLogRows are the days' daily_health_logs rows of userID (oldest first): spotting / moods / sleep / BBT,
// the columns the legacy engine and the BBT chart read.
func DailyLogRows(userID uint64, days []Day) []store.DailyHealthLog {
	out := make([]store.DailyHealthLog, len(days))
	for i, d := range days {
		r := store.DailyHealthLog{
			ID: uint64(i + 1), UserID: userID, LogDate: d.Date, //nolint:gosec // small
			SleepQuality: sql.NullString{String: "good", Valid: true},
			CreatedAt:    sql.NullTime{Time: d.Date.TehranMidnight(), Valid: true},
			UpdatedAt:    sql.NullTime{Time: d.Date.TehranMidnight(), Valid: true},
		}
		for _, e := range d.Entries {
			switch {
			case e.Category == "bleeding" && e.Param == "flow":
				r.BleedingIntensity = sql.NullString{String: e.Code.String, Valid: true}
			case e.Category == "mood":
				raw, _ := json.Marshal([]string{e.Item})
				r.Moods = sql.Null[json.RawMessage]{V: raw, Valid: true}
			case e.Category == "pain":
				r.StomachAcheIntensity = sql.NullString{String: e.Code.String, Valid: true}
			case e.Category == "symptoms":
				r.BloatingIntensity = sql.NullString{String: e.Code.String, Valid: true}
			}
		}
		if s := d.BBTString(); s != "" {
			r.BasalBodyTemperature = sql.NullString{String: s, Valid: true}
		}
		out[i] = r
	}
	return out
}

// User is one fixture user of a life-stage mode.
type User struct {
	Mode enums.LifeMode
	// Periods is the history size; LogDays the daily-log span ending Today.
	Periods, LogDays int
	// Goal is user_profiles.user_goal; TTC users also have pregnancy_intention = trying.
	Goal enums.UserGoal
	// LMP is the pregnancy's last menstrual period (pregnancy mode only; zero otherwise).
	LMP civildate.Date
}

// Users is one user per life-stage mode the hot routes serve, in a stable order.
func Users() []User {
	return []User{
		{Mode: enums.LifeModeCycle, Periods: 24, LogDays: LogDays, Goal: enums.UserGoalNonTtc},
		{Mode: enums.LifeModeTTC, Periods: 24, LogDays: LogDays, Goal: enums.UserGoalTtc},
		{Mode: enums.LifeModePregnancy, Periods: 6, LogDays: 120, Goal: enums.UserGoalNonTtc, LMP: Today.AddDays(-20 * 7)},
		{Mode: enums.LifeModePostpartum, Periods: 6, LogDays: 60, Goal: enums.UserGoalNonTtc},
		{Mode: enums.LifeModeMenopause, Periods: 6, LogDays: 90, Goal: enums.UserGoalNonTtc},
		{Mode: enums.LifeModeTeen, Periods: 6, LogDays: 90, Goal: enums.UserGoalNonTtc},
	}
}

// UserFor is the fixture user of mode (ok false for a mode without a fixture).
func UserFor(mode enums.LifeMode) (User, bool) {
	for _, u := range Users() {
		if u.Mode == mode {
			return u, true
		}
	}
	return User{}, false
}
