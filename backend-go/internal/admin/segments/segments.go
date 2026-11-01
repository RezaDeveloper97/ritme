package segments

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// MinGroup is the k-anonymity floor of a segment size (k = 10, as every admin aggregate): smaller counts are null.
const MinGroup = 10

// Route is the registrar Routes takes (the admin core routes file's `h`).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve /segments.
type Handlers struct {
	q      *store.Queries
	logger *slog.Logger
}

// New wires the handlers over db.
func New(db *sql.DB, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{q: store.New(db), logger: logger}
}

// Routes mounts /segments (capability «segments»: GET view, writes manage — httpadmin.RequirePermission).
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/segments", kit.Admin(h.List))
	route(fiber.MethodPost, "/segments", kit.Admin(h.Store))
	route(fiber.MethodGet, "/segments/:id", kit.Admin(h.Show))
	route(fiber.MethodPut, "/segments/:id", kit.Admin(h.Update))
	route(fiber.MethodDelete, "/segments/:id", kit.Admin(h.Destroy))
}

// Parse reads a stored filters JSON (unknown keys or values = invalid: a broken row matches nobody).
func Parse(raw json.RawMessage) (Filters, bool) {
	var m map[string]string
	if json.Unmarshal(raw, &m) != nil {
		return Filters{}, false
	}
	var f Filters
	for k, v := range m {
		if !f.Set(k, v) {
			return Filters{}, false
		}
	}
	return f, f.Invalid() == "" && f.Q == ""
}

func segmentJSON(s store.AdminSegment) *jsonx.OrderedMap {
	f, ok := Parse(s.Filters)
	var created any
	if s.CreatedBy.Valid {
		created = s.CreatedBy.Int64
	}
	return jsonx.Obj("id", s.ID, "name", s.Name, "filters", jsonx.Obj(f.Map()...), "valid", ok,
		"created_by", created, "created_at", httpadmin.Time(s.CreatedAt), "updated_at", httpadmin.Time(s.UpdatedAt))
}

// List is GET /segments: every saved segment by name, plus the filter vocabulary (`options`).
func (h *Handlers) List(c fiber.Ctx) error {
	rows, err := h.q.ListAdminSegments(c.Context())
	if err != nil {
		return err
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, s := range rows {
		items = append(items, segmentJSON(s))
	}
	return httpadmin.OK(c, jsonx.Obj("items", items, "options", Options()))
}

func (h *Handlers) load(c fiber.Ctx) (store.AdminSegment, error) {
	id, ok := httpadmin.ID(c, "id")
	if !ok {
		return store.AdminSegment{}, httpadmin.NotFound("Segment")
	}
	s, err := h.q.GetAdminSegment(c.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		return s, httpadmin.NotFound("Segment")
	}
	return s, err
}

// Show is GET /segments/:id: the segment and its size today (null under MinGroup users; aggregates only).
func (h *Handlers) Show(c fiber.Ctx) error {
	s, err := h.load(c)
	if err != nil {
		return err
	}
	out := segmentJSON(s)
	var size any
	if f, ok := Parse(s.Filters); ok {
		n, err := h.q.CountAdminDirectory(c.Context(), f.CountParams(httpadmin.Now(c)))
		if err != nil {
			return err
		}
		if n >= MinGroup {
			size = n
		}
	}
	out.Set("size", size)
	out.Set("size_suppressed", size == nil)
	return httpadmin.OK(c, out)
}

// input validates {name, filters} (filters in the directory vocabulary, no free-text q).
func input(c fiber.Ctx) (string, json.RawMessage, error) {
	data, err := httpadmin.Validate(c, validation.Rules{
		validation.F("name", "required|string|max:80"),
		validation.F("filters", "present|array"),
	})
	if err != nil {
		return "", nil, err
	}
	var body struct {
		Filters map[string]string `json:"filters"`
	}
	const msg = "The filters must use the directory filters (no free-text search) with allowed values."
	dec := json.NewDecoder(bytes.NewReader(c.Body()))
	if dec.Decode(&body) != nil {
		return "", nil, httpadmin.FieldError("filters", msg)
	}
	var f Filters
	for k, v := range body.Filters {
		if !f.Set(k, v) || k == "q" {
			return "", nil, httpadmin.FieldError("filters."+k, msg)
		}
	}
	if bad := f.Invalid(); bad != "" {
		return "", nil, httpadmin.FieldError("filters."+bad, msg)
	}
	clean := map[string]string{}
	for _, k := range Keys {
		if v := f.Get(k); v != "" {
			clean[k] = v
		}
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", nil, err
	}
	return httpadmin.String(data, "name"), raw, nil
}

func duplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func nameTaken() error { return httpadmin.FieldError("name", "The name has already been taken.") }

// Store is POST /segments {name, filters} → 201.
func (h *Handlers) Store(c fiber.Ctx) error {
	name, raw, err := input(c)
	if err != nil {
		return err
	}
	var by sql.NullInt64
	if a := httpadmin.CurrentAdmin(c); a != nil {
		by = sql.NullInt64{Int64: int64(a.ID), Valid: true} //nolint:gosec // ids fit int64
	}
	id, err := h.q.CreateAdminSegment(c.Context(), store.CreateAdminSegmentParams{
		Name: name, Filters: raw, CreatedBy: by, Now: httpadmin.DBTime(httpadmin.Now(c))})
	if duplicate(err) {
		return nameTaken()
	}
	if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "segment.create", "segment", uint64(id)) //nolint:gosec // G115: auto-increment id
	s, err := h.q.GetAdminSegment(c.Context(), uint64(id))                //nolint:gosec // G115: auto-increment id
	if err != nil {
		return err
	}
	return httpadmin.Created(c, segmentJSON(s), "Segment created.")
}

// Update is PUT /segments/:id {name, filters}.
func (h *Handlers) Update(c fiber.Ctx) error {
	s, err := h.load(c)
	if err != nil {
		return err
	}
	name, raw, err := input(c)
	if err != nil {
		return err
	}
	if _, err := h.q.UpdateAdminSegment(c.Context(), store.UpdateAdminSegmentParams{
		Name: name, Filters: raw, Now: httpadmin.DBTime(httpadmin.Now(c)), ID: s.ID}); duplicate(err) {
		return nameTaken()
	} else if err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "segment.update", "segment", s.ID)
	s, err = h.q.GetAdminSegment(c.Context(), s.ID)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, segmentJSON(s), "Segment updated.")
}

// Destroy is DELETE /segments/:id. A message rule naming the id simply stops matching (the fact no longer has it).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	s, err := h.load(c)
	if err != nil {
		return err
	}
	if _, err := h.q.DeleteAdminSegment(c.Context(), s.ID); err != nil {
		return err
	}
	httpadmin.Audit(c, h.logger, "segment.delete", "segment", s.ID)
	return httpadmin.OK(c, jsonx.Obj("id", s.ID), "Segment deleted.")
}

// Lookup reads one saved segment's filters (for `GET /users/directory?segment=`); ok = false when the id is unknown
// or the stored filters are invalid.
func Lookup(ctx context.Context, q *store.Queries, id uint64) (Filters, bool, error) {
	s, err := q.GetAdminSegment(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Filters{}, false, nil
	}
	if err != nil {
		return Filters{}, false, err
	}
	f, ok := Parse(s.Filters)
	return f, ok, nil
}

// Resolver evaluates the saved segments for one user: the `segment` message fact (facts.SegmentSource).
type Resolver struct{ q *store.Queries }

// NewResolver reads db.
func NewResolver(db *sql.DB) *Resolver { return &Resolver{q: store.New(db)} }

// UserSegments returns the ids (decimal strings) of every saved segment whose filters match userID's directory row at
// now — two queries (her row, the segment list). An unknown user or a broken segment matches nothing.
func (r *Resolver) UserSegments(ctx context.Context, userID uint64, now time.Time) ([]string, error) {
	row, err := r.q.GetSegmentSubject(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("segments: subject: %w", err)
	}
	segs, err := r.q.ListAdminSegments(ctx)
	if err != nil {
		return nil, fmt.Errorf("segments: list: %w", err)
	}
	sub := SubjectOf(row)
	out := []string{}
	for _, s := range segs {
		if f, ok := Parse(s.Filters); ok && f.Match(sub, now) {
			out = append(out, strconv.FormatUint(s.ID, 10))
		}
	}
	return out, nil
}
