package home

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Section is HomeSectionInterface: one self-contained home widget.
type Section interface {
	// Key is the stable key (also the {section} route parameter).
	Key() string
	// Order is the display order (ascending).
	Order() int
	// Supports reports whether the section appears for the context.
	Supports(hc *Context) bool
	// Build returns the section, or nil to omit it.
	Build(hc *Context) (*Rendered, error)
}

// Rendered is one rendered section (HomeSection.php). Title / Subtitle nil = null; Meta is
// always empty in the ported sections, so it serialises as null.
type Rendered struct {
	Key      string
	Type     string
	Title    *string
	Subtitle *string
	Order    int
	Action   *jsonx.OrderedMap
	Data     *jsonx.OrderedMap
}

// JSON is HomeSection::toArray().
func (s *Rendered) JSON() *jsonx.OrderedMap {
	var title, subtitle, action any
	if s.Title != nil {
		title = *s.Title
	}
	if s.Subtitle != nil {
		subtitle = *s.Subtitle
	}
	if s.Action != nil {
		action = s.Action
	}
	return jsonx.Obj(
		"key", s.Key,
		"type", s.Type,
		"title", title,
		"subtitle", subtitle,
		"order", s.Order,
		"action", action,
		"data", s.Data,
		"meta", nil,
	)
}

// action is AbstractHomeSection::action().
func action(key, label string) *jsonx.OrderedMap { return jsonx.Obj("key", key, "label", label) }

func str(s string) *string { return &s }

// Page is HomePageService: the section registry.
type Page struct {
	sections []Section
	logger   *slog.Logger
}

// NewPage returns the page with the given sections (registration order = available_sections
// order); nil sections means the 17 production sections.
func NewPage(logger *slog.Logger, sections ...Section) *Page {
	if sections == nil {
		sections = DefaultSections()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Page{sections: sections, logger: logger}
}

// DefaultSections are HomePageService's sections in registration order.
func DefaultSections() []Section {
	return []Section{
		headerSection{}, weekCalendarSection{}, nextPeriodSection{}, cyclePredictionSection{},
		recommendationsSection{}, tasksSection{}, challengeSection{}, doctorReminderSection{},
		medicationReminderSection{}, smartTipSection{}, affirmationSection{}, weeklySummarySection{},
		vitalsSection{}, statusChartsSection{}, articlesSection{}, myCyclesSection{}, cycleSummarySection{},
	}
}

// Build is build(): every supported section that builds, sorted by order.
func (p *Page) Build(hc *Context) []*jsonx.OrderedMap {
	type built struct {
		order int
		json  *jsonx.OrderedMap
	}
	var out []built
	for _, s := range p.sections {
		if !s.Supports(hc) {
			continue
		}
		if res := p.safeBuild(s, hc); res != nil {
			out = append(out, built{res.Order, res.JSON()})
		}
	}
	slices.SortStableFunc(out, func(a, b built) int { return a.order - b.order })
	list := make([]*jsonx.OrderedMap, len(out))
	for i, b := range out {
		list[i] = b.json
	}
	return list
}

// BuildSection is buildSection(): nil when unknown, unsupported, empty or failed.
func (p *Page) BuildSection(key string, hc *Context) *jsonx.OrderedMap {
	s := p.find(key)
	if s == nil || !s.Supports(hc) {
		return nil
	}
	if res := p.safeBuild(s, hc); res != nil {
		return res.JSON()
	}
	return nil
}

// Has is hasSection().
func (p *Page) Has(key string) bool { return p.find(key) != nil }

// Keys is availableKeys().
func (p *Page) Keys() []string {
	keys := make([]string, len(p.sections))
	for i, s := range p.sections {
		keys[i] = s.Key()
	}
	return keys
}

func (p *Page) find(key string) Section {
	for _, s := range p.sections {
		if s.Key() == key {
			return s
		}
	}
	return nil
}

// safeBuild is the PHP try/catch: an error or a panic is logged and the section omitted.
func (p *Page) safeBuild(s Section, hc *Context) (res *Rendered) {
	defer func() {
		if r := recover(); r != nil {
			p.fail(s, hc, fmt.Errorf("panic: %v", r))
			res = nil
		}
	}()
	res, err := s.Build(hc)
	if err != nil {
		p.fail(s, hc, err)
		return nil
	}
	return res
}

func (p *Page) fail(s Section, hc *Context, err error) {
	p.logger.ErrorContext(hc.Ctx, "HomePage section failed to build",
		slog.String("section", s.Key()), slog.Uint64("user_id", hc.UserID), slog.String("error", err.Error()))
}

// phpNumberString is PHP's (string) cast of an int or float.
func phpNumberString(v any) string {
	switch n := v.(type) {
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case float64:
		return phpround.String(n)
	}
	return fmt.Sprint(v)
}
