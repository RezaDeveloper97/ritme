package state

import (
	dto "github.com/prometheus/client_model/go"

	"github.com/ritme/backend-go/internal/cycle/engine"
)

// SetPrepare replaces engine.Prepare (tests count the calls).
func SetPrepare(p *Provider, f func(engine.Input, engine.Params) *engine.State) { p.prepare = f }

// InjectL1 stores st as the L1 entry of (uid, rev, fp) — a stale entry for the verify test.
func InjectL1(p *Provider, uid, rev uint64, fp string, st *engine.State) {
	p.l1.Add(l1Key{uid: uid, rev: rev, fp: fp, build: p.opts.Build}, st)
}

// PurgeL1 empties L1.
func PurgeL1(p *Provider) { p.l1.Purge() }

// L2Key is the unprefixed Redis key of uid's snapshot at rev.
func L2Key(uid, rev uint64) string { return l2Key(uid, rev) }

// VerifyMismatches reads engine_cache_verify_mismatch_total.
func VerifyMismatches() float64 {
	var m dto.Metric
	if err := verifyMismatch.Write(&m); err != nil {
		return -1
	}
	return m.GetCounter().GetValue()
}
