package pelvic

import (
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// OverviewJSON is GET /pelvic's data: {program, diary_today}.
func OverviewJSON(o Overview, loc catalog.Localizer) *jsonx.OrderedMap {
	var program any
	if p := o.Program; p != nil {
		days := make([]*jsonx.OrderedMap, len(p.WeekDays))
		for i, d := range p.WeekDays {
			days[i] = jsonx.Obj("date", d.Date, "done", d.Done)
		}
		var level any
		if p.Level != nil {
			level = LevelJSON(*p.Level, loc)
		}
		program = jsonx.Obj(
			"started_on", p.StartedOn,
			"week", p.Week,
			"weeks_total", WeeksTotal,
			"completed", p.Completed,
			"level", level,
			"streak_days", p.StreakDays,
			"today_done", p.TodayDone,
			"week_days", jsonx.List(days),
		)
	}
	return jsonx.Obj("program", program, "diary_today", DiaryJSON(o.Diary))
}

// LevelJSON is a level with its catalog copy picked for the request locale.
func LevelJSON(l Level, loc catalog.Localizer) *jsonx.OrderedMap {
	return jsonx.Obj(
		"code", l.Item.Code,
		"number", l.Number,
		"title", loc.Text(l.Item.Title),
		"body", loc.Text(l.Item.Body),
		"hold_sec", l.HoldSec,
		"rest_sec", l.RestSec,
		"reps", l.Reps,
		"sets", l.Sets,
		"session_sec", l.SessionSec(),
		"needs_review", l.Item.NeedsReview,
	)
}

// DiaryJSON is one bladder-diary day. uti_alert = any UTI symptom logged (the client shows the pelvic_alerts
// `uti_warning` catalog item).
func DiaryJSON(d Diary) *jsonx.OrderedMap {
	var leak, voids any
	if d.Leak != nil {
		leak = *d.Leak
	}
	if d.NightVoids != nil {
		voids = *d.NightVoids
	}
	return jsonx.Obj(
		"date", d.Date,
		"leak", leak,
		"night_voids", voids,
		"uti_symptoms", jsonx.List(d.UTISymptoms),
		"uti_alert", len(d.UTISymptoms) > 0,
	)
}
