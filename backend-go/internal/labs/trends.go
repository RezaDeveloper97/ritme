package labs

import (
	"context"
	"sort"

	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Point is one value of a marker in one lab.
type Point struct {
	LabID, MarkerID uint64
	Date            civildate.Date
	Value           float64
	State           string
}

// Series is one marker across the user's verified labs, oldest first (nbl_An_Labs, nbl_Lab_Marker «روند»).
type Series struct {
	Key    string // catalog code, else "name:<normalised printed name>"
	Latest Evaluated
	Points []Point
}

// MinTrendPoints: a trend shows only when the marker is in at least this many labs (nbl_An_Labs note).
const MinTrendPoints = 2

// trendRows are the rows trends read: values of verified labs that did not fail.
func trendRows(rows []store.ListUserLabMarkersRow) []store.ListUserLabMarkersRow {
	out := rows[:0:0]
	for _, r := range rows {
		if r.LabVerifiedAt.Valid && r.LabStatus != StatusFailed {
			out = append(out, r)
		}
	}
	return out
}

func rowMarker(r store.ListUserLabMarkersRow) store.LabMarker {
	return store.LabMarker{ID: r.ID, LabID: r.LabID, UserID: r.UserID, Code: r.Code, Name: r.Name, Value: r.Value,
		ValueText: r.ValueText, Unit: r.Unit, RefLow: r.RefLow, RefHigh: r.RefHigh, RefText: r.RefText, Confidence: r.Confidence,
		Source: r.Source, SortOrder: r.SortOrder, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func seriesKey(code, name string) string {
	if code != "" {
		return code
	}
	return "name:" + normalizeName(name)
}

// buildSeries groups the rows (already oldest lab first) into series. A value in another unit than the latest one
// is left out of the points (mg/dL and mmol/L never share a line).
func buildSeries(rows []store.ListUserLabMarkersRow, cat *Catalog) []Series {
	type acc struct {
		evals []Evaluated
		rows  []store.ListUserLabMarkersRow
	}
	groups, order := map[string]*acc{}, []string{}
	for _, r := range trendRows(rows) {
		k := seriesKey(r.Code.String, r.Name)
		g, ok := groups[k]
		if !ok {
			g = &acc{}
			groups[k] = g
			order = append(order, k)
		}
		g.evals = append(g.evals, evaluate(rowMarker(r), cat))
		g.rows = append(g.rows, r)
	}
	out := make([]Series, 0, len(order))
	for _, k := range order {
		g := groups[k]
		latest := g.evals[len(g.evals)-1]
		s := Series{Key: k, Latest: latest, Points: []Point{}}
		for i, e := range g.evals {
			if e.Value == nil || NormalizeUnit(e.Row.Unit.String) != NormalizeUnit(latest.Row.Unit.String) {
				continue
			}
			s.Points = append(s.Points, Point{LabID: e.Row.LabID, MarkerID: e.Row.ID, Date: labDate(g.rows[i].LabTakenOn, g.rows[i].LabCreatedAt),
				Value: *e.Value, State: e.State})
		}
		out = append(out, s)
	}
	return out
}

// Direction is the series' trend direction ("" below MinTrendPoints).
func (s Series) Direction() string {
	if len(s.Points) < MinTrendPoints {
		return ""
	}
	vals := make([]float64, len(s.Points))
	for i, p := range s.Points {
		vals[i] = p.Value
	}
	return Direction(vals)
}

func (l loc) trendSentence(s Series) string {
	d := s.Direction()
	if d == "" {
		return T("trend.single", l.Locale, nil)
	}
	return T("trend."+d, l.Locale, map[string]string{"count": num(min(len(s.Points), TrendWindow), l.Locale)})
}

func (l loc) pointsJSON(s Series) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(s.Points))
	for _, p := range s.Points {
		out = append(out, jsonx.Obj("lab_id", p.LabID, "marker_id", p.MarkerID, "date", p.Date.String(), "value", p.Value, "state", p.State))
	}
	return out
}

// trendJSON is the trend block of a marker (detail screen and trends list).
func (l loc) trendJSON(s Series) *jsonx.OrderedMap {
	var dir any
	if d := s.Direction(); d != "" {
		dir = d
	}
	return jsonx.Obj("count", len(s.Points), "direction", dir, "sentence", l.trendSentence(s), "points", l.pointsJSON(s))
}

// seriesJSON is one marker of GET /labs/trends.
func (l loc) seriesJSON(s Series) *jsonx.OrderedMap {
	e := s.Latest
	var code any
	if e.Marker != nil {
		code = e.Marker.Code
	}
	var refText any
	if e.Range.Text != "" {
		refText = e.Range.Text
	}
	return jsonx.Obj(
		"key", s.Key,
		"code", code,
		"name", l.name(e),
		"unit", strNull(e.Row.Unit),
		"reference_text", refText,
		"latest", jsonx.Obj("lab_id", e.Row.LabID, "marker_id", e.Row.ID, "value", numOrNil(e.Value), "value_text", strNull(e.Row.ValueText),
			"state", e.State, "state_label", T("states."+e.State, l.Locale, nil), "attention", Attention(e.State)),
		"trend", l.trendJSON(s),
	)
}

// sortForDisplay: out-of-range latest values first, then markers with a trend, then catalog order / name.
func sortForDisplay(list []Series, cat *Catalog) {
	rank := func(s Series) (int, int, int) {
		a := 1
		if Attention(s.Latest.State) {
			a = 0
		}
		t := 1
		if len(s.Points) >= MinTrendPoints {
			t = 0
		}
		c := len(cat.Items()) + 1
		if s.Latest.Marker != nil {
			if i, ok := cat.byCode[s.Latest.Marker.Code]; ok {
				c = i
			}
		}
		return a, t, c
	}
	sort.SliceStable(list, func(i, j int) bool {
		ai, ti, ci := rank(list[i])
		aj, tj, cj := rank(list[j])
		if ai != aj {
			return ai < aj
		}
		if ti != tj {
			return ti < tj
		}
		return ci < cj
	})
}

// Trends is GET /labs/trends: how many verified labs, their date span and every marker's series.
type Trends struct {
	Labs        int
	First, Last civildate.Date
	Series      []Series
}

// Trends computes the user's trends.
func (s *Service) Trends(ctx context.Context, userID uint64) (Trends, *Catalog, error) {
	cat, err := s.Catalog(ctx)
	if err != nil {
		return Trends{}, nil, err
	}
	rows, err := s.UserMarkers(ctx, userID)
	if err != nil {
		return Trends{}, nil, err
	}
	t := Trends{Series: buildSeries(rows, cat)}
	seen := map[uint64]bool{}
	for _, r := range trendRows(rows) {
		if seen[r.LabID] {
			continue
		}
		seen[r.LabID] = true
		t.Labs++
		d := labDate(r.LabTakenOn, r.LabCreatedAt)
		if t.First.IsZero() || d.Before(t.First) {
			t.First = d
		}
		if d.After(t.Last) {
			t.Last = d
		}
	}
	sortForDisplay(t.Series, cat)
	return t, cat, nil
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

// trendsJSON is the data of GET /labs/trends.
func (l loc) trendsJSON(t Trends) *jsonx.OrderedMap {
	list := make([]*jsonx.OrderedMap, 0, len(t.Series))
	for _, s := range t.Series {
		list = append(list, l.seriesJSON(s))
	}
	return jsonx.Obj("labs_count", t.Labs, "first_date", dateOrNil(t.First), "last_date", dateOrNil(t.Last),
		"min_points", MinTrendPoints, "markers", list)
}

// HubLabs is the `labs` card of GET /analysis/summary (nbl_An_Hub «روند آزمایش‌ها · فریتین ۳۲ ← ۱۸ ← ۹»): ready when
// the user has a verified lab; the highlight is the first displayed marker with a trend (out-of-range first).
func (s *Service) HubLabs(ctx context.Context, userID uint64, locale, def string) (bool, any, error) {
	t, _, err := s.Trends(ctx, userID)
	if err != nil {
		return false, nil, err
	}
	if t.Labs == 0 {
		return false, nil, nil
	}
	l := loc{Locale: locale, Default: def}
	var highlight any
	for _, sr := range t.Series {
		if len(sr.Points) < MinTrendPoints {
			continue
		}
		pts := sr.Points[max(0, len(sr.Points)-TrendWindow):]
		vals := make([]float64, len(pts))
		for i, p := range pts {
			vals[i] = p.Value
		}
		e := sr.Latest
		highlight = jsonx.Obj("key", sr.Key, "name", l.name(e), "unit", strNull(e.Row.Unit), "values", vals,
			"state", e.State, "state_label", T("states."+e.State, l.Locale, nil), "direction", sr.Direction())
		break
	}
	attention := 0
	for _, sr := range t.Series {
		if Attention(sr.Latest.State) {
			attention++
		}
	}
	return true, jsonx.Obj("labs_count", t.Labs, "first_date", dateOrNil(t.First), "last_date", dateOrNil(t.Last),
		"markers_count", len(t.Series), "attention_count", attention, "highlight", highlight), nil
}
