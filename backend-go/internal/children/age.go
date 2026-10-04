package children

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// AddMonths is d plus n calendar months, clamped to the last day of the target month (31 Jan + 1 month = 28/29 Feb).
func AddMonths(d civildate.Date, n int) civildate.Date {
	y, m := d.Year, int(d.Month)-1+n
	y += m / 12
	m %= 12
	if m < 0 {
		m += 12
		y--
	}
	month := time.Month(m + 1)
	last := time.Date(y, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := min(d.Day, last)
	return civildate.New(y, month, day)
}

// Age is a child's age on a day.
type Age struct {
	Days   int // since birth (0 on the birth day)
	Weeks  int // completed weeks
	Months int // completed calendar months
	Years  int // completed years
	// DaysInMonth are the days after the last completed month («۳ ماه و ۱۲ روز»).
	DaysInMonth int
}

// AgeOn is the age of a child born on birth, on day on (zero before the birth).
func AgeOn(birth, on civildate.Date) Age {
	if on.Before(birth) {
		return Age{}
	}
	months := (on.Year-birth.Year)*12 + int(on.Month) - int(birth.Month)
	if AddMonths(birth, months).After(on) {
		months--
	}
	days := birth.DiffDays(on)
	return Age{
		Days: days, Weeks: days / 7, Months: months, Years: months / 12,
		DaysInMonth: AddMonths(birth, months).DiffDays(on),
	}
}

// Label is the screen's age text: «امروز به دنیا آمده», «۱۲ روز», «۳ ماه و ۱۲ روز», «۴ سال و ۲ ماه».
func (a Age) Label(locale string) string {
	switch {
	case a.Days == 0:
		return T("age.newborn", locale)
	case a.Months == 0:
		return Tp("age.days", map[string]string{"d": num(a.Days, locale)}, locale)
	case a.Years == 0 && a.DaysInMonth == 0:
		return Tp("age.months", map[string]string{"m": num(a.Months, locale)}, locale)
	case a.Years == 0:
		return Tp("age.months_days", map[string]string{"m": num(a.Months, locale), "d": num(a.DaysInMonth, locale)}, locale)
	case a.Months%12 == 0:
		return Tp("age.years", map[string]string{"y": num(a.Years, locale)}, locale)
	}
	return Tp("age.years_months", map[string]string{"y": num(a.Years, locale), "m": num(a.Months%12, locale)}, locale)
}
