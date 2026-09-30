package catalog

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Route registers one admin endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Admin serves /api/admin/v1/catalog (editor and super admins).
type Admin struct {
	q      *store.Queries
	reader *Reader
	logger *slog.Logger
}

// NewAdmin builds the admin handlers; reader is used to flush a group's cache after writes.
func NewAdmin(conn *sql.DB, reader *Reader, logger *slog.Logger) *Admin {
	if logger == nil {
		logger = slog.Default()
	}
	return &Admin{q: store.New(conn), reader: reader, logger: logger}
}

// Routes registers the endpoints; static paths before params.
func (h *Admin) Routes(route Route, kit *httpadmin.Kit) {
	a := kit.Admin
	route(fiber.MethodGet, "/catalog", a(h.Groups))
	route(fiber.MethodGet, "/catalog/:group", a(h.List))
	route(fiber.MethodPost, "/catalog/:group", a(h.Store))
	route(fiber.MethodGet, "/catalog/:group/:id", a(h.Show))
	route(fiber.MethodPut, "/catalog/:group/:id", a(h.Update))
	route(fiber.MethodDelete, "/catalog/:group/:id", a(h.Destroy))
}

// ---------------------------------------------------------------------------
// Output

func itemJSON(it *store.CatalogItem) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", it.ID,
		"group", it.Group,
		"code", it.Code,
		"sort_order", it.SortOrder,
		"is_active", it.IsActive,
		"audiences", form.NullRaw(it.Audiences),
		"title", form.Raw(it.Title),
		"body", form.NullRaw(it.Body),
		"meta", form.NullRaw(it.Meta),
		"needs_review", it.NeedsReview,
		"created_at", httpadmin.Time(it.CreatedAt),
		"updated_at", httpadmin.Time(it.UpdatedAt),
	)
}

// ---------------------------------------------------------------------------
// Reads

// Groups is GET /catalog: every group that has items, with counts.
func (h *Admin) Groups(c fiber.Ctx) error {
	rows, err := h.q.ListCatalogGroups(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, jsonx.Obj("group", r.Group, "items_count", r.ItemsCount, "active_count", r.ActiveCount))
	}
	return httpadmin.OK(c, jsonx.Obj("items", jsonx.List(items)))
}

func group(c fiber.Ctx) (string, error) {
	g := c.Params("group")
	if !ValidGroup(g) {
		return "", httpadmin.NotFound("Catalog group")
	}
	return g, nil
}

// List is GET /catalog/:group?q=&status=all|active|inactive&page=&per_page= (sort_order, id).
func (h *Admin) List(c fiber.Ctx) error {
	g, err := group(c)
	if err != nil {
		return err
	}
	search := strings.TrimSpace(c.Query("q"))
	status := c.Query("status")
	var only *bool
	switch status {
	case "active":
		only = new(bool)
		*only = true
	case "inactive":
		only = new(bool)
	default:
		status = "all"
	}
	lo, hi := form.BoolRange(only)
	pattern, jsonPattern := form.Contains(search), form.ContainsJSON(search)
	total, err := h.q.CountAdminCatalogItems(c.Context(), store.CountAdminCatalogItemsParams{
		CatalogGroup: g, Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminCatalogItems(c.Context(), store.ListAdminCatalogItemsParams{
		CatalogGroup: g, Pattern: pattern, JsonPattern: jsonPattern, ActiveMin: lo, ActiveMax: hi,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, itemJSON(&rows[i]))
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("group", g, "q", search, "status", status))
	return httpadmin.OK(c, page)
}

// find loads the item named by :group + :id (404 for a missing id or an id of another group).
func (h *Admin) find(c fiber.Ctx) (store.CatalogItem, error) {
	g, err := group(c)
	if err != nil {
		return store.CatalogItem{}, err
	}
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.CatalogItem{}, httpadmin.NotFound("Catalog item")
	}
	it, err := h.q.GetCatalogItem(c.Context(), store.GetCatalogItemParams{ID: id, CatalogGroup: g})
	if errors.Is(err, sql.ErrNoRows) {
		return store.CatalogItem{}, httpadmin.NotFound("Catalog item")
	}
	return it, err
}

// Show is GET /catalog/:group/:id.
func (h *Admin) Show(c fiber.Ctx) error {
	it, err := h.find(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("catalog_item", itemJSON(&it)))
}

// respond reloads the row and answers {catalog_item} with 200 or 201.
func (h *Admin) respond(c fiber.Ctx, g string, id uint64, created bool, msg string) error {
	it, err := h.q.GetCatalogItem(c.Context(), store.GetCatalogItemParams{ID: id, CatalogGroup: g})
	if err != nil {
		return err
	}
	body := jsonx.Obj("catalog_item", itemJSON(&it))
	if created {
		return httpadmin.Created(c, body, msg)
	}
	return httpadmin.OK(c, body, msg)
}

// flush drops the group's public cache; a failure is logged (the entry expires by TTL anyway).
func (h *Admin) flush(c fiber.Ctx, g string) {
	if h.reader == nil {
		return
	}
	if err := h.reader.Flush(c.Context(), g); err != nil {
		h.logger.WarnContext(c.Context(), "catalog: cache flush failed", slog.String("group", g),
			slog.String("error", err.Error()))
	}
}

// ---------------------------------------------------------------------------
// Writes

// Store is POST /catalog/:group. is_active and needs_review default to true, sort_order to the end.
func (h *Admin) Store(c fiber.Ctx) error {
	g, err := group(c)
	if err != nil {
		return err
	}
	data, err := h.validate(c, g, nil)
	if err != nil {
		return err
	}
	v := values(c, data, nil)
	if !form.Has(data, "sort_order") {
		next, err := h.q.NextCatalogSortOrder(c.Context(), g)
		if err != nil {
			return err
		}
		v.SortOrder = int32(next) //nolint:gosec // G115: MAX(int)+1 of a small table
	}
	res, err := h.q.CreateCatalogItem(c.Context(), store.CreateCatalogItemParams{
		CatalogGroup: g, Code: httpadmin.String(data, "code"), SortOrder: v.SortOrder, IsActive: v.IsActive,
		Audiences: v.Audiences, Title: v.Title, Body: v.Body, Meta: v.Meta, NeedsReview: v.NeedsReview,
		Now: httpadmin.DBTime(httpadmin.Now(c)),
	})
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	uid := uint64(id) //nolint:gosec // G115: auto-increment ids are positive
	h.flush(c, g)
	httpadmin.Audit(c, h.logger, "catalog_item.create", "catalog_item", uid, slog.String("group", g))
	return h.respond(c, g, uid, true, "Catalog item created.")
}

// Update is PUT /catalog/:group/:id. The code is fixed at creation (clients key on it); an optional
// field that is absent keeps its stored value.
func (h *Admin) Update(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	data, err := h.validate(c, cur.Group, &cur)
	if err != nil {
		return err
	}
	v := values(c, data, &cur)
	if err := h.q.UpdateCatalogItem(c.Context(), store.UpdateCatalogItemParams{
		SortOrder: v.SortOrder, IsActive: v.IsActive, Audiences: v.Audiences, Title: v.Title, Body: v.Body,
		Meta: v.Meta, NeedsReview: v.NeedsReview, Now: httpadmin.DBTime(httpadmin.Now(c)),
		ID: cur.ID, CatalogGroup: cur.Group,
	}); err != nil {
		return err
	}
	h.flush(c, cur.Group)
	httpadmin.Audit(c, h.logger, "catalog_item.update", "catalog_item", cur.ID, slog.String("group", cur.Group))
	return h.respond(c, cur.Group, cur.ID, false, "Catalog item updated.")
}

// Destroy is DELETE /catalog/:group/:id.
func (h *Admin) Destroy(c fiber.Ctx) error {
	cur, err := h.find(c)
	if err != nil {
		return err
	}
	res, err := h.q.DeleteCatalogItem(c.Context(), store.DeleteCatalogItemParams{ID: cur.ID, CatalogGroup: cur.Group})
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpadmin.NotFound("Catalog item") // deleted concurrently
	}
	h.flush(c, cur.Group)
	httpadmin.Audit(c, h.logger, "catalog_item.delete", "catalog_item", cur.ID, slog.String("group", cur.Group))
	return httpadmin.OK(c, jsonx.Obj("id", cur.ID), "Catalog item deleted.")
}

// ---------------------------------------------------------------------------
// Validation

func rules(c fiber.Ctx, create bool) validation.Rules {
	var r validation.Rules
	if create {
		r = append(r, validation.F("code", "required|string|max:"+strconv.Itoa(MaxCodeLen),
			validation.Regex("/"+codePattern+"/")))
	}
	r = append(r, form.Translatable(c, "title", true, "max:"+strconv.Itoa(MaxTitleLen))...)
	r = append(r, form.Translatable(c, "body", false, "max:"+strconv.Itoa(MaxBodyLen))...)
	return append(r,
		validation.F("audiences", "nullable|array|max:"+strconv.Itoa(MaxAudiences)),
		validation.F("audiences.*", "required|string|max:"+strconv.Itoa(MaxAudienceLen),
			validation.Regex("/"+codePattern+"/")),
		validation.F("meta", "nullable|array"),
		validation.F("sort_order", "nullable|integer|min:0"),
		validation.F("is_active", "nullable|boolean"),
		validation.F("needs_review", "nullable|boolean"),
	)
}

func (h *Admin) validate(c fiber.Ctx, g string, cur *store.CatalogItem) (phpval.Map, error) {
	checks := []form.Check{
		// meta: bounded size once encoded.
		func(in phpval.Map, add form.Add) error {
			v, _ := phpval.Get(in, "meta")
			if phpval.IsArray(v) && len(form.JSON(v)) > MaxMetaBytes {
				add("meta", form.Msg(c, "validation.max.string", "meta", "max", strconv.Itoa(MaxMetaBytes)))
			}
			return nil
		},
	}
	if cur == nil {
		// code: unique inside the group.
		checks = append(checks, func(in phpval.Map, add form.Add) error {
			v, _ := phpval.Get(in, "code")
			s, isStr := v.(string)
			if !isStr || !ValidCode(s, MaxCodeLen) {
				return nil // the rules report it
			}
			found, err := h.q.CatalogCodeExists(c.Context(), store.CatalogCodeExistsParams{CatalogGroup: g, Code: s})
			if err != nil {
				return err
			}
			if found {
				add("code", form.Msg(c, "validation.unique", "code"))
			}
			return nil
		})
	}
	return form.Validate(c, rules(c, cur == nil), checks...)
}

// ---------------------------------------------------------------------------
// Validated data → columns

type columns struct {
	SortOrder   int32
	IsActive    bool
	Audiences   db.NullRawJSON
	Title       json.RawMessage
	Body        db.NullRawJSON
	Meta        db.NullRawJSON
	NeedsReview bool
}

// translated is {code: text} over the active languages that have text (nil when none).
func translated(v any, codes []string) json.RawMessage {
	kv := make([]any, 0, 2*len(codes))
	for _, code := range codes {
		t, ok := phpval.Get(v, code)
		if !ok || t == nil {
			continue
		}
		if s := phpval.ToString(t); s != "" {
			kv = append(kv, code, s)
		}
	}
	if len(kv) == 0 {
		return nil
	}
	return form.JSON(jsonx.Obj(kv...))
}

func nullJSON(raw json.RawMessage) db.NullRawJSON {
	if raw == nil {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: raw, Valid: true}
}

// audiences is the distinct audience codes in order (NULL when none).
func audiences(v any) db.NullRawJSON {
	_, vals := phpval.Entries(v)
	seen := map[string]bool{}
	out := make([]string, 0, len(vals))
	for _, x := range vals {
		s := phpval.ToString(x)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(out), Valid: true}
}

// values maps validated data onto the columns. On update (cur != nil) an optional field that was not
// sent keeps its stored value; sent null clears it.
func values(c fiber.Ctx, data phpval.Map, cur *store.CatalogItem) columns {
	codes := i18n.LanguagesOf(c).Codes()
	sent := func(key string) bool { _, ok := data.Get(key); return ok }
	v := columns{IsActive: true, NeedsReview: true}
	if cur != nil {
		v = columns{SortOrder: cur.SortOrder, IsActive: cur.IsActive, Audiences: cur.Audiences, Body: cur.Body,
			Meta: cur.Meta, NeedsReview: cur.NeedsReview}
	}
	title, _ := data.Get("title")
	v.Title = translated(title, codes)
	if sent("body") {
		b, _ := data.Get("body")
		v.Body = nullJSON(translated(b, codes))
	}
	if sent("audiences") {
		a, _ := data.Get("audiences")
		v.Audiences = audiences(a)
	}
	if sent("meta") {
		m, _ := data.Get("meta")
		v.Meta = db.NullRawJSON{}
		if phpval.Count(m) > 0 {
			v.Meta = db.NullRawJSON{V: form.JSON(m), Valid: true}
		}
	}
	if form.Has(data, "sort_order") {
		v.SortOrder = form.Int32(data, "sort_order", 0)
	}
	if form.Has(data, "is_active") {
		v.IsActive = httpadmin.Bool(data, "is_active")
	}
	if form.Has(data, "needs_review") {
		v.NeedsReview = httpadmin.Bool(data, "needs_review")
	}
	return v
}
