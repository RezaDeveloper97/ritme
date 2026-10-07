package telemed

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// View renders directory JSON for one request: translations picked by Loc, catalog labels, public photo URLs.
type View struct {
	Loc    catalog.Localizer
	Labels Labels
	AppURL string
}

// Initial is the first letter of a name (the avatar letter of the artboards), or nil.
func Initial(name string) any {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	r, _ := utf8.DecodeRuneInString(name)
	return string(r)
}

// Rating is the average of the visible ratings with one decimal, rounded half up (nil without reviews).
func Rating(sum, count uint32) any {
	if count == 0 {
		return nil
	}
	tenths := (uint64(sum)*20 + uint64(count)) / (2 * uint64(count))
	return float64(tenths) / 10
}

// Satisfaction is the percent of visible ratings ≥ PositiveRating, rounded half up (nil without reviews).
func Satisfaction(positive, count uint32) any {
	if count == 0 {
		return nil
	}
	return (uint64(positive)*200 + uint64(count)) / (2 * uint64(count))
}

func (v View) text(raw json.RawMessage) any { return v.Loc.Text(raw) }

func (v View) nullText(col db.NullRawJSON) any {
	if !col.Valid {
		return nil
	}
	return v.Loc.Text(col.V)
}

// code is a catalog code with its picked label ({code, title}; title null when the catalog has no active item).
func (v View) code(group, code string) *jsonx.OrderedMap {
	var title any
	if raw := v.Labels.Title(group, code); raw != nil {
		title = v.Loc.Text(raw)
	}
	return jsonx.Obj("code", code, "title", title)
}

func (v View) nullCode(group string, code sql.NullString) any {
	if !code.Valid || code.String == "" {
		return nil
	}
	return v.code(group, code.String)
}

func (v View) photo(p sql.NullString) any {
	if !p.Valid || p.String == "" {
		return nil
	}
	return content.PublicURL(v.AppURL, p.String)
}

func nullInt16(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return n.Int16
}

// base is the part of a doctor shared by the card and the profile.
func (v View) base(d store.TelemedDoctor) *jsonx.OrderedMap {
	name := v.text(d.Name)
	nameStr, _ := name.(string)
	return jsonx.Obj(
		"id", d.ID,
		"kind", d.Kind,
		"name", name,
		"initial", Initial(nameStr),
		"headline", v.nullText(d.Headline),
		"specialty", v.code(GroupSpecialties, d.Specialty),
		"city", v.nullCode(GroupCities, d.City),
		"experience_years", nullInt16(d.ExperienceYears),
		"photo_url", v.photo(d.PhotoPath),
		"rating", Rating(d.RatingSum, d.RatingCount),
		"reviews_count", d.RatingCount,
	)
}

// NextSlot is {starts_at, mode} or nil.
func NextSlot(s Slot, mode string, ok bool) any {
	if !ok {
		return nil
	}
	return jsonx.Obj("starts_at", jsonx.ISO8601(s.Start), "date", civildate.InTehran(s.Start).String(),
		"time", s.Start.In(civildate.Tehran).Format("15:04"), "mode", mode)
}

// CardJSON is one directory entry (nbl_v17_Doctors card): offered modes, the lowest price of the considered modes
// (mode = the filter, "" = all) and the first free slot.
func (v View) CardJSON(c Card, mode string) *jsonx.OrderedMap {
	o := v.base(c.Doctor)
	modes := make([]string, 0, len(c.Visits))
	var from uint64
	for _, vt := range c.Visits {
		modes = append(modes, vt.Mode)
		if (mode == "" || vt.Mode == mode) && (from == 0 || vt.PriceRials < from) {
			from = vt.PriceRials
		}
	}
	o.Set("modes", jsonx.List(modes))
	o.Set("price_from_rials", from)
	o.Set("next_slot", NextSlot(c.Next, c.NextMode, c.HasNext))
	return o
}

// VisitTypeJSON is one offered visit type.
func (v View) VisitTypeJSON(vt store.TelemedVisitType) *jsonx.OrderedMap {
	return jsonx.Obj(
		"mode", vt.Mode,
		"duration_minutes", vt.DurationMinutes,
		"price_rials", vt.PriceRials,
		"note", v.nullText(vt.Note),
		"address", v.nullText(vt.Address),
	)
}

// ReviewJSON is one review: the reviewer appears as the first letter of their name only.
func ReviewJSON(r Review, viewerID uint64) *jsonx.OrderedMap {
	var body any
	if r.Body != "" {
		body = r.Body
	}
	var created any
	if !r.CreatedAt.IsZero() {
		created = jsonx.ISO8601(r.CreatedAt)
	}
	return jsonx.Obj(
		"id", r.ID,
		"rating", r.Rating,
		"body", body,
		"author_initial", Initial(r.UserName),
		"created_at", created,
		"mine", r.UserID == viewerID,
	)
}

// ProfileJSON is GET /telemed/doctors/{id} (nbl_v17_DoctorProfile).
func (v View) ProfileJSON(c Card, reviews []Review, viewerID uint64, canReview bool) *jsonx.OrderedMap {
	d := c.Doctor
	o := v.base(d)
	o.Set("licence_no", d.LicenceNo)
	o.Set("bio", v.nullText(d.Bio))
	o.Set("stats", jsonx.Obj(
		"experience_years", nullInt16(d.ExperienceYears),
		"visits_count", d.VisitsCount,
		"response_minutes", nullInt16(d.ResponseMinutes),
		"satisfaction_percent", Satisfaction(d.PositiveCount, d.RatingCount),
	))
	visits := make([]*jsonx.OrderedMap, 0, len(c.Visits))
	for _, vt := range c.Visits {
		visits = append(visits, v.VisitTypeJSON(vt))
	}
	o.Set("visit_types", jsonx.List(visits))
	insurers := make([]*jsonx.OrderedMap, 0, len(c.Insurers))
	for _, code := range c.Insurers {
		insurers = append(insurers, v.code(GroupInsurers, code))
	}
	o.Set("insurers", jsonx.List(insurers))
	o.Set("next_slot", NextSlot(c.Next, c.NextMode, c.HasNext))
	items := make([]*jsonx.OrderedMap, 0, len(reviews))
	for _, r := range reviews {
		items = append(items, ReviewJSON(r, viewerID))
	}
	o.Set("reviews", jsonx.Obj("count", d.RatingCount, "items", jsonx.List(items)))
	o.Set("can_review", canReview)
	return o
}

// SlotJSON is one free slot.
func SlotJSON(s Slot) *jsonx.OrderedMap {
	return jsonx.Obj("starts_at", jsonx.ISO8601(s.Start), "ends_at", jsonx.ISO8601(s.End),
		"time", s.Start.In(civildate.Tehran).Format("15:04"))
}

// SlotsJSON is GET /telemed/doctors/{id}/slots: the visit type, one entry per day (weekday: 0 = Sunday … 6 =
// Saturday) and the first free slot from the window start on.
func (v View) SlotsJSON(res SlotsResult, in SlotsInput) *jsonx.OrderedMap {
	days := make([]*jsonx.OrderedMap, 0, len(res.Days))
	for _, d := range res.Days {
		slots := make([]*jsonx.OrderedMap, 0, len(d.Slots))
		for _, s := range d.Slots {
			slots = append(slots, SlotJSON(s))
		}
		days = append(days, jsonx.Obj("date", d.Date.String(), "weekday", int(d.Date.Weekday()),
			"count", len(slots), "slots", jsonx.List(slots)))
	}
	return jsonx.Obj(
		"visit_type", v.VisitTypeJSON(res.Visit),
		"from", in.From.String(),
		"days", jsonx.List(days),
		"next_slot", NextSlot(res.Next, res.Visit.Mode, res.HasNext),
	)
}

// Minutes formats minutes after midnight as HH:MM.
func Minutes(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

// sortedKeys are the keys of m in byte order.
func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
