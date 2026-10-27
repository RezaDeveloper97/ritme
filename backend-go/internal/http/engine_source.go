package http

import (
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/etag"
)

// userStates memoises the consumer state source per Deps (one provider, so one L1 cache, for all small consumers).
var userStates sync.Map // *Deps → state.UserStates

// engineStates is the engine state source of the small consumers (T-M9-36, D-136): nil for CYCLE_ENGINE=v1 (the
// consumers keep their legacy path), the shared v6 state provider for shadow and v6.
func engineStates(r fiber.Router, d *Deps) state.UserStates {
	if d.Config == nil || d.Config.CycleEngine == "" || d.Config.CycleEngine == config.CycleEngineV1 {
		return nil
	}
	if v, ok := userStates.Load(d); ok {
		return v.(state.UserStates)
	}
	var kv state.KV
	if d.Cache != nil {
		kv = d.Cache
	}
	q := cyclestore.New(d.DB)
	p, err := state.New(q, kv, clock.Real{}, state.OptionsFrom(d.Config.EngineCache, etag.Build(), d.Logger))
	if err != nil {
		auth.FailStartup(r, d.Logger, err)
		return nil
	}
	src := state.Source{P: p, Revs: q}
	userStates.Store(d, state.UserStates(src))
	return src
}
