package engine

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Jalali (Solar Hijri) calendar conversion, a port of the jalaali-js algorithm (Borkowski's
// 2820-year-free break table, exact for Jalali years -61…3177). The engine needs it for the
// "this month" section; T-M4-02 can reuse it for fa labels (Jalali month names).

// jalaliBreaks are the jalaali-js leap-cycle break years.
var jalaliBreaks = [...]int{
	-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210,
	1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
}

// jalCal returns the Gregorian year of Farvardin 1 of Jalali year jy, the March day it falls
// on, and `leap` (years since the last leap year, 0 = jy itself is leap).
func jalCal(jy int) (gy, march, leap int) {
	gy = jy + 621
	leapJ := -14
	jp := jalaliBreaks[0]
	jump := 0
	for i := 1; i < len(jalaliBreaks); i++ {
		jm := jalaliBreaks[i]
		jump = jm - jp
		if jy < jm {
			break
		}
		leapJ += jump/33*8 + jump%33/4
		jp = jm
	}
	n := jy - jp
	leapJ += n/33*8 + (n%33+3)/4
	if jump%33 == 4 && jump-n == 4 {
		leapJ++
	}
	leapG := gy/4 - (gy/100+1)*3/4 - 150
	march = 20 + leapJ - leapG
	if jump-n < 6 {
		n = n - jump + (jump+4)/33*33
	}
	leap = ((n+1)%33 - 1) % 4
	if leap == -1 {
		leap = 4
	}
	return gy, march, leap
}

// ToJalali converts a Gregorian calendar day to Jalali year, month (1 = Farvardin) and day.
func ToJalali(d civildate.Date) (jy, jm, jd int) {
	jy = d.Year - 621
	gy, march, leap := jalCal(jy)
	k := civildate.New(gy, time.March, march).DiffDays(d)
	if k >= 0 {
		if k <= 185 {
			return jy, 1 + k/31, k%31 + 1
		}
		k -= 186
	} else {
		jy--
		k += 179
		if leap == 1 {
			k++
		}
	}
	return jy, 7 + k/30, k%30 + 1
}

// FromJalali converts a Jalali date to the Gregorian calendar day.
func FromJalali(jy, jm, jd int) civildate.Date {
	gy, march, _ := jalCal(jy)
	return civildate.New(gy, time.March, march).AddDays((jm-1)*31 - jm/7*(jm-7) + jd - 1)
}

// JalaliMonthEnd is the last day of the Jalali month that contains d.
func JalaliMonthEnd(d civildate.Date) civildate.Date {
	jy, jm, _ := ToJalali(d)
	if jm == 12 {
		return FromJalali(jy+1, 1, 1).AddDays(-1)
	}
	return FromJalali(jy, jm+1, 1).AddDays(-1)
}
