package taxonomy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/ritme/backend-go/internal/healthlog/taxonomy" //nolint:revive // test reads like the package
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/resources/translations"
)

func TestMenopausePreset_EverySlotIsLoggableInMenopause(t *testing.T) {
	seen := map[Ref]bool{}
	for _, g := range MenopausePreset() {
		require.NotEmpty(t, g.Refs, g.Code)
		for _, r := range g.Refs {
			assert.False(t, seen[r], "duplicate %v", r)
			seen[r] = true
			c, ok := CategoryByCode(r.Category)
			require.True(t, ok, r.Category)
			p, ok := c.Param(r.Param)
			require.True(t, ok, "%s.%s", r.Category, r.Param)
			assert.True(t, c.Available(p, ModeMenopause), "%v not in menopause mode", r)
			if r.Item == "" {
				continue
			}
			assert.Equal(t, Items, p.Type, "%v", r)
			o, ok := p.Option(r.Item)
			require.True(t, ok, "%v", r)
			assert.True(t, o.OptionAvailable(ModeMenopause), "%v not offered in menopause mode", r)
		}
	}
	assert.Len(t, MenopauseSymptoms(), 13, "nbl_Meno_Log has 13 symptoms")
}

func TestMenopauseItems_HiddenInOtherModes(t *testing.T) {
	c, _ := CategoryByCode("symptoms")
	p, _ := c.Param("general")
	for _, code := range []string{"anxiety", "irritability", "low_mood"} {
		o, ok := p.Option(code)
		require.True(t, ok)
		assert.False(t, o.OptionAvailable(ModeCycle), code)
	}
	b, _ := CategoryByCode("bleeding")
	presence, _ := b.Param("presence")
	assert.False(t, b.Available(presence, ModeCycle))
	assert.True(t, presence.Alert)
	m, _ := CategoryByCode("menopause")
	assert.Equal(t, []string{ModeMenopause}, m.Modes)
}

func TestMenopausePreset_GroupLabels(t *testing.T) {
	store := i18n.NewTranslationStore(translations.FS, "")
	for _, code := range []string{"fa", "en"} {
		ns := store.RawNamespace(code, "log-taxonomy")
		for _, g := range MenopausePreset() {
			v, ok := phpval.Get(ns, "presets.menopause."+g.Code)
			s, _ := v.(string)
			assert.True(t, ok && s != "", "%s: missing presets.menopause.%s", code, g.Code)
		}
	}
}
