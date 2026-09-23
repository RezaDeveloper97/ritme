// Package civildate is a calendar date (year, month, day — no time, no zone) anchored
// to Asia/Tehran, the Laravel app timezone.
//
// Iran observed DST until 2022, so "one day" is not always 24h there. All arithmetic
// here is civil (calendar) arithmetic done in UTC, which is what Carbon's startOfDay
// diffs produce (Carbon::parse('2021-03-20')->diffInDays('2021-03-24') === 4.0 across
// the 2021-03-22 DST jump). Never use time.Sub()/24h on Tehran midnights.
package civildate

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"time"
	_ "time/tzdata" // the runtime image may have no zoneinfo; Asia/Tehran must always load
)

// Tehran is the application timezone (config/app.php 'timezone').
var Tehran = mustLoad("Asia/Tehran")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(fmt.Sprintf("civildate: load %s: %v", name, err))
	}
	return loc
}

// Layout is the Y-m-d format.
const Layout = "2006-01-02"

// Nower is anything that tells the current instant (clock.Clock satisfies it).
type Nower interface{ Now() time.Time }

// Date is a calendar day. The zero value is "no date" (IsZero); use NullDate for
// nullable columns.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// New builds a Date, normalising overflow like PHP/Go (2026-02-30 → 2026-03-02).
func New(year int, month time.Month, day int) Date {
	return FromTime(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// FromTime returns the calendar day of t in t's own location.
func FromTime(t time.Time) Date {
	y, m, d := t.Date()
	return Date{Year: y, Month: m, Day: d}
}

// InTehran returns the Tehran calendar day of the instant t.
func InTehran(t time.Time) Date { return FromTime(t.In(Tehran)) }

// Today is the current Tehran calendar day (Carbon::today() with app tz Asia/Tehran).
func Today(c Nower) Date { return InTehran(c.Now()) }

// Parse reads a strict Y-m-d string (the `date_format:Y-m-d` / `date:Y-m-d` shape).
func Parse(s string) (Date, error) {
	t, err := time.Parse(Layout, s)
	if err != nil {
		return Date{}, fmt.Errorf("civildate: parse %q: %w", s, err)
	}
	return FromTime(t), nil
}

// MustParse is Parse for literals in tests and tables; it panics on error.
func MustParse(s string) Date {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// IsZero reports whether d is the zero Date.
func (d Date) IsZero() bool { return d == Date{} }

// String formats d as Y-m-d (Carbon toDateString()).
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.utc().Format(Layout)
}

func (d Date) utc() time.Time { return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC) }

// Midnight returns 00:00 of d in loc (Carbon::parse('Y-m-d') in that timezone).
func (d Date) Midnight(loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, loc)
}

// TehranMidnight is Midnight(Tehran): what a plain `date` cast holds before serialisation.
func (d Date) TehranMidnight() time.Time { return d.Midnight(Tehran) }

// AddDays moves d by n calendar days (n may be negative).
func (d Date) AddDays(n int) Date { return FromTime(d.utc().AddDate(0, 0, n)) }

// DiffDays returns the signed number of calendar days from d to other
// (Carbon: (int) $d->diffInDays($other)); positive when other is later.
func (d Date) DiffDays(other Date) int {
	return int((other.utc().Unix() - d.utc().Unix()) / secondsPerDay)
}

const secondsPerDay = 24 * 60 * 60

// Before reports whether d is earlier than o.
func (d Date) Before(o Date) bool { return d.Compare(o) < 0 }

// After reports whether d is later than o.
func (d Date) After(o Date) bool { return d.Compare(o) > 0 }

// Compare returns -1, 0 or +1.
func (d Date) Compare(o Date) int { return d.utc().Compare(o.utc()) }

// Weekday is Carbon dayOfWeek: Sunday=0 … Saturday=6.
func (d Date) Weekday() time.Weekday { return d.utc().Weekday() }

// ISOWeekday is Carbon dayOfWeekIso: Monday=1 … Sunday=7.
func (d Date) ISOWeekday() int {
	if w := int(d.Weekday()); w != 0 {
		return w
	}
	return 7
}

// DayOfYear is Carbon dayOfYear (1-based: 1 January = 1).
func (d Date) DayOfYear() int { return d.utc().YearDay() }

// StartOfWeek is the Saturday on or before d (Carbon startOfWeek(Carbon::SATURDAY)).
func (d Date) StartOfWeek() Date {
	back := (int(d.Weekday()) - int(time.Saturday) + 7) % 7
	return d.AddDays(-back)
}

// EndOfWeek is the Friday on or after d (the last day of a Saturday-start week).
func (d Date) EndOfWeek() Date { return d.StartOfWeek().AddDays(6) }

// AgeOn is Carbon's `->age` for a birthday d evaluated on day `on`: completed years
// (a 29 Feb birthday completes its year on 1 March in non-leap years).
func (d Date) AgeOn(on Date) int {
	age := on.Year - d.Year
	if on.Month < d.Month || (on.Month == d.Month && on.Day < d.Day) {
		age--
	}
	return age
}

// MarshalJSON writes "Y-m-d" (the `date:Y-m-d` cast); the zero Date is null.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(d.String())), nil
}

// UnmarshalJSON reads "Y-m-d" or null.
func (d *Date) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*d = Date{}
		return nil
	}
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return fmt.Errorf("civildate: %w", err)
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// Value implements driver.Valuer ("Y-m-d"; the zero Date is written as NULL).
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.String(), nil
}

// Scan implements sql.Scanner for DATE / DATETIME columns. With parseTime=true and
// loc=Asia/Tehran the driver hands a time.Time whose wall clock is the stored value.
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		*d = FromTime(v)
		return nil
	case []byte:
		return d.scanString(string(v))
	case string:
		return d.scanString(v)
	case nil:
		return errors.New("civildate: cannot scan NULL into Date (use NullDate)")
	default:
		return fmt.Errorf("civildate: cannot scan %T into Date", src)
	}
}

func (d *Date) scanString(s string) error {
	if len(s) > len(Layout) {
		s = s[:len(Layout)] // "2026-09-23 00:00:00"
	}
	v, err := Parse(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// NullDate is a nullable Date for NULL-able columns.
type NullDate struct {
	Date  Date
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullDate) Scan(src any) error {
	if src == nil {
		*n = NullDate{}
		return nil
	}
	if err := n.Date.Scan(src); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

// Value implements driver.Valuer.
func (n NullDate) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Date.Value()
}

// MarshalJSON writes "Y-m-d" or null.
func (n NullDate) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Date.MarshalJSON()
}
