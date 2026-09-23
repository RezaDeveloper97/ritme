package ojson

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, s string) *Value {
	t.Helper()
	v, err := Parse([]byte(s))
	require.NoError(t, err, s)
	return v
}

func diffs(t *testing.T, want, got string, rules *Rules) []string {
	t.Helper()
	var out []string
	for _, d := range Compare(mustParse(t, want), mustParse(t, got), rules) {
		out = append(out, d.String())
	}
	return out
}

func TestCompare_KeyOrderIgnored(t *testing.T) {
	assert.Empty(t, diffs(t, `{"a":1,"b":{"x":[1,2],"y":null}}`, `{"b":{"y":null,"x":[1,2]},"a":1}`, nil))
}

func TestCompare_EscapingEquivalent(t *testing.T) {
	// PHP json_encode escapes "/" and non-ASCII by default; Go does not.
	assert.Empty(t, diffs(t, `{"u":"http:\/\/127.0.0.1\/storage\/a.png","fa":"سلام"}`,
		`{"u":"http://127.0.0.1/storage/a.png","fa":"سلام"}`, nil))
	assert.Empty(t, diffs(t, `"<&>"`, `"<&>"`, nil))
}

func TestCompare_TypesStrict(t *testing.T) {
	cases := []struct{ want, got, msg string }{
		{`"65.50"`, `65.5`, "type: want string"},
		{`[]`, `{}`, "type: want array"},
		{`{}`, `null`, "type: want object"},
		{`[]`, `null`, "type: want array"},
		{`true`, `1`, "type: want bool"},
		{`"1"`, `1`, "type: want string"},
		{`"65.50"`, `"65.5"`, `want "65.50", got "65.5"`},
		{`10`, `10.0`, "want number 10, got 10.0"},
		{`1`, `2`, "want number 1, got 2"},
	}
	for _, c := range cases {
		got := diffs(t, c.want, c.got, nil)
		require.Len(t, got, 1, "%s vs %s", c.want, c.got)
		assert.Contains(t, got[0], c.msg)
	}
}

func TestCompare_NumbersSameKindAndValue(t *testing.T) {
	assert.True(t, NumbersEqual("0.5", "5e-1"))
	assert.True(t, NumbersEqual("-0", "0"))
	assert.False(t, NumbersEqual("10", "1e1"))
	assert.False(t, NumbersEqual("0.1", "0.10000001"))
}

func TestCompare_MissingExtraAndLength(t *testing.T) {
	got := diffs(t, `{"a":1,"list":[1,2,3]}`, `{"b":1,"list":[1,2]}`, nil)
	assert.ElementsMatch(t, []string{
		"$.a: missing (want 1)",
		"$.b: unexpected key (got 1)",
		"$.list: array length: want 3, got 2",
	}, got)
}

func TestCompare_PathRendering(t *testing.T) {
	got := diffs(t, `{"data":{"items":[{"id":1}]}}`, `{"data":{"items":[{"id":"1"}]}}`, nil)
	require.Len(t, got, 1)
	assert.True(t, strings.HasPrefix(got[0], "$.data.items[0].id: type"), got[0])
}

func TestRules_IgnoreAndPatterns(t *testing.T) {
	rules, err := NewRules(
		[]string{"data.items[*].created_at", "**.jti"},
		map[string]string{"data.access_token": `^eyJ[\w-]+\.[\w-]+\.[\w-]+$`, "data.id": `^\d+$`},
	)
	require.NoError(t, err)

	want := `{"data":{"access_token":"<pattern>","id":"<pattern>","jti":"<ignored>","items":[{"created_at":"x"},{"created_at":"y"}]}}`
	got := `{"data":{"access_token":"eyJa.b-c.d_e","id":42,"jti":"zzz","items":[{"created_at":"q"},{"created_at":1}]}}`
	assert.Empty(t, diffs(t, want, got, rules))

	bad := `{"data":{"access_token":"nope","id":[1],"jti":"zzz","items":[{"created_at":"q"},{"created_at":1}]}}`
	out := diffs(t, want, bad, rules)
	require.Len(t, out, 2)
	assert.Contains(t, strings.Join(out, "\n"), "$.data.access_token: value \"nope\" does not match")
	assert.Contains(t, strings.Join(out, "\n"), "$.data.id: pattern ^\\d+$ expects a scalar")
}

func TestRules_IgnoredKeyMustStillExist(t *testing.T) {
	rules, err := NewRules([]string{"data.token"}, nil)
	require.NoError(t, err)
	out := diffs(t, `{"data":{"token":"a"}}`, `{"data":{}}`, rules)
	assert.Equal(t, []string{`$.data.token: missing (want "a")`}, out)
}

func TestRule_Match(t *testing.T) {
	r, err := ParseRule("**.created_at")
	require.NoError(t, err)
	assert.True(t, r.Match(Path{"created_at"}))
	assert.True(t, r.Match(Path{"data", "0", "created_at"}))
	assert.False(t, r.Match(Path{"data", "created_at", "x"}))

	r, err = ParseRule("$.data[0].id")
	require.NoError(t, err)
	assert.True(t, r.Match(Path{"data", "0", "id"}))
	assert.False(t, r.Match(Path{"data", "1", "id"}))

	_, err = ParseRule("a..b")
	require.Error(t, err)
}

func TestMask_ReplacesVolatileAndFlagsMismatch(t *testing.T) {
	rules, err := NewRules([]string{"data.jti"}, map[string]string{"data.token": "^eyJ"})
	require.NoError(t, err)
	v := mustParse(t, `{"data":{"token":"eyJx","jti":"abc","keep":1}}`)
	assert.Empty(t, Mask(v, rules))
	assert.Equal(t, `{"data":{"token":"<pattern:^eyJ>","jti":"<ignored>","keep":1}}`, string(Encode(v, "")))

	v = mustParse(t, `{"data":{"token":"nope","jti":"abc"}}`)
	assert.Len(t, Mask(v, rules), 1)
}

func TestEncode_PreservesOrderAndLiterals(t *testing.T) {
	in := `{"z":1,"a":[],"m":{},"n":10.50,"s":"a\/b <&> é","e":1e-7}`
	v := mustParse(t, in)
	assert.Equal(t, `{"z":1,"a":[],"m":{},"n":10.50,"s":"a/b <&> é","e":1e-7}`, string(Encode(v, "")))
	assert.Equal(t, "{\n  \"z\": 1,\n  \"a\": [],\n  \"m\": {},\n  \"n\": 10.50,\n  \"s\": \"a/b <&> é\",\n  \"e\": 1e-7\n}",
		string(Encode(v, "  ")))
}

func TestParse_RejectsTrailingAndInvalid(t *testing.T) {
	_, err := Parse([]byte(`{"a":1} {"b":2}`))
	require.Error(t, err)
	_, err = Parse([]byte(`<html>`))
	require.Error(t, err)
	_, err = Parse([]byte(" \n[1]\n"))
	require.NoError(t, err)
}
