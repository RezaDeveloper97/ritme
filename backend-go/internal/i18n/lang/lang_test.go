package lang_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestTrans_LocaleThenEnglishFallbackThenKey(t *testing.T) {
	tr := lang.Default()
	first := map[string]string{"first": "x"}
	assert.Equal(t, "خطای اعتبارسنجی: x", tr.Trans("profile.validation_failed", first, "fa"))
	assert.Equal(t, "Validation failed: x", tr.Trans("profile.validation_failed", first, "en"))
	assert.Equal(t, "Validation failed: x", tr.Trans("profile.validation_failed", first, "ar"), "ar has no files → en")
	assert.Equal(t, "profile.nope", tr.Trans("profile.nope", nil, "fa"))
	assert.Equal(t, "nogroup.key", tr.Trans("nogroup.key", nil, "en"))
	assert.Equal(t, "profile.errors", tr.Trans("profile.errors", nil, "en"), "an array line is not a string")

	// fa has no "between" line: the framework's English line is used.
	assert.Equal(t, "The :attribute field must be between :min and :max.", tr.Trans("validation.between.numeric", nil, "fa"))
	assert.Equal(t, ":attribute باید حداقل :min باشد.", tr.Trans("validation.min.numeric", nil, "fa"))
}

func TestGet_ArraysAndEmptyArrays(t *testing.T) {
	tr := lang.Default()
	custom, ok := tr.Get("validation.custom", "fa")
	require.True(t, ok)
	_, isMap := custom.(phpval.Map)
	assert.True(t, isMap)

	// en's validation.attributes is [] — an empty array counts as missing (Translator::getLine).
	_, ok = tr.Get("validation.attributes", "en")
	assert.False(t, ok)
	_, ok = tr.Get("validation.attributes", "ar")
	assert.False(t, ok)
	attrs, ok := tr.Get("validation.attributes.log_date", "fa")
	require.True(t, ok)
	assert.Equal(t, "تاریخ ثبت", attrs)
}

func TestMakeReplacements_StrtrCases(t *testing.T) {
	assert.Equal(t, "a Bob BOB bob :x", lang.MakeReplacements("a :Name :NAME :name :x", map[string]string{"name": "bob"}))
	// longest placeholder wins, replaced text is not rescanned
	assert.Equal(t, "1-2 :ab", lang.MakeReplacements(":ab-:a", map[string]string{"ab": "1", "a": "2 :ab"}))
	assert.Equal(t, "no params", lang.MakeReplacements("no params", nil))
	assert.Equal(t, "Éa", lang.UcFirst("éa"))
}
