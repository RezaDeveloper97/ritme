// Package cyclesnap turns the benchfixtures data into cycle-service Snapshots, the per-request engine input of
// /cycle/today, /cycle/date and /cycle/month, without a database (T-M9-08 benchmarks).
//
// It is a separate package from benchfixtures because it imports internal/cycle/service; benchfixtures itself
// stays a leaf.
package cyclesnap

import (
	"context"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// UserID is the fixture user of every snapshot.
const UserID = 1

// Inputs are the loaded rows of one request (what Service.Load reads with its three queries), converted once.
type Inputs struct {
	Profile     *store.UserProfile
	HistoryRows []store.CycleHistory
	LogRows     []store.DailyHealthLog
	From, To    civildate.Date
}

// Load is the fixture equivalent of Service.Load for a history of h periods and the daily logs of from..to.
func Load(h int, from, to civildate.Date) Inputs {
	in := Inputs{Profile: benchfixtures.ProfileRow(UserID), HistoryRows: benchfixtures.HistoryRows(UserID, h), From: from, To: to}
	span := from.DiffDays(benchfixtures.Today) + 1
	if span < 1 {
		span = 1
	}
	for _, r := range benchfixtures.DailyLogRows(UserID, benchfixtures.Days(max(span, benchfixtures.LogDays))) {
		if !r.LogDate.Before(from) && !r.LogDate.After(to) {
			in.LogRows = append(in.LogRows, r)
		}
	}
	return in
}

// Snapshot builds a fresh Snapshot (its legacy engine not built yet), as one request would: the row → model
// conversions run here, so a benchmark that calls Snapshot per iteration pays for them like Load does.
func (in Inputs) Snapshot() *service.Snapshot {
	sn := &service.Snapshot{
		UserID: UserID, Profile: in.Profile, HistoryRows: in.HistoryRows,
		Histories: service.HistoriesFromRows(in.HistoryRows),
		From:      in.From, To: in.To, LogRows: in.LogRows,
		Logs:  make(map[civildate.Date]*legacy.DailyLog, len(in.LogRows)),
		Tips:  recommendation.New(NoTips{}),
		Today: benchfixtures.Today,
	}
	for _, r := range in.LogRows {
		sn.Logs[r.LogDate] = service.DailyLogFromRow(r)
	}
	return sn
}

// NoTips is an empty recommendations table: the engines fall back to their built-in tips, as on a fresh install.
type NoTips struct{}

// ActiveRows implements recommendation.Source.
func (NoTips) ActiveRows(context.Context) ([]recommendation.Row, error) { return nil, nil }

// AnyExists implements recommendation.Source.
func (NoTips) AnyExists(context.Context) (bool, error) { return false, nil }
