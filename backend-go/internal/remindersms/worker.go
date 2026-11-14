package remindersms

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

const (
	// Every is how often the worker looks for due reminders.
	Every = time.Minute
	// Lookback is how late a reminder may still be texted (a restart, a slow tick); older ones are dropped.
	Lookback = 10 * time.Minute
	// LogKeep is how long reminder_sms rows are kept.
	LogKeep = 7 * 24 * time.Hour
	// maxRemindOffset is the longest appointment remind_before ("2d").
	maxRemindOffset = 48 * time.Hour
)

// Audience lists the users who get reminders by SMS now (plus.Service.PlusUsers).
type Audience func(ctx context.Context, now time.Time) ([]uint64, error)

// Worker texts the due reminders of the audience.
type Worker struct {
	q        *store.Queries
	prefs    notifications.Getter
	audience Audience
	sender   Sender
	locale   func(ctx context.Context) string // the wording language of the tokens (the default language)
	dailyCap int
	logger   *slog.Logger
	pruned   time.Time
}

// NewWorker wires the worker.
func NewWorker(db store.DBTX, prefs notifications.Getter, audience Audience, sender Sender,
	locale func(ctx context.Context) string, dailyCap int, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{q: store.New(db), prefs: prefs, audience: audience, sender: sender, locale: locale,
		dailyCap: max(dailyCap, 1), logger: logger}
}

// Loop runs Tick every Every until ctx ends.
func (w *Worker) Loop(ctx context.Context, now func() time.Time) {
	t := time.NewTicker(Every)
	defer t.Stop()
	for {
		if _, err := w.Tick(ctx, now()); err != nil && ctx.Err() == nil {
			w.logger.ErrorContext(ctx, "remindersms: tick", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

// due is one reminder occurrence to text.
type due struct {
	userID, reminderID uint64
	key                string
	at                 time.Time
	category           notifications.Category
	dose               *care.Occurrence // medications: to skip a dose already ticked
	sms                SMS
}

// Tick texts the reminders that came due in the last Lookback and returns how many SMS went out. Each occurrence
// is claimed in reminder_sms first, so several processes never text it twice; one that is switched off, ticked,
// inside quiet hours or over the day's cap stays claimed and silent.
func (w *Worker) Tick(ctx context.Context, now time.Time) (int, error) {
	now = now.In(civildate.Tehran)
	if now.Sub(w.pruned) > time.Hour {
		if err := w.q.PruneReminderSMS(ctx, now.Add(-LogKeep)); err != nil {
			return 0, fmt.Errorf("remindersms: prune: %w", err)
		}
		w.pruned = now
	}
	users, err := w.audience(ctx, now)
	if err != nil {
		return 0, err
	}
	if len(users) == 0 {
		return 0, nil
	}
	from, to := now.Add(-Lookback), now.Add(time.Second)
	locale := w.locale(ctx)

	meds, err := w.q.ListActiveMedicationsOfUsers(ctx, users)
	if err != nil {
		return 0, fmt.Errorf("remindersms: medications: %w", err)
	}
	appts, err := w.q.ListActiveAppointmentsOfUsers(ctx, store.ListActiveAppointmentsOfUsersParams{
		UserIds: users, FromAt: nullTime(from), ToAt: nullTime(to.Add(maxRemindOffset)),
	})
	if err != nil {
		return 0, fmt.Errorf("remindersms: appointments: %w", err)
	}
	var list []due
	for _, o := range care.Occurrences(meds, from, to) {
		list = append(list, due{
			userID: o.Med.Row.UserID, reminderID: o.Med.Row.ID, key: o.Key(), at: o.At,
			category: notifications.Medications, dose: &o,
			sms: SMS{Kind: KindMedication, Name: o.Med.Row.Title, Clock: o.Slot},
		})
	}
	for _, d := range care.AppointmentsDue(appts, from, to) {
		list = append(list, due{
			userID: d.A.Row.UserID, reminderID: d.A.Row.ID, key: d.Key(), at: d.At,
			category: notifications.Appointments,
			sms:      SMS{Kind: KindAppointment, Name: d.A.Row.Title, Clock: d.Visit.Format("15:04"), When: d.When(locale)},
		})
	}
	slices.SortStableFunc(list, func(a, b due) int { return cmp.Or(a.at.Compare(b.at), cmp.Compare(a.reminderID, b.reminderID)) })

	type userState struct {
		prefs  notifications.Preferences
		mobile string
		sent   int // SMS of today, this tick included
		taken  map[civildate.Date][]store.ListIntakesOnDateRow
	}
	states := map[uint64]*userState{}
	sent := 0
	for _, d := range list {
		n, err := w.q.ClaimReminderSMS(ctx, store.ClaimReminderSMSParams{
			UserID: d.userID, ReminderID: d.reminderID, Occurrence: d.key, SentAt: now,
		})
		if err != nil {
			return sent, fmt.Errorf("remindersms: claim: %w", err)
		}
		if n == 0 {
			continue
		}
		u := states[d.userID]
		if u == nil {
			u = &userState{taken: map[civildate.Date][]store.ListIntakesOnDateRow{}}
			if u.prefs, err = notifications.Load(ctx, w.prefs, d.userID); err != nil {
				return sent, err
			}
			mobile, err := w.q.GetUserMobile(ctx, d.userID)
			if err != nil {
				return sent, fmt.Errorf("remindersms: mobile: %w", err)
			}
			u.mobile = mobile.String
			count, err := w.q.CountReminderSMSSince(ctx, store.CountReminderSMSSinceParams{
				UserID: d.userID, SentAt: civildate.InTehran(now).TehranMidnight(),
			})
			if err != nil {
				return sent, fmt.Errorf("remindersms: count: %w", err)
			}
			u.sent = int(count)
			states[d.userID] = u
		}
		if u.mobile == "" || !u.prefs.Enabled(notifications.SMSReminders) || u.sent >= w.dailyCap ||
			!notifications.Decide(u.prefs, d.category, d.at, time.Time{}).Send {
			continue
		}
		if d.dose != nil {
			taken, ok := u.taken[d.dose.Day]
			if !ok {
				if taken, err = w.q.ListIntakesOnDate(ctx, store.ListIntakesOnDateParams{UserID: d.userID, IntakeDate: d.dose.Day}); err != nil {
					return sent, fmt.Errorf("remindersms: intakes: %w", err)
				}
				u.taken[d.dose.Day] = taken
			}
			if slices.ContainsFunc(taken, func(t store.ListIntakesOnDateRow) bool {
				return t.ReminderID == d.reminderID && t.Slot == d.dose.Slot
			}) {
				continue
			}
		}
		d.sms.Discreet = u.prefs.Discreet()
		switch err := w.sender.Send(ctx, u.mobile, d.sms); {
		case errors.Is(err, ErrNotSent):
			continue
		case err != nil:
			w.logger.WarnContext(ctx, "remindersms: not delivered", slog.Uint64("reminder", d.reminderID), slog.String("error", err.Error()))
			continue
		}
		if err := w.q.MarkReminderSMSSent(ctx, store.MarkReminderSMSSentParams{ReminderID: d.reminderID, Occurrence: d.key}); err != nil {
			return sent, fmt.Errorf("remindersms: mark sent: %w", err)
		}
		u.sent++
		sent++
	}
	return sent, nil
}
