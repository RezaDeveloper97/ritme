package engine

import (
	"math"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Day is a calendar day as the number of days since 1970-01-01 (proleptic Gregorian). Differences are plain integer
// subtraction (§2.4 difference_in_days(a, b) = a − b).
type Day int32

// NoDay is "no date" (an open period's end, an unknown anchor).
const NoDay Day = math.MinInt32

// IsSet reports whether d is a date (not NoDay).
func (d Day) IsSet() bool { return d != NoDay }

// MakeDay builds the Day of year-month-day; ok is false for an invalid date (2100-02-29, month 13, day 0). Years
// outside 0..9999 are rejected.
func MakeDay(year int, month time.Month, day int) (Day, bool) {
	if year < 0 || year > 9999 || month < time.January || month > time.December || day < 1 ||
		day > daysIn(year, month) {
		return NoDay, false
	}
	return Day(daysFromCivil(year, int(month), day)), true //nolint:gosec // years 0..9999: |days| < 2^22
}

// DayOf is the Day of a civil date; the zero date and invalid dates answer NoDay.
func DayOf(c civildate.Date) Day {
	if c.IsZero() {
		return NoDay
	}
	d, _ := MakeDay(c.Year, c.Month, c.Day)
	return d
}

// Civil is the civil date of d (the zero date for NoDay).
func (d Day) Civil() civildate.Date {
	if d == NoDay {
		return civildate.Date{}
	}
	y, m, dd := civilFromDays(int64(d))
	return civildate.Date{Year: y, Month: time.Month(m), Day: dd}
}

// AppendISO appends d as YYYY-MM-DD (nothing for NoDay or a day outside the years 0..9999).
func (d Day) AppendISO(b []byte) []byte {
	if d == NoDay {
		return b
	}
	y, m, dd := civilFromDays(int64(d))
	if y < 0 || y > 9999 {
		return b
	}
	const digits = "0123456789"
	return append(b,
		digits[y/1000], digits[y/100%10], digits[y/10%10], digits[y%10], '-',
		digits[m/10], digits[m%10], '-',
		digits[dd/10], digits[dd%10])
}

// String is the ISO date ("" for NoDay).
func (d Day) String() string {
	var buf [10]byte
	return string(d.AppendISO(buf[:0]))
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

func daysIn(y int, m time.Month) int {
	switch m {
	case time.February:
		if isLeap(y) {
			return 29
		}
		return 28
	case time.April, time.June, time.September, time.November:
		return 30
	default:
		return 31
	}
}

// daysFromCivil and civilFromDays are H. Hinnant's integer civil-calendar algorithms
// (http://howardhinnant.github.io/date_algorithms.html).
func daysFromCivil(y, m, d int) int64 {
	if m <= 2 {
		y--
	}
	era := y
	if era < 0 {
		era -= 399
	}
	era /= 400
	yoe := y - era*400
	mp := (m + 9) % 12
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return int64(era)*146097 + int64(doe) - 719468
}

func civilFromDays(z int64) (y, m, d int) {
	z += 719468
	era := z
	if era < 0 {
		era -= 146096
	}
	era /= 146097
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d = int(doy - (153*mp+2)/5 + 1)
	m = int(mp + 3)
	if mp >= 10 {
		m = int(mp - 9)
	}
	y = int(yoe + era*400)
	if m <= 2 {
		y++
	}
	return y, m, d
}
