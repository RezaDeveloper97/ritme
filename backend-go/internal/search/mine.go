package search

import (
	"context"
	"net/url"
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Insight window: the current cycle in cycle-like modes when a period start is known and recent, else the
// last fallbackDays days (pregnancy, postpartum, menopause, or no period logged).
const (
	fallbackDays  = 30
	maxCycleSpan  = 60 // a «current cycle» older than this is stale data: use the fallback window
	windowCycle   = "cycle"
	windowDays    = "days"
	analysisRoute = "/analysis"
)

// cycleModes are the modes whose window is the current cycle.
var cycleModes = []string{taxonomy.ModeCycle, taxonomy.ModeTTC, taxonomy.ModeTeen}

// skippedCategories have no fixed vocabulary to search: free note text, custom items and medicines (care
// reminders cover those by name), and the link tiles owned by other features.
var skippedCategories = []string{"note", "custom", "meds", "pregnancy", "baby"}

// Window is the date range of the log insights.
type Window struct {
	Kind     string // cycle | days
	From, To civildate.Date
}

func (w Window) json() *jsonx.OrderedMap {
	return jsonx.Obj("kind", w.Kind, "from", w.From, "to", w.To)
}

// window resolves the insight window of the user.
func (s *Service) window(ctx context.Context, userID uint64, mode string, today civildate.Date) (Window, error) {
	fallback := Window{Kind: windowDays, From: today.AddDays(-(fallbackDays - 1)), To: today}
	if !slices.Contains(cycleModes, mode) {
		return fallback, nil
	}
	start, ok, err := s.periods.LatestPeriodStart(ctx, userID)
	if err != nil || !ok || start.After(today) || start.DiffDays(today) >= maxCycleSpan {
		return fallback, err
	}
	return Window{Kind: windowCycle, From: start, To: today}, nil
}

// target is one searchable log item: a category, a yes/no or numeric param, or an item / option.
type target struct {
	cat, param, item string
	label, context   string
	scored           bool // pain-like items with a 1–10 score
	rank             int
}

func (t target) id() string {
	id := t.cat
	if t.param != "" {
		id += "." + t.param
	}
	if t.item != "" {
		id += "." + t.item
	}
	return id
}

func (t target) matches(e taxonomy.Entry) bool {
	return e.Category == t.cat && (t.param == "" || e.Param == t.param) && (t.item == "" || e.Item == t.item)
}

// positive is a logged «yes»: not a «no» level / bool, and not the «none» param or option.
func positive(e taxonomy.Entry) bool {
	return (!e.Code.Valid || e.Code.String != taxonomy.No) && e.Param != "none" && e.Item != "none"
}

func label(ns any, path, fallback string) string {
	if v, ok := phpval.Get(ns, path); ok {
		if s, isStr := v.(string); isStr && s != "" {
			return s
		}
	}
	return fallback
}

// logTargets are the taxonomy entries of mode whose label matches the query.
func logTargets(mode string, ns any, m Matcher) []target {
	var out []target
	for _, c := range taxonomy.Categories() {
		if slices.Contains(skippedCategories, c.Code) || !slices.Contains(c.Modes, mode) {
			continue
		}
		catPath := "categories." + c.Code
		catLabel := label(ns, catPath+".title", c.Code)
		if r := m.Best(catLabel); r > 0 {
			out = append(out, target{cat: c.Code, label: catLabel, rank: r + 1}) // the category itself first
		}
		for j := range c.Params {
			p := &c.Params[j]
			if p.Code == "none" || !c.Available(p, mode) {
				continue
			}
			pPath := catPath + ".params." + p.Code
			switch p.Type {
			case taxonomy.Bool, taxonomy.Number, taxonomy.Integer:
				pLabel := label(ns, pPath+".title", p.Code)
				if r := m.Best(pLabel); r > 0 {
					out = append(out, target{cat: c.Code, param: p.Code, label: pLabel, context: catLabel, rank: r})
				}
			case taxonomy.Items, taxonomy.Multi:
				for k := range p.Options {
					o := &p.Options[k]
					if o.Code == "none" || !o.OptionAvailable(mode) {
						continue
					}
					oLabel := label(ns, pPath+".options."+o.Code, o.Code)
					if r := m.Best(oLabel); r > 0 {
						out = append(out, target{cat: c.Code, param: p.Code, item: o.Code, label: oLabel,
							context: catLabel, scored: p.Score != nil, rank: r})
					}
				}
			}
		}
	}
	return out
}

// categoryScored reports whether a category has a scored param (its peak is the max over its items).
func categoryScored(cat string) bool {
	c, ok := taxonomy.CategoryByCode(cat)
	if !ok {
		return false
	}
	for _, p := range c.Params {
		if p.Score != nil {
			return true
		}
	}
	return false
}

// insight counts the days in the window with a positive entry of t, and the peak score (scored items).
type insight struct {
	days     int
	peak     float64
	peakDate civildate.Date
	hasPeak  bool
}

func count(t target, days []healthlog.DayEntries) insight {
	var in insight
	scored := t.scored || (t.param == "" && categoryScored(t.cat))
	for _, d := range days {
		hit := false
		for _, e := range d.Entries {
			if !t.matches(e) || !positive(e) {
				continue
			}
			hit = true
			if !scored || !e.Num.Valid {
				continue
			}
			if v, err := strconv.ParseFloat(e.Num.String, 64); err == nil && (!in.hasPeak || v > in.peak) {
				in.peak, in.peakDate, in.hasPeak = v, d.Date, true
			}
		}
		if hit {
			in.days++
		}
	}
	return in
}

// analysisPath is the analysis screen a log category opens (bloom B-N3-08/09).
func analysisPath(cat string) string {
	switch cat {
	case "bleeding":
		return analysisRoute + "/period"
	case "measurements":
		return analysisRoute + "/body"
	default:
		return analysisRoute + "/symptoms"
	}
}

// mine is the «از ثبت‌های تو» group: log insights of the matched taxonomy items (only those with at least
// one day in the window), an «in analysis» link per matched category, then the user's care reminders.
func (s *Service) mine(ctx context.Context, q Query, m Matcher) ([]Hit, error) {
	mode, err := s.logs.LifeMode(ctx, q.UserID)
	if err != nil {
		return nil, err
	}
	var hits []Hit
	if targets := logTargets(mode, q.Taxonomy, m); len(targets) > 0 {
		w, err := s.window(ctx, q.UserID, mode, q.Today)
		if err != nil {
			return nil, err
		}
		days, err := s.logs.Range(ctx, q.UserID, w.From, w.To)
		if err != nil {
			return nil, err
		}
		hits = insightHits(targets, days, w)
	}
	reminders, err := s.reminders.CareReminders(ctx, q.UserID)
	if err != nil {
		return nil, err
	}
	for i, r := range reminders {
		rank := m.Best(r.Title, r.Subtitle)
		if rank == 0 {
			continue
		}
		hits = append(hits, Hit{
			Type: "reminder", ID: strconv.FormatUint(r.ID, 10), Title: r.Title, Subtitle: r.Subtitle,
			Route: "/reminders/" + url.PathEscape(r.Kind) + "/" + strconv.FormatUint(r.ID, 10),
			Meta:  jsonx.Obj("kind", r.Kind, "active", r.Active),
			rank:  rank,
			order: 1000 + i,
		})
	}
	return hits, nil
}

func insightHits(targets []target, days []healthlog.DayEntries, w Window) []Hit {
	var hits, patterns []Hit
	for i, t := range targets {
		in := count(t, days)
		if in.days == 0 {
			continue
		}
		var peak any
		if in.hasPeak {
			var cycleDay any
			if w.Kind == windowCycle {
				cycleDay = w.From.DiffDays(in.peakDate) + 1
			}
			peak = jsonx.Obj("score", jsonx.Float(in.peak), "date", in.peakDate, "cycle_day", cycleDay)
		}
		query := url.Values{"category": {t.cat}}
		if t.param != "" {
			query.Set("param", t.param)
		}
		if t.item != "" {
			query.Set("item", t.item)
		}
		hits = append(hits, Hit{
			Type: "log_insight", ID: t.id(), Title: t.label, Subtitle: t.context,
			Route: analysisPath(t.cat) + "?" + query.Encode(),
			Meta: jsonx.Obj("category", t.cat, "param", nullable(t.param), "item", nullable(t.item),
				"days", in.days, "window", w.json(), "peak", peak),
			rank:  t.rank,
			order: i,
		})
		if t.param == "" {
			patterns = append(patterns, Hit{
				Type: "log_analysis", ID: t.cat, Title: t.label, Route: analysisPath(t.cat),
				Meta:  jsonx.Obj("category", t.cat),
				rank:  t.rank,
				order: 500 + i,
			})
		}
	}
	return append(hits, patterns...)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
