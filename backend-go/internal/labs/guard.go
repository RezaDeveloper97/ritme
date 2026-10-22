package labs

import (
	"regexp"
	"strings"
	"unicode"
)

// The output filter of the AI lab summary (T-M10-20, SPEC v6 §3.3, D-171): the summary is general information, never
// a diagnosis or a treatment, so any AI text that names a condition or a dose is discarded and the rules summary is
// stored instead (labs_ai_filtered_total). The lists are the one place these words live (TestGuardWordLists); they
// are matched in every language at once (a model may answer in another language than asked), case-insensitively,
// after Persian / Arabic digits become ASCII, Arabic ي/ك become ی/ک and a space or ZWNJ inside a term may be either or
// missing («کم خونی» = «کمخونی»; terms are written with a space). Latin terms are stems matched from a word start («diabet» → diabetes,
// diabetic); other scripts match anywhere (Persian attaches suffixes and prefixes: «دیابتی», «پیش‌دیابت»).

// diagnosticTerms are condition names a lab summary must not use.
var diagnosticTerms = []string{
	// Latin script
	"diabet", "prediabet", "anemi", "anaemi", "hypothyroid", "hyperthyroid", "hashimoto", "graves", "thyroiditis",
	"pcos", "polycystic", "cancer", "tumor", "tumour", "malignan", "leukemi", "leukaemi", "lymphoma", "thalass",
	"kidney disease", "renal failure", "kidney failure", "liver disease", "liver failure", "cirrhosis", "hepatitis",
	"fatty liver", "preeclampsia", "pre-eclampsia", "eclampsia", "celiac", "coeliac", "lupus", "insulin resistance",
	"hemochromatosis", "haemochromatosis", "osteoporosis", "infertil", "hypogonadism", "cushing", "addison",
	"metabolic syndrome", "hyperlipid", "hypercholesterol", "dyslipid",
	// Persian script
	"دیابت", "کم خونی", "آنمی", "کم کاری تیروئید", "پرکاری تیروئید", "هیپوتیروئید", "هیپرتیروئید", "هاشیموتو",
	"گریوز", "تیروئیدیت", "پلی کیستیک", "سرطان", "تومور", "بدخیم", "لوسمی", "لنفوم", "تالاسمی", "نارسایی کلیه",
	"بیماری کلیه", "نارسایی کبد", "بیماری کبد", "سیروز", "هپاتیت", "کبد چرب", "پره اکلامپسی", "اکلامپسی",
	"مسمومیت بارداری", "سلیاک", "لوپوس", "مقاومت به انسولین", "هموکروماتوز", "پوکی استخوان", "ناباروری", "نازایی",
	"کوشینگ", "آدیسون", "سندرم متابولیک", "چربی خون بالا",
}

// doseWords are treatment words that never belong in a lab summary, with or without a number.
var doseWords = []string{
	"dose", "dosage", "tablet", "capsule", "pill", "قرص", "کپسول", "دوز",
}

// doseUnits are units that make a dose when they follow a number («500 mg», «۵۰۰۰۰ واحد»); a lab unit written with
// a slash («mg/dL», «IU/L») is not a dose.
var doseUnits = []string{
	"mg", "mcg", "µg", "μg", "ug", "iu", "units", "unit", "میلی گرم", "میکروگرم", "واحد",
}

var (
	diagnosticRes = compileTerms(diagnosticTerms)
	doseWordRes   = compileTerms(doseWords)
	reDose        = regexp.MustCompile(`(?i)[0-9]+(?:[.,][0-9]+)?[\s\x{200C}]*(` + alternation(doseUnits) + `)`)
)

func isLatin(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func termPattern(t string) string {
	parts := strings.FieldsFunc(t, func(r rune) bool { return r == ' ' || r == '‌' })
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return strings.Join(parts, `[\s\x{200C}]*`)
}

func alternation(terms []string) string {
	out := make([]string, len(terms))
	for i, t := range terms {
		out[i] = termPattern(t)
	}
	return strings.Join(out, "|")
}

func compileTerms(terms []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(terms))
	for _, t := range terms {
		p := termPattern(t)
		if isLatin(t) {
			p = `\b` + p
		}
		out = append(out, regexp.MustCompile(`(?i)`+p))
	}
	return out
}

// normalizeGuard is the matching form of an AI text: Persian / Arabic digits as ASCII, Arabic ي/ك as ی/ک.
func normalizeGuard(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r == 'ي':
			r = 'ی'
		case r == 'ك':
			r = 'ک'
		case r == '٫':
			r = '.'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Filter reasons (labs_ai_filtered_total{reason}).
const (
	FilterInvalid    = "invalid"    // not the JSON shape, or empty
	FilterCategory   = "category"   // names a marker the rules did not put out of range
	FilterDiagnostic = "diagnostic" // a condition name
	FilterDose       = "dose"       // a dose or a treatment word
)

// guardText is the filter reason of an AI text ("" = clean).
func guardText(s string) string {
	n := normalizeGuard(s)
	for _, re := range diagnosticRes {
		if re.MatchString(n) {
			return FilterDiagnostic
		}
	}
	for _, re := range doseWordRes {
		if re.MatchString(n) {
			return FilterDose
		}
	}
	for _, loc := range reDose.FindAllStringIndex(n, -1) {
		if loc[1] >= len(n) || n[loc[1]] != '/' {
			return FilterDose
		}
	}
	return ""
}
