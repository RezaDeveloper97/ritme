package contraception

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var d = civildate.MustParse

func TestPackFor(t *testing.T) {
	p, ok := PackFor(MethodCombinedPill, Pack21Plus7)
	require.True(t, ok)
	assert.Equal(t, Pack{Length: 28, Active: 21}, p)
	assert.Equal(t, KindActive, p.KindOf(21))
	assert.Equal(t, KindBreak, p.KindOf(22))

	p, _ = PackFor(MethodCombinedPill, Pack28)
	assert.Equal(t, KindPlacebo, p.KindOf(22))
	p, _ = PackFor(MethodCombinedPill, Pack24Plus4)
	assert.Equal(t, KindActive, p.KindOf(24))
	assert.Equal(t, KindPlacebo, p.KindOf(25))

	p, _ = PackFor(MethodProgestinPill, Pack21Plus7) // pack type ignored: every day a pill
	assert.Equal(t, KindActive, p.KindOf(28))

	_, ok = PackFor(MethodCopperIUD, "")
	assert.False(t, ok)
}

func TestPosition(t *testing.T) {
	p := Pack{Length: 28, Active: 21}
	start := d("2026-09-27")
	for _, c := range []struct {
		day       string
		idx, n    int
		packStart string
	}{
		{"2026-09-27", 0, 1, "2026-09-27"},
		{"2026-10-04", 0, 8, "2026-09-27"}, // board «بسته … · روز ۸»
		{"2026-10-24", 0, 28, "2026-09-27"},
		{"2026-10-25", 1, 1, "2026-10-25"},
		{"2026-09-26", -1, 28, "2026-08-30"},
	} {
		idx, n := p.Position(start, d(c.day))
		assert.Equal(t, c.idx, idx, c.day)
		assert.Equal(t, c.n, n, c.day)
		assert.Equal(t, c.packStart, p.PackStart(start, d(c.day)).String(), c.day)
	}
}

func takenRange(logs Logs, from, to civildate.Date) {
	for x := from; !x.After(to); x = x.AddDays(1) {
		logs.Days[x] = StatusTaken
	}
}

func TestStreak(t *testing.T) {
	p := Pack{Length: 28, Active: 21}
	start := d("2026-09-01")
	today := d("2026-10-12") // pack 2, day 14

	logs := NewLogs(civildate.Date{})
	assert.Equal(t, 0, p.Streak(start, start, logs), "first day pending: nothing yet")

	// Every pill day taken since the start (break days 22–28 of pack 1 have no log), today still pending.
	for x := start; x.Before(today); x = x.AddDays(1) {
		if p.DayKind(start, x) != KindBreak {
			logs.Days[x] = StatusTaken
		}
	}
	assert.Equal(t, 41, p.Streak(start, today, logs), "41 days before today, today pending")
	logs.Days[today] = StatusTaken
	assert.Equal(t, 42, p.Streak(start, today, logs), "board «۴۲ روز پشت سر هم»")

	delete(logs.Days, d("2026-10-05")) // an unlogged past pill day breaks the run
	assert.Equal(t, 7, p.Streak(start, today, logs))

	logs.Days[today] = StatusMissed
	assert.Equal(t, 0, p.Streak(start, today, logs))
}

func TestMissedCount(t *testing.T) {
	p := Pack{Length: 28, Active: 21}
	start := d("2026-09-27")
	today := d("2026-10-04")
	logs := NewLogs(civildate.Date{})
	takenRange(logs, start, d("2026-10-01"))
	assert.Equal(t, 2, p.MissedCount(start, today, logs), "Oct 2–3 unlogged, today pending")
	logs.Days[d("2026-10-03")] = StatusTaken
	assert.Equal(t, 0, p.MissedCount(start, today, logs))
	logs.Days[today] = StatusMissed
	assert.Equal(t, 1, p.MissedCount(start, today, logs))

	// Break days are skipped: missed day 21, break 22–28, today = next pack day 1 pending.
	logs = NewLogs(civildate.Date{})
	takenRange(logs, start, d("2026-10-16")) // days 1–20
	assert.Equal(t, 1, p.MissedCount(start, d("2026-10-25"), logs))
}

func TestStatusOn(t *testing.T) {
	p := Pack{Length: 28, Active: 21}
	start := d("2026-09-27")
	today := d("2026-10-04")
	logs := NewLogs(civildate.Date{})
	logs.Days[d("2026-10-01")] = StatusTaken
	assert.Equal(t, StatusTaken, p.StatusOn(start, d("2026-10-01"), today, logs))
	assert.Equal(t, StatusMissed, p.StatusOn(start, d("2026-10-02"), today, logs))
	assert.Equal(t, StatusPending, p.StatusOn(start, today, today, logs))
	assert.Equal(t, StatusUpcoming, p.StatusOn(start, d("2026-10-05"), today, logs))
	assert.Empty(t, p.StatusOn(start, d("2026-10-20"), today, logs), "break day")

	// Set up on day 6: earlier unlogged days are untracked, not missed, and neither count as a streak nor a miss.
	logs.Since = d("2026-10-02")
	assert.Equal(t, StatusUntracked, p.StatusOn(start, d("2026-09-30"), today, logs))
	assert.Equal(t, StatusMissed, p.StatusOn(start, d("2026-10-02"), today, logs))
	logs.Days[d("2026-10-02")] = StatusTaken
	logs.Days[d("2026-10-03")] = StatusTaken
	assert.Equal(t, 3, p.Streak(start, today, logs), "Oct 1 (logged afterwards), 2, 3; Sep 30 untracked ends it")
	assert.Equal(t, 0, p.MissedCount(start, today, logs))
}

func TestRefillAt(t *testing.T) {
	p := Pack{Length: 28, Active: 21}
	start := d("2026-09-27")
	// Counted 1 pack left during the first pack: runs out after the second pack, refill 5 days earlier.
	r := p.RefillAt(start, d("2026-10-04"), 1, start)
	assert.Equal(t, Refill{Known: true, PacksLeft: 1, RunsOutOn: d("2026-11-22"), RefillOn: d("2026-11-17")}, r)
	// One pack later the spare pack is in use.
	r = p.RefillAt(start, d("2026-10-30"), 1, start)
	assert.Equal(t, 0, r.PacksLeft)
	assert.Equal(t, d("2026-11-22"), r.RunsOutOn)
	// Never below zero.
	r = p.RefillAt(start, d("2027-03-01"), 1, start)
	assert.Equal(t, 0, r.PacksLeft)
}

func TestPlan(t *testing.T) {
	today := d("2026-09-23")
	iud := Method{Method: MethodCopperIUD, InsertedOn: d("2026-09-01"), IUDLifetimeYears: 10}
	specs := Plan(iud, today)
	require.Len(t, specs, 3)
	assert.Equal(t, Spec{Kind: KindIUDStringCheck, Type: TypeCustom, Due: d("2026-10-01"), Monthly: true}, specs[0])
	assert.Equal(t, KindIUDFollowup, specs[1].Kind)
	assert.Equal(t, d("2026-10-13"), specs[1].Due, "6 weeks after insertion")
	assert.Equal(t, KindIUDReplacement, specs[2].Kind)
	assert.Equal(t, d("2036-09-01"), specs[2].Due)

	iud.FollowupDone = true
	assert.Len(t, Plan(iud, today), 2)

	inj := Plan(Method{Method: MethodInjection, InjectedOn: d("2026-08-04")}, today)
	require.Len(t, inj, 1)
	assert.Equal(t, d("2026-10-27"), inj[0].Due, "12 weeks")

	assert.Empty(t, Plan(Method{Method: MethodImplant}, today), "implant without a date")
	assert.Len(t, Plan(Method{Method: MethodImplant, ReplaceOn: d("2029-01-01")}, today), 1)
	assert.Empty(t, Plan(Method{Method: MethodCondom}, today))

	n := 0
	pill := Method{Method: MethodCombinedPill, PackType: Pack21Plus7, PackStartedOn: d("2026-09-10"), PacksLeft: &n, PacksCountedOn: d("2026-09-10")}
	specs = Plan(pill, today)
	require.Len(t, specs, 1)
	assert.Equal(t, KindPillRefill, specs[0].Kind)
	assert.Equal(t, d("2026-10-03"), specs[0].Due, "pack runs out Oct 8, 5 days earlier")
	pill.PacksLeft = nil
	assert.Empty(t, Plan(pill, today), "no count → no refill reminder")
}

func TestSpecColumns(t *testing.T) {
	c := Spec{Kind: KindIUDFollowup, Type: TypeAppointment, Due: d("2026-10-13"), Topic: "checkup"}.Columns()
	require.NotNil(t, c.ScheduledAt)
	assert.Equal(t, "2026-10-13 09:00:00", c.ScheduledAt.In(civildate.Tehran).Format(time.DateTime))
	assert.Equal(t, "none", c.Recurrence)
	var meta care.AppointmentMeta
	require.NoError(t, json.Unmarshal(c.Meta, &meta))
	assert.Equal(t, care.StatusScheduled, meta.Status)
	assert.Equal(t, "checkup", meta.Topic)
	assert.Equal(t, care.DefaultRemindBefore, meta.RemindBefore)

	m := Spec{Kind: KindIUDStringCheck, Type: TypeCustom, Due: d("2026-10-01"), Monthly: true}.Columns()
	assert.Nil(t, m.ScheduledAt)
	assert.Equal(t, "monthly", m.Recurrence)
	assert.Equal(t, "09:00:00", *m.RecurrenceTime)
	assert.Nil(t, m.Meta)
}

func body(kv ...any) phpval.Map {
	m := phpval.NewMap()
	for i := 0; i < len(kv); i += 2 {
		m.Set(kv[i].(string), kv[i+1])
	}
	return m
}

var now = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func errorKeys(t *testing.T, err error) []string {
	t.Helper()
	var fe *httpx.FailError
	require.True(t, errors.As(err, &fe), "%v", err)
	raw, err2 := json.Marshal(fe.Body())
	require.NoError(t, err2)
	var out struct {
		Errors map[string]any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	keys := make([]string, 0, len(out.Errors))
	for k := range out.Errors {
		keys = append(keys, k)
	}
	return keys
}

func TestValidateMethod(t *testing.T) {
	_, err := validateMethod(body("method", MethodCombinedPill), "en", now)
	assert.ElementsMatch(t, []string{"pack_type", "pack_started_on"}, errorKeys(t, err))

	_, err = validateMethod(body("method", MethodHormonalIUD, "inserted_on", "2026-09-30"), "en", now)
	assert.ElementsMatch(t, []string{"inserted_on", "iud_lifetime_years"}, errorKeys(t, err))

	_, err = validateMethod(body("method", "pills"), "en", now)
	assert.ElementsMatch(t, []string{"method"}, errorKeys(t, err))

	in, err := validateMethod(body("method", MethodCombinedPill, "pack_type", Pack21Plus7,
		"pack_started_on", "2026-09-17", "reminder_time", "21:30", "packs_left", 2, "reminder_enabled", false), "en", now)
	require.NoError(t, err)
	assert.Equal(t, d("2026-09-17"), in.PackStartedOn)
	require.NotNil(t, in.ReminderMinute)
	assert.Equal(t, 21*60+30, *in.ReminderMinute)
	require.NotNil(t, in.PacksLeft)
	assert.Equal(t, 2, *in.PacksLeft)
	require.NotNil(t, in.ReminderEnabled)
	assert.False(t, *in.ReminderEnabled)

	in, err = validateMethod(body("method", MethodCondom), "en", now)
	require.NoError(t, err)
	assert.Equal(t, MethodCondom, in.Method)
	assert.Nil(t, in.ReminderEnabled)
}

func TestMethodFrom_DropsOtherMethodsFields(t *testing.T) {
	n := 1
	m := methodFrom(Input{
		Method: MethodProgestinPill, PackType: Pack24Plus4, PackStartedOn: d("2026-09-01"), PacksLeft: &n,
		InsertedOn: d("2026-01-01"), InjectedOn: d("2026-01-01"),
	}, d("2026-09-30"))
	assert.Empty(t, m.PackType, "pack type is combined-pill only")
	assert.True(t, m.InsertedOn.IsZero())
	assert.True(t, m.InjectedOn.IsZero())
	assert.Equal(t, d("2026-09-29"), m.PacksCountedOn, "count dated to the pack in use")
}

func TestValidatePill_Defaults(t *testing.T) {
	in, err := validatePill(phpval.NewMap(), "en", now)
	require.NoError(t, err)
	assert.Equal(t, PillInput{Date: d("2026-09-23"), Status: StatusTaken}, in)
	_, err = validatePill(body("date", "2026-09-24"), "en", now)
	assert.Equal(t, []string{"date"}, errorKeys(t, err))
	_, err = validatePill(body("status", "skipped"), "en", now)
	assert.Equal(t, []string{"status"}, errorKeys(t, err))
}

func TestPillJSON_Board(t *testing.T) {
	n := 1
	start := d("2026-09-27")
	m := Method{Method: MethodCombinedPill, PackType: Pack21Plus7, PackStartedOn: start, PacksLeft: &n, PacksCountedOn: start}
	p, _ := m.Pack()
	logs := NewLogs(civildate.Date{})
	takenRange(logs, start, d("2026-10-04"))
	j := PillJSON(m, p, d("2026-10-04"), logs)
	raw, err := json.Marshal(j)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.InDelta(t, 1, got["pack_number"], 0)
	assert.InDelta(t, 8, got["pack_day"], 0)
	assert.InDelta(t, 2, got["pack_week"], 0)
	assert.InDelta(t, 8, got["streak_days"], 0)
	assert.Equal(t, "2026-10-25", got["next_pack_on"])
	assert.Equal(t, "2026-11-17", got["refill_on"])
	assert.Len(t, got["days"], 28)
	assert.Equal(t, map[string]any{"date": "2026-10-04", "kind": "active", "status": "taken"}, got["today"])
}

func TestLangFiles_SameKeys(t *testing.T) {
	keys := []string{"messages.validation_failed", "messages.method_saved", "messages.method_removed",
		"messages.pill_saved", "messages.pill_removed", "validation.date_future", "validation.date_before_pack",
		"validation.pill_method_required", "validation.break_day"}
	for _, k := range ReminderKinds {
		keys = append(keys, "reminders."+k)
	}
	for _, key := range keys {
		for _, loc := range []string{"fa", "en"} {
			assert.NotEqual(t, "contraception."+key, T(key, loc), "%s missing in %s", key, loc)
		}
	}
	assert.Len(t, attributes("fa"), len(attributes("en")))
}
