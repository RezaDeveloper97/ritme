// Package content is MessageContentRepository
// (backend/app/Services/MessageSystem/Support/MessageContentRepository.php) plus the code
// fallback copy of every smart-message class (their static contentDefaults()), exported once
// from PHP to defaults.json and embedded.
//
// A Repository is request-scoped: the first lookup in a locale loads every live (active +
// approved) message_contents row of that locale in one query; later lookups in any group are
// answered from memory. Nothing outlives the request, so an admin edit is visible on the next
// request.
//
// It also implements profile.MessageContents (the BMI message reader of T-M2-11).
package content

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Rows loads the live rows of one locale (store.Queries implements it).
type Rows interface {
	ListLiveMessageContents(ctx context.Context, locale string) ([]store.ListLiveMessageContentsRow, error)
}

// Repository is one request's MessageContentRepository.
type Repository struct {
	q Rows

	mu       sync.Mutex
	byLocale map[string]map[string]json.RawMessage // locale → "group|item_key" → raw payload
	decoded  map[string]any                        // "locale|group|item_key" → decoded payload (nil = none)
}

// NewRepository returns an empty request-scoped repository.
func NewRepository(q Rows) *Repository {
	return &Repository{q: q, byLocale: map[string]map[string]json.RawMessage{}, decoded: map[string]any{}}
}

// Payload is MessageContentRepository::payload(): json_decode($payload, true) of the live row,
// ok=false when there is no row or the JSON decodes to null / fails to decode (PHP null).
func (r *Repository) Payload(ctx context.Context, group, itemKey, locale string) (any, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	memo := locale + "|" + group + "|" + itemKey
	if v, seen := r.decoded[memo]; seen {
		return v, v != nil, nil
	}
	rows, err := r.rows(ctx, locale)
	if err != nil {
		return nil, false, err
	}
	var v any
	if raw, found := rows[group+"|"+itemKey]; found {
		if d, decErr := phpval.Decode(raw); decErr == nil {
			v = phpval.Packed(d)
		}
	}
	r.decoded[memo] = v
	return v, v != nil, nil
}

// Resolve is MessageContentRepository::resolve() with the embedded contentDefaults() slice as
// the fallback: the DB payload when a live row exists, otherwise Default(group, itemKey, locale).
//
// PHP declares both the parameter and the return as `array`: a stored payload that decodes to
// a scalar is a TypeError there (HTTP 500), reproduced here as an error.
func (r *Repository) Resolve(ctx context.Context, group, itemKey, locale string) (Payload, error) {
	v, ok, err := r.Payload(ctx, group, itemKey, locale)
	if err != nil {
		return Payload{}, err
	}
	if !ok {
		return Default(group, itemKey, locale), nil
	}
	if !phpval.IsArray(v) {
		return Payload{}, fmt.Errorf("messages/content: %s/%s/%s payload is not an array (PHP TypeError)", group, itemKey, locale)
	}
	return Payload{v: v}, nil
}

// rows is MessageContentRepository::rows() (caller holds mu). mapWithKeys keeps the last row
// of a duplicated key; the unique index makes that moot.
func (r *Repository) rows(ctx context.Context, locale string) (map[string]json.RawMessage, error) {
	if rows, ok := r.byLocale[locale]; ok {
		return rows, nil
	}
	list, err := r.q.ListLiveMessageContents(ctx, locale)
	if err != nil {
		return nil, fmt.Errorf("messages/content: load %q: %w", locale, err)
	}
	rows := make(map[string]json.RawMessage, len(list))
	for _, row := range list {
		rows[row.Group+"|"+row.ItemKey] = row.Payload
	}
	r.byLocale[locale] = rows
	return rows, nil
}

// Payload is one resolved content array (a decoded phpval value: an ordered map, or a list).
type Payload struct{ v any }

// NewPayload wraps a decoded PHP array (tests, fakes).
func NewPayload(v any) Payload { return Payload{v: v} }

// Field is `$p[$key] ?? …`: ok=false when the key is missing or the value is null.
func (p Payload) Field(key string) (any, bool) {
	switch a := p.v.(type) {
	case phpval.Map:
		v, ok := a.Get(key)
		return v, ok && v != nil
	case []any:
		keys, vals := phpval.Entries(a)
		for i, k := range keys {
			if k == key {
				return vals[i], vals[i] != nil
			}
		}
	}
	return nil, false
}

// Or is `$p[$key] ?? $def`.
func (p Payload) Or(key string, def any) any {
	if v, ok := p.Field(key); ok {
		return v
	}
	return def
}
