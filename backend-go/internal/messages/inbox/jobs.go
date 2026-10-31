package inbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/platform/usertz"
)

// Job types and schedules (docs/engine-v6/jobs.md).
const (
	JobSweep    = "messages.inbox_sweep"
	JobPush     = "messages.push_slot"
	SweepCron   = "10 4 * * *"
	PushCron    = "0 * * * *"
	ArchiveDays = 30 // resolved / acknowledged / noted rows are archived 30 days after their last hit
	ActiveDays  = 30 // A-06
	// PushHour is the user-local hour the push slot runs in (09:00–09:59; quiet hours are 23–08, M-09).
	PushHour     = 9
	archiveBatch = 5000
	userPage     = 500
	// NotificationType is user_notifications.type of a push slot row.
	NotificationType = "message_push"
	// PushSlot is the slot name.
	PushSlot = "push"
)

// Sweep is messages.inbox_sweep: it re-evaluates every user with an open row (rows whose condition dropped are
// resolved), then archives the resolved, acknowledged and noted rows whose last hit is ArchiveDays or more Tehran days
// before today. It returns the users evaluated and the rows archived. A user whose evaluation fails is skipped
// (counted, logged without data) and the next run tries again.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int, int64, error) {
	users := 0
	var after uint64
	for {
		ids, err := s.q.ListOpenInboxUsers(ctx, store.ListOpenInboxUsersParams{AfterID: after, Limit: userPage})
		if err != nil {
			return users, 0, fmt.Errorf("inbox sweep: users: %w", err)
		}
		for _, id := range ids {
			if _, err := s.EvaluateUser(ctx, id, now); err != nil {
				if ctx.Err() != nil {
					return users, 0, ctx.Err()
				}
				jobUserErrors.WithLabelValues(JobSweep).Inc()
				s.logger.Warn("inbox sweep: user skipped", "user_id", id, "error", err)
				continue
			}
			users++
		}
		if len(ids) < userPage {
			break
		}
		after = ids[len(ids)-1]
	}
	archived, err := s.Archive(ctx, now)
	return users, archived, err
}

// ArchiveCutoff is the last_at before which a closed-or-seen row is archived: Tehran midnight of today − 29, so a
// row last hit on today − 30 goes, one hit on today − 29 stays.
func ArchiveCutoff(now time.Time) time.Time {
	return civildate.InTehran(now).AddDays(-(ArchiveDays - 1)).TehranMidnight()
}

// Archive archives the stale rows in chunks; idempotent.
func (s *Service) Archive(ctx context.Context, now time.Time) (int64, error) {
	var total int64
	args := store.ArchiveStaleInboxParams{Now: sql.NullTime{Time: now, Valid: true},
		Cutoff: sql.NullTime{Time: ArchiveCutoff(now), Valid: true}}
	for {
		n, err := s.q.ArchiveStaleInbox(ctx, args)
		if err != nil {
			return total, fmt.Errorf("inbox sweep: archive: %w", err)
		}
		total += n
		if n < archiveBatch {
			return total, nil
		}
	}
}

// PushResult counts one push run.
type PushResult struct{ Candidates, Sent int }

// Push is messages.push_slot (M-09): for every user active in the last ActiveDays days whose local time is in
// PushHour, the push slot is selected once (its caps: 1 a local day, 3 in 7 local days; quiet hours 23–08) and the
// winner is written to the in-app notifications inbox through the user's notification preferences — with the neutral
// lock-screen text in every language, never the message text. The shown row is written in the same transaction, so a
// rerun within the local day finds the day cap spent (idempotent).
func (s *Service) Push(ctx context.Context, now time.Time) (PushResult, error) {
	var res PushResult
	comp, err := s.load(ctx)
	if err != nil {
		return res, err
	}
	if len(comp.BySlot[PushSlot]) == 0 {
		return res, nil
	}
	sinceAt := now.In(civildate.Tehran).AddDate(0, 0, -ActiveDays)
	since := sql.NullTime{Time: sinceAt, Valid: true}
	sinceDay := civildate.FromTime(sinceAt)
	var after uint64
	for {
		page, err := s.q.ListActivePushUsers(ctx, store.ListActivePushUsersParams{AfterID: after, Since: since,
			SinceDay: sinceDay, Limit: userPage})
		if err != nil {
			return res, fmt.Errorf("push slot: users: %w", err)
		}
		for _, u := range page {
			if now.In(usertz.ZoneOf(u.Timezone)).Hour() != PushHour {
				continue
			}
			res.Candidates++
			sent, err := s.pushUser(ctx, comp, u.UserID, now)
			if err != nil {
				if ctx.Err() != nil {
					return res, ctx.Err()
				}
				jobUserErrors.WithLabelValues(JobPush).Inc()
				s.logger.Warn("push slot: user skipped", "user_id", u.UserID, "error", err)
				continue
			}
			if sent {
				res.Sent++
			}
		}
		if len(page) < userPage {
			return res, nil
		}
		after = page[len(page)-1].UserID
	}
}

// pushUser runs the push slot for one user at now; it reports whether a notification was written.
func (s *Service) pushUser(ctx context.Context, comp *compiled.Compiled, userID uint64, now time.Time) (bool, error) {
	if s.facts == nil {
		return false, fmt.Errorf("push slot: no fact builder wired")
	}
	f, local, err := s.facts(ctx, userID, now)
	if err != nil {
		return false, err
	}
	if local.Hour() != PushHour { // the stored zone moved under us
		return false, nil
	}
	h, err := rules.LoadHistory(ctx, s.q, userID, rules.CapWindows(local, f))
	if err != nil {
		return false, err
	}
	winners := comp.Select(PushSlot, f, h, userID, local)
	if len(winners) == 0 {
		return false, nil
	}
	ch := winners[0]
	prefs := notifications.Defaults()
	if s.prefs != nil {
		if prefs, err = notifications.Load(ctx, s.prefs, userID); err != nil {
			return false, fmt.Errorf("push slot: preferences: %w", err)
		}
	}
	// notifications.Decide reads the quiet hours on the Tehran wall clock; her quiet hours are local (M-09), so it gets
	// her local wall-clock time expressed in Tehran.
	wall := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), 0, civildate.Tehran)
	if d := notifications.Decide(prefs, CategoryOf(ch.Rule.Kind), wall, time.Time{}); !d.Send {
		return false, nil
	}
	title, body, err := s.neutral(ctx)
	if err != nil {
		return false, err
	}
	data, err := json.Marshal(map[string]any{"rule_id": ch.Rule.ID, "slot": PushSlot,
		"version": feed.ServedVersion(ch.Rule), "variant": ch.Variant})
	if err != nil {
		return false, err
	}
	var link sql.NullString
	if t := ch.Rule.VariantContent(ch.Variant).CTATarget; strings.HasPrefix(t, "/") && !strings.HasPrefix(t, "//") {
		link = sql.NullString{String: t, Valid: true}
	}
	ts := sql.NullTime{Time: now, Valid: true}
	write := func(q store.Querier) error {
		if err := q.InsertPushSlotEvent(ctx, store.InsertPushSlotEventParams{RuleID: ch.Rule.ID, UserID: userID,
			Version: sql.NullInt32{Int32: int32(feed.ServedVersion(ch.Rule)), Valid: true}, //nolint:gosec // small
			Variant: ch.Variant, Now: ts}); err != nil {
			return fmt.Errorf("push slot: shown: %w", err)
		}
		if err := q.InsertPushNotification(ctx, store.InsertPushNotificationParams{UserID: userID, Title: title,
			Body: db.NullRawJSON{V: body, Valid: true}, ActionUrl: link, Data: db.NullRawJSON{V: data, Valid: true},
			Now: ts}); err != nil {
			return fmt.Errorf("push slot: notification: %w", err)
		}
		return nil
	}
	if s.db == nil {
		if err := write(s.q); err != nil {
			return false, err
		}
	} else {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return false, fmt.Errorf("push slot: begin: %w", err)
		}
		if err := write(store.New(tx)); err != nil {
			_ = tx.Rollback()
			return false, err
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("push slot: commit: %w", err)
		}
	}
	pushSent.Inc()
	return true, nil
}

// CategoryOf is the notification-settings switch that gates a push of kind: tips (content, product, marketing) go
// under «پیشنهادها و مقاله‌ها»; health messages (safety, clinical, engine) under the daily reminder switch. There is
// no dedicated messages switch yet.
func CategoryOf(kind string) notifications.Category {
	switch kind {
	case rules.KindContent, rules.KindProduct, rules.KindMarketing:
		return notifications.Articles
	}
	return notifications.DailyLog
}

// NeutralText is the push slot's lock-screen title and body in locale («یادآور ریتمی»): it never names a health detail.
func NeutralText(locale string) (title, body string) {
	return T("push.title", locale), T("push.body", locale)
}

// neutral are the title and body {code: text} of every active language.
func (s *Service) neutral(ctx context.Context) (json.RawMessage, json.RawMessage, error) {
	if s.langs == nil {
		return nil, nil, fmt.Errorf("push slot: no language list wired")
	}
	titles, bodies := map[string]string{}, map[string]string{}
	for _, code := range s.langs(ctx) {
		titles[code], bodies[code] = NeutralText(code)
	}
	t, err := json.Marshal(titles)
	if err != nil {
		return nil, nil, err
	}
	b, err := json.Marshal(bodies)
	return t, b, err
}

// RegisterJobs schedules the sweep (daily, unique per day) and the push slot (hourly) on queue low.
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	if err := q.RegisterPeriodic(queue.Job{
		Type: JobSweep, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{10 * time.Minute, 30 * time.Minute},
		Timeout: 30 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { _, _, err := s.Sweep(ctx, now()); return err }, SweepCron,
		queue.Unique(23*time.Hour)); err != nil {
		return err
	}
	return q.RegisterPeriodic(queue.Job{
		Type: JobPush, Priority: queue.Low, MaxTries: 2, Backoff: []time.Duration{5 * time.Minute},
		Timeout: 20 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { _, err := s.Push(ctx, now()); return err }, PushCron,
		queue.Unique(55*time.Minute))
}
