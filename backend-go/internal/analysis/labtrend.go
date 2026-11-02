package analysis

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/plus"
)

// Lab trend (An_Labs, §4.1 «روند آزمایش», D-175): one marker across the user's consecutive verified lab reports —
// confirmed values only (a value read below lab.ocr min_confidence and not confirmed yet is left out, T-M10-20) —
// ready with P.LabTrendMinReports reports. Plus (§4.3). The sentence describes the direction of the numbers only;
// what they mean is for the user's doctor.

// LabTrends reads the user's lab series (internal/labs.Service.Trends).
type LabTrends interface {
	Trends(ctx context.Context, userID uint64) (labs.Trends, *labs.Catalog, error)
}

// WithLabTrends wires GET /analysis/labs (nil = every marker is empty).
func (h *Handlers) WithLabTrends(t LabTrends) *Handlers {
	h.labTrends = t
	return h
}

// LabSeries is one marker's trend as the analysis reads it.
type LabSeries struct {
	Key, Name, Unit string
	Points          []labs.Point
	Direction       string // "" below the minimum
}

// Ready: §4.1 — the marker in P.LabTrendMinReports reports.
func (s LabSeries) Ready() bool { return len(s.Points) >= P.LabTrendMinReports }

// NewLabSeries maps a labs series (name in the request locale).
func NewLabSeries(s labs.Series, locale, def string) LabSeries {
	out := LabSeries{Key: s.Key, Name: s.Latest.Row.Name, Unit: s.Latest.Row.Unit.String, Points: s.Points}
	if m := s.Latest.Marker; m != nil && len(m.Title) > 0 {
		if n := i18n.PickString(m.Title, locale, def); n != "" {
			out.Name = n
		}
	}
	if out.Ready() {
		out.Direction = s.Direction()
	}
	return out
}

// phrase is the trend sentence.
func (s LabSeries) phrase() Phrase {
	if !s.Ready() {
		return Phrase{Key: "labs.trend.not_enough", Args: []Arg{Num("needed", P.LabTrendMinReports-len(s.Points))}}
	}
	return Phrase{Key: "labs.trend." + s.Direction, Args: []Arg{Num("count", min(len(s.Points), labs.TrendWindow))}}
}

// JSON is the marker body.
func (s LabSeries) JSON(c *Copy) *jsonx.OrderedMap {
	points := make([]*jsonx.OrderedMap, 0, len(s.Points))
	for _, p := range s.Points {
		points = append(points, jsonx.Obj("date", p.Date.String(), "value", p.Value, "state", p.State))
	}
	var dir, unit any
	if s.Direction != "" {
		dir = s.Direction
	}
	if s.Unit != "" {
		unit = s.Unit
	}
	return jsonx.Obj(
		"marker", s.Key, "name", s.Name, "unit", unit,
		"ready", s.Ready(), "reports", len(s.Points), "min_reports", P.LabTrendMinReports,
		"reports_needed", max(0, P.LabTrendMinReports-len(s.Points)),
		"direction", dir, "points", points,
		"finding", s.phrase().JSON(c),
		"note", Phrase{Key: "labs.trend.note"}.JSON(c),
	)
}

// LabTrend is GET /analysis/labs[?marker=]: without marker the list of markers (reports per marker), with it that
// marker's trend. A free user gets 200 with locked true and no data (nothing is read).
func (h *Handlers) LabTrend(c fiber.Ctx) error {
	q := validation.Query(c)
	v := validation.Make(lang.Default(), i18n.Locale(c), q, validation.Rules{
		validation.F("marker", "nullable", "string", "max:191"),
	})
	if v.Fails() {
		return v.Errors()
	}
	marker := ""
	if raw, ok := q.Get("marker"); ok {
		if s, isStr := raw.(string); isStr {
			marker = s
		}
	}
	ri, err := h.info(c)
	if err != nil {
		return err
	}
	return serve(c, h.cache, ri, "labs:"+marker, "-", func(context.Context) (any, error) {
		allowed := ri.ents.allowed(plus.DeepAnalysis)
		var series []LabSeries
		if allowed && h.labTrends != nil {
			t, _, err := h.labTrends.Trends(ri.ctx, ri.uid)
			if err != nil {
				return nil, err
			}
			for _, s := range t.Series {
				series = append(series, NewLabSeries(s, ri.locale, ri.def))
			}
		}
		cp := h.copy(ri)
		sec := plusSection(allowed, func() (bool, any) { return LabTrendBody(series, marker, cp) })
		return jsonx.Obj("plus", sec.Plus, "locked", sec.Locked, "ready", sec.Ready, "data", sec.Data), nil
	})
}

// LabTrendBody is the data of GET /analysis/labs: the marker's trend (an unknown marker → an empty, not-ready body),
// or with marker "" the markers list.
func LabTrendBody(series []LabSeries, marker string, c *Copy) (bool, any) {
	if marker != "" {
		for _, s := range series {
			if s.Key == marker {
				return s.Ready(), s.JSON(c)
			}
		}
		return false, LabSeries{Key: marker}.JSON(c)
	}
	list := make([]*jsonx.OrderedMap, 0, len(series))
	ready := false
	for _, s := range series {
		ready = ready || s.Ready()
		var unit any
		if s.Unit != "" {
			unit = s.Unit
		}
		var dir any
		if s.Direction != "" {
			dir = s.Direction
		}
		list = append(list, jsonx.Obj("marker", s.Key, "name", s.Name, "unit", unit, "reports", len(s.Points), "ready", s.Ready(),
			"reports_needed", max(0, P.LabTrendMinReports-len(s.Points)), "direction", dir, "finding", s.phrase().JSON(c)))
	}
	return ready, jsonx.Obj("min_reports", P.LabTrendMinReports, "markers", list, "note", Phrase{Key: "labs.trend.note"}.JSON(c))
}
