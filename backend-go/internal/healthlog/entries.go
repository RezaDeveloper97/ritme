package healthlog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Log taxonomy v2 (B-N3-01): the day log as health_log_entries rows, and the two-way sync with
// daily_health_logs while both exist:
//
//   - an old-endpoint write (POST /health-logs, the fertility day) projects the columns it wrote onto
//     their v2 slots (source "legacy");
//   - a v2 write (PUT /logs/days/{date}) puts the legacy-representable values of the params it changed
//     back into daily_health_logs through the same updateOrCreate + period-start reconciliation the old
//     endpoint runs, so the cycle engine, fertility, messages and the export keep reading what the user
//     logged. v2-only values (a pain score, pink blood, custom items…) live in v2 only.
//
// No health value is ever logged.

// Source codes of health_log_entries rows.
const (
	SourceManual = "manual"
	SourceLegacy = taxonomy.SourceLegacy
	SourceVoice  = "voice" // B-N3-05: values confirmed from a voice-log suggestion
)

// MaxRangeDays bounds GET /logs/days.
const MaxRangeDays = 366

func toEntry(r store.HealthLogEntry) taxonomy.Entry {
	return taxonomy.Entry{Category: r.Category, Param: r.Param, Item: r.Item, Code: r.ValueCode, Num: r.ValueNum, Text: r.ValueText}
}

func (s *Service) insertEntries(ctx context.Context, userID uint64, date civildate.Date, entries []taxonomy.Entry, source string, now time.Time) error {
	ts := sql.NullTime{Time: now, Valid: true}
	for _, e := range entries {
		if err := s.q.InsertLogEntry(ctx, store.InsertLogEntryParams{
			UserID: userID, LogDate: date, Category: e.Category, Param: e.Param, Item: e.Item,
			ValueCode: e.Code, ValueNum: e.Num, ValueText: e.Text, Source: source, CreatedAt: ts, UpdatedAt: ts,
		}); err != nil {
			return fmt.Errorf("healthlog: insert entry: %w", err)
		}
	}
	return nil
}

// deleteLegacySlots removes the v2 slots the given legacy columns write (nil = every mapped slot).
func (s *Service) deleteLegacySlots(ctx context.Context, userID uint64, date civildate.Date, columns []string) error {
	params, items := taxonomy.SlotsOf(columns)
	for _, p := range params {
		cat, param, _ := strings.Cut(p, ".")
		if err := s.q.DeleteLogEntryParam(ctx, store.DeleteLogEntryParamParams{
			UserID: userID, LogDate: date, Category: cat, Param: param,
		}); err != nil {
			return fmt.Errorf("healthlog: delete entries: %w", err)
		}
	}
	for _, slot := range items {
		parts := strings.SplitN(slot, ".", 3)
		if err := s.q.DeleteLogEntrySlot(ctx, store.DeleteLogEntrySlotParams{
			UserID: userID, LogDate: date, Category: parts[0], Param: parts[1], Item: parts[2],
		}); err != nil {
			return fmt.Errorf("healthlog: delete entry: %w", err)
		}
	}
	return nil
}

// legacyGetter reads raw column values of a row the way the backfill sees them.
func legacyGetter(l *model.DailyHealthLog) func(string) taxonomy.LegacyValue {
	return func(column string) taxonomy.LegacyValue {
		c, ok := model.ColumnByName(column)
		if !ok {
			return nil
		}
		switch f := c.Field(&l.Row).(type) {
		case *sql.NullString:
			if f.Valid {
				return f.String
			}
		case *sql.NullBool:
			if f.Valid {
				return f.Bool
			}
		case *sql.NullInt16:
			if f.Valid {
				return int64(f.Int16)
			}
		case *rootdb.NullRawJSON:
			if f.Valid {
				return f.V
			}
		}
		return nil
	}
}

// syncFromLegacy re-projects the slots of the written columns from the saved legacy row.
func (s *Service) syncFromLegacy(ctx context.Context, l *model.DailyHealthLog, columns []string, now time.Time) error {
	if len(columns) == 0 {
		return nil
	}
	if err := s.deleteLegacySlots(ctx, l.Row.UserID, l.Row.LogDate, columns); err != nil {
		return err
	}
	entries := taxonomy.ProjectColumns(legacyGetter(l), columns)
	return s.insertEntries(ctx, l.Row.UserID, l.Row.LogDate, entries, SourceLegacy, now)
}

// LifeMode is the user's effective life-stage mode (enums.ResolveLifeMode).
func (s *Service) LifeMode(ctx context.Context, userID uint64) (string, error) {
	stored, err := s.q.GetLogLifeMode(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("healthlog: life mode: %w", err)
	}
	pregnant, err := s.q.LogPregnancyActive(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("healthlog: pregnancy mode: %w", err)
	}
	profile, err := s.profile(ctx, userID)
	if err != nil {
		return "", err
	}
	goal := ""
	if profile != nil {
		goal = profile.UserGoal
	}
	return string(enums.ResolveLifeMode(stored.String, pregnant, goal)), nil
}

// Day is the user's v2 entries of one day (insertion order).
func (s *Service) Day(ctx context.Context, userID uint64, date civildate.Date) ([]taxonomy.Entry, error) {
	rows, err := s.q.ListLogEntriesOn(ctx, store.ListLogEntriesOnParams{UserID: userID, LogDate: date})
	if err != nil {
		return nil, fmt.Errorf("healthlog: day entries: %w", err)
	}
	out := make([]taxonomy.Entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, toEntry(r))
	}
	return out, nil
}

// DayEntries is one day of a range.
type DayEntries struct {
	Date    civildate.Date
	Entries []taxonomy.Entry
}

// Range is the user's v2 days in [from, to] that have entries, oldest first.
func (s *Service) Range(ctx context.Context, userID uint64, from, to civildate.Date) ([]DayEntries, error) {
	rows, err := s.q.ListLogEntriesBetween(ctx, store.ListLogEntriesBetweenParams{UserID: userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("healthlog: range entries: %w", err)
	}
	out := []DayEntries{}
	for _, r := range rows {
		if n := len(out); n == 0 || out[n-1].Date != r.LogDate {
			out = append(out, DayEntries{Date: r.LogDate})
		}
		out[len(out)-1].Entries = append(out[len(out)-1].Entries, toEntry(r))
	}
	return out, nil
}

// SaveDay applies validated changes to a day (each change replaces one param) and writes the
// legacy-representable values of the changed params back into daily_health_logs. Returns the new day.
func (s *Service) SaveDay(ctx context.Context, userID uint64, date civildate.Date, changes []taxonomy.Change, locale string, now time.Time) ([]taxonomy.Entry, error) {
	now = dbNow(now)
	var day []taxonomy.Entry
	err := s.inTx(ctx, func(t *Service) error {
		existing, err := t.Day(ctx, userID, date)
		if err != nil {
			return err
		}
		changed := map[string]bool{}
		for _, ch := range changes {
			changed[ch.Key()] = true
			if err := t.q.DeleteLogEntryParam(ctx, store.DeleteLogEntryParamParams{
				UserID: userID, LogDate: date, Category: ch.Category, Param: ch.Param,
			}); err != nil {
				return fmt.Errorf("healthlog: replace entries: %w", err)
			}
			source := SourceManual
			if ch.Source != "" {
				source = ch.Source
			}
			if err := t.insertEntries(ctx, userID, date, ch.Entries, source, now); err != nil {
				return err
			}
		}
		for _, e := range existing {
			if !changed[e.ParamKey()] {
				day = append(day, e)
			}
		}
		for _, ch := range changes {
			day = append(day, ch.Entries...)
		}
		return t.writeBackLegacy(ctx, userID, date, changed, day, locale, now)
	})
	if err != nil {
		return nil, err
	}
	return day, nil
}

// writeBackLegacy updates daily_health_logs from the changed params (no row is created for a day that
// only has v2-only values).
func (s *Service) writeBackLegacy(ctx context.Context, userID uint64, date civildate.Date, changed map[string]bool, day []taxonomy.Entry, locale string, now time.Time) error {
	attrs := phpval.NewMap()
	attrs.Set("log_date", date.String())
	anyValue := false
	for _, lc := range taxonomy.LegacyColumns() {
		if !changed[lc.Category+"."+lc.Param] {
			continue
		}
		v := lc.Reverse(day)
		attrs.Set(lc.Column, v)
		anyValue = anyValue || v != nil
	}
	if attrs.Len() == 1 {
		return nil
	}
	if !anyValue {
		if _, err := s.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: date}); errors.Is(err, sql.ErrNoRows) {
			return nil
		} else if err != nil {
			return fmt.Errorf("healthlog: find log: %w", err)
		}
	}
	_, err := s.store(ctx, userID, attrs, locale, now)
	return err
}

// DeleteDay removes every v2 entry of the day and the legacy row.
func (s *Service) DeleteDay(ctx context.Context, userID uint64, date civildate.Date) error {
	return s.inTx(ctx, func(t *Service) error {
		if err := t.q.DeleteLogEntriesOn(ctx, store.DeleteLogEntriesOnParams{UserID: userID, LogDate: date}); err != nil {
			return fmt.Errorf("healthlog: delete day entries: %w", err)
		}
		row, err := t.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: date})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("healthlog: find log: %w", err)
		}
		if err := t.q.DeleteDailyHealthLog(ctx, row.ID); err != nil {
			return fmt.Errorf("healthlog: delete: %w", err)
		}
		return nil
	})
}

// ProjectLegacyRow is the v2 projection of one daily_health_logs row — exactly what the backfill
// (migration 00020) writes for it.
func ProjectLegacyRow(row store.DailyHealthLog) []taxonomy.Entry {
	return taxonomy.ProjectRow(legacyGetter(model.FromRow(row)))
}
