package civildate

import (
	"context"
	"time"
)

// The user-local day (engine v6 §2.3, D-126): "a day is the user's local day from the device timezone (default
// Asia/Tehran), not server UTC". The usertz middleware attaches the request's zone to the context (X-Timezone header
// → stored user_profiles.timezone → Asia/Tehran); LocalToday / LocalNow read it. Without that middleware — a
// background job, a test, a call site that intentionally stays on Tehran — the zone is Tehran and LocalToday is
// exactly Today.

type zoneKey struct{}

// ZoneResolver returns the request's zone. It may be lazy (the stored zone is read only when a handler asks) and
// must return a non-nil location.
type ZoneResolver func(ctx context.Context) *time.Location

// WithZone returns ctx carrying the request's zone resolver.
func WithZone(ctx context.Context, r ZoneResolver) context.Context {
	return context.WithValue(ctx, zoneKey{}, r)
}

// ZoneKey is the key the resolver is stored under, for middlewares that also keep it in fiber Locals (a fiber.Ctx's
// Value reads its Locals).
func ZoneKey() any { return zoneKey{} }

// Zone is the user's zone for ctx: the resolver's answer, Asia/Tehran without one.
func Zone(ctx context.Context) *time.Location {
	if ctx != nil {
		if r, ok := ctx.Value(zoneKey{}).(ZoneResolver); ok && r != nil {
			if loc := r(ctx); loc != nil {
				return loc
			}
		}
	}
	return Tehran
}

// LocalNow is the request clock's instant in the user's zone.
func LocalNow(ctx context.Context, n Nower) time.Time { return n.Now().In(Zone(ctx)) }

// LocalToday is the user's calendar day (Today, but in the request's zone).
func LocalToday(ctx context.Context, n Nower) Date { return FromTime(LocalNow(ctx, n)) }
