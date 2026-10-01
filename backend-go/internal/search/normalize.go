package search

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// fold maps the letters Persian text is commonly typed with interchangeably onto one form: Arabic yeh /
// alef maksura → Persian yeh, Arabic kaf → Persian keheh, alef variants → alef, ta marbuta / heh with
// yeh → heh, waw with hamza → waw, and Persian / Arabic-Indic digits → ASCII digits.
var fold = map[rune]rune{
	'ي': 'ی', 'ى': 'ی', 'ئ': 'ی',
	'ك': 'ک',
	'أ': 'ا', 'إ': 'ا', 'آ': 'ا', 'ٱ': 'ا',
	'ة': 'ه', 'ۀ': 'ه', 'ہ': 'ه',
	'ؤ': 'و',
	'۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4', '۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9',
	'٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4', '٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9',
}

// separator reports runes that split words: whitespace, ZWNJ / ZWJ and every punctuation or symbol.
func separator(r rune) bool {
	return r == '\u200c' || r == '\u200d' || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// dropped reports runes that carry no meaning for matching: Arabic diacritics (harakat, tanwin, shadda,
// sukun, superscript alef), tatweel, hamza and the other format characters (bidi marks).
func dropped(r rune) bool {
	switch {
	case r >= 'ً' && r <= 'ٟ', r == 'ٰ', r == 'ـ', r == 'ء':
		return true
	case r == '\u200c' || r == '\u200d':
		return false // separators, handled by separator()
	}
	return unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Mn, r)
}

// Normalize is the matching form of s: folded letters and digits, lower case, no diacritics, and every
// run of separators (spaces, ZWNJ, punctuation) collapsed to one space. «ثبت‌های من» and «ثبت هاي من»
// normalise to the same string.
func Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if dropped(r) {
			continue
		}
		if separator(r) {
			space = b.Len() > 0
			continue
		}
		if f, ok := fold[r]; ok {
			r = f
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// compact drops the spaces of a normalised string, so a word typed with or without a ZWNJ / space inside
// («می شود», «میشود», «می‌شود») still matches.
func compact(norm string) string { return strings.ReplaceAll(norm, " ", "") }

// Matcher matches one query against candidate texts. Every query word must occur (as a substring, after
// normalisation and with spaces ignored) in the candidate.
type Matcher struct {
	whole string   // the whole query, compact
	words []string // the query words, compact
}

// NewMatcher prepares q.
func NewMatcher(q string) Matcher {
	norm := Normalize(q)
	m := Matcher{whole: compact(norm)}
	if norm != "" {
		m.words = strings.Split(norm, " ")
	}
	return m
}

// Empty reports a query with nothing to match (only separators or marks).
func (m Matcher) Empty() bool { return m.whole == "" }

// Len is the query length in runes (spaces ignored).
func (m Matcher) Len() int { return utf8.RuneCountInString(m.whole) }

// Match ranks text against the query: 0 = no match, 1 = every word occurs somewhere, 2 = the whole
// query occurs as one phrase, 3 = text starts with it.
func (m Matcher) Match(text string) int {
	if m.Empty() {
		return 0
	}
	c := compact(Normalize(text))
	switch {
	case strings.HasPrefix(c, m.whole):
		return 3
	case strings.Contains(c, m.whole):
		return 2
	}
	// Words: each must occur in the normalised text (word order free).
	norm := Normalize(text)
	for _, w := range m.words {
		if !strings.Contains(norm, w) && !strings.Contains(c, w) {
			return 0
		}
	}
	return 1
}

// Best is the best rank of the query in the primary text, or (capped at 1) in any secondary text.
func (m Matcher) Best(primary string, secondary ...string) int {
	if r := m.Match(primary); r > 0 {
		return r + 1 // a title hit always ranks above a secondary-text hit
	}
	all := primary
	for _, s := range secondary {
		if m.Match(s) > 0 {
			return 1
		}
		all += " " + s
	}
	if m.Match(all) > 0 {
		return 1
	}
	return 0
}
