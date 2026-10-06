// Package messages is the smart-message editor of the admin API (Admin\MessageContentController):
// list message_contents rows filtered by group, locale and approval, edit a row's
// payload and label (keeping the payload's shape), approve / unapprove and
// activate / deactivate. Rows are only served to users when active and approved.
//
// T-M7-06 adds creation of rows for registered (group, item_key) pairs (see ./registry),
// the list of registered rows that are still missing, and schema validation of typed
// (pregnancy v2) payloads on update.
package messages

import (
	"database/sql"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/messages/registry"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Status filter values (?status=).
const (
	StatusApproved = "approved"
	StatusPending  = "pending"
)

// Handlers serve /messages.
type Handlers struct {
	q      *store.Queries
	pq     *pstore.Queries // message_contents writes of T-M7-06 (queries in db/queries/pregnancy/v2_admin.sql)
	logger *slog.Logger
}

// New builds the handlers.
func New(db *sql.DB, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{q: store.New(db), pq: pstore.New(db), logger: logger}
}

// Routes registers the endpoints.
func (h *Handlers) Routes(route func(method, path string, chain httpadmin.Chain), kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/messages", kit.Admin(h.List))
	route(fiber.MethodPost, "/messages", kit.Admin(h.Store))
	route(fiber.MethodGet, "/messages/registry", kit.Admin(h.Registry))
	route(fiber.MethodGet, "/messages/registry/:group/:key", kit.Admin(h.RegistryItem))
	route(fiber.MethodGet, "/messages/:id", kit.Admin(h.Show))
	route(fiber.MethodPut, "/messages/:id", kit.Admin(h.Update))
	route(fiber.MethodPost, "/messages/:id/approve", kit.Admin(h.Approve))
	route(fiber.MethodPost, "/messages/:id/toggle", kit.Admin(h.Toggle))
}

func messageJSON(m *store.MessageContent) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", m.ID,
		"group", m.Group,
		"item_key", m.ItemKey,
		"locale", m.Locale,
		"label", httpadmin.NullString(m.Label),
		"payload", form.Raw(m.Payload),
		"is_active", m.IsActive,
		"is_approved", m.IsApproved,
		"sort_order", m.SortOrder,
		"created_at", httpadmin.Time(m.CreatedAt),
		"updated_at", httpadmin.Time(m.UpdatedAt),
	)
}

// canWrite is the write gate of a group: clinical groups (registry.SuperOnly) are super-admin only (403 otherwise).
func canWrite(c fiber.Ctx, group string) error {
	if !registry.SuperOnly(group) {
		return nil
	}
	if a := httpadmin.CurrentAdmin(c); a == nil || a.Role != httpadmin.RoleSuper {
		return httpadmin.Forbidden()
	}
	return nil
}

func (h *Handlers) find(c fiber.Ctx) (*store.MessageContent, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return nil, httpadmin.NotFound("Message")
	}
	m, err := h.q.GetMessageContent(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpadmin.NotFound("Message")
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// List is GET /messages?group=&locale=&status=approved|pending&page=&per_page=
// (ordered by group, item_key, locale). data also carries the known groups and locales
// for the filter bar.
func (h *Handlers) List(c fiber.Ctx) error {
	group, locale, status := c.Query("group"), c.Query("locale"), c.Query("status")
	var only *bool
	switch status {
	case StatusApproved:
		only = new(bool)
		*only = true
	case StatusPending:
		only = new(bool)
	default:
		status = ""
	}
	lo, hi := form.BoolRange(only)
	total, err := h.q.CountAdminMessages(c.Context(), store.CountAdminMessagesParams{
		GroupPattern: form.Exact(group), LocalePattern: form.Exact(locale), ApprovedMin: lo, ApprovedMax: hi,
	})
	if err != nil {
		return err
	}
	p := httpadmin.PageOf(c)
	rows, err := h.q.ListAdminMessages(c.Context(), store.ListAdminMessagesParams{
		GroupPattern: form.Exact(group), LocalePattern: form.Exact(locale), ApprovedMin: lo, ApprovedMax: hi,
		Limit: int32(p.PerPage), Offset: int32(min(p.Offset(), 1<<30)), //nolint:gosec // G115: bounded by PageOf
	})
	if err != nil {
		return err
	}
	groups, err := h.q.ListMessageGroups(c.Context())
	if err != nil {
		return err
	}
	locales, err := h.q.ListMessageLocales(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for i := range rows {
		items = append(items, messageJSON(&rows[i]))
	}
	nullable := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	page := httpadmin.Page(items, p, int(total))
	page.Set("filters", jsonx.Obj("group", nullable(group), "locale", nullable(locale), "status", nullable(status)))
	page.Set("groups", jsonx.List(groups))
	page.Set("locales", jsonx.List(locales))
	page.Set("registered_groups", jsonx.List(registry.GroupNames()))
	page.Set("super_only_groups", jsonx.List(registry.SuperOnlyGroups))
	missing, err := h.missing(c, group, locale)
	if err != nil {
		return err
	}
	page.Set("missing", jsonx.List(missing))
	return httpadmin.OK(c, page)
}

// Show is GET /messages/:id.
func (h *Handlers) Show(c fiber.Ctx) error {
	m, err := h.find(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("message", messageJSON(m)))
}

var lineBreak = regexp.MustCompile(`\r\n|\r|\n`)

// mergePayload is MessageContentController::update: the stored payload's keys and shape
// are kept. A list field takes a newline-separated string (or a list of strings), trimmed,
// empty lines dropped; a scalar field takes a string, anything else keeps the stored
// value. Keys that are not in the stored payload are ignored.
func mergePayload(original any, input any) any {
	out := phpval.NewMap()
	keys, vals := phpval.Entries(original)
	for i, k := range keys {
		v, _ := phpval.Get(input, k)
		if !phpval.IsArray(input) {
			v = nil
		}
		if phpval.IsArray(vals[i]) {
			var lines []string
			if phpval.IsArray(v) {
				_, items := phpval.Entries(v)
				for _, it := range items {
					if s, ok := it.(string); ok {
						lines = append(lines, s)
					}
				}
			} else {
				lines = lineBreak.Split(phpval.ToString(v), -1)
			}
			list := []any{}
			for _, l := range lines {
				if t := strings.Trim(l, " \t\n\r\x00\x0B"); t != "" {
					list = append(list, t)
				}
			}
			out.Set(k, list)
			continue
		}
		if s, ok := v.(string); ok {
			out.Set(k, s)
		} else {
			out.Set(k, vals[i])
		}
	}
	return out
}

// Update is PUT /messages/:id {payload: {field: string | [strings]}, label?}. An absent
// label keeps the stored one; an empty label clears it. Rows of a typed registry item
// (pregnancy v2 groups) are validated against their schema instead (typedPayload).
func (h *Handlers) Update(c fiber.Ctx) error {
	m, err := h.find(c)
	if err != nil {
		return err
	}
	if err := canWrite(c, m.Group); err != nil {
		return err
	}
	data, err := form.Validate(c, validation.Rules{
		validation.F("payload", "nullable|array"),
		validation.F("label", "nullable|string|max:255"),
	})
	if err != nil {
		return err
	}
	original, err := phpval.Decode(m.Payload)
	if err != nil {
		original = phpval.NewMap()
	}
	input, _ := data.Get("payload")
	var payload any
	if item, ok := registry.Lookup(m.Group, m.ItemKey); ok && item.Typed {
		if payload, err = typedPayload(c, item, original, input); err != nil {
			return err
		}
	} else {
		payload = mergePayload(original, input)
	}
	label := m.Label
	if _, sent := data.Get("label"); sent {
		label = form.Str(data, "label")
	}
	if err := h.q.UpdateMessageContent(c.Context(), store.UpdateMessageContentParams{
		Payload: form.JSON(payload), Label: label,
		Now: httpadmin.DBTime(httpadmin.Now(c)), ID: m.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message.update", "message", m.ID)
	return h.respond(c, m.ID, "Message updated.")
}

// Approve is POST /messages/:id/approve: toggles is_approved (approve / unapprove).
func (h *Handlers) Approve(c fiber.Ctx) error {
	m, err := h.find(c)
	if err != nil {
		return err
	}
	if err := canWrite(c, m.Group); err != nil {
		return err
	}
	if err := h.q.ToggleMessageApproved(c.Context(), store.ToggleMessageApprovedParams{
		Now: httpadmin.DBTime(httpadmin.Now(c)), ID: m.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message.approve", "message", m.ID, slog.Bool("is_approved", !m.IsApproved))
	msg := "Message approved."
	if m.IsApproved {
		msg = "Message approval withdrawn."
	}
	return h.respond(c, m.ID, msg)
}

// Toggle is POST /messages/:id/toggle: toggles is_active.
func (h *Handlers) Toggle(c fiber.Ctx) error {
	m, err := h.find(c)
	if err != nil {
		return err
	}
	if err := canWrite(c, m.Group); err != nil {
		return err
	}
	if err := h.q.ToggleMessageActive(c.Context(), store.ToggleMessageActiveParams{
		Now: httpadmin.DBTime(httpadmin.Now(c)), ID: m.ID,
	}); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "message.toggle", "message", m.ID, slog.Bool("is_active", !m.IsActive))
	return h.respond(c, m.ID, "Message status changed.")
}

func (h *Handlers) respond(c fiber.Ctx, id uint64, msg string) error {
	m, err := h.q.GetMessageContent(c.Context(), id)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, jsonx.Obj("message", messageJSON(&m)), msg)
}
