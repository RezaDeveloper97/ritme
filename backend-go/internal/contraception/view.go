package contraception

import (
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// date is a Y-m-d string, nil when zero.
func date(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func orNil[T comparable](v T) any {
	var zero T
	if v == zero {
		return nil
	}
	return v
}

// OverviewJSON is GET /contraception's data: {tracking, method, pill, reminders}.
func OverviewJSON(o Overview) *jsonx.OrderedMap {
	var method, pill any
	if m := o.Method; m != nil {
		method = MethodJSON(*m, o.Reminder, o.Today)
		if p, ok := m.Pack(); ok {
			pill = PillJSON(*m, p, o.Today, o.Logs)
		}
	}
	reminders := make([]*jsonx.OrderedMap, 0, len(o.Reminders))
	for _, r := range o.Reminders {
		reminders = append(reminders, ReminderJSON(r))
	}
	return jsonx.Obj("tracking", o.Tracking, "method", method, "pill", pill, "reminders", jsonx.List(reminders))
}

// MethodJSON is the saved method: the setup fields (null when not the method's) plus the dates derived from them.
// `reminder` is the one pill reminder (B-N1-09) for pill methods, null otherwise; `packs_left` is the count as of
// today (a pack started since the count used one), so sending it back unchanged keeps the stock.
func MethodJSON(m Method, r PillReminder, today civildate.Date) *jsonx.OrderedMap {
	var reminder, packsLeft any
	if IsPill(m.Method) {
		reminder = jsonx.Obj("enabled", r.Enabled, "time", notifications.FormatClock(r.Minute))
	}
	if rf := m.Refill(today); rf.Known {
		packsLeft = rf.PacksLeft
	}
	return jsonx.Obj(
		"method", m.Method,
		"pack_type", orNil(m.PackType),
		"pack_started_on", date(m.PackStartedOn),
		"packs_left", packsLeft,
		"reminder", reminder,
		"inserted_on", date(m.InsertedOn),
		"iud_lifetime_years", orNil(m.IUDLifetimeYears),
		"followup_on", date(m.FollowupOn()),
		"followup_done", m.FollowupDone,
		"iud_replace_on", date(m.IUDReplaceOn()),
		"injected_on", date(m.InjectedOn),
		"next_injection_on", date(m.NextInjectionOn()),
		"replace_on", date(m.ReplaceOn),
	)
}

// PillJSON is the pack screen (nbl_Contra_Pill) as of today.
func PillJSON(m Method, p Pack, today civildate.Date, logs Logs) *jsonx.OrderedMap {
	idx, n := p.Position(m.PackStartedOn, today)
	days := p.Days(m.PackStartedOn, today, logs)
	grid := make([]*jsonx.OrderedMap, len(days))
	for i, d := range days {
		grid[i] = jsonx.Obj("day", d.N, "date", d.Date.String(), "kind", d.Kind, "status", orNil(d.Status))
	}
	refill := m.Refill(today)
	var packsLeft, runsOut, refillOn any
	if refill.Known {
		packsLeft, runsOut, refillOn = refill.PacksLeft, refill.RunsOutOn.String(), refill.RefillOn.String()
	}
	return jsonx.Obj(
		"pack_number", idx+1,
		"pack_day", n,
		"pack_week", (n-1)/7+1,
		"pack_length", p.Length,
		"active_days", p.Active,
		"today", jsonx.Obj(
			"date", today.String(),
			"kind", p.KindOf(n),
			"status", orNil(p.StatusOn(m.PackStartedOn, today, today, logs)),
		),
		"days", jsonx.List(grid),
		"streak_days", p.Streak(m.PackStartedOn, today, logs),
		"missed_count", p.MissedCount(m.PackStartedOn, today, logs),
		"next_pack_on", p.PackStart(m.PackStartedOn, today).AddDays(p.Length).String(),
		"packs_left", packsLeft,
		"runs_out_on", runsOut,
		"refill_on", refillOn,
	)
}

// ReminderJSON is a care reminder the method created, as stored now (title, date and switch as the user left them).
func ReminderJSON(r LinkedReminder) *jsonx.OrderedMap {
	row := r.Row
	var due, at any
	switch {
	case row.ScheduledAt.Valid:
		t := row.ScheduledAt.Time.In(civildate.Tehran)
		due = civildate.FromTime(t).String()
		at = t.Format("2006-01-02 15:04:05")
	case row.StartsOn.Valid:
		due = row.StartsOn.Date.String()
	}
	return jsonx.Obj(
		"kind", r.Kind,
		"reminder_id", row.ID,
		"type", row.Type,
		"title", row.Title,
		"due_on", due,
		"scheduled_at", at,
		"recurrence", row.Recurrence,
		"is_active", row.IsActive,
	)
}
