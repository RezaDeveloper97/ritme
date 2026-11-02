package pregnancy

import (
	"context"
	"time"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// EndForLoss stops the user's pregnancy after a pregnancy loss (CB-LOSS-01): pregnancy mode goes off exactly as
// POST /pregnancy/deactivate does (pregnancy_mode 0, cycle_mode 1 — every reader of pregnancy content keys on that
// flag: home, Today, weeks, calendar, day log, the message engine, the alert engine, the companion pregnancy view),
// and every open pregnancy alert is closed — the preg.* rows of the alerts inbox are archived (T-M10-08b; the legacy
// pregnancy_alerts rows live there since goose 00114, T-M10-08c) — so no badge, urgent modal or alert nudge outlives the
// pregnancy. The profile row and
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
	if err := archivePregnancyAlerts(ctx, q, userID, now); err != nil {
		return false, err
	}
	return active, nil
}
