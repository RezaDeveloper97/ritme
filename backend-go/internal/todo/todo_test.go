package todo

import (
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var (
	now   = time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)
	today = civildate.InTehran(now)
)

func ids(ts []Task) []uint64 {
	out := []uint64{}
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

func TestBuildBoard_GroupsAndOrder(t *testing.T) {
	tasks := []Task{
		{ID: 1, DueDate: today, DueTime: "18:30"},
		{ID: 2, DueDate: today, DueTime: "10:00"},
		{ID: 3, DueDate: today, DoneAt: now.Add(-time.Hour)},               // done today → stays, last
		{ID: 4, DueDate: today.AddDays(-2)},                                // overdue → today, before untimed? dated earlier first
		{ID: 5, DueDate: today.AddDays(1)},                                 // tomorrow
		{ID: 6},                                                            // no date → later
		{ID: 7, DueDate: today.AddDays(9)},                                 // later, dated before undated
		{ID: 8, DueDate: today, DoneAt: now.AddDate(0, 0, -2)},             // done before today → done section
		{ID: 9, DueDate: today.AddDays(-5), DoneAt: now.Add(-time.Minute)}, // done today, due earlier → today
		{ID: 10, DoneAt: now.AddDate(0, 0, -1)},                            // done yesterday → done, newest first
	}
	b := BuildBoard(tasks, today)
	assert.Equal(t, []uint64{4, 2, 1, 9, 3}, ids(b.Today))
	assert.Equal(t, []uint64{5}, ids(b.Tomorrow))
	assert.Equal(t, []uint64{7, 6}, ids(b.Later))
	assert.Equal(t, []uint64{10, 8}, ids(b.Done))
}

func TestTaskJSON_OverdueAndList(t *testing.T) {
	j := TaskJSON(Task{ID: 1, Title: "x", Category: CategoryWork, DueDate: today.AddDays(-1), ItemsTotal: 3, ItemsOpen: 2}, today)
	b, err := json.Marshal(j)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, true, m["overdue"])
	assert.Equal(t, map[string]any{"total": float64(3), "open": float64(2)}, m["list"])
	assert.Nil(t, m["note"])
	assert.Nil(t, m["due_time"])
	assert.Equal(t, false, m["done"])
}

func TestPeriodSuggestion_Window(t *testing.T) {
	for _, tc := range []struct {
		days int
		mode enums.LifeMode
		ok   bool
	}{
		{5, enums.LifeModeCycle, true},
		{1, "", true},
		{3, enums.LifeModeTTC, true},
		{2, enums.LifeModeTeen, true},
		{6, enums.LifeModeCycle, false},
		{0, enums.LifeModeCycle, false},
		{-2, enums.LifeModeCycle, false},
		{3, enums.LifeModePregnancy, false},
		{3, enums.LifeModePostpartum, false},
		{3, enums.LifeModeMenopause, false},
		{3, enums.LifeModeCompanion, false},
	} {
		sg, ok := PeriodSuggestion(today.AddDays(tc.days), today, tc.mode)
		assert.Equal(t, tc.ok, ok, "days=%d mode=%s", tc.days, tc.mode)
		if ok {
			assert.Equal(t, tc.days, sg.Days)
			assert.Equal(t, SuggestionPeriodSupplies, sg.Key)
		}
	}
	_, ok := PeriodSuggestion(civildate.Date{}, today, enums.LifeModeCycle)
	assert.False(t, ok)
	sg := Suggestion{PeriodStart: today.AddDays(1)}
	assert.Equal(t, today, sg.TaskDue(today))
	sg.PeriodStart = today.AddDays(5)
	assert.Equal(t, today.AddDays(4), sg.TaskDue(today))
}

func TestSuggestionCopy_PickAndJSON(t *testing.T) {
	cp := SuggestionCopy{SuggestionPeriodSupplies: {
		"fa": {"prompt": "پریودت {days} روز دیگر است؛ «{title}»؟", "task_title": "خرید پد"},
	}}
	// Request-locale field, then default-language field, then the embedded fallback.
	got := cp.Pick(SuggestionPeriodSupplies, "en", "fa")
	assert.Equal(t, "خرید پد", got["task_title"])
	assert.Equal(t, "Pads", got["item_title"])
	assert.Equal(t, "Add", got["action"])

	fa := cp.Pick(SuggestionPeriodSupplies, "fa", "fa")
	b, err := json.Marshal(SuggestionJSON(Suggestion{Key: SuggestionPeriodSupplies, PeriodStart: today.AddDays(5), Days: 5}, fa, "fa"))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "پریودت ۵ روز دیگر است؛ «خرید پد»؟", m["prompt"])
	assert.Equal(t, "2026-10-11", m["period_start"])
	assert.Equal(t, "shopping", m["category"])

	// Tomorrow variant from the fallback copy.
	b, err = json.Marshal(SuggestionJSON(Suggestion{Key: SuggestionPeriodSupplies, PeriodStart: today.AddDays(1), Days: 1},
		SuggestionCopy{}.Pick(SuggestionPeriodSupplies, "en", "fa"), "en"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "Your period may start tomorrow. Add “Buy pads”?", m["prompt"])
}

func TestPlanReminders(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "Call mom", DueDate: today, DueTime: "18:30", Remind: true},
		{ID: 2, Title: "no remind", DueDate: today, DueTime: "10:00"},
		{ID: 3, Title: "done", DueDate: today, DueTime: "10:00", Remind: true, DoneAt: now},
		{ID: 4, Title: "tomorrow", DueDate: today.AddDays(1), DueTime: "10:00", Remind: true},
		{ID: 5, Title: "late night", DueDate: today, DueTime: "23:30", Remind: true},
	}
	quiet := notifications.Preferences{QuietEnabled: true, QuietStart: notifications.DefaultQuietStart, QuietEnd: notifications.DefaultQuietEnd}
	occ := PlanReminders(quiet, tasks, today, "en")
	require.Len(t, occ, 2)
	assert.Equal(t, uint64(1), occ[0].Task.ID)
	assert.Equal(t, time.Date(2026, 10, 6, 18, 30, 0, 0, civildate.Tehran), occ[0].At)
	assert.True(t, occ[0].Decision.Send)
	assert.Equal(t, "Call mom", occ[0].Message.Body)
	assert.Equal(t, "Task reminder", occ[0].Message.Title)
	assert.False(t, occ[1].Decision.Send) // quiet hours 23:00–08:00
	assert.Equal(t, notifications.ReasonQuietHours, occ[1].Decision.Reason)

	off := notifications.Preferences{Categories: map[notifications.Category]bool{ReminderCategory: false}}
	assert.False(t, PlanReminders(off, tasks, today, "en")[0].Decision.Send)
}

func body(kv ...any) phpval.Map {
	m := phpval.NewMap()
	for i := 0; i+1 < len(kv); i += 2 {
		m.Set(kv[i].(string), kv[i+1])
	}
	return m
}

func errorKeys(t *testing.T, err error) []string {
	t.Helper()
	var fe *httpx.FailError
	require.ErrorAs(t, err, &fe)
	b, mErr := json.Marshal(fe.Body())
	require.NoError(t, mErr)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	errs, _ := m["errors"].(map[string]any)
	keys := []string{}
	for k := range errs {
		keys = append(keys, k)
	}
	return keys
}

func TestValidateTask(t *testing.T) {
	in, err := ValidateTask(body("title", "Buy pads", "category", "shopping", "due_date", "2026-10-07", "due_time", "18:00",
		"remind", true), nil, "en", now)
	require.NoError(t, err)
	assert.Equal(t, TaskInput{Title: "Buy pads", Category: "shopping", DueDate: today.AddDays(1), DueTime: "18:00", Remind: true}, in)

	_, err = ValidateTask(body("category", "food"), nil, "en", now)
	assert.ElementsMatch(t, []string{"title", "category"}, errorKeys(t, err))

	_, err = ValidateTask(body("title", "x", "category", "work", "remind", true, "due_date", "2026-10-07"), nil, "en", now)
	assert.Equal(t, []string{"remind"}, errorKeys(t, err))

	_, err = ValidateTask(body("title", "x", "category", "work", "due_date", "2035-01-01"), nil, "en", now)
	assert.Equal(t, []string{"due_date"}, errorKeys(t, err))

	_, err = ValidateTask(body("title", "x", "category", "work", "due_time", "25:00"), nil, "en", now)
	assert.Equal(t, []string{"due_time"}, errorKeys(t, err))

	// Partial update keeps the current fields and reads done.
	cur := Task{ID: 1, Title: "Old", Category: "work", DueDate: today, DueTime: "10:00", Remind: true}
	in, err = ValidateTask(body("done", true), &cur, "en", now)
	require.NoError(t, err)
	assert.Equal(t, "Old", in.Title)
	assert.True(t, in.Remind)
	assert.True(t, in.Done && in.DoneSet)

	// Clearing the date of a task with a reminder needs the reminder off too.
	_, err = ValidateTask(body("due_date", nil), &cur, "en", now)
	assert.Equal(t, []string{"remind"}, errorKeys(t, err))
	in, err = ValidateTask(body("due_date", nil, "remind", false), &cur, "en", now)
	require.NoError(t, err)
	assert.True(t, in.DueDate.IsZero())

	_, err = ValidateTask(body("title", ""), &cur, "en", now)
	assert.Equal(t, []string{"title"}, errorKeys(t, err))
}

func TestValidateItem(t *testing.T) {
	in, err := ValidateItem(body("title", "Milk"), nil, "fa", now)
	require.NoError(t, err)
	assert.Equal(t, "Milk", in.Title)
	_, err = ValidateItem(body("due_date", "2026-13-01"), nil, "fa", now)
	assert.ElementsMatch(t, []string{"title", "due_date"}, errorKeys(t, err))
	in, err = ValidateItem(body("done", "1"), &Item{ID: 1, Title: "Milk"}, "fa", now)
	require.NoError(t, err)
	assert.True(t, in.Done && in.DoneSet)
	assert.Equal(t, "Milk", in.Title)
}

// Every embedded locale has the same keys as the English fallback.
func TestLangFiles_SameKeys(t *testing.T) {
	read := func(locale string) map[string]any {
		b, err := fs.ReadFile(langFS, "lang/"+locale+"/todo.json")
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		return m
	}
	var keys func(prefix string, m map[string]any) []string
	keys = func(prefix string, m map[string]any) []string {
		var out []string
		for k, v := range m {
			if sub, ok := v.(map[string]any); ok {
				out = append(out, keys(prefix+k+".", sub)...)
			} else {
				out = append(out, prefix+k)
			}
		}
		return out
	}
	en := keys("", read("en"))
	entries, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	for _, e := range entries {
		assert.ElementsMatch(t, en, keys("", read(e.Name())), e.Name())
	}
}
