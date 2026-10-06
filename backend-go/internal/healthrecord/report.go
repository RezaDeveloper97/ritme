package healthrecord

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The doctor report (bloom B-N6-04, nbl_Record_Export / nbl_Record_Preview, D-65): the record built for
// AudienceShare over a chosen window and a chosen set of sections. GET /health-record/report is the owner's preview
// (the PDF is drawn on the device from it); internal/sharelinks freezes the same body into a 7-day share link.

// Report windows («بازه زمانی»): the last 3 / 6 / 12 months, or a custom start date up to today.
const (
	ReportRange3M     = analysis.Range3M
	ReportRange6M     = analysis.Range6M
	ReportRange1Y     = analysis.Range1Y
	ReportRangeCustom = "custom"
)

// ReportRanges are the accepted `range` values.
var ReportRanges = []string{ReportRange3M, ReportRange6M, ReportRange1Y, ReportRangeCustom}

// Report limits.
const (
	MaxQuestionLen = 300 // «سؤال برای پزشک (اختیاری)»
	MaxCustomYears = 3   // a custom window starts at most this many years back
)

// ReportRequest is a validated report selection.
type ReportRequest struct {
	Range    string
	From     civildate.Date // custom only
	Sections []string       // built-in section keys, screen order
	Question string         // "" = none
}

// Options are the Build options of the request for today.
func (r ReportRequest) Options(today civildate.Date, locale, def string) Options {
	o := Options{Today: today, Locale: locale, DefaultLocale: def, Sections: r.Sections}
	if r.Range == ReportRangeCustom {
		o.From = r.From
	} else {
		o.CycleRange = r.Range
		o.VitalsDays = analysis.NewRange(r.Range, today).Days()
	}
	return o
}

// Window is the report window [from, to].
func (r ReportRequest) Window(today civildate.Date) (civildate.Date, civildate.Date) {
	if r.Range == ReportRangeCustom {
		return r.From, today
	}
	rg := analysis.NewRange(r.Range, today)
	return rg.From, rg.To
}

// WindowJSON is {key, from, to, days}.
func (r ReportRequest) WindowJSON(today civildate.Date) *jsonx.OrderedMap {
	from, to := r.Window(today)
	return jsonx.Obj("key", r.Range, "from", from.String(), "to", to.String(), "days", from.DiffDays(to)+1)
}

// ParseReportRequest validates {range, from?, sections[], question?} (the query of GET /health-record/report with
// sections comma-separated, or the body of POST /health-record/share-links). The 422 is the controller envelope.
func ParseReportRequest(data phpval.Map, locale string, now time.Time, withQuestion bool) (ReportRequest, error) {
	today := civildate.InTehran(now)
	keys := []string{"range", "from", "sections"}
	if withQuestion {
		keys = append(keys, "question")
	}
	data = pick(data, keys...)
	if v, ok := data.Get("sections"); ok {
		if s, isStr := v.(string); isStr { // a query string: "basics,vitals"
			list := []any{}
			for _, k := range strings.Split(s, ",") {
				if k = strings.TrimSpace(k); k != "" {
					list = append(list, k)
				}
			}
			data.Set("sections", list)
		}
	}
	rules := validation.Rules{
		validation.F("range", "required", "string", validation.In(ReportRanges...)),
		validation.F("from", "nullable", "required_if:range,"+ReportRangeCustom, "date",
			"after_or_equal:"+civildate.FromTime(today.Midnight(time.UTC).AddDate(-MaxCustomYears, 0, 0)).String(), "before:today"),
		validation.F("sections", "required", "array", "min:1", "max:"+strconv.Itoa(len(Sections))),
		validation.F("sections.*", "required", "string", validation.In(Sections...)),
	}
	if withQuestion {
		rules = append(rules, validation.F("question", "nullable", "string", "max:"+strconv.Itoa(MaxQuestionLen)))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return ReportRequest{}, err
	}
	req := ReportRequest{Range: str(data, "range")}
	if req.Range == ReportRangeCustom {
		if t, err := civildate.ParseLenient(str(data, "from"), now, civildate.Tehran); err == nil {
			req.From = civildate.InTehran(t)
		}
	}
	v, _ := data.Get("sections")
	_, vals := phpval.Entries(v)
	picked := map[string]bool{}
	for _, x := range vals {
		picked[phpval.ToString(x)] = true
	}
	for _, k := range Sections { // screen order, duplicates dropped
		if picked[k] {
			req.Sections = append(req.Sections, k)
		}
	}
	if withQuestion {
		req.Question = strings.Join(strings.Fields(str(data, "question")), " ")
	}
	return req, nil
}

// ShareJSON is the record as a third party sees it: rec.JSON() without any row id (the share audience already drops
// notes and `editable`). Every share output (preview, PDF source, share link snapshot) goes through it.
func ShareJSON(rec *Record) *jsonx.OrderedMap {
	body := rec.JSON()
	for _, s := range rec.Sections {
		if s.Data == nil {
			continue
		}
		if v, ok := s.Data.Get("items"); ok {
			if items, isList := v.([]*jsonx.OrderedMap); isList {
				for _, it := range items {
					it.Delete("id")
				}
			}
		}
	}
	return body
}

// BuildReport builds the share-audience record of userID for req.
// The body is {range, record}.
func (s *Service) BuildReport(ctx context.Context, userID uint64, req ReportRequest, now time.Time, locale, def string,
) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	rec, err := s.Build(ctx, userID, AudienceShare, req.Options(today, locale, def))
	if err != nil {
		return nil, err
	}
	return jsonx.Obj("range", req.WindowJSON(today), "record", ShareJSON(rec)), nil
}

// Report is GET /health-record/report?range=3m|6m|1y|custom&from=&sections=a,b (nbl_Record_Preview): the doctor
// report of the owner for the chosen window and sections, built for the share audience. Free; the PDF is drawn on the
// device from this body.
func (h *Handlers) Report(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	req, err := ParseReportRequest(validation.Query(c), locale, now, false)
	if err != nil {
		return err
	}
	body, err := h.svc.BuildReport(c, userID, req, now, locale, i18n.LanguagesOf(c).DefaultCode())
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}
