package projections

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/family"
)

// auditSection is the companion_audit_logs section a domain read is recorded under: the legacy section name where
// one exists (the owner's /companions/audit trail and the companion reads share the row and its window), the v6 code
// otherwise.
func auditSection(d family.Domain) string {
	if d == family.DomainCyclePhase {
		return "cycle"
	}
	return d.String()
}

// Auditor records member reads in the owner's audit trail before any data is built (fail closed: no row → no data),
// coalesced to one row per (owner, member, domain) per family.ReadAuditWindow (F-16, D-203): Redis SET NX first,
// the DB window always (paths without Redis share it).
type Auditor struct {
	db    store.DBTX
	redis *family.ReadAudit
}

// NewAuditor returns the auditor on db (redis may be nil).
func NewAuditor(db store.DBTX, redis *family.ReadAudit) *Auditor {
	return &Auditor{db: db, redis: redis}
}

// Read records the read of d by s.ActorID through s.LinkID. The owner herself is never audited.
func (a *Auditor) Read(ctx context.Context, s family.Snapshot, d family.Domain, now time.Time) error {
	if s.OwnerID == s.ActorID {
		return nil
	}
	key := d.String()
	claim := a.redis.Claim(ctx, s.OwnerID, s.ActorID, key, now)
	if claim == family.ClaimSkip {
		return nil
	}
	q := store.New(a.db)
	sec := sql.NullString{String: auditSection(d), Valid: true}
	actor := sql.NullInt64{Int64: int64(s.ActorID), Valid: true}        //nolint:gosec // ids fit
	link := sql.NullInt64{Int64: int64(s.LinkID), Valid: s.LinkID != 0} //nolint:gosec // ids fit
	_, err := q.RecentReadAudit(ctx, store.RecentReadAuditParams{OwnerID: s.OwnerID, ActorID: actor, CompanionID: link,
		Section: sec, Since: sql.NullTime{Time: now.Add(-family.ReadAuditWindow), Valid: true}})
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("projections: recent audit: %w", err)
	}
	if err := q.InsertAudit(ctx, store.InsertAuditParams{OwnerID: s.OwnerID, ActorID: actor, CompanionID: link,
		Section: sec, Action: "read", Now: sql.NullTime{Time: now, Valid: true}}); err != nil {
		if claim == family.ClaimRecord {
			a.redis.Release(ctx, s.OwnerID, s.ActorID, key)
		}
		return fmt.Errorf("projections: audit: %w", err)
	}
	return nil
}
