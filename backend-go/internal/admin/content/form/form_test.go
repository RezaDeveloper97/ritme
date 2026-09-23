package form

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestIsURL(t *testing.T) {
	for _, ok := range []string{"https://ritme.app", "http://a.b/c?d=1", "ftp://files.example.com/x"} {
		assert.True(t, IsURL(ok), ok)
	}
	for _, bad := range []string{"", "ritme.app", "javascript:alert(1)", "https://", "http://a b", "/internal/path", "data:text/html,x"} {
		assert.False(t, IsURL(bad), bad)
	}
}

func TestParseTime(t *testing.T) {
	for in, want := range map[string]string{
		"2026-09-23":                "2026-09-23T00:00:00+03:30",
		"2026-09-23 10:05":          "2026-09-23T10:05:00+03:30",
		"2026-09-23T10:05:07":       "2026-09-23T10:05:07+03:30",
		"2026-09-23T06:35:00Z":      "2026-09-23T10:05:00+03:30",
		"2026-09-23T10:05:00+03:30": "2026-09-23T10:05:00+03:30",
	} {
		got, ok := ParseTime(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got.Format("2006-01-02T15:04:05-07:00"), in)
	}
	_, ok := ParseTime("tomorrow")
	assert.False(t, ok)
}

func TestLikePatterns(t *testing.T) {
	assert.Equal(t, "%", Exact(""))
	assert.Equal(t, `a\%b\_c\\`, Exact(`a%b_c\`))
	assert.Equal(t, `%x\%%`, Contains("x%"))
	assert.Equal(t, `%\\u0622\\u0628%`, ContainsJSON("آب"), "json_encode escapes, LIKE-escaped")
	assert.Equal(t, `%a\\/b%`, ContainsJSON("a/b"))
}

func TestKeepAndInts(t *testing.T) {
	m := phpval.NewMap()
	m.Set("n", "12")
	m.Set("big", "99999999999")
	m.Set("null", nil)
	assert.EqualValues(t, 12, Int(m, "n", 0))
	assert.EqualValues(t, 2147483647, Int(m, "big", 0))
	assert.EqualValues(t, 7, Int(m, "missing", 7))
	assert.EqualValues(t, 7, Int(m, "null", 7))
	assert.False(t, Str(m, "null").Valid)
	assert.False(t, KeepStr(m, "null", Str(m, "n")).Valid, "sent null clears")
	assert.Equal(t, "12", KeepStr(m, "missing", Str(m, "n")).String, "absent keeps")
	assert.True(t, NullInt16(m, "big").Valid)
	assert.EqualValues(t, 32767, NullInt16(m, "big").Int16)

	tr := phpval.NewMap()
	tr.Set("fa", "x")
	tr.Set("en", nil)
	m.Set("title", tr)
	assert.JSONEq(t, `{"fa":"x","en":null}`, string(NullJSON(m, "title").V))
	assert.JSONEq(t, `{"fa":"x"}`, string(Clean(m, "title").V))
	empty := phpval.NewMap()
	empty.Set("fa", nil)
	m.Set("blank", empty)
	assert.False(t, Clean(m, "blank").Valid)
}
