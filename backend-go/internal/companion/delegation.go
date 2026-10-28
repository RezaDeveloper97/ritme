package companion

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Delegation is the companion side of «ثبت برای …» for other domains (care implements its Delegation interface
// with it). Authorize checks the live grant and, for someone else's data, writes the audit row first (fail closed:
// no audit row → no access); Notify tells the owner about a write after it succeeded (best effort, logged).
type Delegation struct {
	conn      Conn
	base      clock.Clock
	logger    *slog.Logger
	readAudit *family.ReadAudit
}

// NewDelegation returns the delegation on conn; base is the fallback clock (the request clock in ctx wins).
func NewDelegation(conn Conn, base clock.Clock, logger *slog.Logger) *Delegation {
	if logger == nil {
		logger = slog.Default()
	}
	return &Delegation{conn: conn, base: base, logger: logger}
}

// WithReadAudit coalesces read-audit rows in Redis (F-16, D-203); without it the DB window check is used.
func (d *Delegation) WithReadAudit(r *family.ReadAudit) *Delegation {
	d.readAudit = r
	return d
}

func (d *Delegation) svc(ctx context.Context) *Service {
	return NewService(d.conn, clock.FromContext(ctx, d.base), Options{ReadAudit: d.readAudit})
}

// Authorize reports whether actorID may read (write=false) or write the section of ownerID's data through an active
// link — family.Decide over the request's pair snapshot (T-M11-05) — and records the access in the audit trail before the caller touches the data. A failed audit insert is an
// error (the caller must not proceed). The owner herself is always allowed and never audited. A write that later
// fails validation still leaves its (attempt) audit row.
func (d *Delegation) Authorize(ctx context.Context, ownerID, actorID uint64, section Section, write bool) (bool, error) {
	if ownerID == actorID {
		return true, nil
	}
	if c, ok := ctx.(fiber.Ctx); ok { // a member reads or writes the owner's data: never cache the answer (T-M9-02)
		NoStore(c)
	}
	svc := d.svc(ctx)
	level, linkID, err := svc.access(ctx, ownerID, actorID, section)
	if err != nil {
		return false, err
	}
	allowed := level.CanRead()
	action := ActionRead
	if write {
		allowed, action = level.CanWrite(), ActionWrite
	}
	if !allowed {
		return false, nil
	}
	if err := svc.Audit(ctx, ownerID, actorID, linkID, section, action); err != nil {
		return false, err
	}
	return true, nil
}

// RecheckTx re-decides a member write inside the caller's write transaction (CMP-L4, T-M11-05 Scope 5): the link and
// grant rows are re-read with shared locks (family.RecheckTx), so a revoke, expiry or grant change committed after
// Authorize wins and cannot commit before this transaction ends. It reports whether the write may still go ahead (the
// same rule as Authorize: a level that writes). The owner herself is always allowed.
func (d *Delegation) RecheckTx(ctx context.Context, tx *sql.Tx, ownerID, actorID uint64, section Section) (bool, error) {
	if ownerID == actorID {
		return true, nil
	}
	snap, err := family.RecheckTx(ctx, tx, ownerID, actorID)
	if err != nil {
		return false, err
	}
	now := clock.FromContext(ctx, d.base).Now()
	return effectiveLevel(snap, section, now).CanWrite(), nil
}

// Notify writes the owner-inbox notice of a successful companion write (created: a new record, else an edit).
// Best effort: the write is already committed and audited, so a failure is logged (no payload), not returned.
func (d *Delegation) Notify(ctx context.Context, ownerID, actorID uint64, section Section, created bool, languages []string) {
	if ownerID == actorID {
		return
	}
	notice, ok := writeNotice(section, created)
	if !ok {
		return
	}
	svc := d.svc(ctx)
	_, linkID, err := svc.access(ctx, ownerID, actorID, section)
	if err == nil && linkID != 0 {
		var link Link
		if link, err = svc.OwnerLink(ctx, ownerID, linkID); err == nil {
			err = svc.NotifyOwner(ctx, link, notice, languages)
		}
	}
	if err != nil {
		d.logger.ErrorContext(ctx, "companion write notice failed", slog.Uint64("owner_id", ownerID),
			slog.String("section", string(section)), slog.String("error", err.Error()))
	}
}

func writeNotice(section Section, created bool) (Notice, bool) {
	switch {
	case section == SectionMeds && created:
		return NoticeRecordedMeds, true
	case section == SectionMeds:
		return NoticeUpdatedMeds, true
	case section == SectionAppointments && created:
		return NoticeRecordedAppointment, true
	case section == SectionAppointments:
		return NoticeUpdatedAppointment, true
	}
	return "", false
}
