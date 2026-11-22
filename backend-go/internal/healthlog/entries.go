package healthlog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/messages/events"
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
	rows := make([]sourcedEntry, len(entries))
	for i, e := range entries {
		rows[i] = sourcedEntry{e, source}
	}
	return s.insertBatch(ctx, userID, date, rows, now)
}

// sourcedEntry is one health_log_entries row to insert.
type sourcedEntry struct {
	entry  taxonomy.Entry
	source string
}

// maxBatchRows is InsertLogEntriesBatch's ceiling (its row index reaches 999).
const maxBatchRows = 1000

// insertBatch inserts rows in order with one multi-row INSERT per maxBatchRows (T-M12-31, P-06): the ids follow
// the slice order, as the per-row inserts did.
func (s *Service) insertBatch(ctx context.Context, userID uint64, date civildate.Date, rows []sourcedEntry, now time.Time) error {
	ts := sql.NullTime{Time: now, Valid: true}
	for start := 0; start < len(rows); start += maxBatchRows {
		part := rows[start:min(start+maxBatchRows, len(rows))]
		batch := make([][7]any, len(part))
		for i, r := range part {
			e := r.entry
			batch[i] = [7]any{e.Category, e.Param, e.Item, nullable(e.Code), nullable(e.Num), nullable(e.Text), r.source}
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false) // the text goes to the DB as written (JSON_VALUE decodes \uXXXX too)
		if err := enc.Encode(batch); err != nil {
			return fmt.Errorf("healthlog: encode entries: %w", err)
		}
		if err := s.q.InsertLogEntriesBatch(ctx, store.InsertLogEntriesBatchParams{
			UserID: userID, LogDate: date, Batch: buf.String(), Now: ts,
		}); err != nil {
			return fmt.Errorf("healthlog: insert entries: %w", err)
		}
	}
	return nil
}

// nullable is a NULL-able column value in the batch JSON (null = SQL NULL).
func nullable(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

// deleteParams removes every slot of the given "category.param" keys of a day in one statement (no keys: no query).
func (s *Service) deleteParams(ctx context.Context, userID uint64, date civildate.Date, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, k := range keys {
		if strings.ContainsRune(k, ',') { // FIND_IN_SET list separator; taxonomy codes never contain one
			return fmt.Errorf("healthlog: delete entries: invalid param key %q", k)
		}
	}
	if err := s.q.DeleteLogEntryParams(ctx, store.DeleteLogEntryParamsParams{
		UserID: userID, LogDate: date, ParamKeys: strings.Join(keys, ","),
	}); err != nil {
		return fmt.Errorf("healthlog: delete entries: %w", err)
	}
	return nil
}

// deleteLegacySlots removes the v2 slots the given legacy columns write (nil = every mapped slot).
func (s *Service) deleteLegacySlots(ctx context.Context, userID uint64, date civildate.Date, columns []string) error {
	params, items := taxonomy.SlotsOf(columns)
	if err := s.deleteParams(ctx, userID, date, params); err != nil {
		return err
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
	return s.SaveDayWith(ctx, userID, date, func([]taxonomy.Entry) ([]taxonomy.Change, error) { return changes, nil }, locale, now)
}

// SaveDayWith is SaveDay with the changes built from the stored day as read under the user lock (T-M12-31, P-06):
// PUT /logs/days/{date} validates against that read instead of one of its own, so a save reads the day once and
// never builds its changes on a day another save changed in between. An error from build (a validation failure)
// rolls back with nothing written and no revision bump.
//
// The write is one DELETE of every changed param and one multi-row INSERT of the new rows (it was a DELETE per
// param and an INSERT per row), then the legacy write-back.
func (s *Service) SaveDayWith(ctx context.Context, userID uint64, date civildate.Date, build func(existing []taxonomy.Entry) ([]taxonomy.Change, error), locale string, now time.Time) ([]taxonomy.Entry, error) {
	now = dbNow(now)
	var day []taxonomy.Entry
	err := s.inTxLocked(ctx, userID, func(t *Service) error {
		existing, err := t.Day(ctx, userID, date)
		if err != nil {
			return err
		}
		changes, err := build(existing)
		if err != nil {
			return err
		}
		day, err = t.applyChanges(ctx, userID, date, existing, changes, locale, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	events.Goal(ctx, userID, "log_saved") // T-M10-06, M-17: server-matched message goal (after commit)
	return day, nil
}

// applyChanges replaces the changed params of the day (existing = its rows, in id order) and returns the new day:
// the unchanged rows, then the new ones in change order. Of two changes to one param the last wins, at its own
// place — what the per-param delete + insert left behind.
func (s *Service) applyChanges(ctx context.Context, userID uint64, date civildate.Date, existing []taxonomy.Entry, changes []taxonomy.Change, locale string, now time.Time) ([]taxonomy.Entry, error) {
	last := map[string]int{}
	for i, ch := range changes {
		last[ch.Key()] = i
	}
	changed := map[string]bool{}
	keys := make([]string, 0, len(last))
	var rows []sourcedEntry
	for i, ch := range changes {
		if last[ch.Key()] != i {
			continue
		}
		changed[ch.Key()] = true
		keys = append(keys, ch.Key())
		source := SourceManual
		if ch.Source != "" {
			source = ch.Source
		}
		for _, e := range ch.Entries {
			rows = append(rows, sourcedEntry{e, source})
		}
	}
	if err := s.deleteParams(ctx, userID, date, keys); err != nil {
		return nil, err
	}
	if err := s.insertBatch(ctx, userID, date, rows, now); err != nil {
		return nil, err
	}
	var day []taxonomy.Entry
	for _, e := range existing {
		if !changed[e.ParamKey()] {
			day = append(day, e)
		}
	}
	for _, r := range rows {
		day = append(day, r.entry)
	}
	return day, s.writeBackLegacy(ctx, userID, date, changed, day, locale, now)
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

// DeleteDay removes every v2 entry of the day and the legacy row; a deleted flow day rebuilds its bleeding_log
// period (T-M9-28).
func (s *Service) DeleteDay(ctx context.Context, userID uint64, date civildate.Date, now time.Time) error {
	return s.inTxLocked(ctx, userID, func(t *Service) error {
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
		_, err = t.bridge(ctx, userID, bleedingChange{date: date, before: row.BleedingIntensity.String}, "", dbNow(now))
		return err
	})
}

// ProjectLegacyRow is the v2 projection of one daily_health_logs row — exactly what the backfill
// (migration 00020) writes for it.
func ProjectLegacyRow(row store.DailyHealthLog) []taxonomy.Entry {
	return taxonomy.ProjectRow(legacyGetter(model.FromRow(row)))
}
