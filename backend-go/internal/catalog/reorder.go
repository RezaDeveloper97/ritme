package catalog

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// maxGroupRows bounds the rows one reorder reads (a group is a short editorial list).
const maxGroupRows = 10000

// Reorder is POST /catalog/:group/reorder {ids:[…]}: every item id of the group exactly once, in
// the new order; sort_order becomes 1…n in one transaction (all rows or none). Only sort_order is
// written, so a concurrent PUT of other columns is not overwritten. Partial lists,
// duplicates and ids of another group → 422 on `ids` / `ids.N`.
func (h *Admin) Reorder(c fiber.Ctx) error {
	g, err := group(c)
	if err != nil {
		return err
	}
	if err := h.canWrite(c, g); err != nil {
		return err
	}
	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)

	rows, err := q.ListAdminCatalogItems(c.Context(), store.ListAdminCatalogItemsParams{
		CatalogGroup: g, Pattern: form.Contains(""), JsonPattern: form.ContainsJSON(""),
		ActiveMin: false, ActiveMax: true, Limit: maxGroupRows, Offset: 0,
	})
	if err != nil {
		return err
	}
	byID := make(map[uint64]*store.CatalogItem, len(rows))
	all := make([]uint64, 0, len(rows))
	for i := range rows {
		byID[rows[i].ID] = &rows[i]
		all = append(all, rows[i].ID)
	}
	ids, err := validateReorder(c, all)
	if err != nil {
		return err
	}

	now := httpadmin.DBTime(httpadmin.Now(c))
	for i, id := range ids {
		it := byID[id]
		pos := int32(i + 1) //nolint:gosec // G115: bounded by maxGroupRows
		if it.SortOrder == pos {
			continue
		}
		if err := q.SetCatalogSortOrder(c.Context(), store.SetCatalogSortOrderParams{
			SortOrder: pos, Now: now, ID: it.ID, CatalogGroup: g,
		}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	h.flush(c, g)
	httpadmin.Audit(c, h.logger, "catalog_item.reorder", "catalog_item", 0,
		slog.String("group", g), slog.Int("count", len(ids)))
	out := make([]*jsonx.OrderedMap, 0, len(ids))
	for i, id := range ids {
		out = append(out, jsonx.Obj("id", id, "sort_order", i+1))
	}
	return httpadmin.OK(c, jsonx.Obj("items", jsonx.List(out)), "Order saved.")
}

// validateReorder checks ids is a permutation of the group's ids (all of them, each once).
func validateReorder(c fiber.Ctx, all []uint64) ([]uint64, error) {
	known := make(map[uint64]bool, len(all))
	for _, id := range all {
		known[id] = true
	}
	var ids []uint64
	_, err := validateForm(c, validation.Rules{
		validation.F("ids", "required|array"),
		validation.F("ids.*", "required|integer|min:1"),
	}, func(in phpval.Map, add form.Add) error {
		v, _ := phpval.Get(in, "ids")
		if !phpval.IsArray(v) {
			return nil
		}
		keys, vals := phpval.Entries(v)
		seen := map[uint64]bool{}
		for i, x := range vals {
			if x == nil || !phpval.IsNumeric(x) || phpval.ToFloat(x) < 1 {
				return nil // the rules report it
			}
			id := uint64(phpval.ToFloat(x))
			if seen[id] {
				field := "ids." + keys[i]
				add(field, form.Msg(c, "validation.distinct", field))
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) != len(all) {
			add("ids", form.Msg(c, "validation.in", "ids"))
			return nil
		}
		for _, id := range ids {
			if !known[id] {
				add("ids", form.Msg(c, "validation.in", "ids"))
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}
