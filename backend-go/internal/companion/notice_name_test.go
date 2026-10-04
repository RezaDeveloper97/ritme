package companion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoticeName(t *testing.T) {
	for _, tc := range []struct {
		in          string
		fromAccount bool
		want        string
	}{
		{"  Ali   Ahmadi ", true, "Ali Ahmadi"},
		{"علی" + "\u200c" + "رضا", true, "علی" + "\u200c" + "رضا"}, // ZWNJ kept
		{"Ali\u202eimdahA", true, "AliimdahA"},                     // bidi override dropped
		{"Ali\nline", true, "Ali line"},
		{"Ali 2", true, "Ali 2"},
		{"call 0912 345 6789", true, ""},
		{"تماس ۰۹۱۲۳۴۵۶۷۸۹", true, ""}, // Persian digits
		{"0912-345-6789", true, ""},
		{"visit ritme-help.com", true, ""},
		{"https://x.y", true, ""},
		{"me@example.org", true, ""},
		{"t.me/scam", true, ""},
		{"Room 101", false, "Room 101"}, // the owner's own label is trusted
	} {
		assert.Equal(t, tc.want, noticeName(tc.in, tc.fromAccount), tc.in)
	}
	long := noticeName(strings.Repeat("ب", 100), true)
	assert.Equal(t, MaxNoticeNameRunes+1, len([]rune(long)), "clipped + ellipsis")
	assert.True(t, strings.HasSuffix(long, "…"))
}
