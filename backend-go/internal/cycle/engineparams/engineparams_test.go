package engineparams

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// These tests swap the package-global clinical-config snapshot, so they never run in parallel.

func TestFor_DefaultsWithoutApprovedVersions(t *testing.T) {
	clinicalconfig.Reset()
	for _, d := range []civildate.Date{{}, civildate.MustParse("2026-10-21")} {
		assert.Equal(t, engine.DefaultParams(), For(d), "version 1 of every key == the engine's code defaults")
		assert.Equal(t, "p1", Fingerprint(d))
	}
}

func TestFor_ApprovedVersionAppliesFromItsEffectiveDate(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	clinicalconfig.SetVersions([]clinicalconfig.Version{{
		Key: clinicalconfig.KeyLuteal, Version: 2, Value: clinicalconfig.Value{"default": 13, "min": 10, "max": 16},
		EffectiveFrom: civildate.MustParse("2026-11-01"),
	}})
	before, on := civildate.MustParse("2026-10-31"), civildate.MustParse("2026-11-01")

	assert.Equal(t, engine.DefaultParams(), For(before))
	assert.Equal(t, "p1", Fingerprint(before))

	p := For(on)
	assert.Equal(t, 13, p.Luteal.Default)
	want := engine.DefaultParams()
	want.Luteal.Default = 13
	assert.Equal(t, want, p, "only the approved field changes")
	assert.Equal(t, "p1:engine.luteal@2", Fingerprint(on))
}

// Every field of every key in Keys reaches Params (except engine.fertile_window.after, ignored by v6).
func TestFor_EveryFieldIsRead(t *testing.T) {
	t.Cleanup(clinicalconfig.Reset)
	d := civildate.MustParse("2026-10-21")
	for _, key := range Keys() {
		def, ok := clinicalconfig.Lookup(key)
		require.True(t, ok, key)
		for _, f := range def.Fields {
			if key == clinicalconfig.KeyFertileWindow && f.Name == "after" {
				continue
			}
			v := def.DefaultValue()
			v[f.Name] = f.Max
			if f.Max == f.Default {
				v[f.Name] = f.Min
			}
			clinicalconfig.SetVersions([]clinicalconfig.Version{{Key: key, Version: 2, Value: v, EffectiveFrom: d}})
			assert.NotEqual(t, engine.DefaultParams(), For(d), "%s.%s is not read", key, f.Name)
		}
	}
}

func TestKeys_CoverEveryEngineKey(t *testing.T) {
	keys := Keys()
	for _, def := range clinicalconfig.Defs {
		if strings.HasPrefix(def.Key, "engine.") || strings.HasPrefix(def.Key, "cycle.figo.") ||
			strings.HasPrefix(def.Key, "cycle.teen.") {
			assert.Contains(t, keys, def.Key, "engineparams.Keys misses %s", def.Key)
		}
	}
	for _, k := range keys {
		_, ok := clinicalconfig.Lookup(k)
		assert.True(t, ok, "%s is not in the registry", k)
	}
	assert.Len(t, slices.Compact(slices.Sorted(slices.Values(keys))), len(keys), "no duplicates")
}
