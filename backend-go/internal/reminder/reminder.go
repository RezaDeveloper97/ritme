// Package reminder is ReminderController (GET /reminders/enums, GET/POST /reminders,
// PUT/DELETE /reminders/{id}) and the Reminder model's JSON.
package reminder

import (
	"database/sql"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/reminder/store"
)

// JSON is Reminder::toArray() for a loaded row: every column in table order; scheduled_at
// and the timestamps are datetime casts, starts_on / ends_on plain `date` casts (Tehran
// midnight in UTC → the previous day), recurrence_time uncast ("16:00:00"), is_active
// boolean, meta array.
func JSON(r store.Reminder) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", r.ID,
		"user_id", r.UserID,
		"type", r.Type,
		"title", r.Title,
		"subtitle", nullString(r.Subtitle),
		"notes", nullString(r.Notes),
		"scheduled_at", nullDateTime(r.ScheduledAt),
		"recurrence", r.Recurrence,
		"recurrence_time", nullString(r.RecurrenceTime),
		"starts_on", nullDateCast(r.StartsOn),
		"ends_on", nullDateCast(r.EndsOn),
		"is_active", r.IsActive,
		"meta", decodeArray(r.Meta.V, r.Meta.Valid),
		"created_at", nullDateTime(r.CreatedAt),
		"updated_at", nullDateTime(r.UpdatedAt),
	)
}

// List is a collection of reminders ([] when empty).
func List(rows []store.Reminder) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		out = append(out, JSON(r))
	}
	return out
}

func nullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func nullDateTime(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.DateTime(t.Time)
}

func nullDateCast(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return jsonx.DateCast(d.Date)
}

// dateTimeValue is the datetime cast of an instant set on a fresh model (seconds only:
// fromDateTime() stores 'Y-m-d H:i:s').
func dateTimeValue(t time.Time) jsonx.LaravelDateTime {
	return jsonx.DateTime(t.Truncate(time.Second))
}
