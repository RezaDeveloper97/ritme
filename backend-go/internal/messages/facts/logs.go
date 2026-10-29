package facts

import (
	"context"
	"strconv"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Log facts: activity (days_since_last_log, logs_30d) from the day log (health_log_entries), and the symptom pattern
// symptom_history.<code> — the number of cycles in which the symptom was logged on a day of the same main phase as
// today (the engine resolves every log day; cycles are told apart by their start). The pattern needs a known main
// phase today; it is absent otherwise, and 0 when the symptom was never logged in that phase.

// LogStats is the user's log activity up to today.
type LogStats struct {
	Last    civildate.Date // zero = never logged
	Days30d int            // distinct log days in the last 30 days, today included
}

// SymptomEntry is one logged symptom item (health_log_entries pain.location, symptoms.digestive, symptoms.general).
type SymptomEntry struct {
	Date     civildate.Date
	Category string
	Param    string
	Item     string
	Level    string // value_code: yes | no | mild | moderate | severe
	Score    string // value_num (pain 1–10), "" when none
}

// LogSource reads the user's day log.
type LogSource interface {
	LogStats(ctx context.Context, since, today civildate.Date) (LogStats, error)
	SymptomEntries(ctx context.Context, from, to civildate.Date) ([]SymptomEntry, error)
}

// Groups of this file.
const (
	GroupLogs     = "logs"
	GroupSymptoms = "symptoms"
)

// SymptomHistoryDays is the look-back of symptom_history (about six cycles).
const SymptomHistoryDays = 180

// SymptomPrefix starts every symptom_history fact name.
const SymptomPrefix = "symptom_history."

// symptomCodes map a symptom code onto the log items that count as it (category.param.item).
var symptomCodes = map[string][]string{
	"cramps":            {"pain.location.abdomen", "pain.location.pelvis"},
	"headache":          {"pain.location.head"},
	"backache":          {"pain.location.back"},
	"breast_pain":       {"pain.location.breast"},
	"nausea":            {"symptoms.digestive.nausea"},
	"bloating":          {"symptoms.digestive.bloating"},
	"constipation":      {"symptoms.digestive.constipation"},
	"diarrhea":          {"symptoms.digestive.diarrhea"},
	"heartburn":         {"symptoms.digestive.heartburn"},
	"fatigue":           {"symptoms.general.fatigue"},
	"dizziness":         {"symptoms.general.dizziness"},
	"breast_tenderness": {"symptoms.general.breast_tenderness"},
	"swelling":          {"symptoms.general.swelling"},
	"hot_flashes":       {"symptoms.general.hot_flashes"},
	"chills":            {"symptoms.general.chills"},
	"insomnia":          {"symptoms.general.insomnia"},
}

// itemCodes is symptomCodes inverted: log item → symptom codes.
var itemCodes = func() map[string][]string {
	out := map[string][]string{}
	for code, items := range symptomCodes {
		for _, it := range items {
			out[it] = append(out[it], code)
		}
	}
	return out
}()

// SymptomCodes are the symptom_history codes, sorted.
func SymptomCodes() []string {
	m := map[string]bool{}
	for c := range symptomCodes {
		m[c] = true
	}
	return setOf(m)
}

func init() {
	Register(
		Def{Name: "days_since_last_log", Type: TypeInt, Unit: "day", Group: GroupLogs,
			Description: desc("روز از آخرین ثبت (بدون ثبت وجود ندارد)", "Days since the last log (absent when never logged).")},
		Def{Name: "logs_30d", Type: TypeInt, Unit: "day", Group: GroupLogs,
			Description: desc("روزهای ثبت در ۳۰ روز اخیر", "Days with a log in the last 30 days.")},
	)
	for _, c := range SymptomCodes() {
		Register(Def{Name: SymptomPrefix + c, Type: TypeInt, Unit: "cycle", Group: GroupSymptoms,
			Description: desc("تعداد سیکل\u200cهایی که «"+c+"» در همان فاز امروز ثبت شده", "Cycles in which «"+c+"» was logged in today's phase.")})
	}
	RegisterGroup(GroupLogs, logFacts)
	RegisterGroup(GroupSymptoms, symptomFacts)
}

func logFacts(ctx context.Context, f *Facts) (Values, error) {
	if f.src.Logs == nil {
		return nil, nil
	}
	st, err := f.src.Logs.LogStats(ctx, f.today.AddDays(-29), f.today)
	if err != nil {
		return nil, err
	}
	v := Values{"logs_30d": float64(st.Days30d)}
	if !st.Last.IsZero() {
		v["days_since_last_log"] = float64(st.Last.DiffDays(f.today))
	}
	return v, nil
}

// present reports whether a symptom entry says the symptom was there (a level other than «no», or a score > 0).
func (e SymptomEntry) present() bool {
	switch e.Level {
	case "yes", "mild", "moderate", "severe":
		return true
	}
	n, err := strconv.ParseFloat(e.Score, 64)
	return err == nil && n > 0
}

func symptomFacts(ctx context.Context, f *Facts) (Values, error) {
	m, err := f.engineState(ctx)
	if err != nil || m == nil || f.src.Logs == nil {
		return nil, err
	}
	phase := mainPhaseOf(&m.status, m.strip)
	if m.status.MainPhase == engine.MainPhaseUnknown {
		return nil, nil
	}
	entries, err := f.src.Logs.SymptomEntries(ctx, f.today.AddDays(-(SymptomHistoryDays - 1)), f.today)
	if err != nil {
		return nil, err
	}
	today := engine.DayOf(f.today)
	cycles := map[string]map[engine.Day]bool{} // code → cycle starts
	resolved := map[civildate.Date]*engine.Status{}
	for _, e := range entries {
		codes := itemCodes[e.Category+"."+e.Param+"."+e.Item]
		if len(codes) == 0 || !e.present() || e.Date.After(f.today) {
			continue
		}
		s, ok := resolved[e.Date]
		if !ok {
			st := m.state.Resolve(engine.DayOf(e.Date), today)
			s = &st
			resolved[e.Date] = s
		}
		start := s.Anchors.CurrentPeriodStart
		if !start.IsSet() || s.MainPhase == engine.MainPhaseUnknown || mainPhaseOf(s, m.strip) != phase {
			continue
		}
		for _, c := range codes {
			if cycles[c] == nil {
				cycles[c] = map[engine.Day]bool{}
			}
			cycles[c][start] = true
		}
	}
	v := Values{}
	for _, c := range SymptomCodes() {
		v[SymptomPrefix+c] = float64(len(cycles[c]))
	}
	return v, nil
}
