package sanitizer

import (
	"regexp"
	"strings"
)

var (
	blockEnd = regexp.MustCompile(`(?i)<(br|/p|/div|/li|/h[1-6])[^>]*>`)
	// PCRE /\s+/u (PHP enables UCP with /u): ASCII whitespace, NEL and every Unicode
	// separator (Zs, Zl, Zp).
	unicodeSpace = regexp.MustCompile(`[\t\n\x0B\f\r \x{85}\p{Z}]+`)
)

// PlainText is HtmlSanitizer::toPlainText: readable one-line text for a rich-text
// value (block ends become spaces, tags stripped, entities decoded, whitespace
// collapsed). ok=false is PHP's null (empty input or nothing left).
func PlainText(html string) (string, bool) {
	if html == "" {
		return "", false
	}
	spaced := blockEnd.ReplaceAllString(html, " ")
	text := DecodeEntities(StripTags(spaced))
	text = strings.Trim(unicodeSpace.ReplaceAllString(text, " "), " \t\n\r\x00\x0B")
	if text == "" {
		return "", false
	}
	return text, true
}
