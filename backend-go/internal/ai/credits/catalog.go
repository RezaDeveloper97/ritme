package credits

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/credits/store"
	"github.com/ritme/backend-go/internal/platform/configcache"
)

// Catalogue tables; the admin writers publish them (configcache.Changed) after their commit.
const (
	FeaturesTable = "ai_features"
	ModelsTable   = "ai_models"
)

// Feature is one AI feature slot as the admin configured it.
type Feature struct {
	Key    ai.Feature
	Title  json.RawMessage // translatable JSON keyed by language code
	Cost   uint32          // credits of one use
	Active bool
	// Models are the usable models of the slot, primary first: active, and capable of what the slot needs.
	Models []store.AiModel
}

// Usable reports whether the feature can run: switched on, declared in code and with at least one model.
func (f Feature) Usable() bool { return f.Active && len(f.Models) > 0 }

type snapshot struct {
	features []Feature // display order
	byKey    map[ai.Feature]Feature
}

// Catalog is the process snapshot of the feature slots and their models (ai_features + ai_models). It is the
// ai.Router of the Client and the price list of the credit gate.
type Catalog struct {
	q     *store.Queries
	cache *configcache.Cache[snapshot]
}

// NewCatalog wires the catalogue on db.
func NewCatalog(db *sql.DB) *Catalog {
	c := &Catalog{q: store.New(db)}
	c.cache = configcache.New("ai_catalog", []string{FeaturesTable, ModelsTable}, c.load)
	return c
}

// ModelCapabilities decodes a model's capabilities column.
func ModelCapabilities(m store.AiModel) []ai.Capability {
	var caps []ai.Capability
	_ = json.Unmarshal(m.Capabilities, &caps) // an unreadable column = no capability: the model serves nothing
	return caps
}

// Serves reports whether model has every capability in needs.
func Serves(m store.AiModel, needs []ai.Capability) bool {
	caps := ModelCapabilities(m)
	for _, n := range needs {
		if !slices.Contains(caps, n) {
			return false
		}
	}
	return true
}

func (c *Catalog) load(ctx context.Context) (snapshot, error) {
	feats, err := c.q.ListFeatures(ctx)
	if err != nil {
		return snapshot{}, fmt.Errorf("credits: features: %w", err)
	}
	models, err := c.q.ListModels(ctx)
	if err != nil {
		return snapshot{}, fmt.Errorf("credits: models: %w", err)
	}
	byID := make(map[int64]store.AiModel, len(models))
	for _, m := range models {
		byID[int64(m.ID)] = m //nolint:gosec // G115: ids are positive
	}
	s := snapshot{byKey: make(map[ai.Feature]Feature, len(feats))}
	for _, row := range feats {
		f := Feature{Key: ai.Feature(row.Key), Title: row.Title, Cost: row.CreditCost, Active: row.IsActive}
		// A slot whose key is no longer declared in code has no caller: it stays listed for the admin, unusable.
		if spec, ok := ai.SpecOf(f.Key); ok {
			for _, id := range []sql.NullInt64{row.ModelID, row.FallbackModelID} {
				m, found := byID[id.Int64]
				if id.Valid && found && m.IsActive && Serves(m, spec.Needs) {
					f.Models = append(f.Models, m)
				}
			}
		}
		s.features = append(s.features, f)
		s.byKey[f.Key] = f
	}
	return s, nil
}

// Features are the slots in display order.
func (c *Catalog) Features(ctx context.Context) ([]Feature, error) {
	s, err := c.cache.Get(ctx)
	if err != nil {
		return nil, err
	}
	return s.features, nil
}

// Feature is the slot of key (false when it has no row).
func (c *Catalog) Feature(ctx context.Context, key ai.Feature) (Feature, bool, error) {
	s, err := c.cache.Get(ctx)
	if err != nil {
		return Feature{}, false, err
	}
	f, ok := s.byKey[key]
	return f, ok, nil
}

// Routes implements ai.Router: the feature's usable models, primary first. A switched-off or model-less feature
// has none (the Client answers ErrUnavailable).
func (c *Catalog) Routes(ctx context.Context, feature ai.Feature) ([]ai.Route, error) {
	f, ok, err := c.Feature(ctx, feature)
	if err != nil {
		return nil, err
	}
	if !ok || !f.Active {
		return nil, nil
	}
	out := make([]ai.Route, len(f.Models))
	for i, m := range f.Models {
		out[i] = ai.Route{Provider: m.Provider, Model: m.UpstreamModel, Priced: true,
			InputPriceMilli: m.InputPriceMilli, OutputPriceMilli: m.OutputPriceMilli, AudioPriceMilli: m.AudioPriceMilli}
	}
	return out, nil
}

// Sync creates the ai_features row of every feature declared in code (ai.RegisterFeature) that has none yet — a
// new slot shows up in the admin panel, without a model and at 1 credit, as soon as the code that declares it is
// deployed. Existing rows are never touched. Call once at start-up.
func (c *Catalog) Sync(ctx context.Context, now time.Time) error {
	for i, spec := range ai.RegisteredFeatures() {
		err := c.q.InsertFeatureIfMissing(ctx, store.InsertFeatureIfMissingParams{
			Key: string(spec.Key), Title: json.RawMessage(`{}`), SortOrder: uint32(100 + i), //nolint:gosec // G115: small index
			CreatedAt: stampOf(now), UpdatedAt: stampOf(now),
		})
		if err != nil {
			return fmt.Errorf("credits: sync feature %s: %w", spec.Key, err)
		}
	}
	configcache.Changed(ctx, FeaturesTable)
	return nil
}
