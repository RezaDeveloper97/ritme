package companion

import (
	"context"
	"log/slog"

	"github.com/ritme/backend-go/internal/platform/clock"
)

// Delegation is the companion side of «ثبت برای …» for other domains (care implements its Delegation interface
// with it). Authorize checks the live grant and, for someone else's data, writes the audit row first (fail closed:
// no audit row → no access); Notify tells the owner about a write after it succeeded (best effort, logged).
type Delegation struct {
	conn   Conn
	base   clock.Clock
	logger *slog.Logger
}

// NewDelegation returns the delegation on conn; base is the fallback clock (the request clock in ctx wins).
func NewDelegation(conn Conn, base clock.Clock, logger *slog.Logger) *Delegation {
	if logger == nil {
		logger = slog.Default()
	}
	return &Delegation{conn: conn, base: base, logger: logger}
}

func (d *Delegation) svc(ctx context.Context) *Service {
	return NewService(d.conn, clock.FromContext(ctx, d.base), Options{})
}

// Authorize reports whether actorID may read (write=false) or write the section of ownerID's data through an active
// link, and records the access in the audit trail before the caller touches the data. A failed audit insert is an
// error (the caller must not proceed). The owner herself is always allowed and never audited. A write that later
// fails validation still leaves its (attempt) audit row.
func (d *Delegation) Authorize(ctx context.Context, ownerID, actorID uint64, section Section, write bool) (bool, error) {
	if ownerID == actorID {
		return true, nil
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
