package checkups

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func decoded(t *testing.T, raw string) any {
	t.Helper()
	v, err := phpval.Decode([]byte(raw))
	require.NoError(t, err)
	return v
}

func TestTranslated_KeepsActiveLanguagesWithText(t *testing.T) {
	codes := []string{"fa", "en"}
	v := decoded(t, `{"en":"Eye","fa":"چشم","de":"Auge"}`)
	assert.JSONEq(t, `{"fa":"چشم","en":"Eye"}`, string(translated(v, codes)))

	b, err := jsonx.Marshal(translatedObj(v, codes), 0)
	require.NoError(t, err)
	assert.Equal(t, "{\"fa\":\"\\u0686\\u0634\\u0645\",\"en\":\"Eye\"}", string(b), "language order, PHP escaping")

	assert.Nil(t, translated(decoded(t, `{"fa":"","en":null}`), codes))
	assert.Nil(t, translated(nil, codes))
}

func TestSteps_EmptyListIsNull(t *testing.T) {
	data := phpval.NewMap()
	data.Set("prep_steps", decoded(t, `[]`))
	assert.False(t, steps(data, "prep_steps", func(v any) any { return v }).Valid)
	assert.False(t, steps(data, "missing", func(v any) any { return v }).Valid)

	data.Set("prep_steps", decoded(t, `[{"fa":"a","en":"b"}]`))
	col := steps(data, "prep_steps", func(v any) any { return translatedObj(v, []string{"fa", "en"}) })
	require.True(t, col.Valid)
	assert.JSONEq(t, `[{"fa":"a","en":"b"}]`, string(col.V))
}

func TestRules_OptionalTextsBecomeAllLanguagesOnceSent(t *testing.T) {
	codes := []string{"fa", "en"}
	attrs := func(sub bool) map[string]string {
		in := phpval.NewMap()
		if sub {
			in.Set("subtitle", decoded(t, `{"fa":"x"}`))
		}
		out := map[string]string{}
		for _, f := range rules(in, codes, true) {
			names := ""
			for _, r := range f.Rules {
				names += r.Name + ","
			}
			out[f.Attr] = names
		}
		return out
	}
	off, on := attrs(false), attrs(true)
	assert.Contains(t, off["subtitle.en"], "Nullable")
	assert.Contains(t, on["subtitle.en"], "Required")
	assert.Contains(t, on["title.en"], "Required", "title needs every active language")
	assert.Contains(t, on, "key")
	for _, f := range rules(phpval.NewMap(), codes, false) {
		assert.NotEqual(t, "key", f.Attr, "the key is fixed after create")
	}
}
