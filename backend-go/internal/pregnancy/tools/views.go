package tools

import (
	"database/sql"
	"embed"
	"io/fs"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// The controller messages are data: lang/<code>/pregnancy_tools.json (English fallback). The 5-1-1 copy (call /
// hospital) is the admin-editable pregnancy_alert row; the screens' copy is the frontend's.
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the pregnancy_tools line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("pregnancy_tools."+key, nil, locale) }

func iso(t time.Time) any { return jsonx.ISO8601(t.In(civildate.Tehran)) }

func isoNull(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return iso(t.Time)
}

func seconds(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return int64(d / time.Second)
}

func secondsOrNil(d time.Duration, ok bool) any {
	if !ok {
		return nil
	}
	return seconds(d)
}

// KickJSON is one kick-count session at now.
func KickJSON(k store.PregnancyKickSession, now time.Time) *jsonx.OrderedMap {
	end := now
	if k.EndedAt.Valid {
		end = k.EndedAt.Time
	}
	elapsed := end.Sub(k.StartedAt)
	var toTarget any
	if k.TenthKickAt.Valid {
		toTarget = seconds(k.TenthKickAt.Time.Sub(k.StartedAt))
	}
	var week any
	if k.PregnancyWeek.Valid {
		week = int(k.PregnancyWeek.Int16)
	}
	reached := int(k.Kicks) >= KickTarget
	return jsonx.Obj(
		"id", k.ID,
		"started_at", iso(k.StartedAt),
		"ended_at", isoNull(k.EndedAt),
		"is_active", k.ActiveLock.Valid,
		"kicks", int(k.Kicks),
		"target", KickTarget,
		"window_minutes", KickWindowMinutes,
		"elapsed_seconds", seconds(elapsed),
		"time_to_target_seconds", toTarget,
		"reached_target", reached,
		// Fewer than the target once the window has passed: the «با پزشک یا ماما تماس بگیر» guidance.
		"low_count", !reached && elapsed >= KickWindowMinutes*time.Minute,
		"last_kick_at", isoNull(k.LastKickAt),
		"pregnancy_week", week,
	)
}

// KickOverviewJSON is GET /pregnancy/kick-sessions.
func KickOverviewJSON(o KickOverview, now time.Time) *jsonx.OrderedMap {
	var active any
	if o.Active != nil {
		active = KickJSON(*o.Active, now)
	}
	history := make([]any, 0, len(o.History))
	for _, k := range o.History {
		history = append(history, KickJSON(k, now))
	}
	return jsonx.Obj(
		"target", KickTarget,
		"window_minutes", KickWindowMinutes,
		"active", active,
		"today", jsonx.Obj("date", civildate.InTehran(now).String(), "kicks", o.Today, "sessions", o.Count),
		"history", history,
	)
}

// ParamsJSON are the live 5-1-1 thresholds.
func ParamsJSON(p labor.Params) *jsonx.OrderedMap {
	return jsonx.Obj(
		"interval_max_minutes", int(p.IntervalMax/time.Minute),
		"duration_min_seconds", int(p.DurationMin/time.Second),
		"run_minutes", int(p.Run/time.Minute),
	)
}

// TimingJSON is a contraction session at now; withList adds every contraction (newest first).
func TimingJSON(t Timing, p labor.Params, now time.Time, withList bool) *jsonx.OrderedMap {
	cs := t.Labor()
	var running any
	done := 0
	for _, c := range t.Contractions {
		if !c.EndedAt.Valid {
			running = jsonx.Obj("id", c.ID, "started_at", iso(c.StartedAt), "elapsed_seconds", seconds(now.Sub(c.StartedAt)))
			continue
		}
		done++
	}
	recent := labor.Recent(cs, p.Run)
	r := labor.FiveOneOne(cs, p)
	end := now
	if t.Session.EndedAt.Valid {
		end = t.Session.EndedAt.Time
	}
	out := jsonx.Obj(
		"id", t.Session.ID,
		"started_at", iso(t.Session.StartedAt),
		"ended_at", isoNull(t.Session.EndedAt),
		"is_active", t.Session.ActiveLock.Valid,
		"elapsed_seconds", seconds(end.Sub(t.Session.StartedAt)),
		"count", done,
		"running", running,
		// The «میانگین مدت / میانگین فاصله» of the last run_minutes.
		"avg_duration_seconds", secondsOrNil(recent.AvgDuration, recent.Count > 0),
		"avg_interval_seconds", secondsOrNil(recent.AvgInterval, recent.Count > 1),
		"five_one_one", jsonx.Obj(
			"met", r.Met,
			"count", r.Count,
			"span_minutes", int(r.Span/time.Minute),
			"avg_interval_seconds", secondsOrNil(r.AvgInterval, r.Count > 1),
			"avg_duration_seconds", secondsOrNil(r.AvgDuration, r.Count > 0),
		),
		"alert_at", isoNull(t.Session.AlertAt),
	)
	if withList {
		list := make([]any, 0, len(t.Contractions))
		gaps := labor.Intervals(cs)
		for i := len(t.Contractions) - 1; i >= 0; i-- {
			c := t.Contractions[i]
			var dur, gap any
			if c.EndedAt.Valid {
				dur = seconds(c.EndedAt.Time.Sub(c.StartedAt))
			}
			if i > 0 {
				gap = seconds(gaps[i])
			}
			list = append(list, jsonx.Obj(
				"id", c.ID, "started_at", iso(c.StartedAt), "ended_at", isoNull(c.EndedAt),
				"duration_seconds", dur, "interval_seconds", gap,
			))
		}
		out.Set("contractions", list)
	}
	return out
}

// TimingOverviewJSON is GET /pregnancy/contractions.
func TimingOverviewJSON(active *Timing, history []Timing, p labor.Params, now time.Time) *jsonx.OrderedMap {
	var a any
	if active != nil {
		a = TimingJSON(*active, p, now, true)
	}
	list := make([]any, 0, len(history))
	for _, t := range history {
		list = append(list, TimingJSON(t, p, now, false))
	}
	return jsonx.Obj("params", ParamsJSON(p), "active", a, "history", list)
}
