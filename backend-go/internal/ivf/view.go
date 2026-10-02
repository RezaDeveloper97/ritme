package ivf

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// wallClock is the Tehran wall-clock datetime format of the responses (like /care/appointments).
const wallClock = "2006-01-02 15:04:05"

func dateJSON(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func timeJSON(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.In(civildate.Tehran).Format(wallClock)
}

func strJSON(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func intJSON(n int, ok bool) any {
	if !ok {
		return nil
	}
	return n
}

// CycleJSON is a cycle with its timeline as of today.
func CycleJSON(c *Cycle, today civildate.Date) any {
	if c == nil {
		return nil
	}
	steps := c.Timeline()
	timeline := make([]*jsonx.OrderedMap, len(steps))
	for i, s := range steps {
		timeline[i] = jsonx.Obj("stage", s.Stage, "status", s.Status, "date", dateJSON(s.Date))
	}
	status := "closed"
	if c.Open {
		status = "open"
	}
	stageDay, sdOK := c.StageDay(today)
	toBeta, tbOK := c.DaysToBeta(today)
	if !c.Open {
		sdOK, tbOK = false, false
	}
	return jsonx.Obj(
		"id", c.ID,
		"number", c.Number,
		"protocol", strJSON(c.Protocol),
		"stage", c.Stage,
		"status", status,
		"started_on", dateJSON(c.StartedOn),
		"stim_started_on", dateJSON(c.StimStartedOn),
		"retrieval_at", timeJSON(c.RetrievalAt),
		"transfer_at", timeJSON(c.TransferAt),
		"beta_on", dateJSON(c.BetaOn),
		"next_scan_at", timeJSON(c.NextScanAt),
		"notify_companion", c.NotifyCompanion,
		"outcome", strJSON(c.Outcome),
		"outcome_on", dateJSON(c.OutcomeOn),
		"stage_day", intJSON(stageDay, sdOK),
		"days_to_beta", intJSON(toBeta, tbOK),
		"timeline", timeline,
	)
}

// DoseJSON is one scheduled dose.
func DoseJSON(d Dose) *jsonx.OrderedMap {
	m := d.Med
	return jsonx.Obj(
		"med_id", m.ID,
		"reminder_id", m.ReminderID(),
		"name", m.Name(),
		"dose", m.Care.Meta.Dose,
		"unit", m.Care.Meta.Unit,
		"route", m.Route,
		"role", m.Role,
		"is_trigger", m.IsTrigger(),
		"date", d.Date.String(),
		"slot", d.Slot,
		"taken", d.Taken,
		"site", strJSON(d.Site),
	)
}

func dosesJSON(day civildate.Date, doses []Dose) *jsonx.OrderedMap {
	list := make([]*jsonx.OrderedMap, len(doses))
	for i, d := range doses {
		list[i] = DoseJSON(d)
	}
	return jsonx.Obj("date", day.String(), "doses", list)
}

// MedJSON is a medicine with its inventory (null when not tracked).
func MedJSON(l MedLine) *jsonx.OrderedMap {
	m := l.Med
	r := m.Care.Row
	var inventory any
	if s := m.Stock; s != nil {
		inv := l.Inventory
		inventory = jsonx.Obj(
			"stock_units", s.Units,
			"stock_unit", s.Unit,
			"doses_per_unit", s.DosesPerUnit,
			"counted_at", timeJSON(s.CountedAt),
			"doses_left", inv.DosesLeft,
			"units_left", inv.UnitsLeft,
			"days_left", intJSON(inv.DaysLeft, inv.Scheduled),
			"runs_out_on", dateJSON(inv.RunsOutOn),
			"low", inv.Low,
		)
	}
	var notes any
	if r.Notes.Valid {
		notes = r.Notes.String
	}
	return jsonx.Obj(
		"id", m.ID,
		"reminder_id", m.ReminderID(),
		"name", m.Name(),
		"role", m.Role,
		"route", m.Route,
		"is_trigger", m.IsTrigger(),
		"trigger_at", timeJSON(m.TriggerAt),
		"dose", m.Care.Meta.Dose,
		"unit", m.Care.Meta.Unit,
		"times", m.Care.Meta.Times,
		"starts_on", dateJSON(nullDate(r.StartsOn)),
		"ends_on", dateJSON(nullDate(r.EndsOn)),
		"is_active", r.IsActive,
		"notes", notes,
		"inventory", inventory,
	)
}

// HomeJSON is GET /ivf's data.
func HomeJSON(v HomeView) *jsonx.OrderedMap {
	var next, scan any
	if a := v.NextAppointment; a != nil {
		next = jsonx.Obj("kind", a.Kind, "appointment", a.Appt.JSON(v.Now))
	}
	if v.LatestScan != nil {
		scan = ScanJSON(*v.LatestScan, v.Cycle)
	}
	notify := v.Cycle != nil && v.Cycle.NotifyCompanion && v.Companion.Linked
	return jsonx.Obj(
		"enabled", v.Enabled,
		"cycle", CycleJSON(v.Cycle, v.Today),
		"cycles_count", v.CyclesCount,
		"today", dosesJSON(v.Today, v.Doses),
		"next_appointment", next,
		"latest_scan", scan,
		"companion", jsonx.Obj(
			"linked", v.Companion.Linked,
			"notify", notify,
			"shares_meds", v.Companion.Meds,
			"shares_appointments", v.Companion.Appointments,
		),
	)
}

// MedsJSON is GET /ivf/meds's data.
func MedsJSON(v MedsView) *jsonx.OrderedMap {
	var trigger, last any
	if t := v.Trigger; t != nil {
		trigger = jsonx.Obj(
			"med_id", t.Med.ID,
			"name", t.Med.Name(),
			"dose", t.Med.Care.Meta.Dose,
			"unit", t.Med.Care.Meta.Unit,
			"trigger_at", timeJSON(t.Med.TriggerAt),
			"taken", t.Taken,
		)
	}
	if l := v.LastSite; l != nil {
		last = jsonx.Obj("site", l.Site, "date", l.Date.String(), "slot", l.Slot)
	}
	meds := make([]*jsonx.OrderedMap, len(v.Meds))
	for i, l := range v.Meds {
		meds[i] = MedJSON(l)
	}
	return jsonx.Obj(
		"cycle", CycleJSON(v.Cycle, v.Today),
		"trigger", trigger,
		"today", dosesJSON(v.Today, v.TodayDose),
		"tomorrow", dosesJSON(v.Today.AddDays(1), v.Tomorrow),
		"sites", jsonx.Obj("codes", v.Sites, "last", last, "suggested", strJSON(v.NextSite)),
		"meds", meds,
	)
}

func ovaryJSON(o Ovary) *jsonx.OrderedMap {
	return jsonx.Obj(Bins[0], o[0], Bins[1], o[1], Bins[2], o[2], Bins[3], o[3], "total", o.Total())
}

// ScanJSON is one scan; stim_day is its stimulation day in cycle c.
func ScanJSON(s Scan, c *Cycle) *jsonx.OrderedMap {
	var stimDay any
	if c != nil {
		stimDay = intJSON(c.StimDay(s.Date))
	}
	return jsonx.Obj(
		"date", s.Date.String(),
		"stim_day", stimDay,
		"right", ovaryJSON(s.Right),
		"left", ovaryJSON(s.Left),
		"total", ovaryJSON(s.Right.Plus(s.Left)),
		"endometrium_mm", strJSON(s.EndometriumMM),
		"e2", strJSON(s.E2),
		"e2_unit", strJSON(s.E2Unit),
		"notes", strJSON(s.Notes),
	)
}

// ScansJSON is GET /ivf/scans's data: the scans (oldest first) and the growth chart's two series.
func ScansJSON(v ScansView) *jsonx.OrderedMap {
	scans := make([]*jsonx.OrderedMap, len(v.Scans))
	growth := make([]*jsonx.OrderedMap, len(v.Scans))
	for i, s := range v.Scans {
		scans[i] = ScanJSON(s, v.Cycle)
		g := s.GrowthOf()
		var stimDay any
		if v.Cycle != nil {
			stimDay = intJSON(v.Cycle.StimDay(s.Date))
		}
		growth[i] = jsonx.Obj("date", g.Date.String(), "stim_day", stimDay, "follicles_10_14", g.Mid,
			"follicles_15_plus", g.Mature)
	}
	return jsonx.Obj("cycle", CycleJSON(v.Cycle, v.Today), "scans", scans, "growth", growth)
}

// TWWJSON is GET /ivf/tww's data.
func TWWJSON(v TWWView) *jsonx.OrderedMap {
	var transferOn, betaOn, since, toBeta, todayMood any
	if c := v.Cycle; c != nil {
		transferOn, betaOn = dateJSON(c.TransferOn()), dateJSON(c.BetaOn)
		since = intJSON(c.DaysSinceTransfer(v.Today))
		toBeta = intJSON(c.DaysToBeta(v.Today))
	}
	moods := make([]*jsonx.OrderedMap, len(v.Moods))
	for i, m := range v.Moods {
		moods[i] = jsonx.Obj("date", m.Date.String(), "mood", m.Mood)
		if m.Date == v.Today {
			todayMood = m.Mood
		}
	}
	type group struct {
		dose  Dose
		slots []*jsonx.OrderedMap
		taken int
	}
	var order []uint64
	groups := map[uint64]*group{}
	for _, d := range v.Luteal {
		g, ok := groups[d.Med.ID]
		if !ok {
			g = &group{dose: d}
			groups[d.Med.ID] = g
			order = append(order, d.Med.ID)
		}
		g.slots = append(g.slots, jsonx.Obj("slot", d.Slot, "taken", d.Taken))
		if d.Taken {
			g.taken++
		}
	}
	luteal := make([]*jsonx.OrderedMap, 0, len(order))
	for _, id := range order {
		g := groups[id]
		m := g.dose.Med
		luteal = append(luteal, jsonx.Obj(
			"med_id", m.ID,
			"name", m.Name(),
			"dose", m.Care.Meta.Dose,
			"unit", m.Care.Meta.Unit,
			"route", m.Route,
			"doses", g.slots,
			"taken", g.taken,
			"total", len(g.slots),
		))
	}
	return jsonx.Obj(
		"cycle", CycleJSON(v.Cycle, v.Today),
		"transfer_on", transferOn,
		"beta_on", betaOn,
		"days_since_transfer", since,
		"days_to_beta", toBeta,
		"today", jsonx.Obj("date", v.Today.String(), "mood", todayMood),
		"moods", moods,
		"luteal_support", luteal,
	)
}

// OutcomeJSON is the closed cycle with the follow-ups to offer.
func OutcomeJSON(c Cycle, today civildate.Date) *jsonx.OrderedMap {
	return jsonx.Obj("cycle", CycleJSON(&c, today), "next_steps", NextSteps(c.Outcome))
}
