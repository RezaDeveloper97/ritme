package children

import (
	"database/sql"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Loc is a request's locale and default language (translatable catalog columns).
type Loc struct {
	Locale  string
	Default string
}

// text picks a translatable column (nil when empty).
func (l Loc) text(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	if s := i18n.PickString(raw, l.Locale, l.Default); s != "" {
		return s
	}
	return nil
}

// initial is the avatar letter («آ» for آوا).
func initial(name string) string {
	r, _ := utf8.DecodeRuneInString(strings.TrimSpace(name))
	if r == utf8.RuneError {
		return ""
	}
	return string(r)
}

func labelOf(prefix string, ns sql.NullString, locale string) any {
	if !ns.Valid {
		return nil
	}
	return T(prefix+ns.String, locale)
}

func decJSON(ns sql.NullString) any {
	if f, ok := parseDec(ns); ok {
		return f
	}
	return nil
}

// ChildJSON is the profile part every child payload starts with.
func ChildJSON(a Access, ownerName string, today civildate.Date, locale string) *jsonx.OrderedMap {
	c := a.Child
	age := AgeOn(c.BirthDate, today)
	var owner any
	if a.Role == RoleShared && ownerName != "" {
		owner = ownerName
	}
	return jsonx.Obj(
		"id", c.ID,
		"name", c.Name,
		"initial", initial(c.Name),
		"birth_date", c.BirthDate.String(),
		"sex", nullStr(c.Sex),
		"sex_label", labelOf("sexes.", c.Sex, locale),
		"delivery_type", nullStr(c.DeliveryType),
		"delivery_type_label", labelOf("delivery_types.", c.DeliveryType, locale),
		"birth", jsonx.Obj(
			"weight_kg", decJSON(c.BirthWeightKg),
			"length_cm", decJSON(c.BirthLengthCm),
			"head_cm", decJSON(c.BirthHeadCm),
		),
		"age", jsonx.Obj(
			"days", age.Days, "weeks", age.Weeks, "months", age.Months, "years", age.Years,
			"days_in_month", age.DaysInMonth, "label", age.Label(locale),
		),
		"role", a.Role,
		"can_edit", a.CanEdit(),
		"owner_name", owner,
	)
}

// VisitJSON is a visit card (summary form: no doses unless withDoses).
func VisitJSON(v Visit, today civildate.Date, loc Loc, withDoses bool) *jsonx.OrderedMap {
	out := jsonx.Obj(
		"code", v.Code,
		"age_months", v.AgeMonths,
		"label", VisitLabel(v.AgeMonths, loc.Locale),
		"due_date", v.DueDate.String(),
		"remind_on", v.RemindOn.String(),
		"days_left", today.DiffDays(v.DueDate),
		"status", v.Status,
		"status_label", T("vaccine_status."+v.Status, loc.Locale),
		"given", v.Given,
		"total", len(v.Doses),
	)
	names := make([]any, 0, len(v.Doses))
	for _, d := range v.Doses {
		names = append(names, loc.text(d.Title))
	}
	out.Set("dose_names", names)
	if withDoses {
		doses := make([]*jsonx.OrderedMap, 0, len(v.Doses))
		for _, d := range v.Doses {
			var givenOn, note any
			if d.Given != nil {
				givenOn = d.Given.GivenOn.String()
				note = nullStr(d.Given.Note)
			}
			st := v.DoseStatus(d)
			doses = append(doses, jsonx.Obj(
				"code", d.Code, "title", loc.text(d.Title), "protects_against", loc.text(d.Body),
				"status", st, "status_label", T("vaccine_status."+st, loc.Locale),
				"given_on", givenOn, "note", note, "needs_review", d.NeedsCheck,
			))
		}
		out.Set("doses", doses)
	}
	return out
}

// ScheduleSummaryJSON is {given, total, completed_visits, up_to_date, complete, next}.
func ScheduleSummaryJSON(s Schedule, today civildate.Date, loc Loc) *jsonx.OrderedMap {
	var next any
	if s.Next != nil {
		next = VisitJSON(*s.Next, today, loc, false)
	}
	return jsonx.Obj(
		"given", s.GivenDoses,
		"total", s.TotalDoses,
		"completed_visits", s.CompletedVisits,
		"up_to_date", s.UpToDate,
		"complete", s.Next == nil && s.TotalDoses > 0,
		"next", next,
	)
}

// MilestoneBand is one age band's milestones with the child's checks.
type MilestoneBand struct {
	Months     int
	Items      []Entry
	Checked    map[string]civildate.Date
	Activities []Entry
	Note       *Entry
}

// CheckedCount is how many of the band's items were seen.
func (b MilestoneBand) CheckedCount() int {
	n := 0
	for _, it := range b.Items {
		if _, ok := b.Checked[it.Code]; ok {
			n++
		}
	}
	return n
}

// MilestoneSummaryJSON is {band_months, label, checked, total}.
func MilestoneSummaryJSON(b MilestoneBand, locale string) any {
	if b.Months < 0 {
		return nil
	}
	return jsonx.Obj("band_months", b.Months, "label", VisitLabel(b.Months, locale), "checked", b.CheckedCount(), "total", len(b.Items))
}

// MilestoneBandJSON is the full band (GET /children/{id}/milestones).
func MilestoneBandJSON(b MilestoneBand, loc Loc) *jsonx.OrderedMap {
	items := make([]*jsonx.OrderedMap, 0, len(b.Items))
	for _, it := range b.Items {
		var on any
		d, ok := b.Checked[it.Code]
		if ok {
			on = d.String()
		}
		items = append(items, jsonx.Obj("code", it.Code, "title", loc.text(it.Title), "domain", it.Domain, "checked", ok, "checked_on", on))
	}
	acts := make([]*jsonx.OrderedMap, 0, len(b.Activities))
	for _, a := range b.Activities {
		acts = append(acts, jsonx.Obj("code", a.Code, "title", loc.text(a.Title), "body", loc.text(a.Body)))
	}
	var note any
	if b.Note != nil {
		note = loc.text(b.Note.Body)
	}
	return jsonx.Obj(
		"months", b.Months,
		"label", VisitLabel(b.Months, loc.Locale),
		"checked", b.CheckedCount(),
		"total", len(b.Items),
		"items", items,
		"activities", acts,
		"doctor_note", note,
	)
}

// LearnTip is a child_learn entry with its linked article (if published).
type LearnTip struct {
	Entry
	Article *store.ListLearnArticlesRow
}

// LearnTipJSON is one learn card.
func LearnTipJSON(t LearnTip, loc Loc) *jsonx.OrderedMap {
	var article any
	if t.Article != nil {
		article = jsonx.Obj("id", t.Article.ID, "slug", t.Article.Slug)
	}
	return jsonx.Obj(
		"code", t.Code,
		"topic", t.Topic,
		"topic_label", T("topics."+t.Topic, loc.Locale),
		"title", loc.text(t.Title),
		"body", loc.text(t.Body),
		"minutes", t.Minutes,
		"featured", t.Featured,
		"from_months", t.FromMonths,
		"to_months", t.ToMonths,
		"article", article,
	)
}

// TopicsJSON are the learn tabs (all first).
func TopicsJSON(locale string) []*jsonx.OrderedMap {
	out := []*jsonx.OrderedMap{jsonx.Obj("code", "all", "label", T("topics.all", locale))}
	for _, t := range Topics {
		out = append(out, jsonx.Obj("code", t, "label", T("topics."+t, locale)))
	}
	return out
}
