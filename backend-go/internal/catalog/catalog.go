// Package catalog is the admin-editable content catalog (CB-CORE-03, docs/canvas-build/catalog.md): small
// clinical/content lists of the canvas-build epics (warning signs, missed-pill steps, FAQs, kit items, score
// items, …) stored as `catalog_items` rows grouped by `group` and identified by `code`, with translatable
// title/body, an optional audience filter and free-form meta.
//
//	GET /api/v1/catalog/{group}          one group's active items, localized (cached per group)
//	/api/admin/v1/catalog[/{group}[/{id}]] admin CRUD (admin.go)
//
// Groups are data, not code: any group name matching GroupPattern is valid and an unused one is simply empty.
// The table holds no user data; every admin write flushes the group's cache entry.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/platform/cache"
)

// Limits of groups, codes and the admin form.
const (
	MaxCodeLen      = 64    // `group` / `code` columns
	MaxAudiences    = 20    // audiences per item
	MaxAudienceLen  = 32    // one audience code
	MaxTitleLen     = 255   // per language
	MaxBodyLen      = 5000  // per language
	MaxMetaBytes    = 16384 // encoded meta JSON
	DefaultCacheTTL = 10 * time.Minute
)

// codePattern: lowercase snake case starting with a letter — groups, codes and audience codes.
const codePattern = `^[a-z][a-z0-9_]*$`

var codeRe = regexp.MustCompile(codePattern)

// ValidCode reports whether s is a valid group / item / audience code of at most max bytes.
func ValidCode(s string, maxLen int) bool {
	return len(s) > 0 && len(s) <= maxLen && codeRe.MatchString(s)
}

// ValidGroup reports whether s can name a catalog group.
func ValidGroup(s string) bool { return ValidCode(s, MaxCodeLen) }

// Item is one active catalog item as cached (locale-independent; localized per request).
type Item struct {
	Code        string          `json:"code"`
	Audiences   []string        `json:"audiences"` // nil = everyone
	Title       json.RawMessage `json:"title"`
	Body        json.RawMessage `json:"body,omitempty"`
	Meta        json.RawMessage `json:"meta,omitempty"`
	NeedsReview bool            `json:"needs_review"`
}

// For reports whether the item is shown to audience ("" = no filter; no audiences = everyone).
func (it Item) For(audience string) bool {
	if audience == "" || len(it.Audiences) == 0 {
		return true
	}
	for _, a := range it.Audiences {
		if a == audience {
			return true
		}
	}
	return false
}

// Lister is the query the reader needs (store.Querier satisfies it).
type Lister interface {
	ListActiveCatalogItems(ctx context.Context, group string) ([]store.ListActiveCatalogItemsRow, error)
}

// Reader reads a group's active items through a Redis cache (nil cache: straight from the DB).
type Reader struct {
	q      Lister
	cache  *cache.Client
	ttl    time.Duration
	logger *slog.Logger
}

// NewReader builds a reader. cache and logger may be nil; ttl <= 0 uses DefaultCacheTTL.
func NewReader(q Lister, c *cache.Client, ttl time.Duration, logger *slog.Logger) *Reader {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	return &Reader{q: q, cache: c, ttl: ttl, logger: logger}
}

// CacheKey is the cache entry of a group (under cache.Client's prefix).
func CacheKey(group string) string { return "catalog.group." + group }

// Items returns the group's active items in display order. A cache failure falls back to the DB.
func (r *Reader) Items(ctx context.Context, group string) ([]Item, error) {
	if r.cache != nil {
		raw, err := r.cache.Get(ctx, CacheKey(group))
		if err == nil {
			var items []Item
			if json.Unmarshal([]byte(raw), &items) == nil && items != nil {
				return items, nil
			}
		} else if !errors.Is(err, cache.ErrMiss) {
			r.logger.WarnContext(ctx, "catalog: cache read failed", slog.String("error", err.Error()))
		}
	}
	items, err := r.load(ctx, group)
	if err != nil {
		return nil, err
	}
	if r.cache != nil {
		if b, err := json.Marshal(items); err == nil {
			if err := r.cache.Set(ctx, CacheKey(group), string(b), r.ttl); err != nil {
				r.logger.WarnContext(ctx, "catalog: cache write failed", slog.String("error", err.Error()))
			}
		}
	}
	return items, nil
}

// Flush drops a group's cache entry (after every admin write to it).
func (r *Reader) Flush(ctx context.Context, group string) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Delete(ctx, CacheKey(group))
}

func (r *Reader) load(ctx context.Context, group string) ([]Item, error) {
	rows, err := r.q.ListActiveCatalogItems(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("catalog: list %q: %w", group, err)
	}
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		it := Item{Code: row.Code, Title: row.Title, NeedsReview: row.NeedsReview}
		if row.Audiences.Valid {
			var auds []string
			if json.Unmarshal(row.Audiences.V, &auds) == nil && len(auds) > 0 {
				it.Audiences = auds
			}
		}
		if row.Body.Valid {
			it.Body = row.Body.V
		}
		if row.Meta.Valid {
			it.Meta = row.Meta.V
		}
		items = append(items, it)
	}
	return items, nil
}
