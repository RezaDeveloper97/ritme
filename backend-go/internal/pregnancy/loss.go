package pregnancy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// EndForLoss stops the user's pregnancy after a pregnancy loss (CB-LOSS-01): pregnancy mode goes off exactly as
// POST /pregnancy/deactivate does (pregnancy_mode 0, cycle_mode 1 — every reader of pregnancy content keys on that
// flag: home, Today, weeks, calendar, day log, the message engine, the alert engine, the companion pregnancy view),
// and every open pregnancy alert is closed, so no badge or alert nudge outlives the pregnancy. The profile row and
// its history stay. q may be a transaction. Reports whether a pregnancy was active.
func EndForLoss(ctx context.Context, q store.Querier, userID uint64, now time.Time) (bool, error) {
	p, err := LoadProfile(ctx, q, userID)
	if err != nil {
		return false, err
	}
	active := p != nil && p.PregnancyMode
	if active {
		if err := saveProfile(ctx, q, userID, p, jsonx.Obj("pregnancy_mode", false, "cycle_mode", true), now); err != nil {
			return false, err
		}
	}
	if _, err := q.DismissOpenAlerts(ctx, store.DismissOpenAlertsParams{UserID: userID, Now: sql.NullTime{Time: dbNow(now), Valid: true}}); err != nil {
		return false, fmt.Errorf("pregnancy: dismiss alerts: %w", err)
	}
	return active, nil
}
