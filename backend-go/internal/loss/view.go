package loss

import (
	"database/sql"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date
}

func isoOrNull(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.ISO8601(t.Time)
}

// appointmentJSON is the linked care appointment while it is scheduled (the /care/appointments resource), else null.
func appointmentJSON(a *care.Appointment, st State) any {
	if a == nil || a.Meta.Status != care.StatusScheduled {
		return nil
	}
	return a.JSON(st.Now)
}

// StateJSON is GET /loss: codes and dates only — every label comes from the catalog (loss_* groups). Without a
// recorded loss everything is null and losses_count is 0.
func StateJSON(st State) *jsonx.OrderedMap {
	if st.Loss == nil {
		return jsonx.Obj("loss", nil, "losses_count", 0, "recurrent_hint", false,
			"followup", nil, "mood", nil, "note", nil)
	}
	l := st.Loss
	var nextStep any
	if l.NextStep.Valid {
		nextStep = l.NextStep.String
	}
	from := civildate.InTehran(l.CreatedAt.Time)
	if l.OccurredOn.Valid {
		from = l.OccurredOn.Date
	}
	var today any
	moods := make([]*jsonx.OrderedMap, 0, len(st.Moods))
	for _, m := range st.Moods {
		if m.LogDate == st.Today {
			today = m.Mood
		}
		moods = append(moods, jsonx.Obj("date", m.LogDate, "mood", m.Mood))
	}
	var bleedingToday any
	if st.BleedingToday != "" {
		bleedingToday = st.BleedingToday
	}
	return jsonx.Obj(
		"loss", jsonx.Obj(
			"id", l.ID,
			"type", l.LossType,
			"occurred_on", nullDate(l.OccurredOn),
			"notify_companion", l.NotifyCompanion,
			"companion_notified", l.CompanionNotifiedAt.Valid,
			"content_stopped", l.ContentStoppedAt.Valid,
			"next_step", nextStep,
			"created_at", isoOrNull(l.CreatedAt),
		),
		"losses_count", st.Count,
		"recurrent_hint", st.Count >= RecurrentLosses,
		"followup", jsonx.Obj(
			"bleeding", jsonx.Obj(
				"stopped", l.BleedingStoppedOn.Valid,
				"stopped_on", nullDate(l.BleedingStoppedOn),
				"today", bleedingToday,
			),
			"beta", jsonx.Obj(
				"negative", l.BetaNegativeOn.Valid,
				"negative_on", nullDate(l.BetaNegativeOn),
				"next_on", nullDate(l.BetaNextOn),
				"appointment", appointmentJSON(st.Beta, st),
			),
			"visit", jsonx.Obj(
				"suggested_on", from.AddDays(VisitAfterDays),
				"appointment", appointmentJSON(st.Visit, st),
			),
		),
		"mood", jsonx.Obj("today", today, "recent", moods),
		"note", jsonx.Obj("has_note", l.PrivateNote.Valid && l.PrivateNote.String != "", "updated_at", isoOrNull(l.NoteUpdatedAt)),
	)
}

// NoteJSON is GET /loss/note.
func NoteJSON(note string, updatedAt sql.NullTime) *jsonx.OrderedMap {
	var n any
	if note != "" {
		n = note
	}
	return jsonx.Obj("note", n, "updated_at", isoOrNull(updatedAt))
}
