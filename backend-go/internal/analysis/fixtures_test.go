package analysis

import (
	"database/sql"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/resources/translations"
)

// Seeded histories for the golden tests. Everything is a pure function of the day index: no randomness,
// a fixed request day (the contract clock, 2026-09-23).

var fixtureToday = civildate.MustParse("2026-09-23")

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

func single(cat, param, code string) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Code: ns(code)}
}

func item(cat, param, it, level string) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Item: it, Code: ns(level)}
}

func multi(cat, param, it string) taxonomy.Entry { return item(cat, param, it, taxonomy.Yes) }

func number(cat, param string, v float64) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Num: ns(strconv.FormatFloat(v, 'f', 2, 64))}
}

// history builds confirmed, closed periods from the newest start back through lengths (newest first).
func history(latest civildate.Date, lengths []int, bleed func(i int) int) ([]model.History, []civildate.Date) {
	starts := []civildate.Date{latest}
	for _, l := range lengths {
		starts = append(starts, starts[len(starts)-1].AddDays(-l))
	}
	var out []model.History
	for i, s := range starts {
		h := model.History{ID: int64(i + 1), PeriodStart: s, IsConfirmed: true, Source: "user_logged"}
		if b := bleed(i); b > 0 {
			h.PeriodEnd = s.AddDays(b - 1)
		}
		out = append(out, h)
	}
	return out, starts
}

// regularFixture: 7 completed cycles (29, 30, 28, 29, 38, 29, 29 newest first) and the current one from
// 2026-09-14; 5-day periods (one of 6); a full daily log: flow, cramps (milder on exercise days),
// headache 2 days before the period in 4 of 6 recent cycles, bloating, sleep with short nights and
// irritable mood after half of them, energy high in the follicular phase, daily weight with a slow
// loss and premenstrual water weight, weekly blood pressure, a fasting sugar every 10 days.
func regularFixture() *Input {
	hist, starts := history(civildate.MustParse("2026-09-14"), []int{29, 30, 28, 29, 38, 29, 29}, func(i int) int {
		if i == 0 {
			return 0 // the current period: 5 days logged but not closed yet → closed below
		}
		if i == 3 {
			return 6
		}
		return 5
	})
	hist[0].PeriodEnd = starts[0].AddDays(4)
	in := &Input{
		Today: fixtureToday, Histories: hist, Profile: &model.Profile{},
		HeightCM: 165, Birthday: civildate.MustParse("1996-04-12"), DeepAnalysis: true,
	}
	var rows []DayEntries
	first := starts[len(starts)-1]
	for ci := len(starts) - 1; ci >= 0; ci-- {
		start := starts[ci]
		length := 29
		if ci > 0 {
			length = start.DiffDays(starts[ci-1])
		}
		bleed := 5
		if ci == 3 {
			bleed = 6
		}
		for n := 1; n <= length; n++ {
			d := start.AddDays(n - 1)
			if d.After(fixtureToday) {
				break
			}
			idx := first.DiffDays(d)
			var es []taxonomy.Entry
			exercise := idx%3 == 0
			if n <= bleed {
				es = append(es, single("bleeding", "flow", []string{"medium", "heavy", "heavy", "medium", "light", "light"}[n-1]))
				if n <= 4 {
					exercise = n%2 == 0
					level := "severe"
					if exercise {
						level = "mild"
					}
					es = append(es, item("pain", "location", "abdomen", level))
				}
			}
			if exercise {
				es = append(es, multi("activity", "types", "walking"), number("activity", "duration", 30))
			}
			if ci%3 != 0 && (n == length-2 || n == length-1) {
				es = append(es, item("pain", "location", "head", "moderate"))
			}
			if n >= length-4 {
				es = append(es, item("symptoms", "digestive", "bloating", "mild"))
			}
			if n == length+5 && ci == 0 {
				es = append(es, single("bleeding", "spotting", taxonomy.Yes))
			}
			if n == 12 && ci == 2 {
				es = append(es, single("bleeding", "spotting", taxonomy.Yes))
			}
			short := idx%4 == 0
			if short {
				es = append(es, single("sleep", "duration", "3_6"))
			} else {
				es = append(es, single("sleep", "duration", "6_9"))
			}
			switch {
			case short && idx%8 == 0, idx%13 == 0:
				es = append(es, multi("mood", "moods", "irritable"))
			case n >= 6 && n <= 14:
				es = append(es, multi("mood", "moods", "happy"))
			case n%3 == 0:
				es = append(es, multi("mood", "moods", "calm"))
			}
			energy := "medium"
			switch {
			case n >= 6 && n <= 13:
				energy = "high"
			case n > length-12 && n%5 != 0:
				energy = "low"
			}
			es = append(es, single("appetite_energy", "energy", energy))
			w := 60.0 - 0.01*float64(idx)
			if n > length-4 {
				w += 0.6
			}
			es = append(es, number("measurements", "weight", w))
			if idx%7 == 0 {
				es = append(es, number("measurements", "bp_systolic", float64(118+idx%5)),
					number("measurements", "bp_diastolic", float64(76+idx%3)))
			}
			if idx%10 == 0 {
				es = append(es, number("measurements", "blood_sugar", 92))
			}
			rows = append(rows, DayEntries{Date: d, Entries: es})
		}
	}
	in.Days = BuildDays(rows)
	return in
}

// irregularFixture: lengths 24 / 35 / 29 / 41 / 26 (newest first), a few logs, no Plus.
func irregularFixture() *Input {
	hist, starts := history(civildate.MustParse("2026-09-09"), []int{24, 35, 29, 41, 26}, func(int) int { return 4 })
	var rows []DayEntries
	for _, s := range starts {
		rows = append(rows, DayEntries{Date: s, Entries: []taxonomy.Entry{
			single("bleeding", "flow", "heavy"), item("pain", "location", "back", "moderate"),
		}})
	}
	return &Input{
		Today: fixtureToday, Histories: hist, Profile: &model.Profile{CycleDuration: model.Int(30)},
		Birthday: civildate.MustParse("1990-01-01"), Days: BuildDays(rows),
	}
}

// emptyFixture: a new user — no history, no logs.
func emptyFixture() *Input {
	return &Input{Today: fixtureToday, Days: map[civildate.Date]*Day{}}
}

// teenFixture: age 16, two short cycles (21, 22) — FIGO adult ranges do not apply, not enough cycles for
// a pattern.
func teenFixture() *Input {
	hist, _ := history(civildate.MustParse("2026-09-10"), []int{21, 22}, func(int) int { return 9 })
	return &Input{
		Today: fixtureToday, Histories: hist, Profile: &model.Profile{},
		Birthday: civildate.MustParse("2010-05-01"), Days: map[civildate.Date]*Day{},
	}
}

// fixtures by name.
var fixtures = map[string]func() *Input{
	"regular":   regularFixture,
	"irregular": irregularFixture,
	"empty":     emptyFixture,
	"teen":      teenFixture,
}

// copyFor reads the seeded analysis + log-taxonomy namespaces for locale (default fa).
func copyFor(t *testing.T, locale string) *Copy {
	t.Helper()
	store := i18n.NewTranslationStore(translations.FS, "")
	a := store.NamespaceMessages(locale, Namespace, "fa")
	tx := store.NamespaceMessages(locale, "log-taxonomy", "fa")
	require.NotNil(t, a)
	return NewCopy(a, tx)
}

// withRange returns a copy of the fixture with the range set.
func ranged(in *Input, key string) *Input {
	out := *in
	out.Range = NewRange(key, in.Today)
	return &out
}
