package i18n_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestPick(t *testing.T) {
	title := raw(`{"fa":"سلام","en":"Hello","ar":""}`)
	assert.JSONEq(t, `"Hello"`, string(i18n.Pick(title, "en", "fa")))
	assert.JSONEq(t, `"سلام"`, string(i18n.Pick(title, "ar", "fa")), "empty ar → default fa")
	assert.JSONEq(t, `"سلام"`, string(i18n.Pick(title, "xx", "fa")), "missing locale → default")
	assert.JSONEq(t, `"Hello"`, string(i18n.Pick(raw(`{"fa":null,"en":"Hello"}`), "ar", "fa")), "→ first non-empty")
	assert.JSONEq(t, `"مرحبا"`, string(i18n.Pick(raw(`{"de":"","ar":"مرحبا"}`), "en", "fa")))
	assert.JSONEq(t, `null`, string(i18n.Pick(raw(`{"fa":"","en":null}`), "en", "fa")))
	assert.JSONEq(t, `"plain"`, string(i18n.Pick(raw(`"plain"`), "en", "fa")), "non-array returned as is")
	assert.JSONEq(t, `null`, string(i18n.Pick(nil, "en", "fa")))
	assert.JSONEq(t, `{"a":1}`, string(i18n.Pick(raw(`{"en":{"a":1},"fa":{"a":2}}`), "en", "fa")))
	assert.JSONEq(t, `0`, string(i18n.Pick(raw(`{"en":0,"fa":1}`), "en", "fa")), `0 is not empty`)
	assert.JSONEq(t, `"x"`, string(i18n.Pick(raw(`["","x"]`), "en", "fa")), "list: first non-empty")

	assert.Equal(t, "سلام", i18n.PickString(title, "ar", "fa"))
	assert.Equal(t, "", i18n.PickString(raw(`{"fa":""}`), "ar", "fa"))
}

func TestPickChain_LocaleFaEn(t *testing.T) {
	v := raw(`{"en":"Hello","fa":"سلام"}`)
	assert.JSONEq(t, `"Hello"`, string(i18n.PickChain(v, append([]string{"en"}, i18n.LegacyPair...)...)))
	assert.JSONEq(t, `"سلام"`, string(i18n.PickChain(v, append([]string{"ar"}, i18n.LegacyPair...)...)))
	assert.JSONEq(t, `""`, string(i18n.PickChain(raw(`{"ar":"","fa":"x"}`), "ar", "fa", "en")), `"" is set for ??`)
	assert.JSONEq(t, `"x"`, string(i18n.PickChain(raw(`{"ar":null,"fa":"x"}`), "ar", "fa", "en")))
	assert.JSONEq(t, `null`, string(i18n.PickChain(raw(`{"de":"x"}`), "ar", "fa", "en")))
	assert.JSONEq(t, `"s"`, string(i18n.PickChain(raw(`"s"`), "ar", "fa", "en")))
}

func TestPickOrWhole_PregnancyWeeklyContent(t *testing.T) {
	v := raw(`{"en":{"title":"Week 8"},"fa":{"title":"هفته ۸"}}`)
	assert.JSONEq(t, `{"title":"هفته ۸"}`, string(i18n.PickOrWhole(v, "fa")))
	assert.JSONEq(t, string(v), string(i18n.PickOrWhole(v, "ar")), "missing locale → the whole object")
	assert.JSONEq(t, `["a"]`, string(i18n.PickOrWhole(raw(`["a"]`), "en")))
}

func TestClean(t *testing.T) {
	assert.JSONEq(t, `{"fa":"x","ar":"y"}`, string(i18n.Clean(raw(`{"fa":"x","en":"","ar":"y","de":null}`))))
	assert.Nil(t, i18n.Clean(raw(`{"fa":"","en":null}`)))
	assert.Nil(t, i18n.Clean(raw(`"x"`)))
}

func TestTranslatableRules_OnlyDefaultRequired(t *testing.T) {
	langs := i18n.Languages{{Code: "fa", IsDefault: true}, {Code: "en"}, {Code: "ar"}}
	rules := i18n.TranslatableRules("title", true, langs, "max:5")

	data, _ := phpval.Decode([]byte(`{"title":{"en":"toolong","ar":5}}`))
	v := validation.Make(lang.Default(), "en", data, rules)
	assert.True(t, v.Fails())
	assert.Equal(t, []string{"title.fa", "title.en", "title.ar"}, v.Fields())
	assert.Equal(t, []string{"The title.fa field is required."}, v.Messages("title.fa"))
	assert.Equal(t, []string{"The title.en field must not be greater than 5 characters."}, v.Messages("title.en"))
	assert.Equal(t, []string{"The title.ar field must be a string."}, v.Messages("title.ar"))

	ok, _ := phpval.Decode([]byte(`{"title":{"fa":"سلام"}}`))
	assert.True(t, validation.Make(lang.Default(), "fa", ok, rules).Passes(), "only the default language is required")

	optional := i18n.TranslatableRules("excerpt", false, langs)
	none, _ := phpval.Decode([]byte(`{}`))
	assert.True(t, validation.Make(lang.Default(), "fa", none, optional).Passes())
}
