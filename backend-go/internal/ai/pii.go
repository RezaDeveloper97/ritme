package ai

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PII minimisation (B-N6-05). Every text the Client sends to a provider (chat messages and system prompt, the
// voice-log transcript handed to the parser, extraction hints) goes through Redact first, and every string value
// an extraction returns too (a lab sheet's printed name must not come back as a «field»). The filter replaces:
//
//   - e-mail addresses → [email]
//   - Iranian mobile numbers, with or without +98 / 0098 / 0 prefix and with spaces or dashes between groups
//     (09xx xxx xxxx, +98 9xx-xxx-xxxx) → [phone]
//   - national ids written with dashes (xxx-xxxxxx-x) → [id]
//   - any run of 10 or more digits (national ids, landlines, card / account numbers, IBAN digits) → [number]
//   - the user's own names (Subject.Names: each word of users.name with ≥ 2 letters, and the full name) as whole
//     words, case-insensitive → [name]
//
// Digits are matched in Latin, Persian (۰–۹) and Arabic-Indic (٠–٩) forms. Health values never have 10 digits, so
// the number rule does not touch them. What it cannot catch: other people's names, addresses, and anything inside
// an image or PDF — documents are sent as they are (the consent texts ask users to cover printed personal details,
// and extraction schemas never ask for identity fields). Redaction is best effort, not anonymisation: consent
// is still required for every feature (internal/ai/access).

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

const digitClass = `[0-9۰-۹٠-٩]`

func digits(n string) string { return digitClass + "{" + n + "}" }

var (
	reEmail = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	rePhone = regexp.MustCompile(`(?:(?:\+|00|۰۰|٠٠)(?:98|۹۸|٩٨)[\s\-]?|[0۰٠])?[9۹٩]` + digits("2") +
		`[\s\-]?` + digits("3") + `[\s\-]?` + digits("4"))
	reDashedID = regexp.MustCompile(digits("3") + `-` + digits("6") + `-` + digitClass)
	reLongRun  = regexp.MustCompile(digitClass + `{10,}`)
)

func isDigit(r rune) bool { return unicode.IsDigit(r) }

// replaceBounded replaces every match of re whose neighbours satisfy !edge (so a match is not the middle of a
// longer number / word).
func replaceBounded(s string, re *regexp.Regexp, token string, edge func(rune) bool) string {
	idx := re.FindAllStringIndex(s, -1)
	if len(idx) == 0 {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range idx {
		start, end := m[0], m[1]
		if start < last {
			continue
		}
		if edge != nil {
			if r, _ := utf8.DecodeLastRuneInString(s[:start]); start > 0 && edge(r) {
				continue
			}
			if r, _ := utf8.DecodeRuneInString(s[end:]); end < len(s) && edge(r) {
				continue
			}
		}
		b.WriteString(s[last:start])
		b.WriteString(token)
		last = end
	}
	b.WriteString(s[last:])
	return b.String()
}

// nameTokens are the redactable forms of names: the full trimmed names and each word with ≥ 2 letters, longest
// first (so a full name is replaced before its parts).
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
		add(strings.Join(strings.Fields(n), " "))
		for _, w := range strings.FieldsFunc(n, func(r rune) bool { return unicode.IsSpace(r) || r == '‌' || r == '-' || r == '.' }) {
			add(w)
		}
	}
	slices.SortStableFunc(out, func(a, b string) int { return utf8.RuneCountInString(b) - utf8.RuneCountInString(a) })
	return out
}

// Redact strips PII (see the comment at the top of this file) from text. names are the user's own names.
func Redact(text string, names []string) string {
	if text == "" {
		return text
	}
	text = reEmail.ReplaceAllString(text, RedactedEmail)
	text = replaceBounded(text, rePhone, RedactedPhone, isDigit)
	text = replaceBounded(text, reDashedID, RedactedID, isDigit)
	text = reLongRun.ReplaceAllString(text, RedactedNumber)
	for _, n := range nameTokens(names) {
		re, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(n))
		if err != nil {
			continue
		}
		text = replaceBounded(text, re, RedactedName, isWordRune)
	}
	return text
}

// redactCtx is Redact with the names of the Subject on ctx.
func redactCtx(ctx context.Context, text string) string { return Redact(text, SubjectFrom(ctx).Names) }
