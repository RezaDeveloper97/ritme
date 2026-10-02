package fertility

import (
	"context"
	"fmt"

	"github.com/ritme/backend-go/internal/fertility/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Read side for the TTC analysis (B-N3-11, internal/analysis ttc.go). fertility_logs (LH test,
// cervical mucus) is not synced into the taxonomy v2 rows (QUESTIONS #80), so the analysis reads
// these rows here and merges them with health_log_entries itself.

// Signal is one fertility_logs day: its LH test and cervical mucus ("" when not logged).
type Signal struct {
	Date  civildate.Date
	LH    string
	Mucus string
}

// Signals are the user's fertility_logs LH tests and mucus from `from` to `to` (inclusive), oldest first.
func (s *Service) Signals(ctx context.Context, userID uint64, from, to civildate.Date) ([]Signal, error) {
	rows, err := store.New(s.db).ListFertilitySignals(ctx, store.ListFertilitySignalsParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("fertility: load signals: %w", err)
	}
	out := make([]Signal, 0, len(rows))
	for _, r := range rows {
		out = append(out, Signal{Date: r.LogDate, LH: r.LhTest.String, Mucus: r.CervicalMucus.String})
	}
	return out, nil
}

// FirstSignal is the user's first TTC-specific log day: the earliest fertility_logs row or taxonomy v2
// BBT / LH / pregnancy-test entry (legacy BBT is synced into those). The zero Date when there is none.
func (s *Service) FirstSignal(ctx context.Context, userID uint64) (civildate.Date, error) {
	q := store.New(s.db)
	logs, err := q.FirstFertilityLogDate(ctx, userID)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("fertility: first log: %w", err)
	}
	entries, err := q.FirstFertilityEntryDate(ctx, userID)
	if err != nil {
		return civildate.Date{}, fmt.Errorf("fertility: first entry: %w", err)
	}
	var first civildate.Date
	for _, d := range append(logs, entries...) {
		if first.IsZero() || d.Before(first) {
			first = d
		}
	}
	return first, nil
}
