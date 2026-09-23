package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func TestToJalali(t *testing.T) {
	cases := []struct {
		g          string
		jy, jm, jd int
	}{
		{"2026-09-23", 1405, 7, 1},
		{"2026-09-22", 1405, 6, 31},
		{"2026-03-21", 1405, 1, 1},
		{"2026-03-20", 1404, 12, 29},
		{"2025-03-21", 1404, 1, 1},
		{"2025-03-20", 1403, 12, 30}, // 1403 is a leap year
		{"2024-03-20", 1403, 1, 1},
		{"2022-03-25", 1401, 1, 5},
		{"2032-06-15", 1411, 3, 26},
		{"1992-06-15", 1371, 3, 25},
	}
	for _, c := range cases {
		jy, jm, jd := ToJalali(d(c.g))
		assert.Equal(t, [3]int{c.jy, c.jm, c.jd}, [3]int{jy, jm, jd}, c.g)
		assert.Equal(t, d(c.g), FromJalali(c.jy, c.jm, c.jd), "back to %s", c.g)
	}
}

// Every day 1900–2100 round-trips and consecutive days advance the Jalali date by one.
func TestJalaliRoundTrip(t *testing.T) {
	day := civildate.New(1900, time.January, 1)
	end := civildate.New(2100, time.December, 31)
	py, pm, pd := ToJalali(day)
	for day = day.AddDays(1); !day.After(end); day = day.AddDays(1) {
		jy, jm, jd := ToJalali(day)
		require.Equal(t, day, FromJalali(jy, jm, jd), "round trip %s", day)
		switch {
		case jd == pd+1 && jm == pm && jy == py:
		case jd == 1 && jm == pm+1 && jy == py:
		case jd == 1 && jm == 1 && pm == 12 && jy == py+1:
		default:
			t.Fatalf("%s: %d-%d-%d does not follow %d-%d-%d", day, jy, jm, jd, py, pm, pd)
		}
		py, pm, pd = jy, jm, jd
	}
}

func TestJalaliMonthEnd(t *testing.T) {
	assert.Equal(t, d("2026-10-22"), JalaliMonthEnd(d("2026-09-23"))) // Mehr 1405: 30 days
	assert.Equal(t, d("2026-09-22"), JalaliMonthEnd(d("2026-08-23"))) // Shahrivar: 31 days
	assert.Equal(t, d("2025-03-20"), JalaliMonthEnd(d("2025-03-01"))) // Esfand 1403 (leap): 30 days
	assert.Equal(t, d("2026-03-20"), JalaliMonthEnd(d("2026-03-01"))) // Esfand 1404: 29 days
}
