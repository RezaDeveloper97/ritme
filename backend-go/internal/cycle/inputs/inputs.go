// Package inputs loads engine.Input — every v6 engine input of a user — from the database in a fixed budget of
// four queries (T-M9-25, docs/engine-v6 gap 02 §2/§5):
//
//  1. ListEngineInputProfiles: one joined row (declared data, lengths switch, life mode, medications, active
//     pregnancy, postpartum delivery + breastfeeding, active contraception method, teen age band);
//  2. ListEngineInputPeriods: every cycle_histories row, narrow projection;
//  3. ListEngineInputSkips: every cycle_skips row;
//  4. ListEngineInputMarkers: LH / BBT / pregnancy tests (UNION ALL over three tables) inside the marker window.
//
// Load is LoadMany of one id, so a batch of users costs the same four queries (IN lists of ≤ BatchSize ids). The
// mapping rules (mapping.go) are pure and unit tested; Encode/Decode (json.go) are the L2 snapshot format (T-M9-31).
// Nothing here caches (T-M9-31) or is wired into a consumer yet (T-M9-32…38).
package inputs

import (
	"context"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// BatchSize is the largest IN list of one query; LoadMany splits bigger id sets into batches.
const BatchSize = 500

// Querier is the part of the cycle store the loader reads (*store.Queries satisfies it; tests fake it).
type Querier interface {
	ListEngineInputProfiles(ctx context.Context, userIDs []uint64) ([]store.ListEngineInputProfilesRow, error)
	ListEngineInputPeriods(ctx context.Context, userIDs []uint64) ([]store.ListEngineInputPeriodsRow, error)
	ListEngineInputSkips(ctx context.Context, userIDs []uint64) ([]store.ListEngineInputSkipsRow, error)
	ListEngineInputMarkers(ctx context.Context, arg store.ListEngineInputMarkersParams) ([]store.ListEngineInputMarkersRow, error)
}

// Load returns the engine input of uid as of today (the user's civil day; it bounds the marker window). A user
// without any row (no users row, or a users row and nothing else) is the zero Input, not an error.
func Load(ctx context.Context, q Querier, uid uint64, today civildate.Date) (engine.Input, error) {
	m, err := LoadMany(ctx, q, []uint64{uid}, today)
	if err != nil {
		return engine.Input{}, err
	}
	return m[uid], nil
}

// LoadMany returns the engine input of every id (duplicates collapse; a missing user maps to the zero Input) in
// four queries per batch of BatchSize ids. Every user shares today.
func LoadMany(ctx context.Context, q Querier, uids []uint64, today civildate.Date) (map[uint64]engine.Input, error) {
	ids := slices.Clone(uids)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	out := make(map[uint64]engine.Input, len(ids))
	for chunk := range slices.Chunk(ids, BatchSize) {
		if err := loadBatch(ctx, q, chunk, engine.DayOf(today), out); err != nil {
			return nil, err
		}
	}
	for _, id := range ids {
		if _, ok := out[id]; !ok {
			out[id] = engine.Input{}
		}
	}
	return out, nil
}

// loadBatch runs the four queries for ≤ BatchSize distinct ids and stores each found user's Input in out.
func loadBatch(ctx context.Context, q Querier, ids []uint64, today engine.Day, out map[uint64]engine.Input) error {
	profiles, err := q.ListEngineInputProfiles(ctx, ids)
	if err != nil {
		return fmt.Errorf("inputs: profiles: %w", err)
	}
	if len(profiles) == 0 {
		return nil // no such users: zero inputs, no further query
	}
	found := make([]uint64, 0, len(profiles))
	for _, p := range profiles {
		found = append(found, p.UserID)
	}

	periodRows, err := q.ListEngineInputPeriods(ctx, found)
	if err != nil {
		return fmt.Errorf("inputs: periods: %w", err)
	}
	skipRows, err := q.ListEngineInputSkips(ctx, found)
	if err != nil {
		return fmt.Errorf("inputs: skips: %w", err)
	}

	type partial struct {
		in   engine.Input
		from engine.Day // the user's marker window start
	}
	users := make(map[uint64]*partial, len(profiles))
	for _, p := range profiles {
		users[p.UserID] = &partial{in: mapProfile(p)}
	}
	for uid, rows := range groupBy(periodRows, func(r store.ListEngineInputPeriodsRow) uint64 { return r.UserID }) {
		u := users[uid]
		if u == nil {
			continue
		}
		periods, seed := mapPeriods(rows)
		u.in.Periods = periods
		if !u.in.Declared.LastPeriodStart.OK() {
			u.in.Declared.LastPeriodStart = seed // pre-T-M9-27 data: the onboarding seed row is the declared anchor
		}
	}
	for uid, rows := range groupBy(skipRows, func(r store.ListEngineInputSkipsRow) uint64 { return r.UserID }) {
		if u := users[uid]; u != nil {
			u.in.Skips = mapSkips(rows)
		}
	}

	// One marker query for the batch: from the earliest user window to today + 1; each user is then narrowed to her
	// own window in mapMarkers.
	to := today + 1
	from := to
	for _, u := range users {
		u.from = markerWindowFrom(u.in.Periods, today)
		from = min(from, u.from)
	}
	markerRows, err := q.ListEngineInputMarkers(ctx, store.ListEngineInputMarkersParams{
		EntryUserIds: found, FertilityUserIds: found, DailyUserIds: found,
		FromDate: from.Civil(), ToDate: to.Civil(),
	})
	if err != nil {
		return fmt.Errorf("inputs: markers: %w", err)
	}
	for uid, rows := range groupBy(markerRows, func(r store.ListEngineInputMarkersRow) uint64 { return r.UserID }) {
		if u := users[uid]; u != nil {
			u.in.LH, u.in.BBT, u.in.PregnancyTests = mapMarkers(rows, u.from, to)
		}
	}

	for uid, u := range users {
		out[uid] = u.in
	}
	return nil
}

// groupBy splits rows by key, keeping the row order inside each group.
func groupBy[R any](rows []R, key func(R) uint64) map[uint64][]R {
	g := map[uint64][]R{}
	for _, r := range rows {
		k := key(r)
		g[k] = append(g[k], r)
	}
	return g
}
