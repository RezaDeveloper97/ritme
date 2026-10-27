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

// CycleEngine is the rollout flag of the v6 engine on /cycle/today and /cycle/date (T-M9-32, DECISIONS.md X-03):
//
//	v1      the v1.1 resolver and the legacy engine (the Laravel-compatible payload; the contract goldens)
//	shadow  serve v1 unchanged and compute v6 after the response in a bounded pool, recording classified differences
//	v6      serve cycle_view from the v6 engine (§2.35 + today_ui + limits) with the legacy keys rebuilt (X-05, D-120)
type CycleEngine string

// The CYCLE_ENGINE values.
const (
	CycleEngineV1     CycleEngine = "v1"
	CycleEngineShadow CycleEngine = "shadow"
	CycleEngineV6     CycleEngine = "v6"
)

// ParseCycleEngine reads a CYCLE_ENGINE value (case-insensitive; "" is the default v1).
func ParseCycleEngine(s string) (CycleEngine, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(CycleEngineV1):
		return CycleEngineV1, true
	case string(CycleEngineShadow):
		return CycleEngineShadow, true
	case string(CycleEngineV6):
		return CycleEngineV6, true
	}
	return CycleEngineV1, false
}

// loadCycleEngine fails the start on an unknown value: a typo must never silently serve another engine.
func loadCycleEngine(e *env) CycleEngine {
	raw := e.raw("CYCLE_ENGINE")
	m, ok := ParseCycleEngine(raw)
	if !ok {
		e.fail("CYCLE_ENGINE: %q must be one of v1, shadow, v6", raw)
	}
	return m
}
