package checkups

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// performerIcons are the icon of a custom checkup by who performs it (frontend icon names).
var performerIcons = map[string]string{"self": "heart", "doctor": "stethoscope", "lab": "flask", "dentist": "tooth"}

// customLeadDays is the reminder lead of a custom checkup: 3 days for a monthly one, a week up to
// six months, two weeks beyond.
func customLeadDays(intervalMonths int) uint16 {
	switch {
	case intervalMonths <= 1:
		return 3
	case intervalMonths <= 6:
		return 7
	default:
		return 14
	}
}

// customColumns are the checkup_types columns of a validated custom checkup. The user's texts
// are stored under the request language (the bilingual pick falls back to them in any other).
type customColumns struct {
	Title          json.RawMessage
	Subtitle       rootdb.NullRawJSON
	PerformedBy    string
	Icon           sql.NullString
	IntervalMonths uint16
	RemindLeadDays uint16
	LastDoneOn     civildate.Date
}

func customColumnsOf(data phpval.Map, locale string) (customColumns, error) {
	title, err := json.Marshal(map[string]string{locale: *str(data, "title")})
	if err != nil {
		return customColumns{}, fmt.Errorf("checkups: custom title: %w", err)
	}
	performer := *str(data, "performed_by")
	months := int(phpval.ToFloat(mustGet(data, "interval_months")))
	col := customColumns{
		Title: title, PerformedBy: performer,
		Icon:           sql.NullString{String: performerIcons[performer], Valid: performerIcons[performer] != ""},
		IntervalMonths: uint16(months), //nolint:gosec // validated 1..120
		RemindLeadDays: customLeadDays(months),
		LastDoneOn:     date(data, "last_done_on"),
	}
	if note := str(data, "note"); note != nil {
		b, err := json.Marshal(map[string]string{locale: *note})
		if err != nil {
			return customColumns{}, fmt.Errorf("checkups: custom note: %w", err)
		}
		col.Subtitle = rootdb.NullRawJSON{V: b, Valid: true}
	}
	return col, nil
}

func mustGet(data phpval.Map, key string) any {
	v, _ := data.Get(key)
	return v
}

// seedRecord records last_done_on as a normal result, unless the latest record already has
// that date.
func seedRecord(c fiber.Ctx, q *store.Queries, userID, typeID uint64, doneOn civildate.Date, now time.Time) error {
	if doneOn.IsZero() {
		return nil
	}
	latest, err := q.ListCheckupRecordsOfType(c, store.ListCheckupRecordsOfTypeParams{UserID: userID, CheckupTypeID: typeID, Limit: 1})
	if err != nil {
		return fmt.Errorf("checkups: latest record: %w", err)
	}
	if len(latest) > 0 && latest[0].DoneOn == doneOn {
		return nil
	}
	if _, err := q.InsertCheckupRecord(c, store.InsertCheckupRecordParams{
		UserID: userID, CheckupTypeID: typeID, DoneOn: doneOn, Result: "normal",
		CreatedAt: stamp(now), UpdatedAt: stamp(now),
	}); err != nil {
		return fmt.Errorf("checkups: seed record: %w", err)
	}
	return nil
}

// itemOf is the recomputed plan item of a type (null when it is not in the plan).
func (h *Handlers) itemOf(c fiber.Ctx, userID, typeID uint64, l Lang) (any, error) {
	p, err := LoadPlan(c, h.db, userID, h.clockOf(c))
	if err != nil {
		return nil, err
	}
	if it, ok := p.Item(typeID); ok {
		return p.ItemJSON(it, l), nil
	}
	return nil, nil
}

// StoreCustom is POST /checkups/custom: the user's own checkup (title, interval_months,
// performed_by, note, optional last_done_on seeding a first record) → 201 with the item.
func (h *Handlers) StoreCustom(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	data := pick(phpval.NewMap(), validation.Input(c), customFields)
	if err := validate(l.Locale, data, customRules(), now); err != nil {
		return err
	}
	col, err := customColumnsOf(data, l.Locale)
	if err != nil {
		return err
	}
	tx, err := h.db.BeginTx(c, nil)
	if err != nil {
		return fmt.Errorf("checkups: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := store.New(tx)
	if err := checkCustomCap(c, q, userID, l.Locale); err != nil {
		return err
	}
	id, err := q.InsertCustomCheckupType(c, store.InsertCustomCheckupTypeParams{
		UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // ids fit int64
		Title:  col.Title, Subtitle: col.Subtitle, PerformedBy: col.PerformedBy, Icon: col.Icon,
		IntervalMonths: col.IntervalMonths, RemindLeadDays: col.RemindLeadDays, SortOrder: customSortOrder,
		CreatedAt: stamp(now), UpdatedAt: stamp(now),
	})
	if err != nil || id <= 0 {
		return fmt.Errorf("checkups: insert custom (id %d): %w", id, err)
	}
	if err := seedRecord(c, q, userID, uint64(id), col.LastDoneOn, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("checkups: commit: %w", err)
	}
	item, err := h.itemOf(c, userID, uint64(id), l)
	if err != nil {
		return err
	}
	return httpx.Created(c, item, T("messages.custom_created", l.Locale))
}

func (h *Handlers) findCustom(c fiber.Ctx, userID uint64) (store.CheckupType, error) {
	id, ok := parseID(c.Params("id"))
	if !ok {
		return store.CheckupType{}, notFound(c, "checkup_not_found")
	}
	t, err := h.q.GetCustomCheckupType(c, store.GetCustomCheckupTypeParams{
		ID: id, UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // ids fit int64
	})
	if errors.Is(err, sql.ErrNoRows) {
		return t, notFound(c, "checkup_not_found")
	}
	if err != nil {
		return t, fmt.Errorf("checkups: find custom: %w", err)
	}
	return t, nil
}

// UpdateCustom is PUT /checkups/custom/{id}: a partial update of the user's own checkup (only
// the keys sent change); a new last_done_on adds a record. 200 with the item.
func (h *Handlers) UpdateCustom(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	t, err := h.findCustom(c, userID)
	if err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	base := phpval.NewMap()
	base.Set("title", i18n.PickString(t.Title, l.Locale, l.Default))
	base.Set("interval_months", int(t.IntervalMonths))
	base.Set("performed_by", t.PerformedBy)
	if t.Subtitle.Valid {
		base.Set("note", i18n.PickString(t.Subtitle.V, l.Locale, l.Default))
	}
	data := pick(base, validation.Input(c), customFields)
	if err := validate(l.Locale, data, customRules(), now); err != nil {
		return err
	}
	col, err := customColumnsOf(data, l.Locale)
	if err != nil {
		return err
	}
	tx, err := h.db.BeginTx(c, nil)
	if err != nil {
		return fmt.Errorf("checkups: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := store.New(tx)
	if err := q.UpdateCustomCheckupType(c, store.UpdateCustomCheckupTypeParams{
		Title: col.Title, Subtitle: col.Subtitle, PerformedBy: col.PerformedBy, Icon: col.Icon,
		IntervalMonths: col.IntervalMonths, RemindLeadDays: col.RemindLeadDays, UpdatedAt: stamp(now),
		ID: t.ID, UserID: t.UserID,
	}); err != nil {
		return fmt.Errorf("checkups: update custom: %w", err)
	}
	if err := seedRecord(c, q, userID, t.ID, col.LastDoneOn, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("checkups: commit: %w", err)
	}
	item, err := h.itemOf(c, userID, t.ID, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, item, T("messages.custom_updated", l.Locale))
}

// DestroyCustom is DELETE /checkups/custom/{id}; its records and settings go with it (cascade).
func (h *Handlers) DestroyCustom(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c, "checkup_not_found")
	}
	n, err := h.q.DeleteCustomCheckupType(c, store.DeleteCustomCheckupTypeParams{
		ID: id, UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // ids fit int64
	})
	if err != nil {
		return fmt.Errorf("checkups: delete custom: %w", err)
	}
	if n == 0 {
		return notFound(c, "checkup_not_found")
	}
	return httpx.OK(c, nil, T("messages.custom_deleted", i18n.Locale(c)))
}

// UpdateSettings is PUT /checkups/{id}/settings {enabled?, remind?}: the plan switch and the
// reminder bell of a type the user sees (partial; a missing row counts as both on).
func (h *Handlers) UpdateSettings(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	id, ok := parseID(c.Params("id"))
	if !ok {
		return notFound(c, "checkup_not_found")
	}
	if _, err := h.visibleType(c, id, userID); err != nil {
		return err
	}
	l, now := langOf(c), h.now(c)
	cur, err := h.q.GetCheckupSetting(c, store.GetCheckupSettingParams{UserID: userID, CheckupTypeID: id})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		cur.Enabled, cur.Remind = true, true
	case err != nil:
		return fmt.Errorf("checkups: find setting: %w", err)
	}
	base := phpval.NewMap()
	base.Set("enabled", cur.Enabled)
	base.Set("remind", cur.Remind)
	data := pick(base, validation.Input(c), []string{"enabled", "remind"})
	if err := validate(l.Locale, data, validation.Rules{
		validation.F("enabled", "required", "boolean"),
		validation.F("remind", "required", "boolean"),
	}, now); err != nil {
		return err
	}
	enabled, remind := phpval.Truthy(mustGet(data, "enabled")), phpval.Truthy(mustGet(data, "remind"))
	if err := h.q.UpsertCheckupSetting(c, store.UpsertCheckupSettingParams{
		UserID: userID, CheckupTypeID: id, Enabled: enabled, Remind: remind, Now: stamp(now),
	}); err != nil {
		return fmt.Errorf("checkups: save setting: %w", err)
	}
	item, err := h.itemOf(c, userID, id, l)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("settings", jsonx.Obj("enabled", enabled, "remind", remind), "item", item),
		T("messages.settings_updated", l.Locale))
}
