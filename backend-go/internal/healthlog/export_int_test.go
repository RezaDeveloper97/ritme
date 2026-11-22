package healthlog

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// SaveDayReference is SaveDay as it was before T-M12-31 (a DELETE per changed param and an INSERT per row, in
// change order), kept test-only as the reference of TestSaveDay_MatchesReferencePath: the batched path must leave
// exactly the rows, legacy row and periods this one does.
func (s *Service) SaveDayReference(ctx context.Context, userID uint64, date civildate.Date, changes []taxonomy.Change, locale string, now time.Time) ([]taxonomy.Entry, error) {
	now = dbNow(now)
	var day []taxonomy.Entry
	err := s.inTxLocked(ctx, userID, func(t *Service) error {
		existing, err := t.Day(ctx, userID, date)
		if err != nil {
			return err
		}
		ts := sql.NullTime{Time: now, Valid: true}
		changed := map[string]bool{}
		for _, ch := range changes {
			changed[ch.Key()] = true
			if err := t.q.DeleteLogEntryParam(ctx, store.DeleteLogEntryParamParams{
				UserID: userID, LogDate: date, Category: ch.Category, Param: ch.Param,
			}); err != nil {
				return fmt.Errorf("reference: replace entries: %w", err)
			}
			source := SourceManual
			if ch.Source != "" {
				source = ch.Source
			}
			for _, e := range ch.Entries {
				if err := t.q.InsertLogEntry(ctx, store.InsertLogEntryParams{
					UserID: userID, LogDate: date, Category: e.Category, Param: e.Param, Item: e.Item,
					ValueCode: e.Code, ValueNum: e.Num, ValueText: e.Text, Source: source, CreatedAt: ts, UpdatedAt: ts,
				}); err != nil {
					return fmt.Errorf("reference: insert entry: %w", err)
				}
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
