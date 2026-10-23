package config

import "strings"

// Engine state cache defaults (T-M9-31, docs/engine-v6/README.md §3).
const (
	DefaultEngineL1Entries    = 20000
	DefaultEngineVerifySample = 1.0
)

// EngineCache holds the v6 engine state cache settings (internal/cycle/state).
type EngineCache struct {
	// Enabled is false only for CYCLE_ENGINE_CACHE=off (the same kill switch as the v1 cache): every state is then
	// loaded from the database and prepared on each call, L1 and L2 untouched.
	Enabled bool
	// L1Entries (ENGINE_L1_ENTRIES, default 20 000, ≥ 1) is the size of the in-process LRU of prepared states.
	L1Entries int
	// Verify (ENGINE_CACHE_VERIFY, default false; tests turn it on) rebuilds a state from the database on a cache hit
	// and counts engine_cache_verify_mismatch_total when it differs.
	Verify bool
	// VerifySample (ENGINE_CACHE_VERIFY_SAMPLE, default 1, 0..1) is the share of hits verified when Verify is on
	// (stage runs a small sample).
	VerifySample float64
}

func loadEngineCache(e *env) EngineCache {
	c := EngineCache{
		Enabled:      !strings.EqualFold(e.raw("CYCLE_ENGINE_CACHE"), "off"),
		L1Entries:    e.integer("ENGINE_L1_ENTRIES", DefaultEngineL1Entries),
		Verify:       e.boolean("ENGINE_CACHE_VERIFY", false),
		VerifySample: e.float("ENGINE_CACHE_VERIFY_SAMPLE", DefaultEngineVerifySample),
	}
	if c.L1Entries < 1 {
		e.fail("ENGINE_L1_ENTRIES: %d must be at least 1", c.L1Entries)
	}
	if c.VerifySample < 0 || c.VerifySample > 1 {
		e.fail("ENGINE_CACHE_VERIFY_SAMPLE: %v must be between 0 and 1", c.VerifySample)
	}
	return c
}
