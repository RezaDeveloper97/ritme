package notifications

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/store"
)

func at(hh, mm int) time.Time { return time.Date(2026, 10, 1, hh, mm, 0, 0, civildate.Tehran) }

func TestDefaults_MatchBoard(t *testing.T) {
	p := Defaults()
	for _, c := range []Category{BeforePeriod, PMS, DailyLog, Medications, Appointments, Checkups, Learning, Companion} {
		assert.True(t, p.Enabled(c), c)
	}
	for _, c := range []Category{FertileWindow, Vitals, Articles, Category("marketing")} {
		assert.False(t, p.Enabled(c), c)
	}
	assert.True(t, p.NeutralCopy)
	assert.Equal(t, "23:00", FormatClock(p.QuietStart))
	assert.Equal(t, "08:00", FormatClock(p.QuietEnd))
}

func TestInQuietHours_WrapsMidnight(t *testing.T) {
	p := Defaults() // 23:00 → 08:00
	cases := map[time.Time]bool{
		at(22, 59): false, at(23, 0): true, at(0, 30): true, at(7, 59): true, at(8, 0): false, at(12, 0): false,
	}
	for now, want := range cases {
		assert.Equal(t, want, p.InQuietHours(now), now.Format("15:04"))
	}
	// A UTC instant is judged on Tehran wall-clock: 20:00Z = 23:30 Tehran.
	assert.True(t, p.InQuietHours(time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC)))
}

func TestInQuietHours_SameDayDisabledAndEmpty(t *testing.T) {
	p := Defaults()
	p.QuietStart, p.QuietEnd = 13*60, 15*60
	assert.True(t, p.InQuietHours(at(14, 0)))
	assert.False(t, p.InQuietHours(at(15, 0)))
	assert.False(t, p.InQuietHours(at(23, 30)))

	p.QuietEnabled = false
	assert.False(t, p.InQuietHours(at(14, 0)))

	p.QuietEnabled, p.QuietEnd = true, 13*60 // start == end → empty window
	assert.False(t, p.InQuietHours(at(13, 0)))
}

func TestDecide(t *testing.T) {
	p := Defaults()

	d := Decide(p, BeforePeriod, at(9, 0), time.Time{})
	assert.Equal(t, Decision{Send: true}, d)

	d = Decide(p, FertileWindow, at(9, 0), time.Time{})
	assert.Equal(t, ReasonCategoryOff, d.Reason)
	assert.True(t, d.DeferUntil.IsZero(), "a switched-off category is dropped, not deferred")

	d = Decide(p, BeforePeriod, at(23, 30), time.Time{})
	assert.False(t, d.Send)
	assert.Equal(t, ReasonQuietHours, d.Reason)
	assert.Equal(t, time.Date(2026, 10, 2, 8, 0, 0, 0, civildate.Tehran), d.DeferUntil)

	d = Decide(p, BeforePeriod, at(1, 0), time.Time{})
	assert.Equal(t, at(8, 0), d.DeferUntil, "after midnight the window ends the same morning")

	p.Categories[Articles] = true
	d = Decide(p, Articles, at(12, 0), at(12, 0).AddDate(0, 0, -3))
	assert.Equal(t, ReasonWeeklyCap, d.Reason)
	assert.Equal(t, at(12, 0).AddDate(0, 0, 4), d.DeferUntil)
	assert.True(t, Decide(p, Articles, at(12, 0), at(12, 0).AddDate(0, 0, -7)).Send)
}

func TestRender_NeutralCopyHidesHealthData(t *testing.T) {
	msg := Message{Category: BeforePeriod, Title: "پریودت ۲ روز دیگر است", Body: "روز ۲۶ سیکل", URL: "/calendar"}

	for _, loc := range []string{"fa", "en", "de"} {
		push := Render(Defaults(), msg, loc)
		assert.NotContains(t, push.Title+push.Body, "پریود", loc)
		assert.NotContains(t, push.Title+push.Body, "سیکل", loc)
		assert.NotEmpty(t, push.Title, loc)
		assert.Equal(t, "/calendar", push.URL)
	}
	assert.Equal(t, "یادآور ریتمی", Render(Defaults(), msg, "fa").Title)
	assert.Equal(t, "Ritme reminder", Render(Defaults(), msg, "de").Title, "unknown language falls back to English")

	p := Defaults()
	p.NeutralCopy = false
	assert.Equal(t, Push{Title: msg.Title, Body: msg.Body, URL: "/calendar"}, Render(p, msg, "fa"))
}

func TestFromRow_StoredChoicesAndBadValues(t *testing.T) {
	row := store.NotificationPreference{
		Categories:        db.NullRawJSON{V: json.RawMessage(`{"fertile_window":true,"pms":false,"bogus":true}`), Valid: true},
		QuietHoursEnabled: false,
		QuietStart:        "22:30:00",
		QuietEnd:          "nonsense",
		NeutralCopy:       false,
	}
	p := FromRow(row)
	assert.True(t, p.Enabled(FertileWindow))
	assert.False(t, p.Enabled(PMS))
	assert.True(t, p.Enabled(BeforePeriod), "missing key = default")
	assert.NotContains(t, p.Categories, Category("bogus"))
	assert.Equal(t, 22*60+30, p.QuietStart)
	assert.Equal(t, DefaultQuietEnd, p.QuietEnd)
	assert.False(t, p.QuietEnabled)
	assert.False(t, p.NeutralCopy)
	assert.JSONEq(t, `{"fertile_window":true,"pms":false}`, string(p.CategoriesJSON()))
}

func body(t *testing.T, raw string) phpval.Map {
	t.Helper()
	v, err := phpval.Decode([]byte(raw))
	require.NoError(t, err)
	m, ok := v.(phpval.Map)
	require.True(t, ok)
	return m
}

func TestApply_PartialUpdate(t *testing.T) {
	p, err := Apply(Defaults(), body(t, `{"categories":{"vitals":true,"pms":false,"x":true},"quiet_hours":{"start":"22:00"},"neutral_copy":false}`), "en", at(9, 0))
	require.NoError(t, err)
	assert.True(t, p.Enabled(Vitals))
	assert.False(t, p.Enabled(PMS))
	assert.Equal(t, 22*60, p.QuietStart)
	assert.Equal(t, DefaultQuietEnd, p.QuietEnd, "omitted keys keep their value")
	assert.True(t, p.QuietEnabled)
	assert.False(t, p.NeutralCopy)
	assert.Empty(t, Defaults().Categories, "Apply must not mutate its input")
}

func TestApply_Validation(t *testing.T) {
	_, err := Apply(Defaults(), body(t, `{"categories":{"pms":"maybe"},"quiet_hours":{"start":"25:00","enabled":"x"},"neutral_copy":"yes"}`), "en", at(9, 0))
	require.Error(t, err)
	var fail *httpx.FailError
	if assert.ErrorAs(t, err, &fail) {
		assert.Equal(t, 422, fail.Status)
	}
}

func TestLangFiles_SameKeys(t *testing.T) {
	for _, key := range []string{"messages.validation_failed", "messages.saved", "push.neutral_title", "push.neutral_body"} {
		for _, loc := range []string{"fa", "en"} {
			assert.NotEqual(t, "notifications."+key, T(key, loc), "%s missing in %s", key, loc)
		}
	}
	assert.Len(t, attributes("fa"), len(attributes("en")))
}
