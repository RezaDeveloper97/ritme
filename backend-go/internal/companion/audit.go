package companion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/companion/store"
)

// ReadAuditWindow coalesces companion read audits (B-N4-08b, CMP-L1): a read of the same section of the same owner by
// the same companion through the same link within this window is not written again, so reloading the companion home
// cannot push the owner's write and lifecycle rows out of sight. Writes and lifecycle events are always written.
const ReadAuditWindow = 15 * time.Minute

// readAuditedRecently reports whether the read is already in the trail within ReadAuditWindow.
func readAuditedRecently(ctx context.Context, q *store.Queries, ownerID, actorID, companionID uint64, section Section, now time.Time) (bool, error) {
	if companionID == 0 {
		return false, nil // no link to coalesce on: always record
	}
	_, err := q.RecentReadAudit(ctx, store.RecentReadAuditParams{
		OwnerID: ownerID, ActorID: nid(actorID), CompanionID: nid(companionID), Section: nstr(string(section)),
		Since: nt(now.Add(-ReadAuditWindow)),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("companion: recent audit: %w", err)
	}
	return true, nil
}

// AuditPage is one page of the owner's audit trail, newest first: entries older than beforeID (0 = from the newest),
// only of action when it is set, at most limit (1–MaxAuditEntries, default 50).
func (s *Service) AuditPage(ctx context.Context, ownerID, beforeID uint64, action Action, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > MaxAuditEntries {
		limit = 50
	}
	rows, err := store.New(s.conn).ListOwnerAuditPage(ctx, store.ListOwnerAuditPageParams{
		OwnerID: ownerID, BeforeID: beforeID, Action: string(action), Limit: int32(limit), //nolint:gosec // ≤ MaxAuditEntries
	})
	if err != nil {
		return nil, fmt.Errorf("companion: audit: %w", err)
	}
	out := make([]AuditEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditEntry{
			ID: r.ID, ActorID: uint64(r.ActorID.Int64), CompanionID: uint64(r.CompanionID.Int64), //nolint:gosec // positive ids
			Section: Section(r.Section.String), Action: Action(r.Action), At: r.CreatedAt.Time,
		})
	}
	return out, nil
}

// AuditActions are the audit actions GET /companions/audit filters on.
var AuditActions = []Action{ActionRead, ActionWrite, ActionInvited, ActionAccepted, ActionRevoked, ActionGrantsChanged}

func auditActionStrings() []string {
	out := make([]string, len(AuditActions))
	for i, a := range AuditActions {
		out[i] = string(a)
	}
	return out
}
