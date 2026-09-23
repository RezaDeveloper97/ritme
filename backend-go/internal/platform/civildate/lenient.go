package civildate

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrLenientFormat is returned by ParseLenient for input Carbon::parse would reject
// (or that is outside the supported subset, see ParseLenient).
var ErrLenientFormat = errors.New("civildate: unrecognised date/time")

var (
	reYMD      = regexp.MustCompile(`^(\d{4})-(\d{1,2})(?:-(\d{1,2}))?`)
	reYMDSlash = regexp.MustCompile(`^(\d{4})/(\d{1,2})/(\d{1,2})`)
	reCompact  = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)
	reDMY      = regexp.MustCompile(`^(\d{1,2})-(\d{1,2})-(\d{4})`)
	reMDY      = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{4})`)
	reTime     = regexp.MustCompile(`^[Tt ](\d{1,2}):(\d{2})(?::(\d{2})(?:[.,](\d{1,9}))?)?`)
	reZone     = regexp.MustCompile(`^ ?(?:([Zz])|([+-])(\d{2}):?(\d{2}))$`)
)

// ParseLenient is the subset of Carbon::parse($s) (PHP strtotime) that can reach an
// API parameter such as /cycle/date/{date}, evaluated with `now` as the current
// instant and loc as the default timezone (Asia/Tehran in the app).
//
// Supported, with Carbon's results (see lenient_test.go for the PHP transcript):
//   - "" and "now" → now; "today"/"midnight", "tomorrow", "yesterday" → that day 00:00
//   - Y-m-d, Y-m, Y/m/d, Ymd, d-m-Y, m/d/Y (1–2 digit month/day)
//   - month 0 and day 0 roll back, day overflow rolls forward (2026-02-30 → 2026-03-02);
//     month > 12 or day > 31 is an error
//   - optional time " H:i[:s[.u]]" or "TH:i[:s]", hour ≤ 23
//   - optional zone "Z" or "±hh:mm" / "±hhmm" (the result keeps that fixed offset)
//
// Anything else (relative phrases like "+1 day", military zone letters, weekday
// names …) returns ErrLenientFormat; Laravel would accept some of them, the API
// clients only ever send Y-m-d.
func ParseLenient(s string, now time.Time, loc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	now = now.In(loc)
	switch strings.ToLower(s) {
	case "", "now":
		return now, nil
	case "today", "midnight":
		return FromTime(now).Midnight(loc), nil
	case "tomorrow":
		return FromTime(now).AddDays(1).Midnight(loc), nil
	case "yesterday":
		return FromTime(now).AddDays(-1).Midnight(loc), nil
	}

	y, m, d, rest, ok := parseDatePart(s)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: %q", ErrLenientFormat, s)
	}
	if m < 0 || m > 12 || d < 0 || d > 31 {
		return time.Time{}, fmt.Errorf("%w: %q", ErrLenientFormat, s)
	}

	var hh, mm, ss, nsec int
	if t := reTime.FindStringSubmatch(rest); t != nil {
		hh, mm = atoi(t[1]), atoi(t[2])
		if t[3] != "" {
			ss = atoi(t[3])
		}
		if t[4] != "" {
			frac := (t[4] + "000000000")[:9]
			nsec = atoi(frac)
		}
		if hh > 23 || mm > 59 || ss > 59 {
			return time.Time{}, fmt.Errorf("%w: %q", ErrLenientFormat, s)
		}
		rest = rest[len(t[0]):]
	}

	zone := loc
	if rest != "" {
		z := reZone.FindStringSubmatch(rest)
		if z == nil {
			return time.Time{}, fmt.Errorf("%w: %q", ErrLenientFormat, s)
		}
		if z[1] != "" {
			zone = time.UTC
		} else {
			off := atoi(z[3])*3600 + atoi(z[4])*60
			if z[2] == "-" {
				off = -off
			}
			zone = time.FixedZone("", off)
		}
	}
	return time.Date(y, time.Month(m), d, hh, mm, ss, nsec, zone), nil
}

func parseDatePart(s string) (y, m, d int, rest string, ok bool) {
	if c := reCompact.FindStringSubmatch(s); c != nil {
		return atoi(c[1]), atoi(c[2]), atoi(c[3]), "", true
	}
	if c := reYMD.FindStringSubmatch(s); c != nil {
		d = 1
		if c[3] != "" {
			d = atoi(c[3])
		}
		return atoi(c[1]), atoi(c[2]), d, s[len(c[0]):], true
	}
	if c := reYMDSlash.FindStringSubmatch(s); c != nil {
		return atoi(c[1]), atoi(c[2]), atoi(c[3]), s[len(c[0]):], true
	}
	if c := reDMY.FindStringSubmatch(s); c != nil {
		return atoi(c[3]), atoi(c[2]), atoi(c[1]), s[len(c[0]):], true
	}
	if c := reMDY.FindStringSubmatch(s); c != nil {
		return atoi(c[3]), atoi(c[1]), atoi(c[2]), s[len(c[0]):], true
	}
	return 0, 0, 0, "", false
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s) // only called on \d+ captures
	return n
}
