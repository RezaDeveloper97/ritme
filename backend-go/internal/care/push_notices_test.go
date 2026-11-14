package care

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoticeText(t *testing.T) {
	raw := json.RawMessage(`{"fa":"سلام","en":"Hello","de":"Hallo"}`)
	assert.Equal(t, "Hello", noticeText(raw, "en"))
	assert.Equal(t, "سلام", noticeText(raw, "tr"), "fa before en")
	assert.Equal(t, "Hello", noticeText(json.RawMessage(`{"en":"Hello","fa":" "}`), "fa"))
	assert.Equal(t, "Hallo", noticeText(json.RawMessage(`{"tr":"","de":"Hallo"}`), "fa"), "any language, in code order")
	assert.Equal(t, "plain", noticeText(json.RawMessage(`"plain"`), "fa"))
	assert.Empty(t, noticeText(json.RawMessage(`[1]`), "fa"))
	assert.Empty(t, noticeText(nil, "fa"))
}

func TestNoticeURL(t *testing.T) {
	path := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	assert.Equal(t, "/fa/companions", noticeURL(path("/companions"), "fa"))
	assert.Equal(t, "/en/learn/unlocked/7", noticeURL(path("/learn/unlocked/7"), "en"))
	for _, bad := range []string{"", "//evil.example", "https://evil.example", "companions", "/a\\b", "/a\nb"} {
		assert.Equal(t, "/fa/home", noticeURL(path(bad), "fa"), bad)
	}
	assert.Equal(t, "/fa/home", noticeURL(sql.NullString{}, "fa"))
}
