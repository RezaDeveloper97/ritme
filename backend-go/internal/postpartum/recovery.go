package postpartum

import (
	"database/sql"
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// The recovery log (nbl_v15_Recovery) is a view over log taxonomy v2 slots in health_log_entries — the same rows the
// postpartum log sheet writes through PUT /logs/days/{date} (B-N3-06), so both screens always agree:
//
//	lochia_amount  bleeding.lochia_amount         none|spotting|light|medium|heavy
//	lochia_color   bleeding.lochia_color          red|pink_brown|yellow_white
//	pain_level     pain.none (none) / the level of every pain.location item (mild|moderate|severe)
//	pain_locations pain.location items            stitches|abdomen|breast|back|head
//	breasts        breasts.symptoms items         engorgement|nipple_pain|redness ([] = normal: every item "no")
//	feeds_count    baby.feeds_count               0–30 (manual; feeding sessions arrive with B-N5-03)
//	sleep_hours    sleep.hours (+ sleep.duration) 0–24, step 0.5; the duration bucket follows for the analysis engine
//
// The alerts read bleeding.lochia_amount and bleeding.clot_size (logged from the sheet).
const (
	catBleeding = "bleeding"
	catPain     = "pain"
	catBreasts  = "breasts"
	catBaby     = "baby"
	catSleep    = "sleep"
)

// Value sets of the recovery form.
var (
	LochiaAmounts  = []string{"none", "spotting", "light", "medium", "heavy"}
	LochiaColors   = []string{"red", "pink_brown", "yellow_white"}
	PainLevels     = []string{"none", "mild", "moderate", "severe"}
	PainLocations  = []string{"stitches", "abdomen", "breast", "back", "head"}
	BreastSymptoms = []string{"engorgement", "nipple_pain", "redness"}
)

// Recovery form limits.
const (
	MaxFeeds      = 30
	MaxSleepHours = 24
)

// Recovery is one day of the recovery log. Empty strings / nil = not logged.
type Recovery struct {
	LochiaAmount  string
	LochiaColor   string
	PainLevel     string
	PainLocations []string
	Breasts       []string // nil = not logged, [] = normal
	FeedsCount    *int
	SleepHours    *float64
	LargeClots    bool // bleeding.clot_size = large (from the log sheet)
}

func num(e taxonomy.Entry) (float64, bool) {
	if !e.Num.Valid {
		return 0, false
	}
	f, err := strconv.ParseFloat(e.Num.String, 64)
	return f, err == nil
}

// RecoveryOf reads the day's entries.
func RecoveryOf(entries []taxonomy.Entry) Recovery {
	r := Recovery{}
	painNone := false
	levelRank := -1
	for _, e := range entries {
		switch e.ParamKey() {
		case catBleeding + ".lochia_amount":
			r.LochiaAmount = e.Code.String
		case catBleeding + ".lochia_color":
			r.LochiaColor = e.Code.String
		case catBleeding + ".clot_size":
			r.LargeClots = e.Code.String == "large"
		case catPain + ".none":
			painNone = e.Code.String == taxonomy.Yes
		case catPain + ".location":
			if !slices.Contains(PainLocations, e.Item) {
				continue
			}
			r.PainLocations = append(r.PainLocations, e.Item)
			if i := slices.Index(PainLevels, e.Code.String); i > levelRank {
				levelRank = i
			}
		case catBreasts + ".symptoms":
			if r.Breasts == nil {
				r.Breasts = []string{}
			}
			if slices.Contains(BreastSymptoms, e.Item) && e.Code.String != taxonomy.No {
				r.Breasts = append(r.Breasts, e.Item)
			}
		case catBaby + ".feeds_count":
			if f, ok := num(e); ok {
				n := int(f)
				r.FeedsCount = &n
			}
		case catSleep + ".hours":
			if f, ok := num(e); ok {
				r.SleepHours = &f
			}
		}
	}
	switch {
	case levelRank > 0:
		r.PainLevel = PainLevels[levelRank]
	case painNone:
		r.PainLevel = "none"
	}
	if r.Breasts != nil {
		r.Breasts = ordered(r.Breasts, BreastSymptoms)
	}
	if r.PainLocations != nil {
		r.PainLocations = ordered(r.PainLocations, PainLocations)
	}
	return r
}

// ordered keeps the codes of list that are in allowed, in allowed's order, without duplicates.
func ordered(list, allowed []string) []string {
	out := []string{}
	for _, a := range allowed {
		if slices.Contains(list, a) {
			out = append(out, a)
		}
	}
	return out
}

// RecoveryInput is a validated PUT /postpartum/recovery: Set lists the fields sent (a nil value clears the field).
type RecoveryInput struct {
	Set           map[string]bool
	LochiaAmount  string
	LochiaColor   string
	PainLevel     string
	PainLocations []string
	Breasts       []string
	BreastsNull   bool
	FeedsCount    *int
	SleepHours    *float64
}

func code(cat, param, v string) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Code: sql.NullString{String: v, Valid: true}}
}

func number(cat, param string, f float64) taxonomy.Entry {
	return taxonomy.Entry{Category: cat, Param: param, Num: sql.NullString{String: strconv.FormatFloat(f, 'f', 2, 64), Valid: true}}
}

// sleepBucket is the sleep.duration option of a number of hours.
func sleepBucket(h float64) string {
	switch {
	case h < 3:
		return "0_3"
	case h < 6:
		return "3_6"
	case h < 9:
		return "6_9"
	}
	return "9_plus"
}

// Changes are the taxonomy changes of the input (each replaces one param of the day).
func (in RecoveryInput) Changes() []taxonomy.Change {
	var out []taxonomy.Change
	add := func(cat, param string, es ...taxonomy.Entry) {
		out = append(out, taxonomy.Change{Category: cat, Param: param, Entries: es})
	}
	if in.Set["lochia_amount"] {
		if in.LochiaAmount == "" {
			add(catBleeding, "lochia_amount")
		} else {
			add(catBleeding, "lochia_amount", code(catBleeding, "lochia_amount", in.LochiaAmount))
		}
	}
	if in.Set["lochia_color"] {
		if in.LochiaColor == "" {
			add(catBleeding, "lochia_color")
		} else {
			add(catBleeding, "lochia_color", code(catBleeding, "lochia_color", in.LochiaColor))
		}
	}
	if in.Set["pain_level"] || in.Set["pain_locations"] {
		switch in.PainLevel {
		case "":
			add(catPain, "none")
			add(catPain, "location")
		case "none":
			add(catPain, "none", code(catPain, "none", taxonomy.Yes))
			add(catPain, "location")
		default:
			var es []taxonomy.Entry
			for _, loc := range in.PainLocations {
				e := code(catPain, "location", in.PainLevel)
				e.Item = loc
				es = append(es, e)
			}
			add(catPain, "none")
			add(catPain, "location", es...)
		}
	}
	if in.Set["breasts"] {
		if in.BreastsNull {
			add(catBreasts, "symptoms")
		} else {
			var es []taxonomy.Entry
			for _, s := range BreastSymptoms {
				level := taxonomy.No
				if slices.Contains(in.Breasts, s) {
					level = taxonomy.Yes
				}
				e := code(catBreasts, "symptoms", level)
				e.Item = s
				es = append(es, e)
			}
			add(catBreasts, "symptoms", es...)
		}
	}
	if in.Set["feeds_count"] {
		if in.FeedsCount == nil {
			add(catBaby, "feeds_count")
		} else {
			add(catBaby, "feeds_count", number(catBaby, "feeds_count", float64(*in.FeedsCount)))
		}
	}
	if in.Set["sleep_hours"] {
		if in.SleepHours == nil {
			add(catSleep, "hours")
			add(catSleep, "duration")
		} else {
			add(catSleep, "hours", number(catSleep, "hours", *in.SleepHours))
			add(catSleep, "duration", code(catSleep, "duration", sleepBucket(*in.SleepHours)))
		}
	}
	return out
}

// Alerts are the alert keys (guide.AlertGroup) a day raises.
func (r Recovery) Alerts() []string {
	var out []string
	if r.LochiaAmount == "heavy" {
		out = append(out, guide.AlertHeavyBleeding)
	}
	if r.LargeClots {
		out = append(out, guide.AlertLargeClots)
	}
	return out
}
