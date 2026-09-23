package phpround

import (
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PHP 8.4.6 (prod image php:8.4-apache):
//
//	php -r 'foreach([1.955,2.5,-2.5,0.285,1.005,-1.005,1.45,5.055,5.045,-0.5,0.5,1.4999999999999998,
//	  8615361018.39,0.1+0.2,98.45,72.456,-0.285] as $v) foreach([2,1,0] as $p)
//	  echo "{",var_export($v,true),", $p, ",var_export(round($v,$p),true),"},\n";'
//
// plus: var_dump(round(1234.5678,-2), round(-1234.5678,-2), round(1250,-2), round(-1250,-2),
// round(5.0E-324, 2), round(1e300, 2), round(8.61536101839e+09, 6))
// → 1200, -1200, 1300, -1300, 0, 1.0E+300, 8615361018.390001
func TestRound_Golden(t *testing.T) {
	cases := []struct {
		v    float64
		p    int
		want float64
	}{
		{1.955, 2, 1.96}, {1.955, 1, 2.0}, {1.955, 0, 2.0},
		{2.5, 2, 2.5}, {2.5, 1, 2.5}, {2.5, 0, 3.0},
		{-2.5, 2, -2.5}, {-2.5, 1, -2.5}, {-2.5, 0, -3.0},
		{0.285, 2, 0.29}, {0.285, 1, 0.3}, {0.285, 0, 0.0},
		{1.005, 2, 1.01}, {1.005, 1, 1.0}, {1.005, 0, 1.0},
		{-1.005, 2, -1.01}, {-1.005, 1, -1.0}, {-1.005, 0, -1.0},
		{1.45, 2, 1.45}, {1.45, 1, 1.5}, {1.45, 0, 1.0},
		{5.055, 2, 5.06}, {5.055, 1, 5.1}, {5.055, 0, 5.0},
		{5.045, 2, 5.05}, {5.045, 1, 5.0}, {5.045, 0, 5.0},
		{-0.5, 2, -0.5}, {-0.5, 1, -0.5}, {-0.5, 0, -1.0},
		{0.5, 2, 0.5}, {0.5, 1, 0.5}, {0.5, 0, 1.0},
		{1.4999999999999998, 2, 1.5}, {1.4999999999999998, 1, 1.5}, {1.4999999999999998, 0, 1.0},
		{8615361018.39, 2, 8615361018.39}, {8615361018.39, 1, 8615361018.4}, {8615361018.39, 0, 8615361018.0},
		{0.30000000000000004, 2, 0.3}, {0.30000000000000004, 1, 0.3}, {0.30000000000000004, 0, 0.0},
		{98.45, 2, 98.45}, {98.45, 1, 98.5}, {98.45, 0, 98.0},
		{72.456, 2, 72.46}, {72.456, 1, 72.5}, {72.456, 0, 72.0},
		{-0.285, 2, -0.29}, {-0.285, 1, -0.3}, {-0.285, 0, math.Copysign(0, -1)},
		{1234.5678, -2, 1200}, {-1234.5678, -2, -1200}, {1250, -2, 1300}, {-1250, -2, -1300},
		{5.0e-324, 2, 0}, {1e300, 2, 1e300}, {8.61536101839e+09, 6, 8615361018.390001},
		{0, 2, 0},
	}
	for _, c := range cases {
		got := Round(c.v, c.p)
		assert.Equal(t, math.Float64bits(c.want), math.Float64bits(got), "round(%v, %d) = %v, want %v", c.v, c.p, got, c.want)
	}
	// The naive Go formula disagrees on the headline cases (variables: constants fold exactly).
	a, b := 1.005, 0.285
	assert.InDelta(t, 1.0, math.Round(a*100)/100, 0)
	assert.InDelta(t, 0.28, math.Round(b*100)/100, 0)
	assert.True(t, math.IsNaN(Round(math.NaN(), 2)))
	assert.True(t, math.IsInf(Round(math.Inf(1), 2), 1))
}

// php -r 'foreach([10.0,12.5,0.1+0.2,1e25,1e15,1e-7,0.00001,-0.0,123456789012345678.0,1.5e-5,
//
//	1e20,65.0,1/3,1e16,1e17,9.9e16,0.0001,0.00012345,123456789012345.6,1234567890123456.7,1e14,
//	99999999999999.99,0.1,100000000000001.0,999999999999995.0,601424529962915.0,601424529962905.0,
//	100000000000005.0] as $v) echo json_encode($v)," | ","$v","\n";'
//
// 10 | 10                                   12.5 | 12.5
// 0.30000000000000004 | 0.3                 1.0e+25 | 1.0E+25
// 1000000000000000 | 1.0E+15                1.0e-7 | 1.0E-7
// 1.0e-5 | 1.0E-5                           -0 | -0
// 1.2345678901234568e+17 | 1.2345678901235E+17
// 1.5e-5 | 1.5E-5                           1.0e+20 | 1.0E+20
// 65 | 65                                   0.3333333333333333 | 0.33333333333333
// 10000000000000000 | 1.0E+16               1.0e+17 | 1.0E+17
// 99000000000000000 | 9.9E+16               0.0001 | 0.0001
// 0.00012345 | 0.00012345                   123456789012345.6 | 1.2345678901235E+14
// 1234567890123456.8 | 1.2345678901235E+15  100000000000000 | 1.0E+14
// 99999999999999.98 | 1.0E+14               0.1 | 0.1
// 100000000000001 | 1.0E+14                 999999999999995 | 1.0E+15
// 601424529962915 | 6.0142452996292E+14    601424529962905 | 6.0142452996290E+14
// 100000000000005 | 1.0000000000000E+14
func TestStringAndJSONFloat_Golden(t *testing.T) {
	cases := []struct {
		v          float64
		json, text string
	}{
		{10.0, "10", "10"},
		{12.5, "12.5", "12.5"},
		{0.30000000000000004, "0.30000000000000004", "0.3"},
		{1e25, "1.0e+25", "1.0E+25"},
		{1e15, "1000000000000000", "1.0E+15"},
		{1e-7, "1.0e-7", "1.0E-7"},
		{0.00001, "1.0e-5", "1.0E-5"},
		{math.Copysign(0, -1), "-0", "-0"},
		{123456789012345678.0, "1.2345678901234568e+17", "1.2345678901235E+17"},
		{1.5e-5, "1.5e-5", "1.5E-5"},
		{1e20, "1.0e+20", "1.0E+20"},
		{65.0, "65", "65"},
		{1.0 / 3, "0.3333333333333333", "0.33333333333333"},
		{1e16, "10000000000000000", "1.0E+16"},
		{1e17, "1.0e+17", "1.0E+17"},
		{9.9e16, "99000000000000000", "9.9E+16"},
		{0.0001, "0.0001", "0.0001"},
		{0.00012345, "0.00012345", "0.00012345"},
		{123456789012345.6, "123456789012345.6", "1.2345678901235E+14"},
		{1234567890123456.7, "1234567890123456.8", "1.2345678901235E+15"},
		{1e14, "100000000000000", "1.0E+14"},
		{99999999999999.99, "99999999999999.98", "1.0E+14"},
		{0.1, "0.1", "0.1"},
		{100000000000001.0, "100000000000001", "1.0E+14"},
		{999999999999995.0, "999999999999995", "1.0E+15"},
		{601424529962915.0, "601424529962915", "6.0142452996292E+14"},
		{601424529962905.0, "601424529962905", "6.0142452996290E+14"},
		{100000000000005.0, "100000000000005", "1.0000000000000E+14"},
		{0, "0", "0"},
		{-12.5, "-12.5", "-12.5"},
	}
	for _, c := range cases {
		j, err := JSONFloat(c.v)
		require.NoError(t, err)
		assert.Equal(t, c.json, j, "json_encode(%v)", c.v)
		assert.Equal(t, c.text, String(c.v), "(string)%v", c.v)
	}
	_, err := JSONFloat(math.Inf(1))
	require.ErrorIs(t, err, ErrNonFinite)
	_, err = JSONFloat(math.NaN())
	require.ErrorIs(t, err, ErrNonFinite)
	// php > echo INF, ' ', -INF, ' ', NAN;  →  INF -INF NAN
	assert.Equal(t, "INF", String(math.Inf(1)))
	assert.Equal(t, "-INF", String(math.Inf(-1)))
	assert.Equal(t, "NAN", String(math.NaN()))
	assert.Equal(t, "42", Int(42))
}

// testdata/php_corpus.txt holds 4,000 PHP 8.4.6 results, one per line:
// "<bits of $v> <places> <bits of round($v,$places)> <json_encode($v)> <(string)$v>",
// generated by (bits = bin2hex(strrev(pack('d', $f)))):
//
//	mt_srand(42); 6 shapes of value: mt_rand(-1e5,1e5)/1000, mt_rand(-1e7,1e7)/1e4, (n+0.5)/100·±1,
//	  mt_rand()/mt_getrandmax()*1000, mt_rand(0,99999)/1000*3.3, (n+0.5)/10; places ∈ {0,1,2,3,-1,-2,4}
//	mt_srand(7); v = rand01 * 10**mt_rand(-12,22) * ±1 (every 3rd pre-rounded); places ∈ [-5, 8]
//	mt_srand(3); integers around 1e14..1e16 (the zend_dtoa trailing-zero quirk); places ∈ [-3, 3]
//
// The same generators were run at 350,000 values while porting, with no mismatch.
func TestCorpus_MatchesPHP(t *testing.T) {
	f, err := os.Open("testdata/php_corpus.txt")
	require.NoError(t, err)
	defer f.Close()
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		p := strings.Fields(sc.Text())
		require.Len(t, p, 5, sc.Text())
		vb, err := strconv.ParseUint(p[0], 16, 64)
		require.NoError(t, err)
		places, err := strconv.Atoi(p[1])
		require.NoError(t, err)
		rb, err := strconv.ParseUint(p[2], 16, 64)
		require.NoError(t, err)
		v := math.Float64frombits(vb)

		if got := Round(v, places); math.Float64bits(got) != rb {
			t.Errorf("round(%v, %d): php %v, go %v", v, places, math.Float64frombits(rb), got)
		}
		if got, _ := JSONFloat(v); got != p[3] {
			t.Errorf("json_encode(%v): php %s, go %s", v, p[3], got)
		}
		if got := String(v); got != p[4] {
			t.Errorf("(string)%v: php %s, go %s", v, p[4], got)
		}
		n++
	}
	require.NoError(t, sc.Err())
	assert.Equal(t, 4000, n)
}

// php > $n = fn($v) => strtr((string) $v, ['0'=>'۰','1'=>'۱','2'=>'۲','3'=>'۳','4'=>'۴','5'=>'۵','6'=>'۶','7'=>'۷','8'=>'۸','9'=>'۹']);
// php > echo $n(12.5), ' ', $n(-40), ' ', $n('2026-09-23'), ' ', $n('abc');
// ۱۲.۵ -۴۰ ۲۰۲۶-۰۹-۲۳ abc
func TestPersianDigits(t *testing.T) {
	assert.Equal(t, "۱۲.۵", PersianDigits(String(12.5)))
	assert.Equal(t, "-۴۰", PersianDigits("-40"))
	assert.Equal(t, "۲۰۲۶-۰۹-۲۳", PersianDigits("2026-09-23"))
	assert.Equal(t, "abc", PersianDigits("abc"))
	assert.Equal(t, "۰۱۲۳۴۵۶۷۸۹", PersianDigits("0123456789"))
}
