package clinicalconfig

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Value is a key's value: named numbers.
type Value map[string]float64

// Version is one approved version of a key (version 1 = the code default, EffectiveFrom zero). A numbers key carries
// Value, a rows key Rows. Source is the version's own source: Def.Source for version 1, the stored source column for
// v2+ (Def.Source when that column is empty).
type Version struct {
	Key           string
	Version       int
	Value         Value
	Rows          []Row
	Source        string
	EffectiveFrom civildate.Date
}

// Loader reads every approved stored version (v2+).
type Loader func(ctx context.Context) ([]Version, error)

// RefreshEvery is how stale the cached snapshot may get before a lookup triggers a background reload (another API
// instance may have approved a version). An approval in this instance refreshes at once.
const RefreshEvery = time.Minute

const loadTimeout = 3 * time.Second

// snapshot holds the approved versions per key, ordered by effective date then version.
type snapshot struct {
	byKey map[string][]Version
}

func newSnapshot(vs []Version) *snapshot {
	s := &snapshot{byKey: map[string][]Version{}}
	for _, v := range vs {
		def, ok := byKey[v.Key]
		if !ok || v.Version < 2 || v.EffectiveFrom.IsZero() || (def.IsRows() && len(v.Rows) == 0) {
			continue
		}
		if v.Source == "" {
			v.Source = def.Source
		}
		s.byKey[v.Key] = append(s.byKey[v.Key], v)
	}
	for _, list := range s.byKey {
		sort.SliceStable(list, func(i, j int) bool {
			if c := list[i].EffectiveFrom.Compare(list[j].EffectiveFrom); c != 0 {
				return c < 0
			}
			return list[i].Version < list[j].Version
		})
	}
	return s
}

// at is the stored version valid on d (nil = version 1).
func (s *snapshot) at(key string, d civildate.Date) *Version {
	if s == nil || d.IsZero() {
		return nil
	}
	list := s.byKey[key]
	for i := len(list) - 1; i >= 0; i-- {
		if !list[i].EffectiveFrom.After(d) {
			return &list[i]
		}
	}
	return nil
}

var state struct {
	snap       atomic.Pointer[snapshot]
	loadedAt   atomic.Int64
	refreshing atomic.Bool
	mu         sync.Mutex
	loader     Loader
	logger     *slog.Logger
}

// Install sets the loader (the admin store) and loads once synchronously. A failed load keeps the code defaults.
func Install(l Loader, logger *slog.Logger) {
	state.mu.Lock()
	state.loader, state.logger = l, logger
	state.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	defer cancel()
	_ = Refresh(ctx)
}

// Refresh reloads the snapshot now. On error the previous snapshot stays.
func Refresh(ctx context.Context) error {
	state.mu.Lock()
	l, logger := state.loader, state.logger
	state.mu.Unlock()
	state.loadedAt.Store(time.Now().UnixNano())
	if l == nil {
		return nil
	}
	vs, err := l(ctx)
	if err != nil {
		if logger != nil {
			logger.Warn("clinical config: load failed, keeping the previous thresholds", slog.String("error", err.Error()))
		}
		return err
	}
	state.snap.Store(newSnapshot(vs))
	return nil
}

// SetVersions replaces the snapshot without a loader (tests, tools).
func SetVersions(vs []Version) {
	state.mu.Lock()
	state.loader = nil
	state.mu.Unlock()
	state.snap.Store(newSnapshot(vs))
}

// Reset drops the loader and every stored version (back to the code defaults).
func Reset() { SetVersions(nil) }

func current() *snapshot {
	if time.Since(time.Unix(0, state.loadedAt.Load())) > RefreshEvery && state.refreshing.CompareAndSwap(false, true) {
		go func() {
			defer state.refreshing.Store(false)
			ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
			defer cancel()
			_ = Refresh(ctx)
		}()
	}
	return state.snap.Load()
}

// At is the version of key valid on d: the approved stored version with the latest effective date on or before d,
// else version 1 (the code default). A zero d always answers version 1. Unknown keys answer an empty version 0.
func At(key string, d civildate.Date) Version {
	def, ok := byKey[key]
	if !ok {
		return Version{Key: key}
	}
	if v := current().at(key, d); v != nil {
		return *v
	}
	return Version{Key: key, Version: 1, Value: defaults[def.Key], Rows: defaultRows[def.Key], Source: def.Source}
}

// Rows is the table of a rows key valid on d (nil for a numbers or unknown key). Callers never mutate it.
func Rows(key string, d civildate.Date) []Row { return At(key, d).Rows }

// SourceOf is the source of the version of key valid on d ("" for an unknown key), so a payload can cite it.
func SourceOf(key string, d civildate.Date) string { return At(key, d).Source }

// Num is one field of key valid on d; a field missing from a stored version falls back to its code default.
func Num(key, field string, d civildate.Date) float64 {
	if v, ok := At(key, d).Value[field]; ok {
		return v
	}
	if def, ok := byKey[key]; ok {
		for _, f := range def.Fields {
			if f.Name == field {
				return f.Default
			}
		}
	}
	return 0
}

// Int is Num rounded down to an int (day counts, scores).
func Int(key, field string, d civildate.Date) int { return int(Num(key, field, d)) }

// Fingerprint (numbers and rows keys alike) is "" when every key of keys is at version 1 on d, else "key@v,…" of the keys that are not (the
// payload version suffix clients compare).
func Fingerprint(d civildate.Date, keys ...string) string {
	out := ""
	for _, k := range keys {
		if v := At(k, d); v.Version > 1 {
			if out != "" {
				out += ","
			}
			out += k + "@" + strconv.Itoa(v.Version)
		}
	}
	return out
}
