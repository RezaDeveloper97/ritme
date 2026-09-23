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
