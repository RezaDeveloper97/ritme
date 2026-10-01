package conditions

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var now = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func day(s string) civildate.Date { return civildate.MustParse(s) }

func body(t *testing.T, s string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(s))
	require.NoError(t, err)
	m, ok := v.(phpval.Map)
	require.True(t, ok)
	return m
}

func errorsOf(t *testing.T, err error) map[string]any {
	t.Helper()
	var fe *httpx.FailError
	require.True(t, errors.As(err, &fe), "want a controller 422, got %v", err)
	assert.Equal(t, 422, fe.Status)
	raw, jerr := json.Marshal(fe.Body())
	require.NoError(t, jerr)
	var out struct {
		Errors map[string]any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	return out.Errors
}

func entry(cat, param, item, code, num string) taxonomy.Entry {
	e := taxonomy.Entry{Category: cat, Param: param, Item: item}
	if code != "" {
		e.Code = sql.NullString{String: code, Valid: true}
	}
	if num != "" {
		e.Num = sql.NullString{String: num, Valid: true}
	}
	return e
}

// ---- pain -------------------------------------------------------------------------------------------------------

func TestLevelFor(t *testing.T) {
	for score, want := range map[int]string{1: "mild", 3: "mild", 4: "moderate", 6: "moderate", 7: "severe", 10: "severe"} {
		assert.Equal(t, want, LevelFor(score), "score %d", score)
	}
}

func TestPainFromLog(t *testing.T) {
	d := painFromLog(day("2026-09-20"), []taxonomy.Entry{
		entry("pain", "location", "abdomen", "moderate", "5.00"),
		entry("pain", "location", "back", "severe", "7.00"),
		entry("pain", "relief", "heat", "yes", ""),
		entry("mood", "moods", "sad", "yes", ""),
	})
	require.NotNil(t, d.Score)
	assert.Equal(t, 7, *d.Score, "the highest location score")
	assert.Equal(t, []string{"abdomen", "back"}, d.Locations)
	assert.Equal(t, []string{"heat"}, d.Relief)

	d = painFromLog(day("2026-09-20"), []taxonomy.Entry{entry("pain", "none", "", "yes", "")})
	require.NotNil(t, d.Score)
	assert.Equal(t, 0, *d.Score)

	d = painFromLog(day("2026-09-20"), []taxonomy.Entry{entry("pain", "location", "head", "mild", "")})
	assert.Nil(t, d.Score, "a level-only location (log sheet without score) has no 0–10 score")
	assert.Equal(t, []string{"head"}, d.Locations)
}

func assocItem(code, meta string) catalog.Item {
	it := catalog.Item{Code: code, Title: json.RawMessage(`{"fa":"x","en":"x"}`)}
	if meta != "" {
		it.Meta = json.RawMessage(meta)
	}
	return it
}

func TestSlotOf(t *testing.T) {
	assert.True(t, slotOf(assocItem("bloating", `{"log":"symptoms.digestive.bloating"}`)).valid)
	assert.True(t, slotOf(assocItem("dyspareunia", `{"log":"sex.symptoms.pain_during_intercourse"}`)).valid, "multi param")
	assert.False(t, slotOf(assocItem("x", `{"log":"symptoms.digestive.nope"}`)).valid, "unknown option")
	assert.False(t, slotOf(assocItem("x", `{"log":"sleep.quality.good"}`)).valid, "single params are not slots")
	assert.False(t, slotOf(assocItem("x", `{"log":"pain.location.head"}`)).valid, "items without a yes level")
	assert.False(t, slotOf(assocItem("dyschezia", "")).valid)
}

func TestAssociatedChanges(t *testing.T) {
	items := []catalog.Item{
		assocItem("dyspareunia", `{"log":"sex.symptoms.pain_during_intercourse"}`),
		assocItem("dyschezia", ""),
		assocItem("bloating", `{"log":"symptoms.digestive.bloating"}`),
		assocItem("nausea", `{"log":"symptoms.digestive.nausea"}`),
	}
	stored := []taxonomy.Entry{
		entry("symptoms", "digestive", "nausea", "severe", ""),
		entry("symptoms", "digestive", "heartburn", "yes", ""),
	}

	// cycle mode: every slot is writable
	edit := newDayEdit(stored)
	row := associatedChanges(edit, items, stored, taxonomy.ModeCycle, []string{"nausea"}, []string{"dyspareunia", "dyschezia", "bloating"})
	assert.Equal(t, []string{"dyschezia"}, row, "only the slot-less code stays in the row")
	byParam := map[string][]taxonomy.Entry{}
	for _, ch := range edit.changes() {
		byParam[ch.Key()] = ch.Entries
	}
	assert.Equal(t, []taxonomy.Entry{entry("sex", "symptoms", "pain_during_intercourse", "yes", "")}, byParam["sex.symptoms"])
	assert.Equal(t, []taxonomy.Entry{
		entry("symptoms", "digestive", "heartburn", "yes", ""),
		entry("symptoms", "digestive", "bloating", "yes", ""),
	}, byParam["symptoms.digestive"], "nausea deselected, heartburn untouched, bloating added")

	// teen mode has no sex category: the choice falls back to the row
	edit = newDayEdit(nil)
	row = associatedChanges(edit, items, nil, taxonomy.ModeTeen, nil, []string{"dyspareunia", "nausea"})
	assert.Equal(t, []string{"dyspareunia"}, row)
	require.Len(t, edit.changes(), 1)
	assert.Equal(t, "symptoms.digestive", edit.changes()[0].Key())

	// a stored "severe" level is kept when the item stays selected
	edit = newDayEdit(stored)
	associatedChanges(edit, items, stored, taxonomy.ModeCycle, []string{"nausea"}, []string{"nausea"})
	assert.Empty(t, edit.changes())
}

func TestRowOnly(t *testing.T) {
	items := []catalog.Item{assocItem("bloating", `{"log":"symptoms.digestive.bloating"}`), assocItem("dyschezia", "")}
	got := rowOnly(items, []taxonomy.Entry{entry("symptoms", "digestive", "bloating", "yes", "")}, []string{"bloating", "dyschezia", "retired"})
	assert.Equal(t, []string{"dyschezia", "retired"}, got)
}

func TestPainScoreChanges(t *testing.T) {
	score := func(n int) *int { return &n }
	cur := &PainDay{Locations: []string{}}

	edit := newDayEdit(nil)
	require.NoError(t, painScoreChanges(edit, cur, PainInput{Set: map[string]bool{"score": true, "locations": true}, Score: score(8), Locations: []string{"pelvis", "back"}}))
	ch := edit.changes()
	require.Len(t, ch, 2)
	assert.Empty(t, ch[0].Entries, "pain.none cleared")
	assert.Equal(t, []taxonomy.Entry{
		entry("pain", "location", "pelvis", "severe", "8.00"),
		entry("pain", "location", "back", "severe", "8.00"),
	}, ch[1].Entries)

	edit = newDayEdit(nil)
	require.NoError(t, painScoreChanges(edit, &PainDay{Score: score(5), Locations: []string{"back"}}, PainInput{Set: map[string]bool{"score": true}, Score: score(0)}))
	ch = edit.changes()
	assert.Equal(t, []taxonomy.Entry{entry("pain", "none", "", "yes", "")}, ch[0].Entries)
	assert.Empty(t, ch[1].Entries, "0 = no pain clears the locations")

	err := painScoreChanges(newDayEdit(nil), cur, PainInput{Set: map[string]bool{"score": true}, Score: score(4)})
	var fe *FieldError
	require.ErrorAs(t, err, &fe)
	assert.Equal(t, "locations", fe.Field)

	err = painScoreChanges(newDayEdit(nil), cur, PainInput{Set: map[string]bool{"locations": true}, Locations: []string{"back"}})
	require.ErrorAs(t, err, &fe)
	assert.Equal(t, "score", fe.Field)
}

func TestValidatePain(t *testing.T) {
	ch := PainChoices{Locations: []string{"abdomen", "pelvis", "back"}, Relief: []string{"heat", "painkiller"},
		Types: []string{"cramping", "burning"}, Associated: []string{"bloating"}}
	in, err := validatePain(day("2026-09-23"), body(t, `{"score":"7","locations":["back","abdomen"],"types":["burning","cramping"],"missed_activity":true,"analgesic":" Ibuprofen ","analgesic_time":"08:30","analgesic_effect":"a_little","extra":1}`), ch, "en", now)
	require.NoError(t, err)
	assert.Equal(t, 7, *in.Score)
	assert.Equal(t, []string{"abdomen", "back"}, in.Locations, "known order")
	assert.Equal(t, []string{"cramping", "burning"}, in.Types)
	assert.True(t, *in.MissedActivity)
	assert.Equal(t, "Ibuprofen", *in.Analgesic)
	assert.Equal(t, "08:30", *in.AnalgesicTime)
	assert.False(t, in.Set["relief"])

	in, err = validatePain(day("2026-09-23"), body(t, `{"analgesic":null,"relief":null}`), ch, "en", now)
	require.NoError(t, err)
	assert.True(t, in.Set["analgesic"])
	assert.Nil(t, in.Analgesic)
	assert.Nil(t, in.Relief)

	_, err = validatePain(day("2026-09-23"), body(t, `{"score":11,"locations":["head"],"types":["x"],"associated":["fever"],"analgesic_time":"8:5","analgesic_effect":"maybe","missed_activity":"sometimes"}`), ch, "en", now)
	errs := errorsOf(t, err)
	for _, k := range []string{"score", "locations.0", "types.0", "associated.0", "analgesic_time", "analgesic_effect", "missed_activity"} {
		assert.Contains(t, errs, k)
	}
}

// ---- PMDD -------------------------------------------------------------------------------------------------------

func pmddDay(date string, scores ...int) PMDDDay {
	codes := []string{"sadness", "anxiety", "mood_swings", "anger", "loss_of_interest", "concentration"}
	d := PMDDDay{Date: day(date)}
	for i, s := range scores {
		d.Scores = append(d.Scores, ItemScore{Code: codes[i], Score: s})
	}
	return d
}

func TestPMDDDayMean(t *testing.T) {
	assert.InDelta(t, 2.33, pmddDay("2026-09-01", 1, 2, 4).Mean(), 0.0001)
	assert.InDelta(t, 0.0, pmddDay("2026-09-01").Mean(), 0.0001)
}

func TestOrderScores(t *testing.T) {
	got := orderScores(map[string]int{"anger": 3, "old_item": 2, "sadness": 5}, []string{"sadness", "anxiety", "anger"})
	assert.Equal(t, []ItemScore{{"sadness", 5}, {"anger", 3}, {"old_item", 2}}, got)
}

// ratedCycle rates days [from, to] of a cycle with score s on every item.
func rate(days *[]PMDDDay, from, to civildate.Date, s int) {
	for d := from; !d.After(to); d = d.AddDays(1) {
		*days = append(*days, pmddDay(d.String(), s, s, s, s, s, s))
	}
}

func TestBuildCycles_AndSummarise(t *testing.T) {
	// three recorded starts: 2026-07-01, 2026-07-29 (28 days), 2026-08-26 (28 days); today 2026-09-10 (cycle day 16)
	starts := []PeriodStart{{Start: day("2026-08-26")}, {Start: day("2026-07-29"), BleedingDays: 4}, {Start: day("2026-07-01")}}
	var days []PMDDDay
	// cycle 1 (07-01 … 07-28): follicular days 4–10 low, late luteal (07-19 … 07-28) high
	rate(&days, day("2026-07-04"), day("2026-07-10"), 1)
	rate(&days, day("2026-07-19"), day("2026-07-28"), 4)
	// cycle 2 (07-29 … 08-25): same pattern, only 6 late-luteal days rated → not complete
	rate(&days, day("2026-08-01"), day("2026-08-07"), 2)
	rate(&days, day("2026-08-20"), day("2026-08-25"), 5)
	// current cycle: a few follicular days
	rate(&days, day("2026-08-29"), day("2026-09-02"), 1)

	cycles := BuildCycles(starts, day("2026-09-10"), 28, 5, days)
	require.Len(t, cycles, 3)

	cur := cycles[0]
	assert.False(t, cur.Closed)
	assert.Equal(t, day("2026-09-22"), cur.End, "expected end = start + 28 − 1")
	assert.Equal(t, Window{day("2026-08-26"), day("2026-08-30")}, cur.Period, "profile period length 5")
	assert.Equal(t, Window{day("2026-09-13"), day("2026-09-22")}, cur.LateLuteal)
	assert.Equal(t, 5, cur.FollicularDays)
	assert.False(t, cur.Complete)

	c2 := cycles[1]
	assert.True(t, c2.Closed)
	assert.Equal(t, day("2026-08-25"), c2.End)
	assert.Equal(t, Window{day("2026-07-29"), day("2026-08-01")}, c2.Period, "bleeding_length 4")
	assert.Equal(t, 6, c2.LutealDays)
	assert.False(t, c2.Complete, "6 < 7 late-luteal days")
	require.NotNil(t, c2.LutealMean)
	assert.InDelta(t, 5.0, *c2.LutealMean, 0.001)
	assert.Equal(t, 4, c2.Days[0].CycleDay)
	assert.True(t, c2.Days[0].InPeriod, "08-01 is cycle day 4, the last bleeding day")

	c1 := cycles[2]
	assert.True(t, c1.Complete)
	assert.Equal(t, 10, c1.LutealDays)
	assert.Equal(t, 7, c1.FollicularDays)

	complete, pattern := Summarise(cycles)
	assert.Equal(t, 1, complete)
	assert.Equal(t, PatternLuteal, pattern, "both evaluable closed cycles rise ≥ 30 %")

	// a flat cycle makes the pattern unclear
	rate(&days, day("2026-07-11"), day("2026-07-18"), 4)
	flat := BuildCycles(starts, day("2026-09-10"), 28, 5, append([]PMDDDay{}, pmddDaysWithFlatFollicular(days)...))
	_, pattern = Summarise(flat)
	assert.Equal(t, PatternUnclear, pattern)

	_, pattern = Summarise(BuildCycles(starts, day("2026-09-10"), 28, 5, nil))
	assert.Equal(t, PatternNotEnoughData, pattern)
	assert.Empty(t, BuildCycles(nil, day("2026-09-10"), 28, 5, days), "no recorded period → no cycles")
}

// pmddDaysWithFlatFollicular rewrites cycle 1's follicular days to the luteal level.
func pmddDaysWithFlatFollicular(days []PMDDDay) []PMDDDay {
	w := Window{day("2026-07-04"), day("2026-07-10")}
	out := make([]PMDDDay, len(days))
	for i, d := range days {
		out[i] = d
		if w.Has(d.Date) {
			out[i] = pmddDay(d.Date.String(), 4, 4, 4, 4, 4, 4)
		}
	}
	return out
}

func TestBuildCycles_OverdueAndBadLength(t *testing.T) {
	cycles := BuildCycles([]PeriodStart{{Start: day("2026-08-01")}}, day("2026-09-10"), 0, 0, nil)
	require.Len(t, cycles, 1)
	assert.Equal(t, day("2026-09-10"), cycles[0].End, "overdue: runs until today")
	assert.Equal(t, day("2026-08-05"), cycles[0].Period.To, "default 5-day period")
}

func TestValidatePMDD(t *testing.T) {
	items := []string{"sadness", "anxiety"}
	in, err := validatePMDD(day("2026-09-23"), body(t, `{"scores":{"sadness":"6","anxiety":null}}`), items, "en", now)
	require.NoError(t, err)
	assert.Equal(t, 6, *in.Scores["sadness"])
	assert.Contains(t, in.Scores, "anxiety")
	assert.Nil(t, in.Scores["anxiety"])

	_, err = validatePMDD(day("2026-09-23"), body(t, `{"scores":{"sadness":7}}`), items, "en", now)
	assert.Contains(t, errorsOf(t, err), "scores.sadness")
	_, err = validatePMDD(day("2026-09-23"), body(t, `{"scores":{"joy":3}}`), items, "en", now)
	assert.Contains(t, errorsOf(t, err), "scores.joy")
	_, err = validatePMDD(day("2026-09-23"), body(t, `{}`), items, "en", now)
	assert.Contains(t, errorsOf(t, err), "scores")
}

// ---- PBAC -------------------------------------------------------------------------------------------------------

func TestPBACScore(t *testing.T) {
	large, small, none := ClotLarge, ClotSmall, ClotNone
	// board: 1 light, 2 medium, 3 heavy = 1 + 10 + 60 = 71 («۷۱ امتیاز امروز»)
	assert.Equal(t, 71, PBACDay{Light: 1, Medium: 2, Heavy: 3}.Score())
	assert.Equal(t, 71+5+5, PBACDay{Light: 1, Medium: 2, Heavy: 3, Clots: &large, Flooding: true}.Score())
	assert.Equal(t, 1, PBACDay{Clots: &small}.Score())
	assert.Equal(t, 0, PBACDay{Clots: &none}.Score())
	assert.True(t, PBACDay{Clots: &none}.Logged())
	assert.False(t, PBACDay{}.Logged())
}

func TestClotsFromLog(t *testing.T) {
	assert.Nil(t, clotsFromLog(nil))
	assert.Equal(t, ClotNone, *clotsFromLog([]taxonomy.Entry{entry("bleeding", "clots", "", "no", "")}))
	assert.Equal(t, ClotLarge, *clotsFromLog([]taxonomy.Entry{entry("bleeding", "clots", "", "yes", ""), entry("bleeding", "clot_size", "", "large", "")}))
	assert.Equal(t, ClotSmall, *clotsFromLog([]taxonomy.Entry{entry("bleeding", "clots", "", "yes", ""), entry("bleeding", "clot_size", "", "medium", "")}), "legacy size")
	assert.Equal(t, ClotSmall, *clotsFromLog([]taxonomy.Entry{entry("bleeding", "clots", "", "yes", "")}))
}

func TestPeriodStartFor(t *testing.T) {
	logged := map[civildate.Date]bool{day("2026-09-18"): true, day("2026-09-19"): true, day("2026-09-21"): true, day("2026-09-22"): true}
	is := func(d civildate.Date) bool { return logged[d] }

	s, ok := PeriodStartFor([]PeriodStart{{Start: day("2026-09-20")}}, day("2026-09-22"), is)
	assert.True(t, ok)
	assert.Equal(t, day("2026-09-20"), s, "recorded start within 10 days")

	s, ok = PeriodStartFor([]PeriodStart{{Start: day("2026-08-20")}}, day("2026-09-22"), is)
	assert.True(t, ok)
	assert.Equal(t, day("2026-09-21"), s, "old start → the run of logged days")

	end := day("2026-09-20")
	s, ok = PeriodStartFor([]PeriodStart{{Start: day("2026-09-16"), End: &end}}, day("2026-09-22"), is)
	assert.True(t, ok)
	assert.Equal(t, day("2026-09-21"), s, "the recorded period ended before → the run of logged days")

	_, ok = PeriodStartFor(nil, day("2026-09-20"), is)
	assert.False(t, ok, "nothing logged, no start")
}

func TestValidatePBAC(t *testing.T) {
	in, err := validatePBAC(day("2026-09-23"), body(t, `{"light":1,"medium":"2","heavy":null,"clots":"large","flooding":true}`), ClotChoices, "en", now)
	require.NoError(t, err)
	assert.Equal(t, 1, *in.Light)
	assert.Equal(t, 2, *in.Medium)
	assert.True(t, in.Set["heavy"])
	assert.Nil(t, in.Heavy)
	assert.Equal(t, ClotLarge, *in.Clots)
	assert.True(t, *in.Flooding)

	_, err = validatePBAC(day("2026-09-23"), body(t, `{"light":51,"medium":-1,"clots":"huge","flooding":"x"}`), ClotChoices, "fa", now)
	errs := errorsOf(t, err)
	for _, k := range []string{"light", "medium", "clots", "flooding"} {
		assert.Contains(t, errs, k)
	}
	_, err = validatePBAC(day("2026-09-23"), body(t, `{"clots":"small"}`), []string{}, "en", now)
	assert.Contains(t, errorsOf(t, err), "clots", "no clot param in the mode")
}

func TestPBACJSON_Board(t *testing.T) {
	alert := catalog.Item{Code: "pbac_over_100", Title: json.RawMessage(`{"fa":"بیش از ۱۰۰","en":"Over 100"}`), NeedsReview: true}
	v := PBACView{
		Day:    PBACDay{Date: day("2026-09-21"), Light: 1, Medium: 2, Heavy: 3},
		Period: &PBACPeriod{Start: day("2026-09-20"), End: day("2026-09-29"), DayNumber: 2, Score: 112, DaysLogged: 2},
		Alert:  &alert,
	}
	raw, err := jsonx.Marshal(PBACJSON(v, catalog.Localizer{Locale: "en", Default: "fa"}), 0)
	require.NoError(t, err)
	assert.JSONEq(t, `{"date":"2026-09-21","light":1,"medium":2,"heavy":3,"clots":null,"flooding":false,"score":71,
		"period":{"start":"2026-09-20","end":"2026-09-29","day":2,"score":112,"days_logged":2},"alert_threshold":100,
		"alert":{"code":"pbac_over_100","title":"Over 100","body":null,"meta":null,"audiences":null,"needs_review":true}}`, string(raw))
}

// ---- enrolment, lang ----------------------------------------------------------------------------------------------

func TestValidateEnrol(t *testing.T) {
	p, on, err := validateEnrol(body(t, `{"program":"pmdd"}`), "en", now)
	require.NoError(t, err)
	assert.Equal(t, ProgramPMDD, p)
	assert.Equal(t, "2026-09-23", on.String())

	_, _, err = validateEnrol(body(t, `{"program":"diabetes","enrolled_on":"2026-09-24"}`), "fa", now)
	errs := errorsOf(t, err)
	assert.Contains(t, errs, "program")
	assert.Equal(t, []any{"ثبت برای روزهای آینده ممکن نیست."}, errs["enrolled_on"])

	_, err = validateProgram("nope", "en", now)
	assert.Contains(t, errorsOf(t, err), "program")
}

func TestOverviewJSON(t *testing.T) {
	raw, err := jsonx.Marshal(OverviewJSON([]Enrolment{{Program: ProgramEndo, Enrolled: true, EnrolledOn: day("2026-09-01")}, {Program: ProgramPCOS}}), 0)
	require.NoError(t, err)
	assert.JSONEq(t, `{"programs":[{"code":"endo","enrolled":true,"enrolled_on":"2026-09-01"},{"code":"pcos","enrolled":false,"enrolled_on":null}]}`, string(raw))
}

func TestLangFiles_SameKeys(t *testing.T) {
	for _, key := range []string{"messages.validation_failed", "messages.enrolled", "messages.left", "messages.pain_saved",
		"messages.pmdd_saved", "messages.pbac_saved", "validation.date_future", "validation.program_required",
		"validation.locations_required", "validation.score_required"} {
		for _, loc := range []string{"fa", "en"} {
			assert.NotEqual(t, "conditions."+key, T(key, loc), "%s missing in %s", key, loc)
		}
	}
	assert.Len(t, attributes("fa"), len(attributes("en")))
}
