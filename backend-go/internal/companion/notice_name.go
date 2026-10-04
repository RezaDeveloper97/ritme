package companion

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxNoticeNameRunes clips the companion name embedded in an owner-inbox notice (B-N4-08b, CMP-L5).
const MaxNoticeNameRunes = 40

// noticeLinkRe finds things a name has no business carrying in an app notification: a URL, a domain, an e-mail or a
// handle.
var noticeLinkRe = regexp.MustCompile(`(?i)(https?:|www\.|[a-z0-9-]+\.(com|ir|net|org|me|io|app|info|link|xyz)\b|@|t\.me)`)

// noticeName is the companion name as an owner notice may show it: the companion chooses users.name, so a name that
// carries a run of 3+ digits (a phone number), a URL / domain / e-mail / handle is dropped ("" = the neutral word);
// otherwise control and bidi-override characters are removed, whitespace collapsed and the name clipped to
// MaxNoticeNameRunes. The owner's own label for the link is trusted and only cleaned and clipped.
func noticeName(name string, fromAccount bool) string {
	var b strings.Builder
	digits := 0
	for _, r := range name {
		switch {
		case unicode.IsDigit(r):
			digits++
			if fromAccount && digits >= 3 {
				return ""
			}
		case !unicode.IsSpace(r):
			digits = 0
		}
		if unicode.IsSpace(r) {
			b.WriteRune(' ')
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) { // control, bidi overrides, zero-width marks
			if r == '‌' { // ZWNJ is part of Persian spelling
				b.WriteRune(r)
			}
			continue
		}
		b.WriteRune(r)
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if fromAccount && noticeLinkRe.MatchString(out) {
		return ""
	}
	if utf8.RuneCountInString(out) > MaxNoticeNameRunes {
		out = strings.TrimSpace(string([]rune(out)[:MaxNoticeNameRunes])) + "…"
	}
	return out
}
