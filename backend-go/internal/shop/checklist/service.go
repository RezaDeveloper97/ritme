package checklist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/shop/store"
)

// DueDateSource is the due date of the user's active pregnancy (ok=false when she is not pregnant or not dated).
type DueDateSource interface {
	DueDate(ctx context.Context, userID uint64, today civildate.Date) (civildate.Date, bool, error)
}

// PeriodPredictor is the cycle engine's predicted next period start (ok=false when unknown or not applicable). It is
// the supply reminder's only health input; implementations must not log it.
type PeriodPredictor interface {
	NextPeriodStart(ctx context.Context, userID uint64, today civildate.Date) (civildate.Date, bool, error)
}

// ErrNotFound is a missing (or someone else's) custom row, or an inactive template item.
var ErrNotFound = errors.New("checklist: not found")

// ErrTooMany is the custom-row limit.
var ErrTooMany = errors.New("checklist: too many custom items")

// Service reads and writes the checklist and the supply opt-in.
type Service struct {
	q       store.Querier
	due     DueDateSource
	predict PeriodPredictor
}

// NewService wires the service; due and predict may be nil (no due date / no reminder ever planned).
func NewService(q store.Querier, due DueDateSource, predict PeriodPredictor) *Service {
	return &Service{q: q, due: due, predict: predict}
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func nullQty(n int) sql.NullInt16 {
	if n <= 0 {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: int16(min(n, MaxQuantity)), Valid: true} //nolint:gosec // capped at MaxQuantity
}

func qty(n sql.NullInt16) int {
	if !n.Valid {
		return 0
	}
	return int(n.Int16)
}

// Load assembles userID's list; pick resolves a translatable template column for the request.
func (s *Service) Load(ctx context.Context, userID uint64, today civildate.Date, pick func(string) string) (View, error) {
	tpl, err := s.q.ListLayetteTemplate(ctx)
	if err != nil {
		return View{}, fmt.Errorf("checklist: template: %w", err)
	}
	rows, err := s.q.ListUserChecklistItems(ctx, userID)
	if err != nil {
		return View{}, fmt.Errorf("checklist: user items: %w", err)
	}
	mine := make(map[uint64]store.ShopUserChecklistItem, len(rows))
	var custom []Item
	for _, r := range rows {
		if r.LayetteItemID.Valid {
			mine[uint64(r.LayetteItemID.Int64)] = r //nolint:gosec // ids are positive
			continue
		}
		custom = append(custom, Item{
			ID: r.ID, Group: r.GroupKey, Title: r.CustomTitle.String, State: r.State, Quantity: qty(r.Quantity),
			SortOrder: r.SortOrder,
		})
	}
	items := make([]Item, 0, len(tpl)+len(custom))
	for _, t := range tpl {
		it := Item{
			LayetteItemID: t.ID, Group: t.GroupKey, Title: pick(t.Title), SuggestedQty: qty(t.SuggestedQty),
			IsEssential: t.IsEssential, IsFirstDays: t.IsFirstDays, CategorySlug: t.CategorySlug.String,
			State: StateTodo, SortOrder: t.SortOrder,
		}
		if t.Note.Valid {
			it.Note = pick(t.Note.String)
		}
		if r, ok := mine[t.ID]; ok {
			it.ID, it.State, it.Quantity = r.ID, r.State, qty(r.Quantity)
		}
		items = append(items, it)
	}
	items = append(items, custom...)
	groups, sum := Assemble(items)
	v := View{Groups: groups, Summary: sum, WeeksLeft: -1}
	if s.due != nil {
		due, ok, err := s.due.DueDate(ctx, userID, today)
		if err != nil {
			return View{}, fmt.Errorf("checklist: due date: %w", err)
		}
		if ok {
			v.DueDate, v.WeeksLeft = due, WeeksLeft(due, today)
		}
	}
	return v, nil
}

// SetState records her state (and quantity) on a template item.
func (s *Service) SetState(ctx context.Context, userID, layetteItemID uint64, state string, quantity int, now time.Time) error {
	t, err := s.q.GetActiveLayetteItem(ctx, layetteItemID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("checklist: template item: %w", err)
	}
	err = s.q.SetUserChecklistState(ctx, store.SetUserChecklistStateParams{
		UserID: userID, LayetteItemID: sql.NullInt64{Int64: int64(t.ID), Valid: true}, //nolint:gosec // ids are positive
		GroupKey: t.GroupKey, State: state, Quantity: nullQty(quantity), CreatedAt: nullTime(now), UpdatedAt: nullTime(now),
	})
	if err != nil {
		return fmt.Errorf("checklist: set state: %w", err)
	}
	return nil
}

// CustomInput is a validated custom row.
type CustomInput struct {
	Title, Group, State string
	Quantity            int
}

// AddCustom inserts her own row and returns its id.
func (s *Service) AddCustom(ctx context.Context, userID uint64, in CustomInput, now time.Time) (uint64, error) {
	n, err := s.q.CountCustomChecklistItems(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("checklist: count: %w", err)
	}
	if n >= MaxCustomItems {
		return 0, ErrTooMany
	}
	id, err := s.q.InsertCustomChecklistItem(ctx, store.InsertCustomChecklistItemParams{
		UserID: userID, CustomTitle: sql.NullString{String: in.Title, Valid: true}, GroupKey: in.Group, State: in.State,
		Quantity: nullQty(in.Quantity), Now: nullTime(now),
	})
	if err != nil {
		return 0, fmt.Errorf("checklist: add custom: %w", err)
	}
	return uint64(id), nil //nolint:gosec // auto-increment ids are positive
}

// Custom reads one of her custom rows (ErrNotFound for someone else's).
func (s *Service) Custom(ctx context.Context, userID, id uint64) (CustomInput, error) {
	r, err := s.q.GetCustomChecklistItem(ctx, store.GetCustomChecklistItemParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return CustomInput{}, ErrNotFound
	}
	if err != nil {
		return CustomInput{}, fmt.Errorf("checklist: custom: %w", err)
	}
	return CustomInput{Title: r.CustomTitle.String, Group: r.GroupKey, State: r.State, Quantity: qty(r.Quantity)}, nil
}

// UpdateCustom replaces one of her custom rows.
func (s *Service) UpdateCustom(ctx context.Context, userID, id uint64, in CustomInput, now time.Time) error {
	n, err := s.q.UpdateCustomChecklistItem(ctx, store.UpdateCustomChecklistItemParams{
		CustomTitle: sql.NullString{String: in.Title, Valid: true}, GroupKey: in.Group, State: in.State,
		Quantity: nullQty(in.Quantity), Now: nullTime(now), ID: id, UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("checklist: update custom: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCustom removes one of her custom rows.
func (s *Service) DeleteCustom(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteCustomChecklistItem(ctx, store.DeleteCustomChecklistItemParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("checklist: delete custom: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------- supply reminder

// SupplyState is the opt-in with the day its care reminder is set for (zero: none).
type SupplyState struct {
	Supply
	RemindOn civildate.Date
}

// Supply reads the opt-in (default when she never saved it).
func (s *Service) Supply(ctx context.Context, userID uint64) (Supply, error) {
	r, err := s.q.GetSupplyReminder(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Supply{LeadDays: DefaultLeadDays}, nil
	}
	if err != nil {
		return Supply{}, fmt.Errorf("checklist: supply: %w", err)
	}
	out := Supply{Enabled: r.IsEnabled, LeadDays: int(r.LeadDays)}
	if r.ProductID.Valid {
		out.ProductID = uint64(r.ProductID.Int64) //nolint:gosec // ids are positive
	}
	return out, nil
}

// ProductAvailable reports whether id is an active shop product.
func (s *Service) ProductAvailable(ctx context.Context, id uint64) (bool, error) {
	n, err := s.q.CountActiveProduct(ctx, id)
	if err != nil {
		return false, fmt.Errorf("checklist: product: %w", err)
	}
	return n > 0, nil
}

// SaveSupply stores the opt-in and syncs its care reminder.
func (s *Service) SaveSupply(ctx context.Context, userID uint64, in Supply, today civildate.Date, now time.Time, locale string,
) (SupplyState, error) {
	product := sql.NullInt64{}
	if in.ProductID > 0 {
		product = sql.NullInt64{Int64: int64(in.ProductID), Valid: true} //nolint:gosec // ids are positive
	}
	err := s.q.UpsertSupplyReminder(ctx, store.UpsertSupplyReminderParams{
		UserID: userID, IsEnabled: in.Enabled, LeadDays: uint8(in.LeadDays), //nolint:gosec // validated 1..7
		ProductID: product, CreatedAt: nullTime(now), UpdatedAt: nullTime(now),
	})
	if err != nil {
		return SupplyState{}, fmt.Errorf("checklist: save supply: %w", err)
	}
	return s.SyncSupply(ctx, userID, today, now, locale)
}

// SyncSupply brings userID's care reminder in line with her opt-in and the current prediction: off (or no
// prediction) → no reminder row; on → exactly one, LeadDays before the predicted start. A scheduler may call it daily
// for enabled users to roll the reminder on to the next cycle; GET /shop/supply-reminder calls it too.
func (s *Service) SyncSupply(ctx context.Context, userID uint64, today civildate.Date, now time.Time, locale string,
) (SupplyState, error) {
	sup, err := s.Supply(ctx, userID)
	if err != nil {
		return SupplyState{}, err
	}
	st := SupplyState{Supply: sup}
	var day civildate.Date
	ok := false
	if sup.Enabled && s.predict != nil {
		next, known, err := s.predict.NextPeriodStart(ctx, userID, today)
		if err != nil {
			return SupplyState{}, fmt.Errorf("checklist: prediction: %w", err)
		}
		if known {
			day, ok = RemindOn(sup, next, today)
		}
	}
	if !ok {
		if err := s.q.DeleteSupplyCareReminders(ctx, userID); err != nil {
			return SupplyState{}, fmt.Errorf("checklist: clear reminder: %w", err)
		}
		return st, nil
	}
	st.RemindOn = day
	at := sql.NullTime{Time: ReminderAt(day), Valid: true}
	title := T("reminder.title", locale)
	cur, err := s.q.FindSupplyCareReminder(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		meta, _ := json.Marshal(map[string]string{"source": ReminderSource}) // a string map always marshals
		_, err = s.q.InsertSupplyCareReminder(ctx, store.InsertSupplyCareReminderParams{
			UserID: userID, Title: title, ScheduledAt: at, Meta: db.NullRawJSON{V: meta, Valid: true}, Now: nullTime(now),
		})
	case err == nil:
		if cur.ScheduledAt.Valid && cur.ScheduledAt.Time.Equal(at.Time) && cur.Title == title {
			return st, nil
		}
		err = s.q.UpdateSupplyCareReminder(ctx, store.UpdateSupplyCareReminderParams{
			Title: title, ScheduledAt: at, Now: nullTime(now), ID: cur.ID, UserID: userID,
		})
	}
	if err != nil {
		return SupplyState{}, fmt.Errorf("checklist: set reminder: %w", err)
	}
	return st, nil
}
