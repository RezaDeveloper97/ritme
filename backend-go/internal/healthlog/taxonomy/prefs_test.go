package taxonomy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
)

func TestDefaultTiles_PerMode(t *testing.T) {
	cases := map[string][]string{
		// nbl_Log_Sheet_Cycle
		taxonomy.ModeCycle: {"bleeding", "pain", "mood", "symptoms", "discharge", "sleep", "measurements.weight", "note"},
		taxonomy.ModeTTC:   {"measurements.bbt", "measurements.lh_test", "discharge", "sex", "bleeding", "mood", "symptoms", "note"},
		// nbl_Log_Sheet_Preg
		taxonomy.ModePregnancy: {"pregnancy.kicks", "pregnancy.contractions", "symptoms", "mood", "measurements.weight",
			"measurements.bp_systolic", "sleep", "note"},
		// nbl_Log_Sheet_Post
		taxonomy.ModePostpartum: {"baby.feeding", "bleeding", "pain", "mood", "sleep", "measurements.weight", "meds", "note"},
		taxonomy.ModeMenopause:  {"symptoms", "sleep", "mood", "bleeding", "pain", "urogenital", "measurements.weight", "note"},
		taxonomy.ModeTeen:       {"bleeding", "pain", "mood", "symptoms", "discharge", "sleep", "measurements.weight", "note"},
	}
	for mode, want := range cases {
		got := taxonomy.DefaultTiles(mode, "", nil)
		assert.Equal(t, want, got, mode)
	}
}

func TestDefaultTiles_AlwaysValid(t *testing.T) {
	phases := []string{"", taxonomy.PhaseMenstrual, taxonomy.PhaseFollicular, taxonomy.PhaseFertile, taxonomy.PhaseLuteal,
		taxonomy.PhasePeriodExpected, "unknown"}
	for _, mode := range taxonomy.AllModes {
		for _, phase := range phases {
			got := taxonomy.DefaultTiles(mode, phase, nil)
			assert.Len(t, got, taxonomy.MaxPinned, "%s/%s", mode, phase)
			seen := map[string]bool{}
			for _, k := range got {
				assert.True(t, taxonomy.TileAvailable(k, mode), "%s/%s: %s", mode, phase, k)
				assert.False(t, seen[k], "%s/%s: duplicate %s", mode, phase, k)
				seen[k] = true
			}
		}
	}
}

func TestDefaultTiles_ByPhase(t *testing.T) {
	assert.Equal(t, "bleeding", taxonomy.DefaultTiles(taxonomy.ModeCycle, taxonomy.PhaseMenstrual, nil)[0])
	assert.Equal(t, "discharge", taxonomy.DefaultTiles(taxonomy.ModeCycle, taxonomy.PhaseFertile, nil)[0])
	assert.Equal(t, "mood", taxonomy.DefaultTiles(taxonomy.ModeCycle, taxonomy.PhaseFollicular, nil)[0])
	assert.Equal(t, "measurements.lh_test", taxonomy.DefaultTiles(taxonomy.ModeTTC, taxonomy.PhaseFertile, nil)[0])
	assert.Equal(t, "measurements.pregnancy_test", taxonomy.DefaultTiles(taxonomy.ModeTTC, taxonomy.PhasePeriodExpected, nil)[0])
	// unknown phase → the mode default
	assert.Equal(t, taxonomy.DefaultTiles(taxonomy.ModeCycle, "", nil), taxonomy.DefaultTiles(taxonomy.ModeCycle, "unknown", nil))
	// phase is ignored outside the cycle modes
	assert.Equal(t, taxonomy.DefaultTiles(taxonomy.ModePregnancy, "", nil), taxonomy.DefaultTiles(taxonomy.ModePregnancy, taxonomy.PhaseFertile, nil))

	// teen in the fertile phase: no `sex` tile, topped up instead
	teen := taxonomy.DefaultTiles(taxonomy.ModeTeen, taxonomy.PhaseFertile, nil)
	assert.NotContains(t, teen, "sex")
	assert.Len(t, teen, taxonomy.MaxPinned)
}

func TestDefaultTiles_HiddenAreSkippedAndToppedUp(t *testing.T) {
	got := taxonomy.DefaultTiles(taxonomy.ModeCycle, "", []string{"discharge", "measurements"})
	assert.NotContains(t, got, "discharge")
	assert.NotContains(t, got, "measurements.weight")
	assert.Len(t, got, taxonomy.MaxPinned)
	assert.Equal(t, []string{"bleeding", "pain", "mood", "symptoms", "sleep", "note", "appetite_energy", "skin_hair"}, got)
}

func TestTileAvailable(t *testing.T) {
	assert.True(t, taxonomy.TileAvailable("pain", taxonomy.ModeCycle))
	assert.True(t, taxonomy.TileAvailable("measurements.bbt", taxonomy.ModeTTC))
	assert.False(t, taxonomy.TileAvailable("measurements.bbt", taxonomy.ModePregnancy), "bbt is fertile modes only")
	assert.False(t, taxonomy.TileAvailable("pregnancy", taxonomy.ModeCycle))
	assert.False(t, taxonomy.TileAvailable("pain.nope", taxonomy.ModeCycle))
	assert.False(t, taxonomy.TileAvailable("horoscope", taxonomy.ModeCycle))
	assert.Contains(t, taxonomy.TileKeys(taxonomy.ModePregnancy), "pregnancy.kicks")
	assert.NotContains(t, taxonomy.TileKeys(taxonomy.ModeCycle), "baby")
}

func TestResolve_Defaults(t *testing.T) {
	e := taxonomy.Resolve(taxonomy.ModeCycle, taxonomy.PhaseLuteal, nil)
	assert.True(t, e.Default)
	assert.Equal(t, taxonomy.PhaseLuteal, e.Phase)
	assert.Equal(t, taxonomy.ModeCategories(taxonomy.ModeCycle), e.Order)
	assert.Empty(t, e.Hidden)
	assert.Equal(t, taxonomy.DefaultTiles(taxonomy.ModeCycle, taxonomy.PhaseLuteal, nil), e.Pinned)

	p := taxonomy.Resolve(taxonomy.ModePregnancy, taxonomy.PhaseLuteal, nil)
	assert.Empty(t, p.Phase, "no phase outside the cycle modes")
	assert.Equal(t, "pregnancy", p.Order[len(p.Order)-3], "pregnancy category shows in its mode")
}

func TestResolve_Stored(t *testing.T) {
	e := taxonomy.Resolve(taxonomy.ModeCycle, taxonomy.PhaseMenstrual, &taxonomy.Stored{
		Order:  []string{"note", "mood", "baby", "mood", "horoscope"},
		Hidden: []string{"sex", "baby"},
	})
	assert.False(t, e.Default)
	assert.True(t, e.Custom.Order)
	assert.True(t, e.Custom.Hidden)
	assert.False(t, e.Custom.Pinned)
	require.Len(t, e.Order, len(taxonomy.ModeCategories(taxonomy.ModeCycle)))
	assert.Equal(t, []string{"note", "mood", "bleeding", "pain"}, e.Order[:4], "hers first (unknown/other-mode dropped), the rest after")
	assert.Equal(t, []string{"sex"}, e.Hidden)
	assert.Equal(t, taxonomy.DefaultTiles(taxonomy.ModeCycle, taxonomy.PhaseMenstrual, []string{"sex"}), e.Pinned,
		"tiles stay smart until she pins her own")

	e = taxonomy.Resolve(taxonomy.ModeCycle, "", &taxonomy.Stored{
		Pinned: []string{"note", "pain", "note", "measurements.bbt", "sex", "baby.feeding"},
		Hidden: []string{"sex"},
	})
	assert.Equal(t, []string{"note", "pain", "measurements.bbt"}, e.Pinned, "dups, hidden and other-mode tiles dropped")

	e = taxonomy.Resolve(taxonomy.ModeCycle, "", &taxonomy.Stored{Pinned: []string{}})
	assert.Empty(t, e.Pinned, "an empty pin list is honoured")
	assert.NotNil(t, e.Pinned)
}

func TestCustomParams(t *testing.T) {
	want := map[string]string{
		"custom": "items", "symptoms": "general", "mood": "moods", "activity": "types",
		"appetite_energy": "cravings", "urogenital": "symptoms", "skin_hair": "symptoms",
	}
	got := map[string]string{}
	for _, c := range taxonomy.Categories() {
		if p := c.CustomParam(); p != nil {
			got[c.Code] = p.Code
			assert.Contains(t, []taxonomy.Type{taxonomy.Items, taxonomy.Multi}, p.Type, c.Code)
		}
	}
	assert.Equal(t, want, got)
	assert.Equal(t, "custom_12", taxonomy.CustomItemCode(12))
}
