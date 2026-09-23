package validation_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The goldens in testdata/ were recorded from Laravel by testdata/capture.php
// (clock 2026-09-23 10:00 Asia/Tehran, languages fa/en/ar).
var captureNow = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

func loadJSON(t *testing.T, name string) phpval.Map {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	v, err := phpval.Decode(b)
	require.NoError(t, err)
	if list, ok := v.([]any); ok { // a list of cases keyed by "name"
		m := phpval.NewMap()
		for _, c := range list {
			n, _ := c.(phpval.Map).Get("name")
			m.Set(n.(string), c)
		}
		return m
	}
	return v.(phpval.Map)
}

func get(m phpval.Map, key string) any {
	v, _ := m.Get(key)
	return v
}

// rulesFrom builds a rules array from the JSON form {"attr": "a|b" | ["a", "b"]}.
func rulesFrom(v any) validation.Rules {
	m, _ := v.(phpval.Map)
	var rules validation.Rules
	if m == nil {
		return rules
	}
	for _, attr := range m.Keys() {
		switch r := get(m, attr).(type) {
		case string:
			rules = append(rules, validation.F(attr, r))
		case []any:
			list := make([]string, len(r))
			for i, x := range r {
				list[i] = x.(string)
			}
			rules = append(rules, validation.F(attr, list))
		}
	}
	return rules
}

// kvFrom flattens an ordered {"key": "value"} map to key/value pairs.
func kvFrom(v any) []string {
	m, _ := v.(phpval.Map)
	var kv []string
	if m == nil {
		return kv
	}
	for _, k := range m.Keys() {
		kv = append(kv, k, get(m, k).(string))
	}
	return kv
}

func canonical(t *testing.T, v any) string {
	t.Helper()
	b, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	require.NoError(t, err)
	return string(b)
}

func TestEngine_MatchesLaravel(t *testing.T) {
	cases := loadJSON(t, "engine_cases.json")
	golden := loadJSON(t, "engine_golden.json")
	tr := lang.Default()

	for _, name := range cases.Keys() {
		c := get(cases, name).(phpval.Map)
		want, ok := get(golden, name).(phpval.Map)
		require.True(t, ok, "no golden for %s: re-run testdata/capture.php", name)
		t.Run(name, func(t *testing.T) {
			v := validation.Make(tr, get(c, "locale").(string), get(c, "data"), rulesFrom(get(c, "rules")),
				validation.Now(captureNow),
				validation.Messages(kvFrom(get(c, "messages"))...),
				validation.Attributes(kvFrom(get(c, "attributes"))...))

			assert.Equal(t, get(want, "passes"), v.Passes())
			assert.Equal(t, canonical(t, get(want, "errors")), canonical(t, v.ErrorBag()), "errors")
			assert.Equal(t, get(want, "first"), v.First())
			if get(want, "passes") == true {
				assert.Equal(t, canonical(t, get(want, "validated")), canonical(t, v.Validated()), "validated")
			} else {
				assert.Nil(t, v.Validated())
				require.NotNil(t, v.Errors())
			}
		})
	}
}
