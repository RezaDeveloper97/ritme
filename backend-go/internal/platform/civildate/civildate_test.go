package civildate

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedNow time.Time

func (f fixedNow) Now() time.Time { return time.Time(f) }

func TestToday_UsesTehranCalendarDay(t *testing.T) {
	// 2026-09-22T21:00:00Z is 00:30 on the 23rd in Tehran (+03:30).
	assert.Equal(t, "2026-09-23", Today(fixedNow(time.Date(2026, 9, 22, 21, 0, 0, 0, time.UTC))).String())
	assert.Equal(t, "2026-09-22", Today(fixedNow(time.Date(2026, 9, 22, 20, 29, 0, 0, time.UTC))).String())
	// DST era (+04:30): 2021-06-01T19:30Z is Tehran midnight.
	assert.Equal(t, "2021-06-02", Today(fixedNow(time.Date(2021, 6, 1, 19, 30, 0, 0, time.UTC))).String())
}

func TestNew_NormalisesOverflow(t *testing.T) {
	assert.Equal(t, "2026-03-02", New(2026, 2, 30).String())
	assert.Equal(t, "2025-12-31", New(2026, 1, 0).String())
	assert.Equal(t, "2025-12-10", New(2026, 0, 10).String())
}

// php > use Carbon\Carbon; date_default_timezone_set('Asia/Tehran');
// php > foreach ([["2021-03-20","2021-03-24"],["2021-09-20","2021-09-24"],["2021-03-24","2021-03-20"],
//
//	["2026-09-23","2026-09-20"],["2020-01-01","2022-12-31"]] as [$a,$b])
//	echo (int) Carbon::parse($a)->diffInDays(Carbon::parse($b));
//
// 2021-03-20->2021-03-24 diffInDays=4.0   (across the 2021-03-22 DST start)
// 2021-09-20->2021-09-24 diffInDays=4.0   (across the 2021-09-22 DST end)
// 2021-03-24->2021-03-20 diffInDays=-4.0
// 2026-09-23->2026-09-20 diffInDays=-3.0
// 2020-01-01->2022-12-31 diffInDays=1095.0
func TestDiffDays_CivilAcrossDST(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2021-03-20", "2021-03-24", 4},
		{"2021-09-20", "2021-09-24", 4},
		{"2021-03-24", "2021-03-20", -4},
		{"2026-09-23", "2026-09-20", -3},
		{"2020-01-01", "2022-12-31", 1095},
		{"2026-09-23", "2026-09-23", 0},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, MustParse(c.a).DiffDays(MustParse(c.b)), "%s -> %s", c.a, c.b)
	}
}

func TestAddDays_AcrossDSTAndYears(t *testing.T) {
	assert.Equal(t, "2021-03-23", MustParse("2021-03-21").AddDays(2).String())
	assert.Equal(t, "2021-09-21", MustParse("2021-09-23").AddDays(-2).String())
	assert.Equal(t, "2025-01-01", MustParse("2024-12-31").AddDays(1).String())
	assert.Equal(t, "2024-02-29", MustParse("2024-03-01").AddDays(-1).String())
}

// php > foreach (["2026-09-23","2026-09-19","2026-09-18","2021-06-01","2024-12-31","2024-03-01"] as $d) {
// php >   $c = Carbon::parse($d); echo $c->dayOfWeek, $c->dayOfWeekIso, $c->dayOfYear,
// php >   $c->copy()->startOfWeek(Carbon::SATURDAY)->toDateString(), $c->copy()->endOfWeek(Carbon::FRIDAY)->toDateString(); }
// 2026-09-23 dow=3 iso=3 doy=266 sow=2026-09-19 eow=2026-09-25
// 2026-09-19 dow=6 iso=6 doy=262 sow=2026-09-19 eow=2026-09-25
// 2026-09-18 dow=5 iso=5 doy=261 sow=2026-09-12 eow=2026-09-18
// 2021-06-01 dow=2 iso=2 doy=152 sow=2021-05-29 eow=2021-06-04
// 2024-12-31 dow=2 iso=2 doy=366 sow=2024-12-28 eow=2025-01-03
// 2024-03-01 dow=5 iso=5 doy=61 sow=2024-02-24 eow=2024-03-01
func TestWeekHelpers(t *testing.T) {
	cases := []struct {
		d        string
		dow, iso int
		doy      int
		sow, eow string
	}{
		{"2026-09-23", 3, 3, 266, "2026-09-19", "2026-09-25"},
		{"2026-09-19", 6, 6, 262, "2026-09-19", "2026-09-25"},
		{"2026-09-18", 5, 5, 261, "2026-09-12", "2026-09-18"},
		{"2021-06-01", 2, 2, 152, "2021-05-29", "2021-06-04"},
		{"2024-12-31", 2, 2, 366, "2024-12-28", "2025-01-03"},
		{"2024-03-01", 5, 5, 61, "2024-02-24", "2024-03-01"},
		{"2026-09-20", 0, 7, 263, "2026-09-19", "2026-09-25"}, // Sunday: iso 7
	}
	for _, c := range cases {
		d := MustParse(c.d)
		assert.Equal(t, c.dow, int(d.Weekday()), c.d)
		assert.Equal(t, c.iso, d.ISOWeekday(), c.d)
		assert.Equal(t, c.doy, d.DayOfYear(), c.d)
		assert.Equal(t, c.sow, d.StartOfWeek().String(), c.d)
		assert.Equal(t, c.eow, d.EndOfWeek().String(), c.d)
	}
}

// php > Carbon::setTestNow(Carbon::parse("2025-02-28 10:00"));
// php > foreach (["2000-02-29","2000-02-28","2000-03-01","1990-09-23"] as $b) echo Carbon::parse($b)->age;
// age 2000-02-29=24  age 2000-02-28=25  age 2000-03-01=24  age 1990-09-23=34
// php > Carbon::setTestNow(Carbon::parse("2026-09-23 10:00"));
// age 2000-09-23=26  age 2000-09-24=25  age 2025-09-24=0
func TestAgeOn(t *testing.T) {
	on := MustParse("2025-02-28")
	assert.Equal(t, 24, MustParse("2000-02-29").AgeOn(on))
	assert.Equal(t, 25, MustParse("2000-02-28").AgeOn(on))
	assert.Equal(t, 24, MustParse("2000-03-01").AgeOn(on))
	assert.Equal(t, 34, MustParse("1990-09-23").AgeOn(on))
	on = MustParse("2026-09-23")
	assert.Equal(t, 26, MustParse("2000-09-23").AgeOn(on))
	assert.Equal(t, 25, MustParse("2000-09-24").AgeOn(on))
	assert.Equal(t, 0, MustParse("2025-09-24").AgeOn(on))
	assert.Equal(t, 25, MustParse("2000-02-29").AgeOn(MustParse("2025-03-01")))
}

func TestParse(t *testing.T) {
	d, err := Parse("2026-09-23")
	require.NoError(t, err)
	assert.Equal(t, Date{2026, time.September, 23}, d)
	for _, bad := range []string{"", "2026-9-23", "2026-02-30", "23-09-2026", "2026-09-23T00:00:00"} {
		_, err := Parse(bad)
		assert.Error(t, err, bad)
	}
	assert.Panics(t, func() { MustParse("nope") })
}

func TestMidnight(t *testing.T) {
	m := MustParse("2026-09-23").TehranMidnight()
	assert.Equal(t, "2026-09-22T20:30:00Z", m.UTC().Format(time.RFC3339))
	m = MustParse("2021-06-01").TehranMidnight()
	assert.Equal(t, "2021-05-31T19:30:00Z", m.UTC().Format(time.RFC3339))
}

func TestCompare(t *testing.T) {
	a, b := MustParse("2026-09-22"), MustParse("2026-09-23")
	assert.True(t, a.Before(b))
	assert.True(t, b.After(a))
	assert.Equal(t, 0, a.Compare(MustParse("2026-09-22")))
	assert.True(t, Date{}.IsZero())
	assert.Empty(t, Date{}.String())
}

func TestJSON(t *testing.T) {
	b, err := json.Marshal(struct {
		A Date     `json:"a"`
		B Date     `json:"b"`
		C NullDate `json:"c"`
		D NullDate `json:"d"`
	}{A: MustParse("2026-09-23"), C: NullDate{Date: MustParse("2021-06-01"), Valid: true}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":"2026-09-23","b":null,"c":"2021-06-01","d":null}`, string(b))

	var d Date
	require.NoError(t, json.Unmarshal([]byte(`"2026-09-23"`), &d))
	assert.Equal(t, "2026-09-23", d.String())
	require.NoError(t, json.Unmarshal([]byte(`null`), &d))
	assert.True(t, d.IsZero())
	require.Error(t, json.Unmarshal([]byte(`"x"`), &d))
	require.Error(t, json.Unmarshal([]byte(`12`), &d))
}

func TestSQL(t *testing.T) {
	var d Date
	// parseTime=true&loc=Asia/Tehran: the driver returns Tehran midnight.
	require.NoError(t, d.Scan(time.Date(2026, 9, 23, 0, 0, 0, 0, Tehran)))
	assert.Equal(t, "2026-09-23", d.String())
	require.NoError(t, d.Scan([]byte("2021-06-01")))
	assert.Equal(t, "2021-06-01", d.String())
	require.NoError(t, d.Scan("2021-06-02 00:00:00"))
	assert.Equal(t, "2021-06-02", d.String())
	require.Error(t, d.Scan(nil))
	require.Error(t, d.Scan(42))
	require.Error(t, d.Scan("junk"))

	v, err := d.Value()
	require.NoError(t, err)
	assert.Equal(t, driver.Value("2021-06-02"), v)
	v, err = Date{}.Value()
	require.NoError(t, err)
	assert.Nil(t, v)

	var n NullDate
	require.NoError(t, n.Scan(nil))
	assert.False(t, n.Valid)
	v, err = n.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
	require.NoError(t, n.Scan("2026-09-23"))
	assert.True(t, n.Valid)
	v, err = n.Value()
	require.NoError(t, err)
	assert.Equal(t, driver.Value("2026-09-23"), v)
	require.Error(t, n.Scan(3.5))
}
