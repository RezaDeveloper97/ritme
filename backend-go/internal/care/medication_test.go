package care

import (
	"database/sql"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func ptr(s string) *string { return &s }

func TestSaturdayWeekday(t *testing.T) {
	// 2026-09-19 is a Saturday.
	for i, want := range []int{0, 1, 2, 3, 4, 5, 6, 0} {
		assert.Equal(t, want, SaturdayWeekday(civildate.MustParse("2026-09-19").AddDays(i)))
	}
}

func TestMeta_LegacyColumns(t *testing.T) {
	m := MedicationMeta{V: 1, Dose: ptr("400"), Unit: ptr("mcg"), Times: []string{"08:00", "20:00"}, Weekdays: AllWeekdays}
	assert.Equal(t, "daily", m.Recurrence())
	assert.Equal(t, sql.NullString{String: "08:00:00", Valid: true}, m.RecurrenceTime())
	assert.Equal(t, "۴۰۰ میکروگرم", m.Subtitle("fa").String)
	assert.Equal(t, "400 mcg", m.Subtitle("en").String)
	assert.Equal(t, "400 mcg", m.Subtitle("de").String, "unknown language falls back to English")

	m.Weekdays = []int{0, 3}
	m.Unit = ptr("قاشق")
	assert.Equal(t, "weekly", m.Recurrence())
	assert.Equal(t, "۴۰۰ قاشق", m.Subtitle("fa").String, "free-text unit as typed")

	m.Dose, m.Unit = nil, nil
	assert.False(t, m.Subtitle("fa").Valid)
	m.Times = nil
	assert.False(t, m.RecurrenceTime().Valid)
}

func TestParseMedication(t *testing.T) {
	row := store.Reminder{
		ID: 1, Type: TypeMedication, Title: "x",
		Meta: rootdb.NullRawJSON{V: []byte(`{"v":1,"dose":"1","unit":"mg","form":"syrup","times":["09:00"],"weekdays":[1,5],"amount":2,"duration":"until_date","notify":false}`), Valid: true},
	}
	m := ParseMedication(row).Meta
	assert.Equal(t, "syrup", m.Form)
	assert.Equal(t, []int{1, 5}, m.Weekdays)
	assert.Equal(t, 2, m.Amount)
	assert.False(t, m.Notify)

	// A row made by the legacy POST /reminders (no meta).
	legacy := store.Reminder{
		Type: TypeMedication, Title: "x",
		RecurrenceTime: sql.NullString{String: "16:00:00", Valid: true},
		EndsOn:         civildate.NullDate{Date: civildate.MustParse("2026-10-01"), Valid: true},
	}
	m = ParseMedication(legacy).Meta
	assert.Equal(t, []string{"16:00"}, m.Times)
	assert.Equal(t, AllWeekdays, m.Weekdays)
	assert.Equal(t, DurationUntilDate, m.Duration)
	assert.Equal(t, 1, m.Amount)
	assert.Nil(t, m.Dose)
}

func TestCovers(t *testing.T) {
	m := Medication{
		Row: store.Reminder{
			StartsOn: civildate.NullDate{Date: civildate.MustParse("2026-09-12"), Valid: true},
			EndsOn:   civildate.NullDate{Date: civildate.MustParse("2026-09-21"), Valid: true},
		},
		Meta: MedicationMeta{Weekdays: []int{0, 2}}, // Saturday, Monday
	}
	for d, want := range map[string]bool{
		"2026-09-05": false, // Saturday before start
		"2026-09-12": true,  // Saturday, start
		"2026-09-13": false, // Sunday
		"2026-09-14": true,  // Monday
		"2026-09-21": true,  // Monday, end (inclusive)
		"2026-09-26": false, // Saturday after end
	} {
		assert.Equal(t, want, m.Covers(civildate.MustParse(d)), d)
	}
	m.Row.EndsOn = civildate.NullDate{}
	assert.True(t, m.Covers(civildate.MustParse("2027-01-02")), "open-ended, Saturday")
}

func TestValidateMedication_Defaults(t *testing.T) {
	data := phpval.NewMap()
	data.Set("title", "x")
	data.Set("form", "tablet")
	data.Set("times", []any{"20:00", "08:00"})
	data.Set("starts_on", "2026-09-23")
	data.Set("ends_on", "2026-10-01") // ignored: duration is ongoing
	in, err := validateMedication("en", data, time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran))
	require.NoError(t, err)
	assert.Equal(t, []string{"08:00", "20:00"}, in.Meta.Times)
	assert.Equal(t, AllWeekdays, in.Meta.Weekdays)
	assert.Equal(t, 1, in.Meta.Amount)
	assert.Equal(t, DurationOngoing, in.Meta.Duration)
	assert.True(t, in.Meta.Notify)
	assert.True(t, in.IsActive)
	assert.Nil(t, in.EndsOn)
}

// Every enum value has a label and every message a line in every shipped language.
func TestLabels_Complete(t *testing.T) {
	locales, err := fs.Glob(langFS, "lang/*")
	require.NoError(t, err)
	require.NotEmpty(t, locales)
	groups := map[string][]string{
		"forms": Forms, "units": Units, "durations": Durations, "kinds": AppointmentKinds,
		"topics": AppointmentTopics, "remind_before": RemindBefore,
	}
	for _, dir := range locales {
		locale := dir[len("lang/"):]
		for group, values := range groups {
			for _, v := range values {
				_, ok := Label(group, v, locale)
				assert.True(t, ok, "%s: %s.%s", locale, group, v)
			}
		}
		for _, k := range []string{"messages.validation_failed", "messages.medication_not_found",
			"validation.ends_on_required", "validation.slot_unknown", "validation.date_not_scheduled"} {
			line, ok := translator().Get("care."+k, locale)
			assert.True(t, ok && line != nil, "%s: %s", locale, k)
		}
		assert.Len(t, []rune(T("digits", locale)), 10, locale)
	}
}

func TestIntervalSchedule(t *testing.T) {
	after := "after"
	doses := []DoseSpec{{Time: "08:00", Amount: 1, Unit: "pill", Meal: &after}, {Time: "21:30", Amount: 5, Unit: "drop"}}

	assert.Equal(t, []string{"08:00", "21:30"}, TimesOf(Frequency{Every: 2, Unit: FrequencyDay}, doses))
	assert.Equal(t, []string{"00:00", "08:00", "16:00"}, TimesOf(Frequency{Every: 8, Unit: FrequencyHour}, doses[:1]))
	assert.Equal(t, []string{"01:30", "05:30", "09:30", "13:30", "17:30", "21:30"},
		TimesOf(Frequency{Every: 4, Unit: FrequencyHour}, doses[1:]))
	assert.Len(t, TimesOf(Frequency{Every: 1, Unit: FrequencyHour}, doses[:1]), 24)

	start := civildate.MustParse("2026-11-10")
	med := func(f Frequency, d []DoseSpec) Medication {
		return Medication{
			Row:  store.Reminder{StartsOn: civildate.NullDate{Date: start, Valid: true}},
			Meta: MedicationMeta{V: 1, Frequency: &f, Doses: d, Times: TimesOf(f, d), Weekdays: AllWeekdays},
		}
	}

	everyThird := med(Frequency{Every: 3, Unit: FrequencyDay}, doses)
	assert.False(t, everyThird.Covers(start.AddDays(-1)), "before the start")
	assert.True(t, everyThird.Covers(start))
	assert.False(t, everyThird.Covers(start.AddDays(1)))
	assert.False(t, everyThird.Covers(start.AddDays(2)))
	assert.True(t, everyThird.Covers(start.AddDays(3)))
	assert.Equal(t, []string{"08:00", "21:30"}, everyThird.SlotsOn(start.AddDays(3)))
	assert.Empty(t, everyThird.SlotsOn(start.AddDays(4)))
	assert.Equal(t, "drop", everyThird.Meta.DoseAt("21:30").Unit)
	assert.Nil(t, everyThird.Meta.DoseAt("09:00"))

	hourly := med(Frequency{Every: 8, Unit: FrequencyHour}, doses[:1])
	assert.Equal(t, []string{"08:00", "16:00"}, hourly.SlotsOn(start), "the chain starts at the first dose")
	assert.Equal(t, []string{"00:00", "08:00", "16:00"}, hourly.SlotsOn(start.AddDays(1)))
	assert.Equal(t, 1, hourly.Meta.DoseAt("00:00").Amount)
	assert.Nil(t, hourly.Meta.DoseAt("00:30"))

	assert.Equal(t, "۱ قرص · بعد از غذا", DoseLine(doses[0], "fa"))
	assert.Equal(t, "5 drop", DoseLine(doses[1], "en"))
	assert.Equal(t, "۱ قرص", everyThird.Meta.Subtitle("fa").String, "no strength: the first dose")
	assert.Equal(t, "drops", FormOfDoseUnit("drop"))
}

func TestValidateMedication_Interval(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, civildate.Tehran)
	body := func(freq, doses string) phpval.Map {
		raw, err := phpval.Decode([]byte(`{"title":"آهن","starts_on":"2026-11-10","times":["03:00"],"frequency":` + freq + `,"doses":` + doses + `}`))
		require.NoError(t, err)
		m, ok := raw.(phpval.Map)
		require.True(t, ok)
		return pickMedication(phpval.NewMap(), m)
	}
	in, err := validateMedication("fa", body(`{"every":2,"unit":"hour"}`, `[{"time":"20:00","amount":2,"unit":"drop","meal":"before"}]`), now)
	require.NoError(t, err)
	assert.Equal(t, &Frequency{Every: 2, Unit: "hour"}, in.Meta.Frequency)
	assert.Len(t, in.Meta.Times, 12, "times come from the doses, the sent ones are ignored")
	assert.Equal(t, "drops", in.Meta.Form)
	assert.Equal(t, 2, in.Meta.Amount)
	assert.Equal(t, "daily", in.Meta.Recurrence())

	in, err = validateMedication("fa", body(`null`, `[{"time":"20:00","amount":1,"unit":"pill"},{"time":"08:00","amount":1,"unit":"pill","meal":null}]`), now)
	require.NoError(t, err)
	assert.Equal(t, &Frequency{Every: 1, Unit: "day"}, in.Meta.Frequency, "every day by default")
	assert.Equal(t, []string{"08:00", "20:00"}, in.Meta.Times)
	assert.Equal(t, "08:00", in.Meta.Doses[0].Time, "stored sorted")

	for name, tc := range map[string][2]string{
		"hour interval must divide the day": {`{"every":5,"unit":"hour"}`, `[{"time":"20:00","amount":1,"unit":"pill"}]`},
		"day interval too long":             {`{"every":31,"unit":"day"}`, `[{"time":"20:00","amount":1,"unit":"pill"}]`},
		"unknown frequency unit":            {`{"every":1,"unit":"week"}`, `[{"time":"20:00","amount":1,"unit":"pill"}]`},
		"duplicate times":                   {`null`, `[{"time":"20:00","amount":1,"unit":"pill"},{"time":"20:00","amount":2,"unit":"pill"}]`},
		"unknown dose unit":                 {`null`, `[{"time":"20:00","amount":1,"unit":"bucket"}]`},
		"amount out of range":               {`null`, `[{"time":"20:00","amount":0,"unit":"pill"}]`},
		"unknown meal":                      {`null`, `[{"time":"20:00","amount":1,"unit":"pill","meal":"during"}]`},
		"bad time":                          {`null`, `[{"time":"8pm","amount":1,"unit":"pill"}]`},
		"empty doses":                       {`null`, `[]`},
	} {
		_, err := validateMedication("fa", body(tc[0], tc[1]), now)
		assert.Error(t, err, name)
	}
}

func TestOccurrences(t *testing.T) {
	start := civildate.MustParse("2026-11-10")
	row := func(id uint64, active bool, meta string) store.Reminder {
		return store.Reminder{
			ID: id, UserID: 7, Type: TypeMedication, Title: "آهن", IsActive: active,
			StartsOn: civildate.NullDate{Date: start, Valid: true},
			Meta:     rootdb.NullRawJSON{V: []byte(meta), Valid: true},
		}
	}
	const daily = `{"v":1,"form":"tablet","times":["08:00","20:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true,` +
		`"frequency":{"every":2,"unit":"day"},"doses":[{"time":"08:00","amount":1,"unit":"pill","meal":"after"},{"time":"20:00","amount":2,"unit":"pill","meal":null}]}`
	rows := []store.Reminder{
		row(1, true, daily),
		row(2, false, daily), // switched off
		row(3, true, `{"v":1,"form":"tablet","times":["09:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":false}`), // no notification
	}
	from := time.Date(2026, 11, 10, 8, 0, 0, 0, civildate.Tehran)
	occ := Occurrences(rows, from, from.Add(72*time.Hour))
	require.Len(t, occ, 4, "every second day: the 10th and the 12th, two doses each; the window end is exclusive")
	assert.Equal(t, "1:2026-11-10:08:00", occ[0].ID())
	assert.Equal(t, from, occ[0].At)
	assert.Equal(t, "1:2026-11-12:20:00", occ[3].ID())

	msg := occ[0].Message("fa")
	assert.Equal(t, "وقت داروی توست", msg.Title)
	assert.Equal(t, "آهن · ۱ قرص · بعد از غذا", msg.Body)
	assert.Equal(t, "/fa/reminders", msg.URL)
	assert.Equal(t, "آهن · ۲ قرص", occ[1].Message("fa").Body)

	assert.Empty(t, Occurrences(rows, from.Add(time.Second), from.Add(time.Hour)))
}

func TestValidPushEndpoint(t *testing.T) {
	for _, ok := range []string{
		"https://fcm.googleapis.com/fcm/send/abc",
		"https://updates.push.services.mozilla.com/wpush/v2/x",
		"https://db5p.notify.windows.com/w/?token=1",
		"https://web.push.apple.com/abc",
	} {
		assert.True(t, ValidPushEndpoint(ok), ok)
	}
	for _, bad := range []string{
		"http://fcm.googleapis.com/fcm/send/abc", "https://127.0.0.1/x", "https://fcm.googleapis.com.evil.io/x",
		"https://evilpush.apple.com/x", "https://user@fcm.googleapis.com/x", "https://fcm.googleapis.com:8443/x",
		"https://localhost/x", "", "fcm.googleapis.com/x",
	} {
		assert.False(t, ValidPushEndpoint(bad), bad)
	}
	assert.True(t, validPushKey("BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM", 65))
	assert.False(t, validPushKey("short", 65))
}

func TestAppointmentsDue(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
		require.NoError(t, err)
		return v
	}
	row := func(id uint64, visit, meta string, active bool) store.Reminder {
		return store.Reminder{
			ID: id, UserID: 7, Type: TypeAppointment, Title: "دکتر احمدی", IsActive: active,
			ScheduledAt: sql.NullTime{Time: at(visit), Valid: true},
			Meta:        rootdb.NullRawJSON{V: []byte(meta), Valid: true},
		}
	}
	rows := []store.Reminder{
		row(1, "2026-11-12 10:30", `{"v":1,"remind_before":"1d","status":"scheduled"}`, true),
		row(2, "2026-11-11 11:00", `{"v":1,"remind_before":"1h","status":"scheduled"}`, true),
		row(3, "2026-11-12 10:30", `{"v":1,"remind_before":"1d","status":"cancelled"}`, true),
		row(4, "2026-11-12 10:30", `{"v":1,"remind_before":"1d","status":"scheduled"}`, false),
		row(5, "2026-11-13 10:25", `{"v":1,"remind_before":"2d","status":"scheduled"}`, true),
	}
	due := AppointmentsDue(rows, at("2026-11-11 10:20"), at("2026-11-11 10:31"))
	require.Len(t, due, 2)
	assert.Equal(t, uint64(5), due[0].A.Row.ID)
	assert.Equal(t, "2 روز دیگر ساعت 10:25", due[0].When("fa"))
	assert.Equal(t, uint64(1), due[1].A.Row.ID)
	assert.Equal(t, "2026-11-12 10:30", due[1].Key())
	assert.Equal(t, "فردا ساعت 10:30", due[1].When("fa"))

	due = AppointmentsDue(rows, at("2026-11-11 09:55"), at("2026-11-11 10:01"))
	require.Len(t, due, 1)
	assert.Equal(t, "امروز ساعت 11:00", due[0].When("fa"))
	assert.Equal(t, "دکتر احمدی · امروز ساعت 11:00", due[0].Message("fa").Body)
}
