package loss

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	companionstore "github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/loss/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// followupAppointment is one dated follow-up as an M3 care appointment (so it shows in her /care list and calendar).
// It is created private (care.AppointmentMeta.Private: never shown to a companion) with a neutral title that says
// nothing about the loss («آزمایش» / «ویزیت پزشک»), in case it is seen anywhere else (a calendar export, a screen).
type followupAppointment struct {
	title string // lang key of the neutral title
	topic string // care appointment topic
}

var (
	betaAppointment  = followupAppointment{title: "appointments.beta", topic: "lab"}
	visitAppointment = followupAppointment{title: "appointments.visit", topic: "checkup"}
)

// Followup applies PUT /loss/followup to the newest loss: bleeding stopped (today) or not; the next beta test day
// (a care appointment at BetaHour, created or moved; null cancels it); beta negative (today — clears the next test
// and cancels its upcoming appointment); the visit time (a care appointment; null cancels it). New appointments get
// their neutral title in locale.
func (s *Service) Followup(ctx context.Context, userID uint64, in FollowupInput, now time.Time, locale string) error {
	today := civildate.InTehran(now)
	titles := map[string]string{betaAppointment.title: T(betaAppointment.title, locale), visitAppointment.title: T(visitAppointment.title, locale)}

	return s.inTx(ctx, func(tx *sql.Tx) error {
		q, cq := store.New(tx), carestore.New(tx)
		cur, err := latest(ctx, q, userID)
		if err != nil {
			return err
		}
		p := store.UpdateLossFollowupParams{
			BleedingStoppedOn: cur.BleedingStoppedOn, BetaNextOn: cur.BetaNextOn, BetaNegativeOn: cur.BetaNegativeOn,
			BetaReminderID: cur.BetaReminderID, VisitReminderID: cur.VisitReminderID, Now: nt(now), ID: cur.ID, UserID: userID,
		}
		if in.BleedingStopped != nil {
			switch {
			case !*in.BleedingStopped:
				p.BleedingStoppedOn = civildate.NullDate{}
			case !p.BleedingStoppedOn.Valid:
				p.BleedingStoppedOn = civildate.NullDate{Date: today, Valid: true}
			}
		}
		if in.BetaNextOn != nil {
			p.BetaNextOn = *in.BetaNextOn
			if p.BetaNextOn.Valid {
				p.BetaNegativeOn = civildate.NullDate{}
				at := p.BetaNextOn.Date.TehranMidnight().Add(BetaHour * time.Hour)
				if p.BetaReminderID, err = upsertAppointment(ctx, cq, userID, p.BetaReminderID, at, betaAppointment.topic, titles[betaAppointment.title], now); err != nil {
					return err
				}
			} else if err := cancelUpcoming(ctx, q, cq, userID, p.BetaReminderID, now); err != nil {
				return err
			}
		}
		if in.BetaNegative != nil {
			switch {
			case !*in.BetaNegative:
				p.BetaNegativeOn = civildate.NullDate{}
			case !p.BetaNegativeOn.Valid:
				p.BetaNegativeOn, p.BetaNextOn = civildate.NullDate{Date: today, Valid: true}, civildate.NullDate{}
				if err := cancelUpcoming(ctx, q, cq, userID, p.BetaReminderID, now); err != nil {
					return err
				}
			}
		}
		if in.VisitAt != nil {
			if in.VisitAt.Valid {
				if p.VisitReminderID, err = upsertAppointment(ctx, cq, userID, p.VisitReminderID, in.VisitAt.Time, visitAppointment.topic, titles[visitAppointment.title], now); err != nil {
					return err
				}
			} else if err := cancelUpcoming(ctx, q, cq, userID, p.VisitReminderID, now); err != nil {
				return err
			}
		}
		if err := q.UpdateLossFollowup(ctx, p); err != nil {
			return fmt.Errorf("loss: followup: %w", err)
		}
		return nil
	})
}

// upsertAppointment schedules the follow-up appointment at `at`: the linked one is moved (and re-opened when it was
// cancelled; the user's own edits — title, place, notes, reminder offset — are kept), otherwise a new one is created.
func upsertAppointment(ctx context.Context, q *carestore.Queries, userID uint64, id sql.NullInt64, at time.Time, topic, title string, now time.Time) (sql.NullInt64, error) {
	if id.Valid {
		row, err := q.GetAppointment(ctx, carestore.GetAppointmentParams{ID: uint64(id.Int64), UserID: userID}) //nolint:gosec // positive id
		switch {
		case err == nil:
			a := care.ParseAppointment(row)
			a.Meta.Status, a.Meta.Private = care.StatusScheduled, true
			meta, err := json.Marshal(a.Meta)
			if err != nil {
				return id, fmt.Errorf("loss: appointment meta: %w", err)
			}
			if err := q.UpdateAppointment(ctx, carestore.UpdateAppointmentParams{
				Title: row.Title, Subtitle: row.Subtitle, Notes: row.Notes, ScheduledAt: nt(at), IsActive: true,
				Meta: rootdb.NullRawJSON{V: meta, Valid: true}, UpdatedAt: nt(now), ID: row.ID, UserID: userID,
			}); err != nil {
				return id, fmt.Errorf("loss: move appointment: %w", err)
			}
			return id, nil
		case !errors.Is(err, sql.ErrNoRows):
			return id, fmt.Errorf("loss: appointment: %w", err)
		}
	}
	meta, err := json.Marshal(care.AppointmentMeta{
		V: care.MetaVersion, Kind: care.AppointmentKinds[0], Topic: topic, RemindBefore: care.DefaultRemindBefore,
		Prep: []care.PrepItem{}, Status: care.StatusScheduled, Private: true,
	})
	if err != nil {
		return id, fmt.Errorf("loss: appointment meta: %w", err)
	}
	newID, err := q.InsertAppointment(ctx, carestore.InsertAppointmentParams{
		UserID: userID, Title: title, ScheduledAt: nt(at), IsActive: true,
		Meta: rootdb.NullRawJSON{V: meta, Valid: true}, CreatedAt: nt(now), UpdatedAt: nt(now),
	})
	if err != nil {
		return id, fmt.Errorf("loss: create appointment: %w", err)
	}
	return sql.NullInt64{Int64: newID, Valid: true}, nil
}

// cancelUpcoming cancels the linked follow-up appointment when it is still ahead (a past one stays as history).
func cancelUpcoming(ctx context.Context, q store.Querier, cq *carestore.Queries, userID uint64, id sql.NullInt64, now time.Time) error {
	a, err := appointment(ctx, cq, userID, id)
	if err != nil || a == nil || !a.Upcoming(now) {
		return err
	}
	if err := q.CancelAppointmentReminder(ctx, store.CancelAppointmentReminderParams{Now: nt(now), ID: a.Row.ID, UserID: userID}); err != nil {
		return fmt.Errorf("loss: cancel appointment: %w", err)
	}
	return nil
}

// Mood records the day's mood check-in on the newest loss (one per day, the last answer wins).
func (s *Service) Mood(ctx context.Context, userID uint64, mood string, day civildate.Date, now time.Time) error {
	q := store.New(s.db)
	cur, err := latest(ctx, q, userID)
	if err != nil {
		return err
	}
	if err := q.UpsertLossMood(ctx, store.UpsertLossMoodParams{UserID: userID, LossID: cur.ID, LogDate: day, Mood: mood, Now: nt(now)}); err != nil {
		return fmt.Errorf("loss: mood: %w", err)
	}
	return nil
}

// Note is the newest loss's private note in plain text ("" when none) and when it was written.
func (s *Service) Note(ctx context.Context, userID uint64) (string, sql.NullTime, error) {
	if s.notes.Disabled() {
		return "", sql.NullTime{}, ErrNoteUnavailable
	}
	cur, err := latest(ctx, store.New(s.db), userID)
	if err != nil {
		return "", sql.NullTime{}, err
	}
	if !cur.PrivateNote.Valid || cur.PrivateNote.String == "" {
		return "", sql.NullTime{}, nil
	}
	plain, err := s.notes.Open(cur.PrivateNote.String, userID, cur.ID)
	if err != nil {
		return "", sql.NullTime{}, err
	}
	return plain, cur.NoteUpdatedAt, nil
}

// SetNote encrypts and stores the private note ("" clears it).
func (s *Service) SetNote(ctx context.Context, userID uint64, note string, now time.Time) error {
	if s.notes.Disabled() {
		return ErrNoteUnavailable
	}
	q := store.New(s.db)
	cur, err := latest(ctx, q, userID)
	if err != nil {
		return err
	}
	p := store.SetLossNoteParams{Now: nt(now), ID: cur.ID, UserID: userID}
	if note != "" {
		sealed, err := s.notes.Seal(note, userID, cur.ID)
		if err != nil {
			return err
		}
		p.PrivateNote, p.NoteUpdatedAt = sql.NullString{String: sealed, Valid: true}, nt(now)
	}
	if err := q.SetLossNote(ctx, p); err != nil {
		return fmt.Errorf("loss: note: %w", err)
	}
	return nil
}

// NextStep stores the chosen next step on the newest loss and switches the life-stage mode: ttc → ttc, cycle and
// nothing → cycle (with nothing, only the medical follow-up reminders remain).
func (s *Service) NextStep(ctx context.Context, u *auth.User, choice string, now time.Time) error {
	q := store.New(s.db)
	cur, err := latest(ctx, q, u.ID)
	if err != nil {
		return err
	}
	if err := q.SetLossNextStep(ctx, store.SetLossNextStepParams{
		NextStep: sql.NullString{String: choice, Valid: true}, Now: nt(now), ID: cur.ID, UserID: u.ID,
	}); err != nil {
		return fmt.Errorf("loss: next step: %w", err)
	}
	mode := enums.LifeModeCycle
	if choice == NextTTC {
		mode = enums.LifeModeTTC
	}
	if s.modes == nil {
		return nil
	}
	return s.modes.SwitchLifeMode(ctx, u, mode, now)
}

// DeleteNote erases the newest loss's private note (works even while notes are switched off: nothing is decrypted).
func (s *Service) DeleteNote(ctx context.Context, userID uint64, now time.Time) error {
	q := store.New(s.db)
	cur, err := latest(ctx, q, userID)
	if err != nil {
		return err
	}
	if err := q.SetLossNote(ctx, store.SetLossNoteParams{Now: nt(now), ID: cur.ID, UserID: userID}); err != nil {
		return fmt.Errorf("loss: delete note: %w", err)
	}
	return nil
}

// Delete erases the newest loss record with its moods (FK cascade) and encrypted note, and the one-line notices her
// companions got about it. It never resurrects pregnancy content: pregnancy mode stays off, closed alerts stay closed
// and paused reminders stay paused (the forced pregnancy message mode keys on the ended pregnancy, not on this row).
// Her follow-up appointments stay in her own care list (private). An earlier loss, if any, becomes the newest.
func (s *Service) Delete(ctx context.Context, userID uint64) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		q := store.New(tx)
		cur, err := latest(ctx, q, userID)
		if err != nil {
			return err
		}
		if _, err := q.DeleteLoss(ctx, store.DeleteLossParams{ID: cur.ID, UserID: userID}); err != nil {
			return fmt.Errorf("loss: delete: %w", err)
		}
		if err := companionstore.New(tx).DeletePregnancyNoticesForOwner(ctx, userID); err != nil {
			return fmt.Errorf("loss: delete notices: %w", err)
		}
		return nil
	})
}
