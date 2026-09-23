package checkups

import (
	"embed"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The checkups copy (interval / timing / next-due labels, the calendar and its month names,
// controller messages, validation attribute names) is data: lang/<code>/checkups.json, one file
// per language, read through the platform translator. A language without the file (or a key)
// falls back to English (lang.FallbackLocale). `calendar` is "jalali" or "gregorian".
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the checkups line for key ("messages.record_created") in locale.
func T(key, locale string) string { return translator().Trans("checkups."+key, nil, locale) }

// tr is T with :placeholders; numeric params are written in the locale's digits.
func tr(key, locale string, kv ...string) string {
	params := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		params[kv[i]] = kv[i+1]
	}
	return translator().Trans("checkups."+key, params, locale)
}

// num is n in the locale's digits.
func num(n int, locale string) string { return LocalizeDigits(strconv.Itoa(n), locale) }

// LocalizeDigits writes the ASCII digits of s in the locale's digit set ("digits" line).
func LocalizeDigits(s, locale string) string {
	digits := []rune(T("digits", locale))
	if len(digits) != 10 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(digits[r-'0'])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("checkups.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// calendarDate is d in the locale's calendar: year, month (1-based), day.
func calendarDate(d civildate.Date, locale string) (y, m, day int) {
	if T("calendar", locale) == "jalali" {
		return engine.ToJalali(d)
	}
	return d.Year, int(d.Month), d.Day
}

// YearStart is the first day of the locale-calendar year that contains d (Farvardin 1 for a
// Jalali locale, January 1 otherwise) — the History «امسال» tab.
func YearStart(d civildate.Date, locale string) civildate.Date {
	if T("calendar", locale) == "jalali" {
		jy, _, _ := engine.ToJalali(d)
		return engine.FromJalali(jy, 1, 1)
	}
	return civildate.New(d.Year, 1, 1)
}

// monthName is the locale's name of calendar month m (1-based).
func monthName(m int, locale string) string {
	line, ok := translator().Get("checkups.months", locale)
	if !ok {
		return num(m, locale)
	}
	_, vals := phpval.Entries(line)
	if m < 1 || m > len(vals) {
		return num(m, locale)
	}
	return phpval.ToString(vals[m-1])
}

// FullDate is «۱۵ مهر ۱۴۰۵» / "October 15, 2026".
func FullDate(d civildate.Date, locale string) string {
	y, m, day := calendarDate(d, locale)
	return tr("date.full", locale, "day", num(day, locale), "month", monthName(m, locale), "year", num(y, locale))
}

// MonthYear is «مهر ۱۴۰۵» / "October 2026".
func MonthYear(d civildate.Date, locale string) string {
	y, m, _ := calendarDate(d, locale)
	return tr("date.month_year", locale, "month", monthName(m, locale), "year", num(y, locale))
}

// IntervalLabel is «هر ماه» / «هر ۶ ماه» / «هر ۳ سال» / «هر ۱ تا ۲ سال».
func IntervalLabel(minMonths, maxMonths int, locale string) string {
	if maxMonths > minMonths {
		if minMonths%12 == 0 && maxMonths%12 == 0 {
			return tr("interval.range_years", locale, "from", num(minMonths/12, locale), "to", num(maxMonths/12, locale))
		}
		return tr("interval.range_months", locale, "from", num(minMonths, locale), "to", num(maxMonths, locale))
	}
	switch {
	case minMonths <= 1:
		return T("interval.monthly", locale)
	case minMonths == 12:
		return T("interval.yearly", locale)
	case minMonths%12 == 0:
		return tr("interval.years", locale, "n", num(minMonths/12, locale))
	default:
		return tr("interval.months", locale, "n", num(minMonths, locale))
	}
}

// TimingLabel is «روز ۷ تا ۱۰ سیکل» for a cycle-timed type, else "".
func TimingLabel(t engine.Type, locale string) string {
	if !t.CycleTimed() {
		return ""
	}
	return tr("timing.cycle_days", locale, "from", num(t.CycleDayFrom, locale), "to", num(t.CycleDayTo, locale))
}

// soonLabelDays: a next due date at most this many days away is written as «N روز دیگر».
const soonLabelDays = 30

// NextDueLabel is the item's due line: «۳ روز دیگر», «امروز», «زمانش رسیده»,
// «عقب‌افتاده از فروردین», «از ۱۴۰۹ (۴۰ سالگی)», «مهر ۱۴۰۶», «غیرفعال».
func NextDueLabel(it engine.Item, t engine.Type, today civildate.Date, locale string) string {
	switch it.Status {
	case engine.StatusDisabled:
		return T("due.disabled", locale)
	case engine.StatusNotYet:
		y, _, _ := calendarDate(it.NextDueOn, locale)
		return tr("due.not_yet", locale, "year", num(y, locale), "age", num(t.AgeMin, locale))
	case engine.StatusOverdue:
		since := monthName(monthOf(it.NextDueOn, locale), locale)
		if ty, _, _ := calendarDate(today, locale); yearOf(it.NextDueOn, locale) != ty {
			since = MonthYear(it.NextDueOn, locale)
		}
		return tr("due.overdue_since", locale, "month", since)
	}
	if it.NextDueOn.IsZero() {
		return T("due.now", locale)
	}
	return RelativeDue(it.NextDueOn, today, locale)
}

// RelativeDue is «امروز» / «فردا» / «N روز دیگر» up to a month ahead, then «مهر ۱۴۰۶»;
// a day already passed is «زمانش رسیده».
func RelativeDue(d, today civildate.Date, locale string) string {
	days := today.DiffDays(d)
	switch {
	case days < 0:
		return T("due.now", locale)
	case days == 0:
		return T("due.today", locale)
	case days == 1:
		return T("due.tomorrow", locale)
	case days <= soonLabelDays:
		return tr("due.in_days", locale, "n", num(days, locale))
	default:
		return MonthYear(d, locale)
	}
}

// ReminderLabel is the MarkDone banner's reminder line («۳ روز قبل یادآوری می‌کنیم»).
func ReminderLabel(leadDays int, remind bool, locale string) string {
	switch {
	case !remind:
		return T("reminder.off", locale)
	case leadDays <= 0:
		return T("reminder.same_day", locale)
	default:
		return tr("reminder.days_before", locale, "n", num(leadDays, locale))
	}
}

func monthOf(d civildate.Date, locale string) int {
	_, m, _ := calendarDate(d, locale)
	return m
}

func yearOf(d civildate.Date, locale string) int {
	y, _, _ := calendarDate(d, locale)
	return y
}
