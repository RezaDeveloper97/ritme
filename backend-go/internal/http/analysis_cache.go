package http

import (
	"sync"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/platform/etag"
)

// analysisCaches holds one analysis.Cache per Deps (T-M10-28), so the four analysis route groups share
// one singleflight and one Redis namespace, like engineStates does for the engine.
var analysisCaches sync.Map

func analysisCache(d *Deps) *analysis.Cache {
	if v, ok := analysisCaches.Load(d); ok {
		return v.(*analysis.Cache)
	}
	var kv analysis.KV
	if d.Cache != nil { // avoid a typed-nil interface
		kv = d.Cache
	}
	v, _ := analysisCaches.LoadOrStore(d, analysis.NewCache(kv, etag.Build(), d.Logger))
	return v.(*analysis.Cache)
}
