// Package accountexport adds the canvas health data to the account data export (GET /profile/export, user decision
// 2026-10-12, deviations.md D-96): pelvic floor, contraception, menopause, IVF, condition programs, the loss path,
// insurance, the shop and the mother & child directory. Each section is a Go-only top-level key of the export
// (internal/profile ExportSection), read through the owning domain's sqlc store with every statement scoped by the
// owner's user id, and serialised like the rest of the export (raw model attributes with the model casts,
// internal/profile/model). Text a domain seals at rest (insurance notes and questionnaire evidence, the loss private
// note, shop buyer PII) is opened for the owner — the AD binds each seal to her, so another account's blob never
// opens; an unreadable seal (no key, a rotated-out key) exports as null instead of failing the export. No private
// file bytes, storage paths or hashes are exported, only file metadata. Nothing here logs a value.
package accountexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/insurance"
	"github.com/ritme/backend-go/internal/loss"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/profile"
	"github.com/ritme/backend-go/internal/profile/model"
	"github.com/ritme/backend-go/internal/shop/order"
)

// Box opens the owner's sealed text (*loss.NoteBox on PRIVATE_NOTE_KEY; disabled without a key).
type Box interface {
	SealWith(plain string, ad []byte) (string, error)
	OpenWith(stored string, ad []byte) (string, error)
	Open(stored string, userID, lossID uint64) (string, error)
	Disabled() bool
}

var _ Box = (*loss.NoteBox)(nil)

// Sections are the canvas sections of the export, in output order.
func Sections(db *sql.DB, box Box) []profile.ExportSection {
	return []profile.ExportSection{
		{Key: "pelvic_floor", Load: pelvicFloor(db)},
		{Key: "contraception", Load: contraception(db)},
		{Key: "menopause", Load: menopause(db)},
		{Key: "ivf", Load: ivf(db)},
		{Key: "condition_programs", Load: conditionPrograms(db)},
		{Key: "loss", Load: lossPath(db, box)},
		{Key: "insurance", Load: insuranceData(db, insurance.NewService(db, box))},
		{Key: "shop", Load: shopData(db, order.NewService(db).WithSealer(box))},
		{Key: "directory", Load: directory(db)},
	}
}

// rows serialises a query result as model attributes (casts + hidden columns), then lets each adjust the object.
func rows[T any](key string, list []T, err error, casts model.Casts, hidden []string, each func(r *T, o *jsonx.OrderedMap)) ([]any, error) {
	if err != nil {
		return nil, fmt.Errorf("accountexport: %s: %w", key, err)
	}
	out := make([]any, 0, len(list))
	for i := range list {
		o, _ := model.Attributes(&list[i], casts, hidden...).(*jsonx.OrderedMap)
		if o == nil {
			continue
		}
		if each != nil {
			each(&list[i], o)
		}
		out = append(out, o)
	}
	return out, nil
}

// first is the single row of a per-user table (null when none).
func first(list []any) any {
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

// jsonText decodes JSON kept in longtext columns (MariaDB CHECK json_valid) like the `array` cast.
func jsonText(o *jsonx.OrderedMap, cols ...string) {
	for _, col := range cols {
		v, ok := o.Get(col)
		if s, isStr := v.(string); ok && isStr {
			o.Set(col, model.DecodeJSONColumn(json.RawMessage(s)))
		}
	}
}

// opened is a sealed cell opened for the owner: null for none or an unreadable seal.
func opened(plain string, err error) any {
	if err != nil || plain == "" {
		return nil
	}
	return plain
}

func nonEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// part is one key of a section: its rows, or the single row / null when one is set.
type part struct {
	key  string
	load func() ([]any, error)
	one  bool
}

// object loads every part into one section object (an error in any part fails the section, all errors joined).
func object(parts ...part) (any, error) {
	out := jsonx.NewObject()
	var errs []error
	for _, p := range parts {
		v, err := p.load()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if p.one {
			out.Set(p.key, first(v))
		} else {
			out.Set(p.key, v)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return out, nil
}

// loader is the Load of a section.
type loader = func(ctx context.Context, userID uint64) (any, error)
