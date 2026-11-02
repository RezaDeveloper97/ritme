package inputs

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The mapping rules from rows to engine.Input (T-M9-25 scope §2/§3). Optional dates are always Opt (Some/None):
// Day 0 is the real date 1970-01-01, so a zero value never stands for "unknown".

const (
	// markerPeriods is how many of the newest periods the marker window covers (scope §1.4).
	markerPeriods = 7
	// markerLookback is the minimum marker window in days before today (scope §1.4).
	markerLookback = 400

	// bbtMinCenti / bbtMaxCenti bound a plausible BBT in hundredths of °C (inclusive); readings outside are typos.
	bbtMinCenti = 3400
	bbtMaxCenti = 4300

	// Marker kinds of ListEngineInputMarkers (the v2 param, or the legacy source).
	kindLH          = "lh_test"
	kindBBT         = "bbt"
	kindPregnancy   = "pregnancy_test"
	kindFertilityLH = "fertility_lh"
	kindLegacyBBT   = "legacy_bbt"

	valuePositive = "positive" // LH negative | faint | positive, pregnancy test negative | positive
)

// optDay is Some(day) for a valid stored date, None otherwise.
func optDay(d civildate.NullDate) engine.Opt[engine.Day] {
	if !d.Valid {
		return engine.None[engine.Day]()
	}
	if day := engine.DayOf(d.Date); day != engine.NoDay {
		return engine.Some(day)
	}
	return engine.None[engine.Day]()
}

// mapProfile maps the joined profile row: declared lengths and anchor, the manual-lengths switch, birth date and
// teen band, the effective mode, hormonal contraception, delivery date and breastfeeding.
func mapProfile(r store.ListEngineInputProfilesRow) engine.Input {
	var in engine.Input
	in.Declared.LastPeriodStart = optDay(r.OnboardingLastPeriodStart)
	if r.CycleDuration.Valid && r.CycleDuration.Int16 > 0 {
		in.Declared.CycleLength = engine.Some(int(r.CycleDuration.Int16))
	}
	if r.PeriodDuration.Valid && r.PeriodDuration.Int16 > 0 {
		in.Declared.PeriodLength = engine.Some(int(r.PeriodDuration.Int16))
	}
	in.Declared.Manual = !r.LengthsAuto // lengths_auto = 0 (no cycle_preferences row reads as 1)

	in.BirthDate = optDay(r.Birthday)
	// E-27: a teen age band without a birthday is treated as < 18. Every band (10_12 | 13_15 | 16_17) is a minor.
	in.TeenBandOnly = !in.BirthDate.OK() && r.AgeBand.Valid && r.AgeBand.String != ""

	in.Mode = mapMode(r.LifeMode.String, r.PregnancyActive != 0, r.UserGoal.String)
	in.Hormonal = contraception.HormonalActive(r.ContraceptionMethod.String, r.ContraceptionMethod.Valid,
		medications(r.Medications.V, r.Medications.Valid))
	if in.Hormonal && r.ContraceptionMethod.Valid {
		in.HormonalFrom = hormonalFrom(r)
	}
	if r.DeliveryDate.Valid {
		in.DeliveryDate = optDay(r.DeliveryDate)
	}
	in.Breastfeeding = r.Breastfeeding.Valid && r.Breastfeeding.Bool // I-04: NULL = not asked = no
	return in
}

// hormonalFrom is the start of the saved hormonal method (E-13, T-M9-48d; user decision 2026-11-02): the earlier
// of the day the method was saved (contraception_methods.created_at, Tehran civil day) and the method's own date
// when set — pack_started_on for the pills, inserted_on for the hormonal IUD, injected_on for the injection (the
// implant has none). The other method's columns are ignored, so a date left over from an earlier method never
// counts. None when neither is known (the engine then suppresses the whole history, as for the onboarding answer).
func hormonalFrom(r store.ListEngineInputProfilesRow) engine.Opt[engine.Day] {
	from := engine.NoDay
	if r.ContraceptionCreatedAt.Valid {
		from = engine.DayOf(civildate.InTehran(r.ContraceptionCreatedAt.Time))
	}
	var own civildate.NullDate
	switch r.ContraceptionMethod.String {
	case contraception.MethodCombinedPill, contraception.MethodProgestinPill:
		own = r.ContraceptionPackStartedOn
	case contraception.MethodHormonalIUD:
		own = r.ContraceptionInsertedOn
	case contraception.MethodInjection:
		own = r.ContraceptionInjectedOn
	}
	if d, ok := optDay(own).Get(); ok && (from == engine.NoDay || d < from) {
		from = d
	}
	if from == engine.NoDay {
		return engine.None[engine.Day]()
	}
	return engine.Some(from)
}

// mapMode is the effective mode (enums.ResolveLifeMode: an active pregnancy wins, then the stored postpartum /
// menopause / teen mode, then the ttc goal, else cycle) as the engine's Mode.
func mapMode(stored string, pregnancyActive bool, userGoal string) engine.Mode {
	m, _ := engine.ParseMode(string(enums.ResolveLifeMode(stored, pregnancyActive, userGoal)))
	return m
}

// medications decodes user_life_profiles.medications (a JSON list of codes); NULL or a malformed value is none.
func medications(raw json.RawMessage, valid bool) []string {
	if !valid || len(raw) == 0 {
		return nil
	}
	var list []string
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	return list
}

// mapPeriods maps the cycle_histories rows (I-01, I-02): is_estimated rows are ignored; user_profile_confirmed rows
// are never periods and never the anchor — goose 00103 (T-M9-27, D-123) moved every onboarding seed into
// user_profiles.onboarding_last_period_start (the only declared anchor) or turned it into a user_logged row, and
// nothing writes the source any more; every other row (user_logged confirmed or not, bleeding_log, …) is an actual
// period. An open period has End = NoDay. The result is ordered by start.
func mapPeriods(rows []store.ListEngineInputPeriodsRow) []engine.Period {
	var periods []engine.Period
	for _, r := range rows {
		if r.IsEstimated || r.Source == string(enums.DataSourceUserProfileConfirmed) {
			continue
		}
		start := engine.DayOf(r.PeriodStartDate)
		if start == engine.NoDay {
			continue
		}
		p := engine.Period{Start: start, End: engine.NoDay, Source: engine.PeriodUserLogged}
		if end, ok := optDay(r.PeriodEndDate).Get(); ok {
			p.End = end
		}
		if r.Source == string(enums.DataSourceBleedingLog) {
			p.Source = engine.PeriodBleedingLog
		}
		periods = append(periods, p)
	}
	slices.SortStableFunc(periods, func(a, b engine.Period) int { return cmp.Compare(a.Start, b.Start) })
	return periods
}

// mapSkips passes the skipped predicted starts as stored (E-15; the engine sorts, dedupes and ignores stale ones).
func mapSkips(rows []store.ListEngineInputSkipsRow) []engine.Day {
	var out []engine.Day
	for _, r := range rows {
		if d := engine.DayOf(r.ExpectedStart); d != engine.NoDay {
			out = append(out, d)
		}
	}
	return out
}

// markerWindowFrom is the first day of the marker window: the start of the 7th newest period (the oldest one when
// there are fewer) or today − 400, whichever is earlier. periods must be ordered by start.
func markerWindowFrom(periods []engine.Period, today engine.Day) engine.Day {
	from := today - markerLookback
	if n := len(periods); n > 0 {
		from = min(from, periods[max(0, n-markerPeriods)].Start)
	}
	return from
}

// mapMarkers maps one user's marker rows inside [from, to] (to = today + 1: a log dated after tomorrow is a clock
// or typing error and is dropped; tomorrow is kept for users ahead of the server's day):
//   - LH: positive tests only (faint and negative are not positive, E-16), one per day, from the log sheet
//     (measurements.lh_test) or the /fertility log;
//   - BBT: one reading per day in hundredths of °C; the v2 entry wins over the legacy daily_health_logs column when
//     both exist, and the chosen reading is dropped when outside 34.00–43.00;
//   - pregnancy tests: positive ones only (E-26; the engine ignores negatives), one per day.
//
// Each list is ordered by date; an empty list is nil.
func mapMarkers(rows []store.ListEngineInputMarkersRow, from, to engine.Day) ([]engine.LHTest, []engine.BBTReading, []engine.PregnancyTest) {
	lhDays := map[engine.Day]bool{}
	pregDays := map[engine.Day]bool{}
	bbtV2 := map[engine.Day]string{}
	bbtLegacy := map[engine.Day]string{}
	for _, r := range rows {
		d := engine.DayOf(r.LogDate)
		if d == engine.NoDay || d < from || d > to {
			continue
		}
		switch r.Kind {
		case kindLH, kindFertilityLH:
			if r.ValueCode.String == valuePositive {
				lhDays[d] = true
			}
		case kindPregnancy:
			if r.ValueCode.String == valuePositive {
				pregDays[d] = true
			}
		case kindBBT:
			if r.ValueNum.Valid {
				bbtV2[d] = r.ValueNum.String
			}
		case kindLegacyBBT:
			if r.ValueNum.Valid {
				bbtLegacy[d] = r.ValueNum.String
			}
		}
	}

	var lh []engine.LHTest
	for _, d := range sortedKeys(lhDays) {
		lh = append(lh, engine.LHTest{Date: d, Positive: true})
	}
	var preg []engine.PregnancyTest
	for _, d := range sortedKeys(pregDays) {
		preg = append(preg, engine.PregnancyTest{Date: d, Positive: true})
	}
	for d, v := range bbtV2 {
		bbtLegacy[d] = v // v2 wins
	}
	var bbt []engine.BBTReading
	for _, d := range sortedKeys(bbtLegacy) {
		if c, ok := parseCenti(bbtLegacy[d]); ok && c >= bbtMinCenti && c <= bbtMaxCenti {
			bbt = append(bbt, engine.BBTReading{Date: d, Centi: c})
		}
	}
	return lh, bbt, preg
}

func sortedKeys[V any](m map[engine.Day]V) []engine.Day {
	keys := make([]engine.Day, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// parseCenti reads a DECIMAL text ("36.55", "36.5", "37") as hundredths, exactly (no float): digits after the
// second decimal round half up. Negative, empty and malformed values are not readings.
func parseCenti(s string) (int32, bool) {
	s = strings.TrimSpace(s)
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" || len(whole) > 4 || !digits(whole) || !digits(frac) {
		return 0, false
	}
	var n int32
	for _, c := range whole {
		n = n*10 + c - '0'
	}
	for i := range 2 {
		n *= 10
		if i < len(frac) {
			n += int32(frac[i] - '0')
		}
	}
	if len(frac) > 2 && frac[2] >= '5' {
		n++
	}
	return n, true
}

func digits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
