package learning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	authsms "github.com/ritme/backend-go/internal/auth/sms"
	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// SMS outbox statuses and skip reasons.
const (
	smsSent    = "sent"
	smsSkipped = "skipped"
	smsFailed  = "failed"

	ReasonSMSDisabled = "sms_disabled"
	ReasonRevoked     = "revoked"
	ReasonPhoneCap    = "phone_cap"
	ReasonSenderCap   = "instructor_cap"
	ReasonNoCourses   = "no_courses"
)

// Dispatcher settings.
const (
	// SMSPerPhonePerDay caps course SMS to one number from all instructors per day.
	SMSPerPhonePerDay = 3
	// SMSPerInstructorPerDay caps the SMS one instructor triggers per day.
	SMSPerInstructorPerDay = 300
	// SMSMaxAttempts: a provider failure is retried this many times, SMSRetryAfter apart.
	SMSMaxAttempts = 3
	SMSRetryAfter  = 5 * time.Minute
	smsLease       = 2 * time.Minute
	smsBatch       = 50
	// DispatchEvery is how often the in-process loop looks for due SMS.
	DispatchEvery = time.Minute
)

// DispatchSMS sends the due outbox rows once: each row is claimed (lease) so parallel instances never double-send,
// then the notification policy applies — an account's own settings (category learning off → skipped, quiet hours →
// deferred to the window's end), the default settings for a number without an account — and the daily caps.
// It returns how many SMS went out.
func (s *Service) DispatchSMS(ctx context.Context, now time.Time) (int, error) {
	now = now.In(civildate.Tehran).Truncate(time.Second)
	ids, err := s.q.ListDueSMS(ctx, store.ListDueSMSParams{DueBefore: now, LeaseBefore: nt(now), Limit: smsBatch})
	if err != nil {
		return 0, fmt.Errorf("learning: due sms: %w", err)
	}
	sent := 0
	for _, id := range ids {
		n, err := s.q.ClaimSMS(ctx, store.ClaimSMSParams{LeaseUntil: nt(now.Add(smsLease)), Now: nt(now), ID: id, DueBefore: now, LeaseBefore: nt(now)})
		if err != nil {
			return sent, fmt.Errorf("learning: claim sms: %w", err)
		}
		if n == 0 {
			continue // another instance took it
		}
		ok, err := s.dispatchOne(ctx, id, now)
		if err != nil {
			s.logger.ErrorContext(ctx, "learning: SMS dispatch failed", slog.Uint64("outbox_id", id), slog.String("error", err.Error()))
			continue
		}
		if ok {
			sent++
		}
	}
	return sent, nil
}

func (s *Service) finish(ctx context.Context, id uint64, status, reason string, attempts uint8, now time.Time) error {
	p := store.FinishSMSParams{Status: status, Reason: ns(reason), Attempts: attempts, Now: nt(now), ID: id}
	if status == smsSent {
		p.SentAt = nt(now)
	}
	if err := s.q.FinishSMS(ctx, p); err != nil {
		return fmt.Errorf("learning: finish sms: %w", err)
	}
	return nil
}

func (s *Service) deferSMS(ctx context.Context, id uint64, until time.Time, attempts uint8, reason string, now time.Time) error {
	if err := s.q.DeferSMS(ctx, store.DeferSMSParams{DueAt: nt(until).Time, Attempts: attempts, Reason: ns(reason), Now: nt(now), ID: id}); err != nil {
		return fmt.Errorf("learning: defer sms: %w", err)
	}
	return nil
}

func (s *Service) dispatchOne(ctx context.Context, id uint64, now time.Time) (bool, error) {
	job, err := s.q.GetSMSJob(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("learning: sms job: %w", err)
	}
	if job.GrantStatus == GrantRevoked {
		return false, s.finish(ctx, id, smsSkipped, ReasonRevoked, job.Attempts, now)
	}
	if !s.sms.Delivers() {
		return false, s.finish(ctx, id, smsSkipped, ReasonSMSDisabled, job.Attempts, now)
	}
	prefs := notifications.Defaults()
	if userID := uid(job.UserID); userID != 0 {
		if prefs, err = notifications.Load(ctx, s.prefs, userID); err != nil {
			return false, err
		}
	}
	d := notifications.Decide(prefs, notifications.Learning, now, time.Time{})
	if !d.Send {
		if d.DeferUntil.IsZero() {
			return false, s.finish(ctx, id, smsSkipped, d.Reason, job.Attempts, now)
		}
		return false, s.deferSMS(ctx, id, d.DeferUntil, job.Attempts, d.Reason, now)
	}
	since := nt(now.Add(-24 * time.Hour))
	toPhone, err := s.q.CountSMSSentToPhone(ctx, store.CountSMSSentToPhoneParams{Phone: job.Phone, Since: since})
	if err != nil {
		return false, fmt.Errorf("learning: sms phone cap: %w", err)
	}
	if toPhone >= SMSPerPhonePerDay {
		return false, s.finish(ctx, id, smsSkipped, ReasonPhoneCap, job.Attempts, now)
	}
	byIns, err := s.q.CountSMSSentByInstructor(ctx, store.CountSMSSentByInstructorParams{InstructorID: job.InstructorID, Since: since})
	if err != nil {
		return false, fmt.Errorf("learning: sms instructor cap: %w", err)
	}
	if byIns >= SMSPerInstructorPerDay {
		return false, s.finish(ctx, id, smsSkipped, ReasonSenderCap, job.Attempts, now)
	}
	g, err := s.q.GetInstructorGrant(ctx, store.GetInstructorGrantParams{ID: job.GrantID, InstructorID: job.InstructorID})
	if err != nil {
		return false, fmt.Errorf("learning: sms grant: %w", err)
	}
	courses, err := s.q.ListGrantCourses(ctx, store.ListGrantCoursesParams{CourseID: g.CourseID, GroupID: g.GroupID})
	if err != nil {
		return false, fmt.Errorf("learning: sms courses: %w", err)
	}
	if len(courses) == 0 {
		return false, s.finish(ctx, id, smsSkipped, ReasonNoCourses, job.Attempts, now)
	}
	attempts := job.Attempts + 1
	if err := s.sms.SendUnlocked(ctx, job.Phone, len(courses)); err != nil {
		s.logger.WarnContext(ctx, "learning: course SMS not delivered",
			slog.String("mobile", authsms.MaskMobile(job.Phone)), slog.Int("attempt", int(attempts)), slog.String("error", err.Error()))
		if attempts >= SMSMaxAttempts {
			return false, s.finish(ctx, id, smsFailed, "provider", attempts, now)
		}
		return false, s.deferSMS(ctx, id, now.Add(SMSRetryAfter), attempts, "provider", now)
	}
	return true, s.finish(ctx, id, smsSent, "", attempts, now)
}

// DispatchLoop runs DispatchSMS every DispatchEvery until ctx ends (started with the listener).
func (s *Service) DispatchLoop(ctx context.Context, now func() time.Time) {
	t := time.NewTicker(DispatchEvery)
	defer t.Stop()
	for {
		if _, err := s.DispatchSMS(ctx, now()); err != nil && ctx.Err() == nil {
			s.logger.ErrorContext(ctx, "learning: SMS dispatch", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
