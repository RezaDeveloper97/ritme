package pregnancy

import (
	"database/sql"
	"errors"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// CareItemKinds are pregnancy_care_items.kind values (CARE_ITEM_KINDS in the frontend).
var CareItemKinds = []string{"visit", "test", "scan", "vaccine"}

// Care item limits and defaults.
const (
	MaxRemindBefore     = 30 // days
	DefaultRemindBefore = 1
)

// CodeInUse is the 422 error_code of deleting a care item that appointments reference.
const CodeInUse = "in_use"

func careItemJSON(it *store.PregnancyCareItem, appointments int64) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", it.ID,
		"key", it.Key,
		"title", form.Raw(it.Title),
		"prep", form.NullRaw(it.Prep),
		"kind", it.Kind,
		"week_from", it.WeekFrom,
		"week_to", it.WeekTo,
		"remind_before", it.RemindBefore,
		"sort_order", it.SortOrder,
		"is_active", it.IsActive,
		"appointments_count", appointments,
		"created_at", httpadmin.Time(it.CreatedAt),
		"updated_at", httpadmin.Time(it.UpdatedAt),
	)
}

func (h *Handlers) findCareItem(c fiber.Ctx) (store.PregnancyCareItem, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.PregnancyCareItem{}, httpadmin.NotFound("Care item")
	}
	it, err := h.q.GetCareItem(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return store.PregnancyCareItem{}, httpadmin.NotFound("Care item")
	}
	return it, err
}

func (h *Handlers) respondCareItem(c fiber.Ctx, id uint64, created bool, msg string) error {
	it, err := h.q.GetCareItem(c.Context(), id)
	if err != nil {
		return err
	}
	n, err := h.q.CountAppointmentsOfCareItem(c.Context(), id)
	if err != nil {
		return err
	}
	body := jsonx.Obj("care_item", careItemJSON(&it, n))
	var msgs []string
	if msg != "" {
		msgs = []string{msg}
	}
	if created {
		return httpadmin.Created(c, body, msgs...)
	}
	return httpadmin.OK(c, body, msgs...)
}

// ListCareItems is GET /pregnancy-care-items: every item (active and inactive) in sort order.
func (h *Handlers) ListCareItems(c fiber.Ctx) error {
	rows, err := h.q.ListCareItems(c.Context())
	if err != nil {
		return err
	}
	counts, err := h.q.CountCareItemAppointments(c.Context())
	if err != nil {
		return err
	}
	byID := make(map[uint64]int64, len(counts))
	for _, r := range counts {
		byID[r.ID] = r.Appointments
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, careItemJSON(&rows[i], byID[rows[i].ID]))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items))
}

// CareItemOptions is GET /pregnancy-care-items/options.
func (h *Handlers) CareItemOptions(c fiber.Ctx) error {
	next, err := h.q.NextCareItemSortOrder(c.Context())
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj(
		"kinds", CareItemKinds,
		"min_week", registry.MinWeek,
		"max_week", registry.MaxWeek,
		"max_remind_before", MaxRemindBefore,
		"default_remind_before", DefaultRemindBefore,
		"next_sort_order", next,
	))
}

// ShowCareItem is GET /pregnancy-care-items/:id.
func (h *Handlers) ShowCareItem(c fiber.Ctx) error {
	it, err := h.findCareItem(c)
	if err != nil {
		return err
	}
	return h.respondCareItem(c, it.ID, false, "")
}

func careRules(in phpval.Map, codes []string, create bool) validation.Rules {
	var r validation.Rules
	if create {
		r = append(r, validation.F("key", "required|string|max:64", validation.Regex(keyPattern)))
	}
	week := "required|integer|min:" + strconv.Itoa(registry.MinWeek) + "|max:" + strconv.Itoa(registry.MaxWeek)
	r = append(r, translatable("title", true, codes, 255)...)
	r = append(r, translatable("prep", sent(in, "prep"), codes, 2000)...)
	return append(r,
		validation.F("kind", "required|string", validation.In(CareItemKinds...)),
		validation.F("week_from", week),
		validation.F("week_to", week),
		validation.F("remind_before", "nullable|integer|min:0|max:"+strconv.Itoa(MaxRemindBefore)),
		validation.F("sort_order", "nullable|integer|min:-1000000|max:1000000"),
		validation.F("is_active", "nullable|boolean"),
	)
}

func (h *Handlers) careChecks(c fiber.Ctx, create bool) []form.Check {
	return []form.Check{
		func(in phpval.Map, add form.Add) error {
			from, _ := phpval.Get(in, "week_from")
			to, _ := phpval.Get(in, "week_to")
			if phpval.IsNumeric(from) && phpval.IsNumeric(to) && phpval.ToFloat(to) < phpval.ToFloat(from) {
				add("week_to", form.Msg(c, "validation.gte.numeric", "week_to", "value", phpval.ToString(from)))
			}
			return nil
		},
		func(in phpval.Map, add form.Add) error {
			if !create {
				return nil
			}
			v, _ := phpval.Get(in, "key")
			s, isStr := v.(string)
			if !isStr || s == "" {
				return nil
			}
			_, err := h.q.GetCareItemByKey(c.Context(), s)
			switch {
			case err == nil:
				add("key", form.Msg(c, "validation.unique", "key"))
			case !errors.Is(err, sql.ErrNoRows):
				return err
			}
			return nil
		},
	}
}

type careValues struct {
	title        []byte
	prep         db.NullRawJSON
	kind         string
	weekFrom     uint8
	weekTo       uint8
	remindBefore uint16
	isActive     bool
}

// careData maps validated input; cur (update) supplies absent optional values.
func careData(c fiber.Ctx, data phpval.Map, cur *store.PregnancyCareItem) careValues {
	codes := i18n.LanguagesOf(c).Codes()
	title, _ := data.Get("title")
	v := careValues{
		title:    form.JSON(translatedObj(title, codes)),
		kind:     httpadmin.String(data, "kind"),
		weekFrom: uint8(form.Int(data, "week_from", 1)), //nolint:gosec // G115: validated 1…42
		weekTo:   uint8(form.Int(data, "week_to", 1)),   //nolint:gosec // G115: validated 1…42
		isActive: httpadmin.Bool(data, "is_active"),
	}
	if _, ok := data.Get("prep"); ok || cur == nil {
		p, _ := data.Get("prep")
		v.prep = nullJSON(translatedObj(p, codes))
	} else {
		v.prep = cur.Prep
	}
	switch {
	case form.Has(data, "remind_before"):
		v.remindBefore = uint16(form.Int(data, "remind_before", DefaultRemindBefore)) //nolint:gosec // G115: validated 0…30
	case cur != nil:
		v.remindBefore = cur.RemindBefore
	default:
		v.remindBefore = DefaultRemindBefore
	}
	return v
}

// StoreCareItem is POST /pregnancy-care-items.
func (h *Handlers) StoreCareItem(c fiber.Ctx) error {
	in := validation.Input(c)
	data, err := form.ValidateInput(c, in, careRules(in, i18n.LanguagesOf(c).Codes(), true), h.careChecks(c, true)...)
	if err != nil {
		return err
	}
	v := careData(c, data, nil)
	sortOrder := form.Int(data, "sort_order", 0)
	if !form.Has(data, "sort_order") {
		if sortOrder, err = h.q.NextCareItemSortOrder(c.Context()); err != nil {
			return err
		}
	}
	id, err := h.q.CreateCareItem(c.Context(), store.CreateCareItemParams{
		ItemKey: httpadmin.String(data, "key"), Title: v.title, Prep: v.prep, Kind: v.kind,
		WeekFrom: v.weekFrom, WeekTo: v.weekTo, RemindBefore: v.remindBefore,
		SortOrder: int32(sortOrder), //nolint:gosec // G115: validated / MAX()+1
		IsActive:  v.isActive, Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	uid := uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	httpadmin.Audit(c, h.logger, "pregnancy_care_item.create", "pregnancy_care_item", uid)
	return h.respondCareItem(c, uid, true, "Care item created.")
}

// UpdateCareItem is PUT /pregnancy-care-items/:id (the key is fixed: appointments reference
// it in meta.care_item_key). Absent prep / remind_before / sort_order keep their value;
// is_active follows $request->boolean() (absent = false).
func (h *Handlers) UpdateCareItem(c fiber.Ctx) error {
	cur, err := h.findCareItem(c)
	if err != nil {
		return err
	}
	in := validation.Input(c)
	data, err := form.ValidateInput(c, in, careRules(in, i18n.LanguagesOf(c).Codes(), false), h.careChecks(c, false)...)
	if err != nil {
		return err
	}
	v := careData(c, data, &cur)
	sortOrder := cur.SortOrder
	if form.Has(data, "sort_order") {
		sortOrder = form.Int32(data, "sort_order", 0)
	}
	if _, err := h.q.UpdateCareItem(c.Context(), store.UpdateCareItemParams{
		Title: v.title, Prep: v.prep, Kind: v.kind, WeekFrom: v.weekFrom, WeekTo: v.weekTo,
		RemindBefore: v.remindBefore, SortOrder: sortOrder, IsActive: v.isActive,
		Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_care_item.update", "pregnancy_care_item", cur.ID)
	return h.respondCareItem(c, cur.ID, false, "Care item updated.")
}

// ToggleCareItem is POST /pregnancy-care-items/:id/toggle (is_active flips).
func (h *Handlers) ToggleCareItem(c fiber.Ctx) error {
	cur, err := h.findCareItem(c)
	if err != nil {
		return err
	}
	if _, err := h.q.SetCareItemActive(c.Context(), store.SetCareItemActiveParams{
		IsActive: !cur.IsActive, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: cur.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_care_item.toggle", "pregnancy_care_item", cur.ID,
		slog.Bool("is_active", !cur.IsActive))
	return h.respondCareItem(c, cur.ID, false, "Care item status changed.")
}

// DestroyCareItem is DELETE /pregnancy-care-items/:id: refused with 422 in_use while any
// appointment references the item's key — admin-web offers deactivate instead.
func (h *Handlers) DestroyCareItem(c fiber.Ctx) error {
	cur, err := h.findCareItem(c)
	if err != nil {
		return err
	}
	n, err := h.q.DeleteUnlinkedCareItem(c.Context(), cur.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		count, err := h.q.CountAppointmentsOfCareItem(c.Context(), cur.ID)
		if err != nil {
			return err
		}
		if count == 0 {
			return httpadmin.NotFound("Care item") // deleted concurrently
		}
		return httpadmin.Fail(fiber.StatusUnprocessableEntity, CodeInUse,
			"This care item has "+strconv.FormatInt(count, 10)+" appointments; deactivate it instead.",
			"appointments_count", count)
	}
	httpadmin.Audit(c, h.logger, "pregnancy_care_item.delete", "pregnancy_care_item", cur.ID)
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Care item deleted.")
}

// ReorderCareItems is POST /pregnancy-care-items/reorder {ids:[…]}: every item id exactly once,
// in the new order; sort_order becomes 1…n in one transaction.
func (h *Handlers) ReorderCareItems(c fiber.Ctx) error {
	rows, err := h.q.ListCareItems(c.Context())
	if err != nil {
		return err
	}
	known := make(map[uint64]bool, len(rows))
	for _, r := range rows {
		known[r.ID] = true
	}
	var ids []uint64
	_, err = form.Validate(c, validation.Rules{
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
				f := "ids." + keys[i]
				add(f, form.Msg(c, "validation.distinct", f))
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) != len(rows) {
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
		return err
	}
	tx, err := h.db.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := h.q.WithTx(tx)
	now := httpadmin.DBTime(httpadmin.Now(c))
	for i, id := range ids {
		if _, err := q.SetCareItemSortOrder(c.Context(), store.SetCareItemSortOrderParams{
			SortOrder: int32(i + 1), Now: now, ID: id, //nolint:gosec // G115: bounded by the list size
		}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "pregnancy_care_item.reorder", "pregnancy_care_item", 0, slog.Int("count", len(ids)))
	out := make([]*jsonx.OrderedMap, 0, len(ids))
	for i, id := range ids {
		out = append(out, jsonx.Obj("id", id, "sort_order", i+1))
	}
	return httpadmin.OK(c, jsonx.Obj("items", out), "Order saved.")
}
