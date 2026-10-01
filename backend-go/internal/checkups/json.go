package checkups

import (
	"encoding/json"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Lang is the request language and the default language (the fallback of bilingual columns).
type Lang struct {
	Locale, Default string
}

func (l Lang) pick(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	s := i18n.PickString(raw, l.Locale, l.Default)
	if s == "" {
		return nil
	}
	return s
}

func (l Lang) pickNull(raw rootdb.NullRawJSON) any {
	if !raw.Valid {
		return nil
	}
	return l.pick(raw.V)
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

func emptyToNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ItemJSON is one plan row (GET /checkups items, /checkups/home highlights).
func (p *Plan) ItemJSON(it engine.Item, l Lang) *jsonx.OrderedMap {
	row, t := p.Rows[it.TypeID], p.Types[it.TypeID]
	return jsonx.Obj(
		"id", it.TypeID,
		"key", emptyToNil(t.Key),
		"title", l.pick(row.Title),
		"subtitle", l.pickNull(row.Subtitle),
		"category", string(it.Category),
		"section", string(it.Section),
		"status", string(it.Status),
		"icon", emptyToNil(row.Icon.String),
		"tone", row.Tone,
		"interval_label", IntervalLabel(t.IntervalMonths, t.IntervalMonthsMax, l.Locale),
		"timing_label", emptyToNil(p.TimingLabel(t, l.Locale)),
		"last_done_on", dateOrNil(it.LastDoneOn),
		"next_due_on", dateOrNil(it.NextDueOn),
		"next_due_label", NextDueLabel(it, t, p.Today, l.Locale),
		"is_custom", it.IsCustom,
	)
}

// DetailJSON is GET /checkups/{id}: the item plus the catalog copy, the latest records and the
// user's switches.
func (p *Plan) DetailJSON(it engine.Item, records []store.CheckupRecord, l Lang) *jsonx.OrderedMap {
	row, t := p.Rows[it.TypeID], p.Types[it.TypeID]
	out := p.ItemJSON(it, l)
	title := l.pick(row.Title)
	recs := make([]*jsonx.OrderedMap, 0, len(records))
	for _, r := range records {
		recs = append(recs, RecordJSON(r, title))
	}
	intOrNil := func(n int) any {
		if n <= 0 {
			return nil
		}
		return n
	}
	out.Set("why", l.pickNull(row.Why))
	out.Set("performed_by", row.PerformedBy)
	out.Set("interval_months", t.IntervalMonths)
	out.Set("interval_months_max", intOrNil(t.IntervalMonthsMax))
	out.Set("age_min", intOrNil(t.AgeMin))
	out.Set("age_max", intOrNil(t.AgeMax))
	cycleFrom, cycleTo := t.CycleDayFrom, t.CycleDayTo
	if !p.LifeMode.TracksCycle() { // B-N2-11b: no «روز ۷ تا ۱۰ سیکل» hint without a cycle
		cycleFrom, cycleTo = 0, 0
	}
	out.Set("cycle_day_from", intOrNil(cycleFrom))
	out.Set("cycle_day_to", intOrNil(cycleTo))
	out.Set("remind_lead_days", t.RemindLeadDays)
	out.Set("prep_steps", prepSteps(row.PrepSteps, l))
	out.Set("guide_steps", guideSteps(row.GuideSteps, l))
	out.Set("finding_options", findingOptionsJSON(row.FindingOptions, l))
	out.Set("records", recs)
	out.Set("settings", jsonx.Obj("enabled", it.Enabled, "remind", it.Remind))
	return out
}

func prepSteps(raw rootdb.NullRawJSON, l Lang) []string {
	out := []string{}
	if !raw.Valid {
		return out
	}
	var steps []json.RawMessage
	if json.Unmarshal(raw.V, &steps) != nil {
		return out
	}
	for _, s := range steps {
		if v, ok := l.pick(s).(string); ok {
			out = append(out, v)
		}
	}
	return out
}

func guideSteps(raw rootdb.NullRawJSON, l Lang) []*jsonx.OrderedMap {
	out := []*jsonx.OrderedMap{}
	if !raw.Valid {
		return out
	}
	var steps []struct {
		Title json.RawMessage `json:"title"`
		Body  json.RawMessage `json:"body"`
	}
	if json.Unmarshal(raw.V, &steps) != nil {
		return out
	}
	for _, s := range steps {
		title := l.pick(s.Title)
		if title == nil {
			continue
		}
		out = append(out, jsonx.Obj("title", title, "body", l.pick(s.Body)))
	}
	return out
}

// findingOption is one finding chip of a type (finding_options JSON).
type findingOption struct {
	Key       string          `json:"key"`
	Exclusive bool            `json:"exclusive"`
	Label     json.RawMessage `json:"label"`
}

func findingOptions(raw rootdb.NullRawJSON) []findingOption {
	if !raw.Valid {
		return nil
	}
	var opts []findingOption
	if json.Unmarshal(raw.V, &opts) != nil {
		return nil
	}
	out := opts[:0]
	for _, o := range opts {
		if o.Key != "" {
			out = append(out, o)
		}
	}
	return out
}

// findingKeys are the valid `findings` values of a type.
func findingKeys(raw rootdb.NullRawJSON) []string {
	keys := []string{}
	for _, o := range findingOptions(raw) {
		keys = append(keys, o.Key)
	}
	return keys
}

func findingOptionsJSON(raw rootdb.NullRawJSON, l Lang) []*jsonx.OrderedMap {
	out := []*jsonx.OrderedMap{}
	for _, o := range findingOptions(raw) {
		label := l.pick(o.Label)
		if label == nil {
			label = o.Key
		}
		out = append(out, jsonx.Obj("key", o.Key, "label", label, "exclusive", o.Exclusive))
	}
	return out
}

// RecordJSON is one checkup record; title is the checkup's localized title.
func RecordJSON(r store.CheckupRecord, title any) *jsonx.OrderedMap {
	findings := []string{}
	if r.Findings.Valid {
		_ = json.Unmarshal(r.Findings.V, &findings)
		if findings == nil {
			findings = []string{}
		}
	}
	var note any
	if r.Note.Valid {
		note = r.Note.String
	}
	return jsonx.Obj(
		"id", r.ID,
		"checkup_type_id", r.CheckupTypeID,
		"checkup_title", title,
		"done_on", r.DoneOn.String(),
		"result", r.Result,
		"findings", findings,
		"note", note,
		"has_attachment", r.HasAttachment,
		"next_due_on", nullDate(r.NextDueOn),
	)
}

// historyJSON is a History timeline row (RecordJSON plus the checkup's key, icon and tone).
func historyJSON(r store.ListCheckupRecordHistoryRow, l Lang) *jsonx.OrderedMap {
	out := RecordJSON(r.CheckupRecord, l.pick(r.CheckupTitle))
	out.Set("checkup_key", emptyToNil(r.CheckupKey.String))
	out.Set("checkup_icon", emptyToNil(r.CheckupIcon.String))
	out.Set("checkup_tone", r.CheckupTone)
	return out
}

// SummaryJSON is the ring and counts line.
func SummaryJSON(s engine.Summary) *jsonx.OrderedMap {
	return jsonx.Obj("total", s.Total, "up_to_date", s.UpToDate, "due", s.Due, "overdue", s.Overdue)
}
