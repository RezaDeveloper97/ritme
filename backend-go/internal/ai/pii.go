package ai

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PII minimisation (B-N6-05, hardened in B-N6-05b). Every text the Client sends to a provider (chat messages and
// system prompt, the voice-log transcript handed to the parser, extraction hints) goes through Redact first, and
// every string value an extraction returns too (a lab sheet's printed name must not come back as a «field»).
//
// Matching runs on a normalized copy (normalizeRunes: Persian / Arabic-Indic digits → ASCII, Arabic letter forms →
// Persian, invisible format characters → ZWNJ) and replaces the matched range of the original text, so the rest
// of the text reaches the provider unchanged. The filter replaces:
//
//   - e-mail addresses → [email]
//   - Iranian mobile numbers, with or without +98 / 0098 / 0 prefix and with spaces, dashes, underscores or ZWNJ
//     between groups (09xx xxx xxxx, +98 9xx-xxx-xxxx) → [phone]
//   - national ids written with dashes (xxx-xxxxxx-x) → [id]
//   - any number of 10 or more digits, also when its groups are separated by single spaces, dashes, underscores
//     or ZWNJ (card numbers 6037 9912 3456 7890, national ids, landlines, postal codes, IBAN digits) → [number];
//     dates (1403/07/12, 2026-09-20) are kept and never glued to a neighbouring number
//   - the user's own names (Subject.Names: each word of users.name with ≥ 2 letters, and the full name) as whole
//     words, case-insensitive, tolerant of ي/ی ك/ک and of spaces / ZWNJ between the words → [name]
//   - the same names written in the other script (Latin ↔ Persian: «Fatemeh Hosseini» ↔ «فاطمه حسینی»), by a
//     consonant skeleton of each word (pii_names.go); only names with ≥ 3 consonants, so short words are not hit
//
// Health values never have 10 digits, so the number rule does not touch them. What it cannot catch: other
// people's names, addresses, and anything inside an image or PDF — documents are sent as they are (the consent
// texts ask users to cover printed personal details, and extraction schemas may not ask for identity fields —
// identityKey). Redaction is best effort, not anonymisation: consent is still required for every feature
// (internal/ai/access).

// Subject is the user a call is made for. It travels on the context (WithSubject) from the HTTP middleware to
// the Client: the id for the usage log, the names for redaction. Never sent to a provider.
type Subject struct {
	UserID uint64
	Names  []string // users.name (split into words by Redactor)
}

type subjectKey struct{}

// WithSubject returns ctx carrying s.
func WithSubject(ctx context.Context, s Subject) context.Context {
	return context.WithValue(ctx, subjectKey{}, s)
}

// SubjectFrom is the Subject on ctx (zero when none).
func SubjectFrom(ctx context.Context) Subject {
	s, _ := ctx.Value(subjectKey{}).(Subject)
	return s
}

// Redaction placeholders (ASCII, language neutral; providers keep them as is).
const (
	RedactedEmail  = "[email]"
	RedactedPhone  = "[phone]"
	RedactedID     = "[id]"
	RedactedNumber = "[number]"
	RedactedName   = "[name]"
)

// MinLongRunDigits is the digit count from which a number (with or without separators) is redacted.
const MinLongRunDigits = 10

// sep is what may sit between the digit groups of a phone / card / id number (B-N6-05b): spaces (also NBSP and
// other Unicode spaces), a dash, an underscore and ZWNJ — invisible format characters are mapped to ZWNJ by
// normalizeRunes. A decimal point or a slash is not a separator (health values, dates).
const sep = `[\s\-_\x{200C}]`

var (
	reEmail = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	rePhone = regexp.MustCompile(`(?:(?:\+|00)98` + sep + `?|0)?9[0-9]{2}` + sep + `?[0-9]{3}` + sep + `?[0-9]{4}`)
	// reDashedID: a national id written xxx-xxxxxx-x.
	reDashedID = regexp.MustCompile(`[0-9]{3}-[0-9]{6}-[0-9]`)
	// reDatePII: an ISO / Jalali date, kept even when followed by another number (it is not one long number).
	reDatePII = regexp.MustCompile(`[0-9]{4}[-/][0-9]{1,2}[-/][0-9]{1,2}`)
	// reDigitRun: digits with single separators between groups; redacted when it holds ≥ MinLongRunDigits digits.
	reDigitRun = regexp.MustCompile(`[0-9](?:` + sep + `?[0-9])+`)
)

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// normalizeRunes is the matching copy of text, rune for rune (so a match maps back to the original): Persian and
// Arabic-Indic digits → ASCII, Arabic letter forms → Persian (ي ى → ی, ك → ک, ة → ه, أ إ آ ٱ → ا, ؤ → و) and
// invisible format characters (zero-width space / joiner, bidi marks and embeddings, BOM, word joiner, soft
// hyphen) → ZWNJ, which counts as a separator — so «۰۹۱۲‌۱۲۳…» or a zero-width space inside a number does not hide it.
func normalizeRunes(in []rune) []rune {
	out := make([]rune, len(in))
	for i, r := range in {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r == 'ي' || r == 'ى':
			r = 'ی'
		case r == 'ك':
			r = 'ک'
		case r == 'ة':
			r = 'ه'
		case r == 'أ' || r == 'إ' || r == 'آ' || r == 'ٱ':
			r = 'ا'
		case r == 'ؤ':
			r = 'و'
		case r == '\u200C':
		case r == '\u00AD' || unicode.Is(unicode.Cf, r):
			r = '\u200C'
		}
		out[i] = r
	}
	return out
}

// span is a [start, end) rune range of the text to replace by token.
type span struct {
	start, end int
	token      string
}

// redactor collects the spans to replace on the normalized copy of a text.
type redactor struct {
	orig   []rune
	nr     []rune // normalized runes
	norm   string
	runeAt []int // byte offset in norm → rune index (len(norm)+1 entries)
	taken  []bool
	spans  []span
}

func newRedactor(text string) *redactor {
	orig := []rune(text)
	nr := normalizeRunes(orig)
	norm := string(nr)
	runeAt := make([]int, len(norm)+1)
	i := 0
	for b := range norm {
		runeAt[b] = i
		i++
	}
	runeAt[len(norm)] = len(orig)
	// fill continuation bytes (never looked up for regexp match bounds, kept total for safety)
	for b := 1; b < len(norm); b++ {
		if !utf8.RuneStart(norm[b]) {
			runeAt[b] = runeAt[b-1]
		}
	}
	return &redactor{orig: orig, nr: nr, norm: norm, runeAt: runeAt, taken: make([]bool, len(orig))}
}

func (r *redactor) free(start, end int) bool {
	for i := start; i < end; i++ {
		if r.taken[i] {
			return false
		}
	}
	return true
}

// mark records the rune range [start, end) for token (protect = keep the text but block later rules).
func (r *redactor) mark(start, end int, token string) {
	for i := start; i < end; i++ {
		r.taken[i] = true
	}
	if token != "" {
		r.spans = append(r.spans, span{start, end, token})
	}
}

// bounded reports whether the rune range is not glued to edge runes on either side.
func (r *redactor) bounded(start, end int, edge func(rune) bool) bool {
	if edge == nil {
		return true
	}
	if start > 0 && edge(r.nr[start-1]) {
		return false
	}
	return end >= len(r.nr) || !edge(r.nr[end])
}

// apply replaces (or, with token "", protects) every free, bounded match of re that keep accepts.
func (r *redactor) apply(re *regexp.Regexp, token string, edge func(rune) bool, keep func(m string) bool) {
	for _, m := range re.FindAllStringIndex(r.norm, -1) {
		start, end := r.runeAt[m[0]], r.runeAt[m[1]]
		if start >= end || !r.free(start, end) || !r.bounded(start, end, edge) {
			continue
		}
		if keep != nil && !keep(r.norm[m[0]:m[1]]) {
			continue
		}
		r.mark(start, end, token)
	}
}

func (r *redactor) String() string {
	if len(r.spans) == 0 {
		return string(r.orig)
	}
	slices.SortFunc(r.spans, func(a, b span) int { return a.start - b.start })
	var b strings.Builder
	last := 0
	for _, s := range r.spans {
		b.WriteString(string(r.orig[last:s.start]))
		b.WriteString(s.token)
		last = s.end
	}
	b.WriteString(string(r.orig[last:]))
	return b.String()
}

func countDigits(s string) int {
	n := 0
	for _, c := range s {
		if isDigit(c) {
			n++
		}
	}
	return n
}

// nameTokens are the redactable forms of names (normalized like the text): the full trimmed names and each word
// with ≥ 2 letters, longest first (so a full name is replaced before its parts).
func nameTokens(names []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		t = strings.TrimSpace(t)
		if utf8.RuneCountInString(t) < 2 || seen[strings.ToLower(t)] {
			return
		}
		seen[strings.ToLower(t)] = true
		out = append(out, t)
	}
	for _, n := range names {
		n = string(normalizeRunes([]rune(n)))
		add(strings.Join(strings.FieldsFunc(n, nameSep), " "))
		for _, w := range strings.FieldsFunc(n, func(r rune) bool { return nameSep(r) || r == '-' || r == '.' }) {
			add(w)
		}
	}
	slices.SortStableFunc(out, func(a, b string) int { return utf8.RuneCountInString(b) - utf8.RuneCountInString(a) })
	return out
}

func nameSep(r rune) bool { return unicode.IsSpace(r) || r == '\u200C' }

// namePattern matches token as a whole name: case-insensitive, any run of spaces / ZWNJ between its words.
func namePattern(token string) (*regexp.Regexp, error) {
	words := strings.Split(token, " ")
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return regexp.Compile(`(?i)` + strings.Join(words, `[\s\x{200C}]+`))
}

// Redact strips PII (see the comment at the top of this file) from text. names are the user's own names.
func Redact(text string, names []string) string {
	if text == "" {
		return text
	}
	r := newRedactor(text)
	r.apply(reEmail, RedactedEmail, nil, nil)
	r.apply(rePhone, RedactedPhone, isDigit, nil)
	r.apply(reDashedID, RedactedID, isDigit, nil)
	r.apply(reDatePII, "", isDigit, nil) // protected: a date is not part of a long number
	r.apply(reDigitRun, RedactedNumber, isDigit, func(m string) bool { return countDigits(m) >= MinLongRunDigits })
	for _, n := range nameTokens(names) {
		re, err := namePattern(n)
		if err != nil {
			continue
		}
		r.apply(re, RedactedName, isWordRune, nil)
	}
	r.crossScriptNames(names)
	return r.String()
}

// redactCtx is Redact with the names of the Subject on ctx.
func redactCtx(ctx context.Context, text string) string { return Redact(text, SubjectFrom(ctx).Names) }
