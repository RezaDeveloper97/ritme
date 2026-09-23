// Package checkups is the periodic-checkups domain (docs/checkups/README.md, M4). T-M4-01 ships the
// schema, the default catalog seed and the status engine (package engine); the user API
// (T-M4-02) and the admin API (T-M4-03) build on the helpers here.
package checkups

import (
	"context"
	"fmt"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/checkups/store"
)

// EngineInputs loads the catalog (shared rows + the user's custom checkups, active only), the
// user's records and settings, and converts them to engine inputs. Birthday, pregnancy and the
// cycle prediction are the caller's (profile / pregnancy / cycle services). Three queries.
func EngineInputs(ctx context.Context, q store.Querier, userID uint64) (engine.Input, error) {
	types, err := q.ListActiveCheckupTypesForUser(ctx, int64(userID)) //nolint:gosec // ids fit int64
	if err != nil {
		return engine.Input{}, fmt.Errorf("checkups: list types: %w", err)
	}
	records, err := q.ListCheckupRecordsForUser(ctx, userID)
	if err != nil {
		return engine.Input{}, fmt.Errorf("checkups: list records: %w", err)
	}
	settings, err := q.ListCheckupSettingsForUser(ctx, userID)
	if err != nil {
		return engine.Input{}, fmt.Errorf("checkups: list settings: %w", err)
	}

	in := engine.Input{
		Types:    make([]engine.Type, 0, len(types)),
		Records:  make([]engine.Record, 0, len(records)),
		Settings: make([]engine.Setting, 0, len(settings)),
	}
	for _, t := range types {
		in.Types = append(in.Types, TypeFromRow(t))
	}
	for _, r := range records {
		in.Records = append(in.Records, RecordFromRow(r))
	}
	for _, s := range settings {
		in.Settings = append(in.Settings, engine.Setting{TypeID: s.CheckupTypeID, Enabled: s.Enabled, Remind: s.Remind})
	}
	return in, nil
}

// TypeFromRow converts a checkup_types row (NULL ints become 0 = "not set").
func TypeFromRow(t store.CheckupType) engine.Type {
	return engine.Type{
		ID:                t.ID,
		Key:               t.Key.String,
		UserID:            uint64(max(t.UserID.Int64, 0)),
		Category:          engine.Category(t.Category),
		IntervalMonths:    int(t.IntervalMonths),
		IntervalMonthsMax: int(t.IntervalMonthsMax.Int16),
		AgeMin:            int(t.AgeMin.Int16),
		AgeMax:            int(t.AgeMax.Int16),
		CycleDayFrom:      int(t.CycleDayFrom.Int16),
		CycleDayTo:        int(t.CycleDayTo.Int16),
		RemindLeadDays:    int(t.RemindLeadDays),
		HideInPregnancy:   t.HideInPregnancy,
		IsActive:          t.IsActive,
		SortOrder:         int(t.SortOrder),
	}
}

// RecordFromRow converts a checkup_records row.
func RecordFromRow(r store.CheckupRecord) engine.Record {
	rec := engine.Record{ID: r.ID, TypeID: r.CheckupTypeID, DoneOn: r.DoneOn}
	if r.NextDueOn.Valid {
		rec.NextDueOn = r.NextDueOn.Date
	}
	return rec
}
