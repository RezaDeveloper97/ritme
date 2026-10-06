// Package children is the admin «کودک» module (B-N5-09, docs/go-migration/admin-api.md §18) under
// /api/admin/v1/children/*.
//
// The child catalogs (vaccine schedule, milestones + activities + doctor notes, age notes, learn) are catalog_items
// groups edited through the generic /api/admin/v1/catalog/{group} API (their writes are super-admin only there, see
// catalog.Admin.WithSuperGroups). This module only adds what the generic API cannot serve:
//
//   - GET /children/who — a read-only viewer of the embedded WHO Child Growth Standards (seeds/who): the daily LMS
//     parameters of one indicator and sex, sampled by month / week / day, with the derived P3 / P15 / P50 / P85 / P97
//     values (internal/children/growth). The standards are fixed reference data: no table, no edit, no import.
//
// Any active admin may read (reference data, no user data). No writes, so no audit and no CSRF-relevant routes.
package children

import (
	"log/slog"
	"math"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/children/growth"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/seeds/who"
)

// Route registers one endpoint (httpadmin.Handle with the prefix applied).
type Route func(method, path string, chain httpadmin.Chain)

// Handlers serve /children/*.
type Handlers struct {
	logger *slog.Logger
}

// New wires the module.
func New(logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{logger: logger}
}

// Routes mounts the module (any active admin).
func (h *Handlers) Routes(route Route, kit *httpadmin.Kit) {
	route(fiber.MethodGet, "/children/who", kit.Admin(h.WHO))
}

// Sampling steps of the viewer.
const (
	StepMonth = "month" // day = round(month × 30.4375), months 0–60 (the WHO monthly tables)
	StepWeek  = "week"  // day = 7 × week, weeks 0–265
	StepDay   = "day"   // every day 0–1856
)

// Steps in display order.
var Steps = []string{StepMonth, StepWeek, StepDay}

// DaysPerMonth is the WHO average month length (365.25 / 12).
const DaysPerMonth = 30.4375

// Percentiles of the derived columns with their z-scores.
var percentiles = []struct {
	Key string
	Z   float64
}{
	{"p3", growth.ZP3}, {"p15", growth.ZP15}, {"p50", 0}, {"p85", growth.ZP85}, {"p97", growth.ZP97},
}

// Point is one sampled age: the age in the step's unit and the day index.
type Point struct {
	Age int
	Day int
}

// Points are the sampled ages of step over 0–who.MaxDay.
func Points(step string) []Point {
	var out []Point
	switch step {
	case StepDay:
		for d := 0; d <= who.MaxDay; d++ {
			out = append(out, Point{Age: d, Day: d})
		}
	case StepWeek:
		for w := 0; 7*w <= who.MaxDay; w++ {
			out = append(out, Point{Age: w, Day: 7 * w})
		}
	default:
		for m := 0; ; m++ {
			d := int(math.Round(float64(m) * DaysPerMonth))
			if d > who.MaxDay {
				break
			}
			out = append(out, Point{Age: m, Day: d})
		}
	}
	return out
}

// Row is one sampled day of a table as JSON.
func Row(ind who.Indicator, p who.LMS, age int) *jsonx.OrderedMap {
	o := jsonx.Obj("age", age, "day", p.Day, "l", p.L, "m", p.M, "s", p.S)
	for _, pc := range percentiles {
		o.Set(pc.Key, growth.Round(growth.ValueAt(ind, p, pc.Z), 3))
	}
	return o
}

func indicatorNames() []string {
	out := make([]string, 0, len(who.Indicators))
	for _, i := range who.Indicators {
		out = append(out, string(i))
	}
	return out
}

func sexNames() []string {
	out := make([]string, 0, len(who.Sexes))
	for _, s := range who.Sexes {
		out = append(out, string(s))
	}
	return out
}

// unitOf is the measurement unit of an indicator.
func unitOf(ind who.Indicator) string {
	if ind == who.Weight {
		return "kg"
	}
	return "cm"
}

// WHO is GET /children/who?indicator=weight|length|head&sex=girl|boy&step=month|week|day&page=&per_page=: the
// sampled LMS rows with derived percentiles, paged (per_page ≤ 100), plus the options and the source citation.
// indicator and sex default to weight / girl; an unknown value is a 422.
func (h *Handlers) WHO(c fiber.Ctx) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("indicator", "nullable|string", validation.In(indicatorNames()...)),
		validation.F("sex", "nullable|string", validation.In(sexNames()...)),
		validation.F("step", "nullable|string", validation.In(Steps...)),
	})
	if err != nil {
		return err
	}
	ind := who.Indicator(orDefault(httpadmin.String(data, "indicator"), string(who.Weight)))
	sex := who.Sex(orDefault(httpadmin.String(data, "sex"), string(who.Girl)))
	step := orDefault(httpadmin.String(data, "step"), StepMonth)

	table := who.Table(ind, sex)
	pts := Points(step)
	p := httpadmin.PageOf(c)
	from := min(p.Offset(), len(pts))
	to := min(from+p.PerPage, len(pts))
	items := make([]*jsonx.OrderedMap, 0, to-from)
	for _, pt := range pts[from:to] {
		items = append(items, Row(ind, table[pt.Day], pt.Age))
	}
	page := httpadmin.Page(items, p, len(pts))
	page.Set("filters", jsonx.Obj("indicator", string(ind), "sex", string(sex), "step", step))
	page.Set("unit", unitOf(ind))
	page.Set("options", jsonx.Obj(
		"indicators", jsonx.List(indicatorNames()),
		"sexes", jsonx.List(sexNames()),
		"steps", jsonx.List(Steps),
		"max_day", who.MaxDay,
		"percentiles", jsonx.List([]string{"p3", "p15", "p50", "p85", "p97"}),
	))
	page.Set("source", who.Source)
	page.Set("read_only", true)
	return httpadmin.OK(c, page)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
