package ivf

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"context"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/ivf/store"
)

// LinkedAppointment is a care appointment a cycle date created, as the user left it.
type LinkedAppointment struct {
	Kind string
	Appt care.Appointment
}

// apptSpec is one appointment the cycle wants.
type apptSpec struct {
	Kind string
	At   time.Time
}

// kindTopics are the care appointment topics of the kinds.
var kindTopics = map[string]string{KindScan: "ultrasound", KindRetrieval: "other", KindTransfer: "other", KindBeta: "lab"}

// plan is the appointments an open cycle's dates want: the next scan, retrieval and transfer at their times, the beta
// test on beta_on at BetaHour (Tehran).
func plan(c Cycle) []apptSpec {
	var out []apptSpec
	if !c.NextScanAt.IsZero() {
		out = append(out, apptSpec{KindScan, c.NextScanAt})
	}
	if !c.RetrievalAt.IsZero() {
		out = append(out, apptSpec{KindRetrieval, c.RetrievalAt})
	}
	if !c.TransferAt.IsZero() {
		out = append(out, apptSpec{KindTransfer, c.TransferAt})
	}
	if !c.BetaOn.IsZero() {
		out = append(out, apptSpec{KindBeta, c.BetaOn.TehranMidnight().Add(BetaHour * time.Hour)})
	}
	return out
}

func linkedAppointments(ctx context.Context, q *store.Queries, userID, cycleID uint64) ([]LinkedAppointment, error) {
	rows, err := q.ListReminderLinks(ctx, store.ListReminderLinksParams{UserID: userID, CycleID: cycleID})
	if err != nil {
		return nil, fmt.Errorf("ivf: load appointments: %w", err)
	}
	out := make([]LinkedAppointment, 0, len(rows))
	for _, r := range rows {
		out = append(out, LinkedAppointment{Kind: r.Kind, Appt: care.ParseAppointment(carestore.Reminder(r.Reminder))})
	}
	return out, nil
}

// syncAppointments makes the cycle's linked care appointments match specs: a kind no longer wanted is deleted, a
// kept one is moved to its new time (the user's title, notes, details and switch stay), a new one is created
// (in person, reminded a day before, titled in locale) and linked.
func syncAppointments(ctx context.Context, q *store.Queries, cq *carestore.Queries, userID, cycleID uint64, specs []apptSpec, now time.Time, locale string) error {
	links, err := linkedAppointments(ctx, q, userID, cycleID)
	if err != nil {
		return err
	}
	want := make(map[string]apptSpec, len(specs))
	for _, s := range specs {
		want[s.Kind] = s
	}
	have := map[string]bool{}
	ts := sql.NullTime{Time: now, Valid: true}
	for _, l := range links {
		row := l.Appt.Row
		s, ok := want[l.Kind]
		if !ok {
			if _, err := cq.DeleteAppointment(ctx, carestore.DeleteAppointmentParams{ID: row.ID, UserID: userID}); err != nil {
				return fmt.Errorf("ivf: delete appointment: %w", err)
			}
			continue
		}
		have[l.Kind] = true
		if row.ScheduledAt.Valid && row.ScheduledAt.Time.Equal(s.At) {
			continue
		}
		if err := q.MoveAppointment(ctx, store.MoveAppointmentParams{
			ScheduledAt: sql.NullTime{Time: s.At, Valid: true}, Now: ts, ID: row.ID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("ivf: move appointment: %w", err)
		}
	}
	for _, s := range specs {
		if have[s.Kind] {
			continue
		}
		meta, err := json.Marshal(care.AppointmentMeta{
			V: care.MetaVersion, Kind: "in_person", Topic: kindTopics[s.Kind], RemindBefore: care.DefaultRemindBefore,
			Prep: []care.PrepItem{}, Status: care.StatusScheduled,
		})
		if err != nil {
			return fmt.Errorf("ivf: encode appointment: %w", err)
		}
		id, err := cq.InsertAppointment(ctx, carestore.InsertAppointmentParams{
			UserID: userID, Title: T("reminders."+s.Kind, locale), ScheduledAt: sql.NullTime{Time: s.At, Valid: true},
			IsActive: true, Meta: db.NullRawJSON{V: meta, Valid: true}, CreatedAt: ts, UpdatedAt: ts,
		})
		if err != nil {
			return fmt.Errorf("ivf: insert appointment: %w", err)
		}
		if err := q.InsertReminderLink(ctx, store.InsertReminderLinkParams{
			UserID: userID, CycleID: cycleID, Kind: s.Kind, ReminderID: uint64(id), Now: ts, //nolint:gosec // G115: auto-increment id
		}); err != nil {
			return fmt.Errorf("ivf: link appointment: %w", err)
		}
	}
	return nil
}

// dropUpcoming deletes the cycle's linked appointments still ahead of now (a closed cycle has no more visits).
func dropUpcoming(ctx context.Context, q *store.Queries, cq *carestore.Queries, userID, cycleID uint64, now time.Time) error {
	links, err := linkedAppointments(ctx, q, userID, cycleID)
	if err != nil {
		return err
	}
	for _, l := range links {
		if row := l.Appt.Row; row.ScheduledAt.Valid && row.ScheduledAt.Time.After(now) {
			if _, err := cq.DeleteAppointment(ctx, carestore.DeleteAppointmentParams{ID: row.ID, UserID: userID}); err != nil {
				return fmt.Errorf("ivf: delete appointment: %w", err)
			}
		}
	}
	return nil
}

// NextAppointment is the earliest upcoming (scheduled, not cancelled) linked appointment, nil when none.
func NextAppointment(links []LinkedAppointment, now time.Time) *LinkedAppointment {
	var best *LinkedAppointment
	for i := range links {
		a := links[i].Appt
		if !a.Upcoming(now) {
			continue
		}
		if best == nil || a.Row.ScheduledAt.Time.Before(best.Appt.Row.ScheduledAt.Time) {
			best = &links[i]
		}
	}
	return best
}
