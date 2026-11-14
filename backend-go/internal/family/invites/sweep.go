package invites

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// JobSweep is the daily invite / time-box sweep (docs/engine-v6/jobs.md).
const JobSweep = "family.invites_sweep"

// SweepCron runs the sweep at 04:05 Tehran (after the nightly engine jobs, before the morning).
const SweepCron = "5 4 * * *"

// NotificationType is user_notifications.type of the owner notices written here (no health payload).
const NotificationType = "family"

// pathFamily is the web path the owner notices open.
const pathFamily = "/family"

// Owner notices.
const (
	NoticeInviteExpired = "invite_expired"
	NoticeCarerExpired  = "carer_expired"
	NoticeAccepted      = "accepted"
	NoticeAwaitingOwner = "awaiting_owner"
)

// Languages lists the active language codes (the `languages` table; the list is data).
type Languages func(ctx context.Context) []string

// SweepReport counts one run.
type SweepReport struct {
	Invites  int // invited links whose invites all expired → revoked (expired)
	Links    int // active links past expires_at → revoked (expired)
	Awaiting int // awaiting_owner links not confirmed within ConfirmTTL or past expires_at → revoked (expired)
}

// expiry is which kind of link a sweep candidate is.
type expiry uint8

const (
	expireInvited expiry = iota
	expireActive
	expireAwaiting
)

// Sweep expires (F-12, F-13): invited links without an open, unexpired invite, active links past expires_at, and
// awaiting_owner links the owner did not confirm within ConfirmTTL (or past expires_at).
// Each becomes revoked with revoked_by = expired, with an activity row (invite_expired | link_expire) and an owner
// inbox notice, in one transaction per link under the owner lock (re-checked there, so an owner who extended the link
// or renewed the invite meanwhile wins). Idempotent: a second run finds nothing.
func (s *Service) Sweep(ctx context.Context, now time.Time, langs Languages) (SweepReport, error) {
	now = now.Truncate(time.Second)
	q := store.New(s.conn)
	var rep SweepReport
	seen := map[uint64]bool{}
	for {
		rows, err := q.ListExpiredInvitedLinks(ctx, store.ListExpiredInvitedLinksParams{Now: nt(now), Limit: sweepPage})
		if err != nil {
			return rep, fmt.Errorf("family invites: sweep: %w", err)
		}
		progressed := false
		for _, r := range rows {
			if seen[r.ID] {
				continue
			}
			seen[r.ID], progressed = true, true
			done, err := s.expire(ctx, r.ID, r.OwnerID, r.DisplayName.String, expireInvited, now, langs)
			if err != nil {
				return rep, err
			}
			if done {
				rep.Invites++
			}
		}
		if !progressed || len(rows) < sweepPage {
			break
		}
	}
	for {
		rows, err := q.ListExpiredActiveLinks(ctx, store.ListExpiredActiveLinksParams{Now: nt(now), Limit: sweepPage})
		if err != nil {
			return rep, fmt.Errorf("family invites: sweep: %w", err)
		}
		progressed := false
		for _, r := range rows {
			if seen[r.ID] {
				continue
			}
			seen[r.ID], progressed = true, true
			done, err := s.expire(ctx, r.ID, r.OwnerID, r.DisplayName.String, expireActive, now, langs)
			if err != nil {
				return rep, err
			}
			if done {
				rep.Links++
			}
		}
		if !progressed || len(rows) < sweepPage {
			break
		}
	}
	for {
		rows, err := q.ListStaleAwaitingLinks(ctx, store.ListStaleAwaitingLinksParams{StaleBefore: nt(now.Add(-s.opt.ConfirmTTL)),
			Now: nt(now), Limit: sweepPage})
		if err != nil {
			return rep, fmt.Errorf("family invites: sweep: %w", err)
		}
		progressed := false
		for _, r := range rows {
			if seen[r.ID] {
				continue
			}
			seen[r.ID], progressed = true, true
			done, err := s.expire(ctx, r.ID, r.OwnerID, r.DisplayName.String, expireAwaiting, now, langs)
			if err != nil {
				return rep, err
			}
			if done {
				rep.Awaiting++
			}
		}
		if !progressed || len(rows) < sweepPage {
			break
		}
	}
	return rep, nil
}

func (s *Service) expire(ctx context.Context, linkID, ownerID uint64, label string, kind expiry, now time.Time, langs Languages) (bool, error) {
	done := false
	err := s.inTx(ctx, func(_ *sql.Tx, q *store.Queries) error {
		if err := lock(ctx, q, ownerID); err != nil {
			return err
		}
		c, err := q.GetCompanion(ctx, linkID)
		if err != nil {
			return notFound(err)
		}
		action, notice := ActionInviteExpired, NoticeInviteExpired
		switch kind {
		case expireAwaiting:
			pastEnd := c.ExpiresAt.Valid && !now.Before(c.ExpiresAt.Time)
			stale := c.AcceptedAt.Valid && !c.AcceptedAt.Time.After(now.Add(-s.opt.ConfirmTTL))
			if c.Status != StatusAwaitingOwner || (!pastEnd && !stale) {
				return nil // confirmed or ended meanwhile
			}
		case expireActive:
			if c.Status != StatusActive || !c.ExpiresAt.Valid || now.Before(c.ExpiresAt.Time) {
				return nil // extended or ended meanwhile
			}
			action, notice = ActionLinkExpire, NoticeCarerExpired
		default:
			if c.Status != StatusInvited {
				return nil
			}
			invites, err := q.ListLinkInvites(ctx, store.ListLinkInvitesParams{OwnerID: ownerID, CompanionID: linkID})
			if err != nil {
				return fmt.Errorf("family invites: sweep invites: %w", err)
			}
			for _, inv := range invites {
				if !inv.UsedAt.Valid && !inv.RevokedAt.Valid && inv.ExpiresAt.Valid && now.Before(inv.ExpiresAt.Time) {
					return nil // renewed meanwhile
				}
			}
		}
		if n, err := q.RevokeCompanion(ctx, store.RevokeCompanionParams{ID: linkID, RevokedBy: nstr(RevokedByExpired), Now: nt(now)}); err != nil {
			return fmt.Errorf("family invites: expire: %w", err)
		} else if n == 0 {
			return nil
		}
		if err := cleanupLink(ctx, q, linkID, now); err != nil {
			return err
		}
		if err := activity(ctx, q, ownerID, 0, linkID, action, nil, now); err != nil {
			return err
		}
		done = true
		return notifyOwner(ctx, q, ownerID, linkID, label, notice, langs(ctx), now)
	})
	return done, err
}

// notifyOwner writes an owner inbox row about one of her links. The member is named only by the owner's own label
// (never the member's account name: no CMP-L5 spoofing, no detail), else a neutral word; title/body carry one entry
// per active language.
func notifyOwner(ctx context.Context, q *store.Queries, ownerID, linkID uint64, label, notice string, languages []string, now time.Time) error {
	title, body := map[string]string{}, map[string]string{}
	for _, code := range languages {
		name := label
		if name == "" {
			name = T("notices.someone", code)
		}
		title[code] = T("notices."+notice+"_title", code, "name", name)
		body[code] = T("notices."+notice+"_body", code)
	}
	rawTitle, err := json.Marshal(title)
	if err != nil {
		return fmt.Errorf("family invites: notice: %w", err)
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("family invites: notice: %w", err)
	}
	data, err := json.Marshal(map[string]any{"link_id": linkID, "event": notice})
	if err != nil {
		return fmt.Errorf("family invites: notice: %w", err)
	}
	if err := q.InsertUserNotification(ctx, store.InsertUserNotificationParams{
		UserID: ownerID, Type: NotificationType, Title: rawTitle, Body: rootdb.NullRawJSON{V: rawBody, Valid: true},
		ActionUrl: nstr(pathFamily), Data: rootdb.NullRawJSON{V: data, Valid: true}, Now: nt(now),
	}); err != nil {
		return fmt.Errorf("family invites: notice: %w", err)
	}
	return nil
}

// NotifyOwner is the owner notice of an answered invite (accepted | awaiting_owner), best effort for the caller.
func (s *Service) NotifyOwner(ctx context.Context, a Accepted, languages []string) error {
	notice := NoticeAccepted
	if a.Status == StatusAwaitingOwner {
		notice = NoticeAwaitingOwner
	}
	return notifyOwner(ctx, store.New(s.conn), a.OwnerID, a.LinkID, a.DisplayName, notice, languages, s.now())
}

// RegisterJobs registers family.invites_sweep (daily, low queue, unique for 23 h; idempotent on the link id).
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time, langs Languages) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobSweep, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{10 * time.Minute, 30 * time.Minute},
		Timeout: 10 * time.Minute,
	}, func(ctx context.Context, _ []byte) error {
		_, err := s.Sweep(ctx, now(), langs)
		return err
	}, SweepCron, queue.Unique(23*time.Hour))
}
