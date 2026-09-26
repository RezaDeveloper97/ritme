package v2

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
	"github.com/ritme/backend-go/internal/pregnancy/calc"
)

// The v2 UI copy the server writes (calendar + month names, trimester and week labels, size
// lines, confidence labels, controller messages, validation attribute names) is data:
// lang/<code>/pregnancy_v2.json. A language without the file (or a key) falls back to English.
// Admin-authored text (week details, tips, setup templates) comes from the database instead.
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

// T is the pregnancy_v2 line for key in locale.
func T(key, locale string) string { return translator().Trans("pregnancy_v2."+key, nil, locale) }

// tr is T with :placeholders.
func tr(key, locale string, kv ...string) string {
	params := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		params[kv[i]] = kv[i+1]
	}
	return translator().Trans("pregnancy_v2."+key, params, locale)
}

// num is n in the locale's digits.
func num(n int, locale string) string { return localizeDigits(strconv.Itoa(n), locale) }

func localizeDigits(s, locale string) string {
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

// calendarDate is d in the locale's calendar ("calendar": jalali | gregorian).
func calendarDate(d civildate.Date, locale string) (y, m, day int) {
	if T("calendar", locale) == "jalali" {
		return engine.ToJalali(d)
	}
	return d.Year, int(d.Month), d.Day
}

func monthName(m int, locale string) string {
	line, ok := translator().Get("pregnancy_v2.months", locale)
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

// DayMonth is «۱۵ مهر» / "October 15".
func DayMonth(d civildate.Date, locale string) string {
	_, m, day := calendarDate(d, locale)
	return tr("date.day_month", locale, "day", num(day, locale), "month", monthName(m, locale))
}

// RangeLabel is «۱ مهر تا ۷ مهر».
func RangeLabel(from, to civildate.Date, locale string) string {
	return tr("date.range", locale, "from", DayMonth(from, locale), "to", DayMonth(to, locale))
}

// WeekLabel is «سه‌ماههٔ اول · هفتهٔ ۸ از ۴۰».
func WeekLabel(week int, locale string) string {
	return tr("week_label", locale, "tri", T("trimesters."+strconv.Itoa(calc.Trimester(week-1)), locale),
		"week", num(week, locale), "term", num(TermWeeks, locale))
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("pregnancy_v2.attributes", locale)
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
