package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M9-31: engine state cache settings.
func TestLoad_EngineCache(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, EngineCache{Enabled: true, L1Entries: 20000, Verify: false, VerifySample: 1}, cfg.EngineCache)

	env := minimal()
	env["CYCLE_ENGINE_CACHE"] = "OFF"
	env["ENGINE_L1_ENTRIES"] = "500"
	env["ENGINE_CACHE_VERIFY"] = "1"
	env["ENGINE_CACHE_VERIFY_SAMPLE"] = "0.05"
	cfg, err = LoadFrom(lookup(env))
	require.NoError(t, err)
	assert.Equal(t, EngineCache{Enabled: false, L1Entries: 500, Verify: true, VerifySample: 0.05}, cfg.EngineCache)

	env = minimal()
	env["ENGINE_L1_ENTRIES"] = "0"
	env["ENGINE_CACHE_VERIFY_SAMPLE"] = "2"
	_, err = LoadFrom(lookup(env))
	require.Error(t, err)
	assert.ErrorContains(t, err, "ENGINE_L1_ENTRIES")
	assert.ErrorContains(t, err, "ENGINE_CACHE_VERIFY_SAMPLE")
}

// T-M9-32: the CYCLE_ENGINE rollout flag (X-03): default v1, invalid → fail at start.
func TestLoad_CycleEngine(t *testing.T) {
	cfg, err := LoadFrom(lookup(minimal()))
	require.NoError(t, err)
	assert.Equal(t, CycleEngineV1, cfg.CycleEngine)

	for _, tc := range []struct {
		raw  string
		want CycleEngine
	}{{"v1", CycleEngineV1}, {"SHADOW", CycleEngineShadow}, {" v6 ", CycleEngineV6}, {"", CycleEngineV1}} {
		raw, want := tc.raw, tc.want
		env := minimal()
		env["CYCLE_ENGINE"] = raw
		cfg, err := LoadFrom(lookup(env))
		require.NoError(t, err, raw)
		assert.Equal(t, want, cfg.CycleEngine, raw)
	}

	env := minimal()
	env["CYCLE_ENGINE"] = "v7"
	_, err = LoadFrom(lookup(env))
	require.Error(t, err)
	assert.ErrorContains(t, err, "CYCLE_ENGINE")
}
