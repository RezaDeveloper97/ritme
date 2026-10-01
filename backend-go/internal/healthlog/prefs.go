package healthlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Log preferences and custom items (B-N3-02): per user and mode the category order, hidden categories and
// quick tiles (taxonomy.Resolve fills in the defaults), and the user's custom items.

// MaxCustomItems is how many active custom items a user can have.
const MaxCustomItems = 20

// MaxCustomLabel is the longest custom item label (characters).
const MaxCustomLabel = 40

// Custom item errors (the handler maps them to 422 / 404).
var (
	ErrCustomLimit     = errors.New("healthlog: custom item limit reached")
	ErrCustomDuplicate = errors.New("healthlog: custom item label taken")
	ErrCustomNotFound  = errors.New("healthlog: custom item not found")
)

// TodayPhase is today's main cycle phase for the tile defaults ("" outside the cycle modes, without data or
// when the engine cannot tell). Same inputs as the cycle engine (profile, history, metrics).
func (s *Service) TodayPhase(ctx context.Context, userID uint64, mode string, today civildate.Date) (string, error) {
	if !slices.Contains(taxonomy.PhaseModes, mode) {
		return "", nil
	}
	cq := cyclestore.New(s.raw)
	var profile *cyclestore.UserProfile
	manual := false
	p, err := cq.GetEngineProfileByUserID(ctx, userID)
	switch {
	case err == nil:
		profile = &p.UserProfile
		manual = !p.LengthsAuto
	case !errors.Is(err, sql.ErrNoRows):
		return "", fmt.Errorf("healthlog: phase profile: %w", err)
	}
	rows, err := cq.ListCycleHistoriesNewestFirst(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("healthlog: phase histories: %w", err)
	}
	histories := cycleservice.HistoriesFromRows(rows)
	engineProfile := cycleservice.ProfileFromRow(profile)
	if engineProfile != nil {
		engineProfile.LengthsManual = manual
	}
	st := resolver.Resolve(histories, engineProfile, today, today, metrics.Calculate(histories, engineProfile))
	if st.MainPhase == enums.MainPhaseUnknown || !st.MainPhase.IsValid() {
		return "", nil
	}
	return string(st.MainPhase), nil
}

// storedPrefs reads the user's saved preferences of mode (nil: none).
func (s *Service) storedPrefs(ctx context.Context, userID uint64, mode string) (*taxonomy.Stored, error) {
	row, err := s.q.GetLogPreferences(ctx, store.GetLogPreferencesParams{UserID: userID, Mode: mode})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("healthlog: load preferences: %w", err)
	}
	st := &taxonomy.Stored{}
	for _, f := range []struct {
		src rootdb.NullRawJSON
		dst *[]string
	}{{row.CategoryOrder, &st.Order}, {row.Hidden, &st.Hidden}, {row.Pinned, &st.Pinned}} {
		if !f.src.Valid {
			continue
		}
		list := []string{}
		if err := json.Unmarshal(f.src.V, &list); err != nil {
			continue // a broken list falls back to the default
		}
		*f.dst = list
	}
	return st, nil
}

// Preferences are the user's preferences in force for mode today.
func (s *Service) Preferences(ctx context.Context, userID uint64, mode string, today civildate.Date) (taxonomy.Effective, error) {
	st, err := s.storedPrefs(ctx, userID, mode)
	if err != nil {
		return taxonomy.Effective{}, err
	}
	phase, err := s.TodayPhase(ctx, userID, mode, today)
	if err != nil {
		return taxonomy.Effective{}, err
	}
	return taxonomy.Resolve(mode, phase, st), nil
}

// PrefsChange is a validated PUT: each Set field replaces the stored list (a nil list resets it to the
// default); fields not set are kept.
type PrefsChange struct {
	Order, Hidden, Pinned          []string
	OrderSet, HiddenSet, PinnedSet bool
}

func listJSON(list []string) rootdb.NullRawJSON {
	if list == nil {
		return rootdb.NullRawJSON{}
	}
	b, _ := json.Marshal(list)
	return rootdb.NullRawJSON{V: b, Valid: true}
}

// SavePreferences applies ch to the stored preferences of mode (all three lists back to default → the
// row is removed).
func (s *Service) SavePreferences(ctx context.Context, userID uint64, mode string, ch PrefsChange, now time.Time) error {
	st, err := s.storedPrefs(ctx, userID, mode)
	if err != nil {
		return err
	}
	if st == nil {
		st = &taxonomy.Stored{}
	}
	if ch.OrderSet {
		st.Order = ch.Order
	}
	if ch.HiddenSet {
		st.Hidden = ch.Hidden
	}
	if ch.PinnedSet {
		st.Pinned = ch.Pinned
	}
	if st.Order == nil && st.Hidden == nil && st.Pinned == nil {
		return s.ResetPreferences(ctx, userID, mode)
	}
	err = s.q.UpsertLogPreferences(ctx, store.UpsertLogPreferencesParams{
		UserID: userID, Mode: mode,
		CategoryOrder: listJSON(st.Order), Hidden: listJSON(st.Hidden), Pinned: listJSON(st.Pinned),
		Now: sql.NullTime{Time: dbNow(now), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("healthlog: save preferences: %w", err)
	}
	return nil
}

// ResetPreferences drops the user's preferences of mode (back to the defaults).
func (s *Service) ResetPreferences(ctx context.Context, userID uint64, mode string) error {
	if err := s.q.DeleteLogPreferences(ctx, store.DeleteLogPreferencesParams{UserID: userID, Mode: mode}); err != nil {
		return fmt.Errorf("healthlog: reset preferences: %w", err)
	}
	return nil
}

// CustomItems lists the user's custom items, oldest first; deleted ones too unless activeOnly (logged
// days keep showing their labels).
func (s *Service) CustomItems(ctx context.Context, userID uint64, activeOnly bool) ([]store.HealthLogCustomItem, error) {
	var (
		rows []store.HealthLogCustomItem
		err  error
	)
	if activeOnly {
		rows, err = s.q.ListActiveLogCustomItems(ctx, userID)
	} else {
		rows, err = s.q.ListLogCustomItems(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("healthlog: list custom items: %w", err)
	}
	return rows, nil
}

// CustomSet is the user's active custom items as the day validation reads them.
func (s *Service) CustomSet(ctx context.Context, userID uint64) (taxonomy.CustomItems, error) {
	rows, err := s.CustomItems(ctx, userID, true)
	if err != nil {
		return nil, err
	}
	set := taxonomy.CustomItems{}
	for _, r := range rows {
		set.Add(r.Category, r.Param, taxonomy.CustomItemCode(r.ID))
	}
	return set, nil
}

// NormalizeLabel trims a label and collapses inner whitespace runs to one space.
func NormalizeLabel(label string) string { return strings.Join(strings.Fields(label), " ") }

// labelTaken: another active item of the same category already has this label (case-insensitive).
func labelTaken(rows []store.HealthLogCustomItem, category, label string, except uint64) bool {
	for _, r := range rows {
		if r.ID != except && r.Category == category && strings.EqualFold(r.Label, label) {
			return true
		}
	}
	return false
}

// AddCustomItem creates a custom item in the category's custom param (label already normalized and
// validated). ErrCustomLimit past MaxCustomItems active items, ErrCustomDuplicate for a taken label.
func (s *Service) AddCustomItem(ctx context.Context, userID uint64, category, param, label string, now time.Time) (store.HealthLogCustomItem, error) {
	var out store.HealthLogCustomItem
	err := s.inTx(ctx, func(t *Service) error {
		rows, err := t.CustomItems(ctx, userID, true)
		if err != nil {
			return err
		}
		if len(rows) >= MaxCustomItems {
			return ErrCustomLimit
		}
		if labelTaken(rows, category, label, 0) {
			return ErrCustomDuplicate
		}
		at := sql.NullTime{Time: dbNow(now), Valid: true}
		id, err := t.q.InsertLogCustomItem(ctx, store.InsertLogCustomItemParams{
			UserID: userID, Category: category, Param: param, Label: label, CreatedAt: at, UpdatedAt: at,
		})
		if err != nil {
			return fmt.Errorf("healthlog: add custom item: %w", err)
		}
		out = store.HealthLogCustomItem{ID: uint64(id), UserID: userID, Category: category, Param: param, //nolint:gosec // auto-increment id
			Label: label, CreatedAt: at, UpdatedAt: at}
		return nil
	})
	return out, err
}

// customItem reads one of the user's custom items (ErrCustomNotFound for another user's or unknown id).
func (s *Service) customItem(ctx context.Context, userID, id uint64) (store.HealthLogCustomItem, error) {
	row, err := s.q.GetLogCustomItem(ctx, store.GetLogCustomItemParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return row, ErrCustomNotFound
	}
	if err != nil {
		return row, fmt.Errorf("healthlog: load custom item: %w", err)
	}
	return row, nil
}

// RenameCustomItem changes the label of an active custom item (its code and logged days stay).
func (s *Service) RenameCustomItem(ctx context.Context, userID, id uint64, label string, now time.Time) (store.HealthLogCustomItem, error) {
	var out store.HealthLogCustomItem
	err := s.inTx(ctx, func(t *Service) error {
		row, err := t.customItem(ctx, userID, id)
		if err != nil {
			return err
		}
		if row.DeletedAt.Valid {
			return ErrCustomNotFound
		}
		rows, err := t.CustomItems(ctx, userID, true)
		if err != nil {
			return err
		}
		if labelTaken(rows, row.Category, label, row.ID) {
			return ErrCustomDuplicate
		}
		at := sql.NullTime{Time: dbNow(now), Valid: true}
		if err := t.q.RenameLogCustomItem(ctx, store.RenameLogCustomItemParams{Label: label, UpdatedAt: at, ID: id, UserID: userID}); err != nil {
			return fmt.Errorf("healthlog: rename custom item: %w", err)
		}
		row.Label, row.UpdatedAt = label, at
		out = row
		return nil
	})
	return out, err
}

// DeleteCustomItem soft-deletes an active custom item: new input refuses it, logged days keep it. Pinned
// tiles are unaffected (custom items are not tiles).
func (s *Service) DeleteCustomItem(ctx context.Context, userID, id uint64, now time.Time) (store.HealthLogCustomItem, error) {
	row, err := s.customItem(ctx, userID, id)
	if err != nil {
		return row, err
	}
	if row.DeletedAt.Valid {
		return row, ErrCustomNotFound
	}
	at := sql.NullTime{Time: dbNow(now), Valid: true}
	if err := s.q.SoftDeleteLogCustomItem(ctx, store.SoftDeleteLogCustomItemParams{Now: at, ID: id, UserID: userID}); err != nil {
		return row, fmt.Errorf("healthlog: delete custom item: %w", err)
	}
	row.DeletedAt, row.UpdatedAt = at, at
	return row, nil
}
