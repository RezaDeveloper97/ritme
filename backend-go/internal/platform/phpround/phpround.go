// Package phpround reproduces PHP 8.4 number behaviour that Go does differently:
// round(), float → string conversion (echo / interpolation / (string) cast),
// json_encode's float form, and the Persian-digit strtr() used for fa strings.
package phpround

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Round is PHP 8.4 round($value, $places) with the default PHP_ROUND_HALF_UP mode:
// half away from zero, decided against the decimal the float is closest to, so
// round(1.005, 2) = 1.01 and round(0.285, 2) = 0.29 where math.Round(x*100)/100
// gives 1 and 0.28. Port of _php_math_round / php_round_helper (ext/standard/math.c).
func Round(value float64, places int) float64 {
	if math.IsInf(value, 0) || math.IsNaN(value) || value == 0 {
		return value
	}
	exponent := intPow10(abs(places))

	var scaled float64
	if places > 0 {
		scaled = value * exponent
	} else {
		scaled = value / exponent
	}
	var integral, next float64
	if value >= 0 {
		integral = math.Floor(scaled)
		next = integral + 1
	} else {
		integral = math.Ceil(scaled)
		next = integral - 1
	}
	// value*exponent can land just below the integer the decimal really is
	// (8615361018.39 * 1e6 = 8615361018389999): take the neighbour when it maps
	// back onto value exactly.
	if unscale(next, exponent, places) == value {
		integral = next
	}

	// Beyond float precision: rounding is pointless.
	if math.Abs(integral) >= 1e16 {
		return value
	}

	// The halfway point between integral and its neighbour, expressed at the
	// value's own scale: ties are resolved on the decimal, not on the product.
	edge := math.Abs(unscale(integral+math.Copysign(0.5, integral), exponent, places))
	if math.Abs(value) >= edge {
		integral += math.Copysign(1, value)
	}
	return unscale(integral, exponent, places)
}

func unscale(x, exponent float64, places int) float64 {
	if places > 0 {
		return x / exponent
	}
	return x * exponent
}

// intPow10 is php_intpow10: exact table for 0..22, pow() beyond.
func intPow10(power int) float64 {
	if power < 0 || power > 22 {
		return math.Pow(10, float64(power))
	}
	p := 1.0
	for range power {
		p *= 10
	}
	return p
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// String is PHP's (string)$float / "$float" / echo (ini precision = 14, "%.14G"):
// 12.0 → "12", 12.5 → "12.5", 0.1+0.2 → "0.3", 1e15 → "1.0E+15", 1e-5 → "1.0E-5".
func String(f float64) string {
	return gcvt(f, 14, 'E', false)
}

// Int formats an int the way PHP does (no surprises; here for symmetry with String).
func Int(n int) string { return strconv.Itoa(n) }

// ErrNonFinite is returned by JSONFloat for INF / NAN (json_encode fails on them too).
var ErrNonFinite = errors.New("phpround: json_encode cannot encode INF or NAN")

// JSONFloat is json_encode's float form (serialize_precision = -1, the shortest
// round-trip repr): 10.0 → "10", 0.1+0.2 → "0.30000000000000004", 1e25 → "1.0e+25",
// 1e-5 → "1.0e-5", 1e15 → "1000000000000000".
func JSONFloat(f float64) (string, error) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "", ErrNonFinite
	}
	return gcvt(f, 17, 'e', true), nil
}

// gcvt is zend_gcvt: digits rounded to ndigit significant digits (or the shortest
// repr), exponential when the decimal exponent is < -4 or ≥ ndigit.
func gcvt(f float64, ndigit int, expChar byte, shortest bool) string {
	switch {
	case math.IsNaN(f):
		return "NAN"
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	}
	neg := math.Signbit(f)
	f = math.Abs(f)

	prec := ndigit - 1
	if shortest {
		prec = -1
	}
	e := strconv.FormatFloat(f, 'e', prec, 64) // "d.ddde±XX"
	mant, expPart, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expPart)
	digits := strings.Replace(mant, ".", "", 1)
	if !keepTrailingZeros(f, ndigit, shortest) {
		digits = strings.TrimRight(digits, "0")
	}
	if digits == "" {
		digits = "0"
	}
	decpt := exp + 1
	if f == 0 {
		decpt = 1
	}

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	if (decpt < 0 && decpt < -3) || decpt > ndigit {
		// d.ddd E±x, at least one fractional digit ("1.0E+25").
		b.WriteByte(digits[0])
		b.WriteByte('.')
		if len(digits) > 1 {
			b.WriteString(digits[1:])
		} else {
			b.WriteByte('0')
		}
		b.WriteByte(expChar)
		if decpt-1 < 0 {
			b.WriteByte('-')
		} else {
			b.WriteByte('+')
		}
		b.WriteString(strconv.Itoa(abs(decpt - 1)))
		return b.String()
	}
	switch {
	case decpt <= 0:
		b.WriteString("0.")
		b.WriteString(strings.Repeat("0", -decpt))
		b.WriteString(digits)
	case len(digits) <= decpt:
		b.WriteString(digits)
		b.WriteString(strings.Repeat("0", decpt-len(digits)))
	default:
		b.WriteString(digits[:decpt])
		b.WriteByte('.')
		b.WriteString(digits[decpt:])
	}
	return b.String()
}

// keepTrailingZeros reproduces a zend_dtoa quirk: an integer-valued float in
// [1e14, 1e15) whose 15th digit is an exact tie (…5) rounded down to even at 14
// digits leaves the "small integer" path without the trailing-zero strip, so PHP
// prints 601424529962905.0 as "6.0142452996290E+14" (and 100000000000005.0 as
// "1.0000000000000E+14") while 100000000000001.0 is "1.0E+14".
func keepTrailingZeros(f float64, ndigit int, shortest bool) bool {
	if shortest || ndigit != 14 || f < 1e14 || f >= 1e15 || f != math.Trunc(f) {
		return false
	}
	if math.Mod(f, 10) != 5 {
		return false
	}
	return int64(f/10)%2 == 0 // 14th digit even → no bump
}

var persianDigits = strings.NewReplacer(
	"0", "۰", "1", "۱", "2", "۲", "3", "۳", "4", "۴",
	"5", "۵", "6", "۶", "7", "۷", "8", "۸", "9", "۹",
)

// PersianDigits is strtr($s, ['0' => '۰', …, '9' => '۹']) (HomeContext::num,
// DailyCardBuilder): ASCII digits only; '.', '-' and everything else are kept.
func PersianDigits(s string) string { return persianDigits.Replace(s) }
