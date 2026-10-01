package checkups

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// DB is the handle the handlers read through and open write transactions on (*sql.DB).
type DB interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Handlers are the /api/v1/checkups actions. Mount them behind the locale middleware and auth
// RequireUser.
type Handlers struct {
	db    DB
	q     *store.Queries
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(db DB, base clock.Clock) *Handlers {
	return &Handlers{db: db, q: store.New(db), clock: base}
}

// Records page size of GET /checkups/records; records shown on the detail page.
const (
	recordsPerPage = 20
	detailRecords  = 2
	homeHighlights = 2
)

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func (h *Handlers) clockOf(c fiber.Ctx) clock.Clock { return clock.FromContext(c, h.clock) }

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return h.clockOf(c).Now().In(civildate.Tehran).Truncate(time.Second)
}

func langOf(c fiber.Ctx) Lang {
	return Lang{Locale: i18n.Locale(c), Default: i18n.LanguagesOf(c).DefaultCode()}
}

func notFound(c fiber.Ctx, key string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages."+key, i18n.Locale(c)))
}

func stamp(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

// List is GET /checkups (?filter=all|action|done): the age, the summary (always over the whole
// plan) and the items.
func (h *Handlers) List(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	query := pick(phpval.NewMap(), validation.Query(c), []string{"filter"})
	if err := validate(l.Locale, query, validation.Rules{validation.F("filter", "nullable", validation.In(ListFilters...))}, now); err != nil {
		return err
	}
	filter := "all"
	if s := str(query, "filter"); s != nil {
		filter = *s
	}
	p, err := LoadPlan(c, h.db, userID, h.clockOf(c))
	if err != nil {
		return err
	}
	items := []*jsonx.OrderedMap{}
	for _, it := range p.Result.Items {
		if matchesFilter(it, filter) {
			items = append(items, p.ItemJSON(it, l))
		}
	}
	var age any
	if p.Result.Age != nil {
		age = *p.Result.Age
	}
	return httpx.OK(c, jsonx.Obj("age", age, "summary", SummaryJSON(p.Result.Summary), "items", items))
}

// matchesFilter: action = due or overdue; done = recorded and not needing action.
func matchesFilter(it engine.Item, filter string) bool {
	switch filter {
	case "action":
		return it.Status == engine.StatusDue || it.Status == engine.StatusOverdue
	case "done":
		return !it.LastDoneOn.IsZero() && (it.Status == engine.StatusUpToDate || it.Status == engine.StatusSoon)
	default:
		return true
	}
}

// Home is GET /checkups/home: the summary and up to two due/overdue items (the monthly self-exam
// first, then cycle-timed, then overdue before due), in three queries. `data: null` when nothing applies.
func (h *Handlers) Home(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	p, err := LoadPlan(c, h.db, userID, h.clockOf(c))
	if err != nil {
		return err
	}
	if p.Result.Summary.Total == 0 {
		return httpx.OK(c, nil)
	}
	return httpx.OK(c, jsonx.Obj("summary", SummaryJSON(p.Result.Summary), "highlights", Highlights(p, langOf(c))))
}

// Highlights are the home card's action rows.
func Highlights(p *Plan, l Lang) []*jsonx.OrderedMap {
	var action []engine.Item
	for _, it := range p.Result.Items {
		if it.Status == engine.StatusDue || it.Status == engine.StatusOverdue {
			action = append(action, it)
		}
	}
	rank := func(it engine.Item) int {
		t := p.Types[it.TypeID]
		if t.CycleTimed() && t.Category == engine.CategoryMonthly {
			return 0 // the self-exam guide leads (artboard v14_Main)
		}
		r := 1
		if !t.CycleTimed() {
			r += 2
		}
		if it.Status != engine.StatusOverdue {
			r++
		}
		return r
	}
	slices.SortStableFunc(action, func(a, b engine.Item) int { return cmp.Compare(rank(a), rank(b)) })
	out := []*jsonx.OrderedMap{}
	for _, it := range action[:min(len(action), homeHighlights)] {
		out = append(out, p.ItemJSON(it, l))
	}
	return out
}

// Show is GET /checkups/{id}: the item, the catalog copy, the latest two records, the switches.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c, "checkup_not_found")
	}
	p, err := LoadPlan(c, h.db, userID, h.clockOf(c))
	if err != nil {
		return err
	}
	it, ok := p.Item(id)
	if !ok {
		return notFound(c, "checkup_not_found")
	}
	records, err := h.q.ListCheckupRecordsOfType(c, store.ListCheckupRecordsOfTypeParams{
		UserID: userID, CheckupTypeID: id, Limit: detailRecords,
	})
	if err != nil {
		return fmt.Errorf("checkups: list records: %w", err)
	}
	return httpx.OK(c, p.DetailJSON(it, records, langOf(c)))
}

// visibleType is an active type the user sees (shared for her life mode, or her custom), else the 404.
func (h *Handlers) visibleType(c fiber.Ctx, id, userID uint64) (store.CheckupType, error) {
	mode, err := UserLifeMode(c, h.q, userID)
	if err != nil {
		return store.CheckupType{}, err
	}
	t, err := h.q.GetCheckupTypeForUser(c, store.GetCheckupTypeForUserParams{
		ID: id, UserID: int64(userID), LifeMode: string(mode), //nolint:gosec // ids fit int64
	})
	if errors.Is(err, sql.ErrNoRows) {
		return t, notFound(c, "checkup_not_found")
	}
	if err != nil {
		return t, fmt.Errorf("checkups: find type: %w", err)
	}
	return t, nil
}

// recordColumns are the checkup_records columns of validated record data.
type recordColumns struct {
	DoneOn        civildate.Date
	Result        string
	Findings      rootdb.NullRawJSON
	Note          sql.NullString
	HasAttachment bool
	NextDueOn     civildate.NullDate
}

func recordColumnsOf(data phpval.Map) (recordColumns, error) {
	col := recordColumns{DoneOn: date(data, "done_on"), Result: *str(data, "result")}
	if findings := stringList(data, "findings"); findings != nil {
		b, err := json.Marshal(findings)
		if err != nil {
			return col, fmt.Errorf("checkups: findings: %w", err)
		}
		col.Findings = rootdb.NullRawJSON{V: b, Valid: true}
	}
	if s := str(data, "note"); s != nil {
		col.Note = sql.NullString{String: *s, Valid: true}
	}
	if v, ok := data.Get("has_attachment"); ok {
		col.HasAttachment = phpval.Truthy(v)
	}
	if d := date(data, "next_due_on"); !d.IsZero() {
		col.NextDueOn = civildate.NullDate{Date: d, Valid: true}
	}
	return col, nil
}

// StoreRecord is POST /checkups/{id}/records: 201 with {item, record} (the recomputed item and
// the new record, whose id keys the on-device attachment).
func (h *Handlers) StoreRecord(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c, "checkup_not_found")
	}
	t, err := h.visibleType(c, id, userID)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	data := pick(phpval.NewMap(), validation.Input(c), recordFields)
	if err := validate(l.Locale, data, recordRules(findingKeys(t.FindingOptions)), now); err != nil {
		return err
	}
	col, err := recordColumnsOf(data)
	if err != nil {
		return err
	}
	newID, err := h.q.InsertCheckupRecord(c, store.InsertCheckupRecordParams{
		UserID: userID, CheckupTypeID: id, DoneOn: col.DoneOn, Result: col.Result, Findings: col.Findings,
		Note: col.Note, HasAttachment: col.HasAttachment, NextDueOn: col.NextDueOn,
		CreatedAt: stamp(now), UpdatedAt: stamp(now),
	})
	if err != nil || newID <= 0 {
		return fmt.Errorf("checkups: insert record (id %d): %w", newID, err)
	}
	body, err := h.recordResult(c, userID, uint64(newID), l)
	if err != nil {
		return err
	}
	return httpx.Created(c, body, T("messages.record_created", l.Locale))
}

// recordResult is {item, record} after a record write (item null when the type left the plan).
func (h *Handlers) recordResult(c fiber.Ctx, userID, recordID uint64, l Lang) (*jsonx.OrderedMap, error) {
	rec, err := h.q.GetCheckupRecord(c, store.GetCheckupRecordParams{ID: recordID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("checkups: reload record: %w", err)
	}
	p, err := LoadPlan(c, h.db, userID, h.clockOf(c))
	if err != nil {
		return nil, err
	}
	var item, title any
	if it, ok := p.Item(rec.CheckupTypeID); ok {
		item = p.ItemJSON(it, l)
		title = l.pick(p.Rows[rec.CheckupTypeID].Title)
	}
	return jsonx.Obj("item", item, "record", RecordJSON(rec, title)), nil
}

func (h *Handlers) findRecord(c fiber.Ctx, userID uint64) (store.CheckupRecord, error) {
	id, ok := parseID(c.Params("recordId"))
	if !ok {
		return store.CheckupRecord{}, notFound(c, "record_not_found")
	}
	rec, err := h.q.GetCheckupRecord(c, store.GetCheckupRecordParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return rec, notFound(c, "record_not_found")
	}
	if err != nil {
		return rec, fmt.Errorf("checkups: find record: %w", err)
	}
	return rec, nil
}

// storedRecord is a record as request data, the base a PUT merges over.
func storedRecord(r store.CheckupRecord) phpval.Map {
	data := phpval.NewMap()
	data.Set("done_on", r.DoneOn.String())
	data.Set("result", r.Result)
	if r.Findings.Valid {
		var keys []string
		if json.Unmarshal(r.Findings.V, &keys) == nil {
			list := make([]any, 0, len(keys))
			for _, k := range keys {
				list = append(list, k)
			}
			data.Set("findings", list)
		}
	}
	if r.Note.Valid {
		data.Set("note", r.Note.String)
	}
	data.Set("has_attachment", r.HasAttachment)
	if r.NextDueOn.Valid {
		data.Set("next_due_on", r.NextDueOn.Date.String())
	}
	return data
}

// UpdateRecord is PUT /checkups/records/{recordId}: a partial update (only the keys sent change;
// `{"has_attachment": false}` is the attachment rollback), validated as a whole; {item, record}.
func (h *Handlers) UpdateRecord(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	rec, err := h.findRecord(c, userID)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	// Findings are checked against the type's options; a type the admin has since switched off
	// keeps accepting the findings already stored.
	var keys []string
	if t, err := h.visibleType(c, rec.CheckupTypeID, userID); err == nil {
		keys = findingKeys(t.FindingOptions)
	} else if f := stringList(storedRecord(rec), "findings"); f != nil {
		keys = f
	}
	data := pick(storedRecord(rec), validation.Input(c), recordFields)
	if err := validate(l.Locale, data, recordRules(keys), now); err != nil {
		return err
	}
	col, err := recordColumnsOf(data)
	if err != nil {
		return err
	}
	if err := h.q.UpdateCheckupRecord(c, store.UpdateCheckupRecordParams{
		DoneOn: col.DoneOn, Result: col.Result, Findings: col.Findings, Note: col.Note,
		HasAttachment: col.HasAttachment, NextDueOn: col.NextDueOn, UpdatedAt: stamp(now), ID: rec.ID, UserID: userID,
	}); err != nil {
		return fmt.Errorf("checkups: update record: %w", err)
	}
	body, err := h.recordResult(c, userID, rec.ID, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, body, T("messages.record_updated", l.Locale))
}

// DestroyRecord is DELETE /checkups/records/{recordId}.
func (h *Handlers) DestroyRecord(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("recordId"))
	if !ok {
		return notFound(c, "record_not_found")
	}
	n, err := h.q.DeleteCheckupRecord(c, store.DeleteCheckupRecordParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("checkups: delete record: %w", err)
	}
	if n == 0 {
		return notFound(c, "record_not_found")
	}
	return httpx.OK(c, nil, T("messages.record_deleted", i18n.Locale(c)))
}

// ListRecords is GET /checkups/records (?filter=all|this_year|with_attachment&type=&page=):
// the History timeline, newest first, {items, meta}. «امسال» is the locale calendar's year.
func (h *Handlers) ListRecords(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	query := pick(phpval.NewMap(), validation.Query(c), []string{"filter", "type"})
	if err := validate(l.Locale, query, validation.Rules{
		validation.F("filter", "nullable", validation.In(RecordFilters...)),
		validation.F("type", "nullable", "integer", "min:1"),
	}, now); err != nil {
		return err
	}
	params := store.ListCheckupRecordHistoryParams{UserID: userID}
	if s := str(query, "type"); s != nil {
		params.TypeID, _ = parseID(*s)
	}
	switch f := str(query, "filter"); {
	case f == nil:
	case *f == "this_year":
		params.FromDate = YearStart(civildate.InTehran(now), l.Locale)
	case *f == "with_attachment":
		params.WithAttachment = 1
	}
	if params.FromDate.IsZero() {
		params.FromDate = civildate.New(1, 1, 1)
	}
	total, err := h.q.CountCheckupRecordHistory(c, store.CountCheckupRecordHistoryParams{
		UserID: params.UserID, TypeID: params.TypeID, FromDate: params.FromDate, WithAttachment: params.WithAttachment,
	})
	if err != nil {
		return fmt.Errorf("checkups: count records: %w", err)
	}
	page := httpx.NewPage([]any{}, int(total), recordsPerPage, httpx.PageParam(c), httpx.RequestURL(c))
	params.Limit, params.Offset = recordsPerPage, int32(min(page.Offset(), 1<<30)) //nolint:gosec // bounded
	rows, err := h.q.ListCheckupRecordHistory(c, params)
	if err != nil {
		return fmt.Errorf("checkups: list records: %w", err)
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, historyJSON(r, l))
	}
	return httpx.OK(c, jsonx.Obj("items", items, "meta", page.Meta()))
}

// PreviewNext is GET /checkups/preview-next?type=&done_on=: when a checkup done on done_on is
// next due, for the MarkDone banner (the user's override is not applied).
func (h *Handlers) PreviewNext(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	query := pick(phpval.NewMap(), validation.Query(c), []string{"type", "done_on"})
	if err := validate(l.Locale, query, validation.Rules{
		validation.F("type", "required", "integer", "min:1"),
		validation.F("done_on", append([]any{"required"}, pastDate()...)...),
	}, now); err != nil {
		return err
	}
	typeID, _ := parseID(*str(query, "type"))
	p, err := LoadPlan(c, h.db, userID, h.clockOf(c))
	if err != nil {
		return err
	}
	t, ok := p.Types[typeID]
	if !ok {
		return notFound(c, "checkup_not_found")
	}
	due := engine.NextDueAfter(t, date(query, "done_on"), p.Input.Cycle)
	return httpx.OK(c, jsonx.Obj(
		"next_due_on", due.From.String(),
		"due_by", due.By.String(),
		"next_due_label", FullDate(due.From, l.Locale),
		"interval_label", IntervalLabel(t.IntervalMonths, t.IntervalMonthsMax, l.Locale),
		"timing_label", emptyToNil(TimingLabel(t, l.Locale)),
		"reminder_label", ReminderLabel(t.RemindLeadDays, p.Remind(typeID), l.Locale),
		"cycle_timed", due.CycleTimed,
	))
}
