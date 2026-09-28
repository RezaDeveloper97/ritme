package profile

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/model"
)

type fakeContent map[string]any

func (f fakeContent) Payload(_ context.Context, group, key, locale string) (any, bool, error) {
	v, ok := f[group+"|"+key+"|"+locale]
	return v, ok, nil
}

func profileWith(weight string, height int16) *model.UserProfile {
	return &model.UserProfile{
		Weight: sql.NullString{String: weight, Valid: weight != ""},
		Height: sql.NullInt16{Int16: height, Valid: height != 0},
	}
}

func TestBmi(t *testing.T) {
	assert.Equal(t, enums.BmiCategoryUnderweight, BmiCategoryOf(18.49))
	assert.Equal(t, enums.BmiCategoryNormal, BmiCategoryOf(18.5))
	assert.Equal(t, enums.BmiCategoryOverweight, BmiCategoryOf(25.0))
	assert.Equal(t, enums.BmiCategoryObese, BmiCategoryOf(30.0))

	msg := phpval.NewMap()
	msg.Set("message", "edited")
	b := Bmi{Content: fakeContent{"bmi_message|normal|en": msg, "bmi_message|normal|fa": "not-a-map"}}

	got, err := b.ForProfile(context.Background(), profileWith("58.50", 162), "en", "fa")
	require.NoError(t, err)
	out, err := jsonx.Marshal(got, jsonx.UnescapedUnicode)
	require.NoError(t, err)
	assert.JSONEq(t, `{"value":22.3,"category":"normal","category_label":"Normal","message":"edited"}`, string(out))

	got, err = b.ForProfile(context.Background(), profileWith("58.50", 162), "fa", "fa")
	require.NoError(t, err)
	m, _ := got.(*jsonx.OrderedMap).Get("message")
	assert.Equal(t, bmiDefaults[enums.BmiCategoryNormal]["fa"], m, "a non-map payload falls back")

	got, err = b.ForProfile(context.Background(), profileWith("58.50", 162), "ar", "fa")
	require.NoError(t, err)
	m, _ = got.(*jsonx.OrderedMap).Get("message")
	assert.Equal(t, bmiDefaults[enums.BmiCategoryNormal]["fa"], m, "unknown locale → default language text")

	got, err = b.ForProfile(context.Background(), profileWith("", 162), "fa", "fa")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestAttributeDirty(t *testing.T) {
	assert.False(t, attributeDirty("60.50", "60.5", model.CastFloat))
	assert.True(t, attributeDirty("60.50", 61.0, model.CastFloat))
	assert.False(t, attributeDirty(int64(165), "165", model.CastInteger))
	assert.False(t, attributeDirty("2026-09-10", "2026-09-10", model.CastDateYMD))
	assert.True(t, attributeDirty("2026-09-10", "2026-09-11", model.CastDateYMD))
	assert.True(t, attributeDirty("non_ttc", nil, ""))
	assert.False(t, attributeDirty(`["pcos"]`, []any{"pcos"}, model.CastArray))
	assert.True(t, attributeDirty(nil, []any{}, model.CastArray))
	assert.False(t, attributeDirty("Sara", "Sara", ""))
	assert.True(t, attributeDirty(nil, "Sara", ""))
}

func TestCycleFieldsChanged(t *testing.T) {
	p := &model.UserProfile{PeriodDuration: sql.NullInt16{Int16: 5, Valid: true}}
	assert.False(t, cycleFieldsChanged(p, map[string]any{"period_duration": "5", "weight": 70}))
	assert.True(t, cycleFieldsChanged(p, map[string]any{"period_duration": int64(6)}))
	assert.False(t, cycleFieldsChanged(p, map[string]any{"birthday": nil}))
	assert.True(t, cycleFieldsChanged(nil, map[string]any{"last_period_start": "2026-09-23"}))
}

func TestNameChangedNotice(t *testing.T) {
	got := NameChangedNotice(7, nil, `<b>"Sara" & 'co'</b>`)
	assert.Equal(t, "✏️ <b>تغییر نام کاربر</b>\nشناسه: <code>7</code>\nقبلی: —\n"+
		"جدید: &lt;b&gt;&quot;Sara&quot; &amp; &#039;co&#039;&lt;/b&gt;", got)
}

// D-22: the fa fallback copy names the app «ریتمی»; en keeps "Ritme".
func TestBmiDefaults_BrandPerLocale(t *testing.T) {
	for cat, byLocale := range bmiDefaults {
		assert.NotContains(t, byLocale["fa"], "Ritme", cat)
	}
	assert.Contains(t, bmiDefaults[enums.BmiCategoryNormal]["fa"], "ریتمی تلاش")
	assert.Contains(t, bmiDefaults[enums.BmiCategoryObese]["fa"], "در ریتمی سعی")
	assert.Contains(t, bmiDefaults[enums.BmiCategoryNormal]["en"], "Ritme aims")
	assert.Contains(t, bmiDefaults[enums.BmiCategoryObese]["en"], "At Ritme")
}
