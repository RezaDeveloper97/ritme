package jsonx

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func mustMarshal(t *testing.T, v any, flags Flags) string {
	t.Helper()
	b, err := Marshal(v, flags)
	require.NoError(t, err)
	return string(b)
}

// php > echo json_encode(['url'=>'https://a.b/c?d=1&e=<f>', 'fa'=>'سلام', 'emoji'=>'😀', 'ls'=>"a\u{2028}b",
// php >   'ctl'=>"\x01\t\n\x08\x0c\"\\", 'empty_arr'=>[], 'empty_obj'=>new stdClass, 'list'=>[1,2], 'f'=>10.0,
// php >   'f2'=>0.1+0.2, 'nested'=>['a'=>[], 'b'=>['x'=>null]]]);
// {"url":"https:\/\/a.b\/c?d=1&e=<f>","fa":"\u0633\u0644\u0627\u0645","emoji":"\ud83d\ude00","ls":"a\u2028b",
// "ctl":"\u0001\t\n\b\f\"\\","empty_arr":[],"empty_obj":{},"list":[1,2],"f":10,"f2":0.30000000000000004,
// "nested":{"a":[],"b":{"x":null}}}
func TestMarshal_PHPDefaultFlags(t *testing.T) {
	v := Obj(
		"url", "https://a.b/c?d=1&e=<f>",
		"fa", "سلام",
		"emoji", "😀",
		"ls", "a\u2028b",
		"ctl", "\x01\t\n\x08\x0c\"\\",
		"empty_arr", NewArray(),
		"empty_obj", NewObject(),
		"list", []int{1, 2},
		"f", 10.0,
		"f2", 0.30000000000000004,
		"nested", Obj("a", PHPArray[int]{}, "b", Obj("x", nil)),
	)
	want := `{"url":"https:\/\/a.b\/c?d=1&e=<f>","fa":"\u0633\u0644\u0627\u0645","emoji":"\ud83d\ude00","ls":"a\u2028b",` +
		`"ctl":"\u0001\t\n\b\f\"\\","empty_arr":[],"empty_obj":{},"list":[1,2],"f":10,"f2":0.30000000000000004,` +
		`"nested":{"a":[],"b":{"x":null}}}`
	assert.Equal(t, want, mustMarshal(t, v, 0))
}

// php > echo json_encode(['url'=>'https://a.b/c','fa'=>'سلام','emoji'=>'😀','ls'=>"a\u{2028}b"], JSON_UNESCAPED_UNICODE);
// {"url":"https:\/\/a.b\/c","fa":"سلام","emoji":"😀","ls":"a\u2028b"}
func TestMarshal_UnescapedUnicode(t *testing.T) {
	v := Obj("url", "https://a.b/c", "fa", "سلام", "emoji", "😀", "ls", "a\u2028b")
	assert.Equal(t, `{"url":"https:\/\/a.b\/c","fa":"سلام","emoji":"😀","ls":"a\u2028b"}`, mustMarshal(t, v, UnescapedUnicode))
	assert.Equal(t, `{"url":"https://a.b/c","fa":"سلام","emoji":"😀","ls":"a\u2028b"}`,
		mustMarshal(t, v, UnescapedUnicode|UnescapedSlashes))
}

// php > echo json_encode(['url'=>'https://a.b/c','fa'=>'سلام','empty_arr'=>[],'empty_obj'=>new stdClass,
// php >   'list'=>[1,[2,[]]]], JSON_PRETTY_PRINT|JSON_UNESCAPED_SLASHES);
func TestMarshal_FrameworkPretty(t *testing.T) {
	v := Obj("url", "https://a.b/c", "fa", "سلام", "empty_arr", Slice[int](nil), "empty_obj", NewObject(),
		"list", []any{1, []any{2, List[int](nil)}})
	want := `{
    "url": "https://a.b/c",
    "fa": "\u0633\u0644\u0627\u0645",
    "empty_arr": [],
    "empty_obj": {},
    "list": [
        1,
        [
            2,
            []
        ]
    ]
}`
	assert.Equal(t, want, mustMarshal(t, v, Framework))
}

func TestMarshal_Error(t *testing.T) {
	_, err := Marshal(make(chan int), 0)
	require.Error(t, err)
	_, err = Marshal(Obj("x", make(chan int)), Framework)
	require.Error(t, err)
	_, err = Marshal(Float(1)/Float(zero()), 0) // +Inf
	require.Error(t, err)
}

func zero() float64 { return 0 }

// Laravel model serialisation (toArray() + json_encode), backend models:
//
//	$r = new App\Models\Reminder; // starts_on/ends_on 'date', scheduled_at 'datetime'
//	$r->forceFill(['starts_on'=>'2026-09-23','ends_on'=>'2021-06-01','scheduled_at'=>'2026-09-23 13:00:00',
//	  'created_at'=>'2021-06-01 08:00:00']);
//	→ {"starts_on":"2026-09-22T20:30:00.000000Z","ends_on":"2021-05-31T19:30:00.000000Z",
//	   "scheduled_at":"2026-09-23T09:30:00.000000Z","created_at":"2021-06-01T03:30:00.000000Z"}
//	$l = new App\Models\DailyHealthLog; // log_date 'date:Y-m-d', weight/BBT 'decimal:2', blood_sugar 'decimal:1'
//	$l->forceFill(['log_date'=>'2026-09-23','weight'=>65.5,'basal_body_temperature'=>'36.555','blood_sugar'=>98.45]);
//	→ {"log_date":"2026-09-23","weight":"65.50","basal_body_temperature":"36.56","blood_sugar":"98.5"}
//	$l->forceFill(['weight'=>null,'blood_sugar'=>'98']);  → {"weight":null,"blood_sugar":"98.0"}
//	Carbon::parse('2026-09-23 13:00:00')->toIso8601String() → "2026-09-23T13:00:00+03:30"
//	Carbon::parse('2021-06-01 13:00:00')->toIso8601String() → "2021-06-01T13:00:00+04:30"
func TestDateTypes_MatchLaravel(t *testing.T) {
	tz := civildate.Tehran
	v := Obj(
		"starts_on", DateCast(civildate.MustParse("2026-09-23")),
		"ends_on", DateCast(civildate.MustParse("2021-06-01")),
		"scheduled_at", DateTime(time.Date(2026, 9, 23, 13, 0, 0, 0, tz)),
		"created_at", DateTime(time.Date(2021, 6, 1, 8, 0, 0, 0, tz)),
		"log_date", civildate.MustParse("2026-09-23"), // YMD
		"iso", ISO8601(time.Date(2026, 9, 23, 13, 0, 0, 0, tz)),
		"iso_dst", ISO8601(time.Date(2021, 6, 1, 13, 0, 0, 0, tz)),
		"iso_from_utc", ISO8601(time.Date(2026, 9, 23, 9, 30, 0, 0, time.UTC)),
		"null_cast", DateCast(civildate.Date{}),
		"null_dt", DateTime(time.Time{}),
		"null_iso", ISO8601(time.Time{}),
		"null_ymd", YMD{}, //nolint:gocritic // the alias under test
	)
	want := `{"starts_on":"2026-09-22T20:30:00.000000Z","ends_on":"2021-05-31T19:30:00.000000Z",` +
		`"scheduled_at":"2026-09-23T09:30:00.000000Z","created_at":"2021-06-01T03:30:00.000000Z",` +
		`"log_date":"2026-09-23","iso":"2026-09-23T13:00:00+03:30","iso_dst":"2021-06-01T13:00:00+04:30",` +
		`"iso_from_utc":"2026-09-23T13:00:00+03:30",` +
		`"null_cast":null,"null_dt":null,"null_iso":null,"null_ymd":null}`
	assert.Equal(t, want, mustMarshal(t, v, UnescapedSlashes))
}

// (string) BigDecimal::of($v)->toScale(2|0, RoundingMode::HALF_UP), brick/math as used by
// HasAttributes::asDecimal:
//
//	"-0.001" => 0.00 | 0        "-0.005" => -0.01 | 0       "0.005" => 0.01 | 0
//	"65" => 65.00 | 65          "65.5" => 65.50 | 66        "-65.555" => -65.56 | -66
//	"1e-3" => 0.00 | 0          "+5.5" => 5.50 | 6          ".5" => 0.50 | 1
//	"5." => 5.00 | 5            "1.5e1" => 15.00 | 15       "00012.30" => 12.30 | 12
//	" 5" => ERR                 "" => ERR
//	65.0 => 65.00   98.45 => 98.45   -0.0 => 0.00   1.0E+20 => 100000000000000000000.00
//	0.30000000000000004 => 0.30   7 => 7.00   true => 1.00
func TestDecimal_MatchesBrick(t *testing.T) {
	cases := []struct {
		in         any
		two, zeroS string
	}{
		{"-0.001", "0.00", "0"}, {"-0.005", "-0.01", "0"}, {"0.005", "0.01", "0"},
		{"65", "65.00", "65"}, {"65.5", "65.50", "66"}, {"-65.555", "-65.56", "-66"},
		{"1e-3", "0.00", "0"}, {"+5.5", "5.50", "6"}, {".5", "0.50", "1"},
		{"5.", "5.00", "5"}, {"1.5e1", "15.00", "15"}, {"00012.30", "12.30", "12"},
		{[]byte("36.555"), "36.56", "37"},
		{65.0, "65.00", "65"}, {98.45, "98.45", "98"}, {math.Copysign(0, -1), "0.00", "0"},
		{1e20, "100000000000000000000.00", "100000000000000000000"},
		{0.30000000000000004, "0.30", "0"}, {7, "7.00", "7"}, {int64(-7), "-7.00", "-7"},
		{true, "1.00", "1"}, {float32(2.5), "2.50", "3"},
	}
	for _, c := range cases {
		d, err := Decimal(c.in, 2)
		require.NoError(t, err, "%v", c.in)
		assert.Equal(t, c.two, d.String(), "%#v scale 2", c.in)
		assert.True(t, d.Valid())
		d, err = Decimal(c.in, 0)
		require.NoError(t, err, "%v", c.in)
		assert.Equal(t, c.zeroS, d.String(), "%#v scale 0", c.in)
	}
	for _, bad := range []any{" 5", "", ".", "abc", "1e", struct{}{}} {
		_, err := Decimal(bad, 2)
		require.Error(t, err, "%#v", bad)
	}
	assert.Panics(t, func() { MustDecimal("x", 2) })

	// Laravel: weight 65.5 (float) decimal:2, blood_sugar 98.45 (float) decimal:1, '98' decimal:1, null.
	v := Obj("weight", MustDecimal(65.5, 2), "bbt", MustDecimal("36.555", 2),
		"blood_sugar", MustDecimal(98.45, 1), "bs2", MustDecimal("98", 1), "null", MustDecimal(nil, 2))
	assert.Equal(t, `{"weight":"65.50","bbt":"36.56","blood_sugar":"98.5","bs2":"98.0","null":null}`, mustMarshal(t, v, 0))
	assert.False(t, NullDecimal.Valid())
}

func TestFloat(t *testing.T) {
	v := Obj("a", Float(10), "b", Float(1e25), "c", Float(0.00001), "d", Float(12.5))
	assert.Equal(t, `{"a":10,"b":1.0e+25,"c":1.0e-5,"d":12.5}`, mustMarshal(t, v, 0))
}

func TestPHPArrayAndSlices(t *testing.T) {
	type s struct {
		A PHPArray[string] `json:"a"`
		B PHPArray[string] `json:"b"`
		C Slice[int]       `json:"c"`
		D Slice[int]       `json:"d"`
		E []string         `json:"e"`
		F []string         `json:"f"`
	}
	got := mustMarshal(t, s{
		B: PHPArray[string]{"z": "<1>", "a": "2"},
		D: Slice[int]{3},
		E: List[string](nil),
		F: List([]string{"x"}),
	}, 0)
	assert.Equal(t, `{"a":[],"b":{"a":"2","z":"<1>"},"c":[],"d":[3],"e":[],"f":["x"]}`, got)
}

func TestOrderedMap(t *testing.T) {
	m := Obj("success", true, "message", "ok", "data", nil)
	m.Set("message", "changed") // keeps its position, like PHP
	m.Set("warning", "w")
	assert.Equal(t, []string{"success", "message", "data", "warning"}, m.Keys())
	assert.Equal(t, 4, m.Len())
	v, ok := m.Get("message")
	assert.True(t, ok)
	assert.Equal(t, "changed", v)
	m.Delete("data")
	m.Delete("missing")
	assert.Equal(t, `{"success":true,"message":"changed","warning":"w"}`, mustMarshal(t, m, 0))

	var zero OrderedMap
	zero.Set("k", 1)
	assert.Equal(t, `{"k":1}`, mustMarshal(t, &zero, 0))
	assert.Equal(t, `[]`, mustMarshal(t, NewArray(), 0))
	assert.Equal(t, `{}`, mustMarshal(t, NewObject(), 0))
	assert.Equal(t, `{"a":1}`, mustMarshal(t, NewArray().Set("a", 1), 0))
	var nilMap *OrderedMap
	assert.Equal(t, `null`, mustMarshal(t, nilMap, 0))

	assert.Panics(t, func() { Obj("a") }) //nolint:staticcheck // odd count is the case under test
	assert.Panics(t, func() { Obj(1, 2) })

	// Nested values keep <, >, & raw (no HTML escaping inside custom marshalers).
	assert.Equal(t, `{"h":"<b>&</b>"}`, mustMarshal(t, Obj("h", "<b>&</b>"), UnescapedSlashes))
	// Raw JSON passes through (re-escaped per flags).
	assert.Equal(t, `{"r":{"fa":"\u0633","u":"a\/b"}}`, mustMarshal(t, Obj("r", Raw(`{"fa":"س","u":"a/b"}`)), 0))
}

func TestStdlibCompat(t *testing.T) {
	// Types also work with plain encoding/json (e.g. in tests or logs).
	b, err := json.Marshal(struct {
		D LaravelDateCast `json:"d"`
	}{DateCast(civildate.MustParse("2026-09-23"))})
	require.NoError(t, err)
	assert.JSONEq(t, `{"d":"2026-09-22T20:30:00.000000Z"}`, string(b))
}
