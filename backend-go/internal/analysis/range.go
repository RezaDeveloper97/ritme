package analysis

import (
	"slices"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Range keys. The hub and the detail screens use 3m / 6m / 1y / all; the symptom trend adds 2w / 1m
// and the body screen 7d / 1m («۳۰ روز»).
const (
	Range7D  = "7d"
	Range2W  = "2w"
	Range1M  = "1m"
	Range3M  = "3m"
	Range6M  = "6m"
	Range1Y  = "1y"
	RangeAll = "all"
)

// RangeKeys are the accepted `range` values; DefaultRange applies when none is sent.
var RangeKeys = []string{Range7D, Range2W, Range1M, Range3M, Range6M, Range1Y, RangeAll}

// DefaultRange is the hub's default selection («۶ ماه»).
const DefaultRange = Range6M

// rangeDays is each range's length in days, today inclusive. `all` reaches back 10 years.
var rangeDays = map[string]int{
	Range7D: 7, Range2W: 14, Range1M: 30, Range3M: 91, Range6M: 182, Range1Y: 365, RangeAll: 3650,
}

// IsRange reports whether key is an accepted range.
func IsRange(key string) bool { return slices.Contains(RangeKeys, key) }

// Range is the selected window [From, To], To = the request day.
type Range struct {
	Key      string
	From, To civildate.Date
}

// NewRange is the window of key ending today (an unknown key falls back to DefaultRange).
func NewRange(key string, today civildate.Date) Range {
	n, ok := rangeDays[key]
	if !ok {
		key, n = DefaultRange, rangeDays[DefaultRange]
	}
	return Range{Key: key, From: today.AddDays(-(n - 1)), To: today}
}

// Days is the window's length (both ends inclusive).
func (r Range) Days() int { return r.From.DiffDays(r.To) + 1 }

// Contains reports whether d lies in the window.
func (r Range) Contains(d civildate.Date) bool { return !d.Before(r.From) && !d.After(r.To) }

// JSON is the `range` object every response carries.
func (r Range) JSON() *jsonx.OrderedMap {
	return jsonx.Obj("key", r.Key, "from", r.From.String(), "to", r.To.String(), "days", r.Days())
}
