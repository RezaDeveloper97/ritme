package healthrecord

// Timeline and category counts of the record (canvas-build CB-REC-01, D-70; nbl_Rec_Timeline, nbl_Rec_Home): the
// user's documents and ready lab sheets (internal/labs, read-only) in one newest-first list grouped by Jalali month.
// Nothing else is merged in: no loss event and no care appointment (private loss follow-ups included) ever appears
// here, and every read is scoped by the user id.

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Timeline filters: all, lab, or one of DocumentKinds.
const FilterAll = "all"

// TimelineFilters are the accepted ?kind= values, in chip order.
var TimelineFilters = append([]string{FilterAll, KindLab}, DocumentKinds...)

// TimelineQuery is a validated GET /health-record/timeline.
type TimelineQuery struct {
	Kind                  string         // FilterAll by default
	Before                civildate.Date // exclusive upper bound (the previous page's next_before); zero = from the newest
	Limit                 int
	Locale, DefaultLocale string
}

// entry is one timeline row before rendering.
type entry struct {
	date   civildate.Date
	isLab  bool
	id     uint64
	render *jsonx.OrderedMap
}

func (s *Documents) labRows(ctx context.Context, userID uint64, locale, def string) ([]*jsonx.OrderedMap, error) {
	if s.labs == nil {
		return nil, nil
	}
	rows, err := s.labs.RecordLabs(ctx, userID, locale, def, timelineLabs)
	if err != nil {
		return nil, fmt.Errorf("healthrecord: timeline labs: %w", err)
	}
	return rows, nil
}

func linkBadges(links []store.ListUserRecordDocumentLinksRow) map[uint64][]*jsonx.OrderedMap {
	out := map[uint64][]*jsonx.OrderedMap{}
	for _, l := range links {
		out[l.DocumentID] = append(out[l.DocumentID], jsonx.Obj("type", l.TargetType, "state", l.State))
	}
	return out
}

func documentEntry(d store.ListRecordDocumentsRow, links []*jsonx.OrderedMap) entry {
	if links == nil {
		links = []*jsonx.OrderedMap{}
	}
	return entry{date: d.SortDate, id: d.ID, render: jsonx.Obj(
		"type", "document",
		"id", d.ID,
		"kind", d.Kind,
		"title", strVal(d.Title),
		"date", d.SortDate.String(),
		"date_known", d.DocumentDate.Valid,
		"ended_on", dateVal(d.EndedOn),
		"centre", strVal(d.Centre),
		"doctor", strVal(d.Doctor),
		"file_count", d.FileCount,
		"review_state", d.ReviewState,
		"lab", nil,
		"links", links,
	)}
}

func labEntry(row *jsonx.OrderedMap) (entry, bool) {
	get := func(k string) any { v, _ := row.Get(k); return v }
	ds, _ := get("date").(string)
	date, err := civildate.Parse(ds)
	if err != nil {
		return entry{}, false
	}
	id, _ := get("id").(uint64)
	return entry{date: date, isLab: true, id: id, render: jsonx.Obj(
		"type", "lab",
		"id", get("id"),
		"kind", KindLab,
		"title", get("title"),
		"date", ds,
		"date_known", true,
		"ended_on", nil,
		"centre", nil,
		"doctor", nil,
		"file_count", nil,
		"review_state", nil,
		"lab", jsonx.Obj("category", get("category"), "marker_count", get("marker_count"),
			"attention_count", get("attention_count"), "all_normal", get("all_normal")),
		"links", []*jsonx.OrderedMap{},
	)}, true
}

// Timeline is GET /health-record/timeline: up to Limit rows older than Before, newest first (documents before lab
// sheets on the same day), always whole days (a page never splits one day), grouped by Jalali month.
func (s *Documents) Timeline(ctx context.Context, userID uint64, tq TimelineQuery) (*jsonx.OrderedMap, error) {
	if tq.Kind == "" {
		tq.Kind = FilterAll
	}
	if tq.Limit <= 0 {
		tq.Limit = DefaultTimelineMax
	}
	var all []entry
	if tq.Kind != KindLab {
		docs, err := s.q.ListRecordDocuments(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("healthrecord: timeline documents: %w", err)
		}
		links, err := s.q.ListUserRecordDocumentLinks(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("healthrecord: timeline links: %w", err)
		}
		badges := linkBadges(links)
		for _, d := range docs {
			if tq.Kind == FilterAll || d.Kind == tq.Kind {
				all = append(all, documentEntry(d, badges[d.ID]))
			}
		}
	}
	if tq.Kind == FilterAll || tq.Kind == KindLab {
		rows, err := s.labRows(ctx, userID, tq.Locale, tq.DefaultLocale)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if e, ok := labEntry(r); ok {
				all = append(all, e)
			}
		}
	}
	all = slices.DeleteFunc(all, func(e entry) bool { return !tq.Before.IsZero() && !e.date.Before(tq.Before) })
	slices.SortStableFunc(all, func(a, b entry) int {
		if c := b.date.Compare(a.date); c != 0 {
			return c
		}
		if a.isLab != b.isLab {
			if a.isLab {
				return 1
			}
			return -1
		}
		return cmp.Compare(b.id, a.id)
	})
	page := all
	var next any
	if len(all) > tq.Limit {
		last := all[tq.Limit-1].date
		n := tq.Limit
		for n < len(all) && all[n].date == last {
			n++
		}
		page = all[:n]
		if n < len(all) {
			next = last.String()
		}
	}
	return jsonx.Obj("kind", tq.Kind, "months", months(page), "next_before", next), nil
}

// months groups newest-first rows by Jalali month.
func months(rows []entry) []*jsonx.OrderedMap {
	out := []*jsonx.OrderedMap{}
	var cur *jsonx.OrderedMap
	var items []*jsonx.OrderedMap
	curKey := ""
	flush := func() {
		if cur != nil {
			cur.Set("items", items)
			out = append(out, cur)
		}
	}
	for _, r := range rows {
		jy, jm, _ := engine.ToJalali(r.date)
		key := fmt.Sprintf("%04d-%02d", jy, jm)
		if key != curKey {
			flush()
			curKey, items = key, nil
			start := engine.FromJalali(jy, jm, 1)
			cur = jsonx.Obj("key", key, "jalali_year", jy, "jalali_month", jm, "start", start.String(),
				"end", engine.JalaliMonthEnd(start).String())
		}
		items = append(items, r.render)
	}
	flush()
	return out
}

// Category keys of the record home grid, in board order.
const (
	CategoryLabs          = "labs"
	CategorySurgeries     = "surgeries"
	CategoryFamilyHistory = "family_history"
)

// Categories is GET /health-record/categories (nbl_Rec_Home grid): counts per document kind, lab sheets, surgeries
// and family history entries, plus the total and the documents waiting for the user's review.
func (s *Documents) Categories(ctx context.Context, userID uint64, locale, def string) (*jsonx.OrderedMap, error) {
	byKind, err := s.q.CountRecordDocumentsByKind(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("healthrecord: count documents: %w", err)
	}
	counts := map[string]int64{}
	var docs int64
	for _, k := range byKind {
		counts[k.Kind] = k.Documents
		docs += k.Documents
	}
	labs, err := s.labRows(ctx, userID, locale, def)
	if err != nil {
		return nil, err
	}
	extras, err := s.GetExtras(ctx, userID)
	if err != nil {
		return nil, err
	}
	review, err := s.q.CountRecordDocumentsInReview(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("healthrecord: count documents: %w", err)
	}
	cat := func(key string, n int64) *jsonx.OrderedMap { return jsonx.Obj("key", key, "count", n) }
	list := []*jsonx.OrderedMap{cat(CategoryLabs, int64(len(labs)))}
	for _, k := range DocumentKinds {
		list = append(list, cat(k, counts[k]))
	}
	list = append(list, cat(CategorySurgeries, int64(len(extras.Surgeries))),
		cat(CategoryFamilyHistory, int64(len(extras.FamilyHistory))))
	return jsonx.Obj(
		"categories", list,
		"documents_count", docs,
		"all_count", docs+int64(len(labs)),
		"needs_review_count", review,
	), nil
}
