package recommendation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/enums"
)

// Source is the database behind the repository (a sqlc adapter in production, a slice in tests).
type Source interface {
	// ActiveRows returns every row with is_active = 1, ordered by sort_order, id.
	ActiveRows(ctx context.Context) ([]Row, error)
	// AnyExists reports whether the table holds any row at all, active or not.
	AnyExists(ctx context.Context) (bool, error)
}

// Repository is RecommendationRepository: one load of the live set per request, matched in Go.
// Create one per request (it memoises); it is safe for concurrent use within that request.
type Repository struct {
	src Source

	mu         sync.Mutex
	rows       []Row
	loaded     bool
	hasContent *bool
	cache      map[string][]Tip
}

// New returns a request-scoped repository over src.
func New(src Source) *Repository {
	return &Repository{src: src, cache: map[string][]Tip{}}
}

// HasContent reports whether any recommendation exists (inactive rows count: an admin who switched
// everything off wants an empty card, not the code fallback).
// PHP: RecommendationRepository::hasContent (:49).
func (r *Repository) HasContent(ctx context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hasContentLocked(ctx)
}

func (r *Repository) hasContentLocked(ctx context.Context) (bool, error) {
	if r.hasContent != nil {
		return *r.hasContent, nil
	}
	rows, err := r.rowsLocked(ctx)
	if err != nil {
		return false, err
	}
	has := len(rows) > 0
	if !has {
		if has, err = r.src.AnyExists(ctx); err != nil {
			return false, fmt.Errorf("recommendation: exists: %w", err)
		}
	}
	r.hasContent = &has
	return has, nil
}

// ForDay returns the day's tips: rows matching phase, canonical sub-phase and today's triggers,
// symptom-triggered rows first, each group in admin order. log may be nil (no log today).
// PHP: RecommendationRepository::forDay (:67).
func (r *Repository) ForDay(ctx context.Context, phase enums.CyclePhase, subphase enums.CycleSubphase, log enums.TriggerLog) ([]Tip, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	canonical := string(subphase.Canonical())
	phaseValue := string(phase)
	triggers := enums.RecommendationTriggerActiveFor(log)
	key := phaseValue + "|" + canonical + "|" + strings.Join(triggers, ",")

	if tips, ok := r.cache[key]; ok {
		return tips, nil
	}
	rows, err := r.rowsLocked(ctx)
	if err != nil {
		return nil, err
	}

	var triggered, plain []Tip
	for _, row := range rows {
		if !row.AppliesTo(&phaseValue, &canonical, triggers) {
			continue
		}
		if row.SymptomTrigger != nil {
			triggered = append(triggered, row.ToTip())
		} else {
			plain = append(plain, row.ToTip())
		}
	}
	tips := append(append(make([]Tip, 0, len(triggered)+len(plain)), triggered...), plain...)
	r.cache[key] = tips
	return tips, nil
}

// Signature digests everything ForDay and HasContent can return, for engine cache keys
// (RecommendationRepository::signature; Go uses its own digest — its cache namespace is separate).
func (r *Repository) Signature(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	has, err := r.hasContentLocked(ctx)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		Has  bool
		Rows []Row
	}{has, r.rows})
	if err != nil {
		return "", fmt.Errorf("recommendation: signature: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:16]), nil
}

func (r *Repository) rowsLocked(ctx context.Context) ([]Row, error) {
	if !r.loaded {
		rows, err := r.src.ActiveRows(ctx)
		if err != nil {
			return nil, fmt.Errorf("recommendation: load: %w", err)
		}
		r.rows, r.loaded = rows, true
	}
	return r.rows, nil
}
