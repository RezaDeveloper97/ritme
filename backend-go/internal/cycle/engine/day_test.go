package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestDay_RoundTripAgainstTime_1900To2100(t *testing.T) {
	end := time.Date(2101, 1, 1, 0, 0, 0, 0, time.UTC)
	prev := NoDay
	n := 0
	for tm := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC); tm.Before(end); tm = tm.AddDate(0, 0, 1) {
		c := civildate.FromTime(tm)
		d := DayOf(c)
		require.Equal(t, tm.Unix()/86400, int64(d), tm.Format(time.DateOnly))
		require.Equal(t, c, d.Civil(), tm.Format(time.DateOnly))
		require.Equal(t, tm.Format(time.DateOnly), d.String())
		if prev != NoDay {
			require.Equal(t, prev+1, d)
		}
		prev = d
		n++
	}
	assert.Equal(t, 73414, n, "every day of 1900-01-01 … 2100-12-31")
}

func TestDay_Validity(t *testing.T) {
	d, ok := MakeDay(2024, time.February, 29)
	assert.True(t, ok)
	assert.Equal(t, "2024-02-29", d.String())
	_, ok = MakeDay(2100, time.February, 29)
	assert.False(t, ok, "2100 is not a leap year")
	_, ok = MakeDay(2000, time.February, 29)
	assert.True(t, ok, "2000 is a leap year")
	for _, bad := range [][3]int{{2026, 13, 1}, {2026, 0, 1}, {2026, 4, 31}, {2026, 1, 0}, {-1, 1, 1}, {10000, 1, 1}} {
		_, ok := MakeDay(bad[0], time.Month(bad[1]), bad[2])
		assert.False(t, ok, "%v", bad)
	}
	assert.Equal(t, NoDay, DayOf(civildate.Date{}), "zero date")
	assert.Equal(t, NoDay, DayOf(civildate.Date{Year: 2100, Month: time.February, Day: 29}), "invalid date")
	assert.Equal(t, Day(0), DayOf(civildate.MustParse("1970-01-01")))
	assert.Equal(t, Day(-1), DayOf(civildate.MustParse("1969-12-31")))
}

func TestDay_ISO(t *testing.T) {
	d := DayOf(civildate.MustParse("2026-07-01"))
	assert.Equal(t, `"2026-07-01"`, string(append(d.AppendISO([]byte(`"`)), '"')))
	assert.Equal(t, "2026-07-10", (d + 9).String(), "§2.4 difference_in_days(2026-07-10, 2026-07-01) = 9")
	assert.Equal(t, "0001-01-01", DayOf(civildate.Date{Year: 1, Month: time.January, Day: 1}).String())
	assert.Empty(t, NoDay.String())
	assert.Empty(t, NoDay.AppendISO(nil))
	assert.Equal(t, civildate.Date{}, NoDay.Civil())
	assert.False(t, NoDay.IsSet())
	assert.True(t, d.IsSet())
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		var buf [16]byte
		_ = d.AppendISO(buf[:0])
	}))
}

func TestOpt(t *testing.T) {
	var none Opt[int]
	v, ok := none.Get()
	assert.False(t, ok)
	assert.Zero(t, v)
	assert.Equal(t, 7, none.Or(7))
	assert.Equal(t, None[int](), none)
	some := Some(3)
	assert.True(t, some.OK())
	assert.Equal(t, 3, some.Or(7))
}
