package labs

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func iso(t time.Time) any { return jsonx.ISO8601(t.In(civildate.Tehran)) }

func isoNull(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return iso(t.Time)
}

func strNull(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func dateNull(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date.String()
}

func numOrNil(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

// Stages of the processing screen (nbl_Lab_Processing: خواندن تصویر → استخراج شاخص‌ها → مقایسه با محدوده مرجع →
// آماده‌سازی توضیح ساده).
const (
	StageQueued     = "queued"
	StageReading    = "reading"
	StageExtracting = "extracting"
	StageReview     = "review"
	StageExplaining = "explaining"
	StageDone       = "done"
	StageFailed     = "failed"
)

func stage(lab store.LabReport) string {
	switch lab.Status {
	case StatusQueued:
		return StageQueued
	case StatusExtracting:
		if lab.Progress < 30 {
			return StageReading
		}
		return StageExtracting
	case StatusNeedsReview:
		return StageReview
	case StatusInterpreting:
		return StageExplaining
	case StatusReady:
		return StageDone
	}
	return StageFailed
}

// title is the lab's title, else the category's default name (nbl_Lab_Intro «آزمایش خون کامل + آهن»).
func (l loc) title(lab store.LabReport) string {
	if lab.Title.Valid && lab.Title.String != "" {
		return lab.Title.String
	}
	return T("categories."+lab.Category, l.Locale, nil)
}

// date is the lab's sheet date, else its upload day.
func labDate(takenOn civildate.NullDate, created sql.NullTime) civildate.Date {
	if takenOn.Valid {
		return takenOn.Date
	}
	if created.Valid {
		return civildate.FromTime(created.Time)
	}
	return civildate.Date{}
}

func countsJSON(c Counts) *jsonx.OrderedMap {
	return jsonx.Obj("total", c.Total, "normal", c.Normal, "attention", c.Attention, "low", c.Low,
		"borderline_low", c.BorderlineLow, "high", c.High, "borderline_high", c.BorderlineHigh, "unknown", c.Unknown)
}

// markerJSON is one marker row of a lab.
func (l loc) markerJSON(e Evaluated) *jsonx.OrderedMap {
	var code, subtitle, category any
	needsReview := false
	if e.Marker != nil {
		code, category, needsReview = e.Marker.Code, e.Marker.Meta.Category, e.Marker.NeedsReview
		if s := l.pick(e.Marker.Meta.Subtitle); s != "" {
			subtitle = s
		}
	}
	var conf any
	if e.Confidence != nil {
		conf = *e.Confidence
	}
	var rangeSource any
	if e.Range.Source != "" {
		rangeSource = e.Range.Source
	}
	var refText any
	if e.Range.Text != "" {
		refText = e.Range.Text
	}
	return jsonx.Obj(
		"id", e.Row.ID,
		"code", code,
		"name", l.name(e),
		"printed_name", e.Row.Name,
		"subtitle", subtitle,
		"category", category,
		"value", numOrNil(e.Value),
		"value_text", strNull(e.Row.ValueText),
		"unit", strNull(e.Row.Unit),
		"reference", jsonx.Obj("low", numOrNil(e.Range.Low), "high", numOrNil(e.Range.High), "text", refText, "source", rangeSource),
		"state", e.State,
		"state_label", T("states."+e.State, l.Locale, nil),
		"attention", Attention(e.State),
		"confidence", conf,
		"low_confidence", e.LowConfidence(),
		"source", e.Row.Source,
		"catalog_needs_review", needsReview,
	)
}

func redFlagsOf(evals []Evaluated, l loc) []RedFlag {
	out := []RedFlag{}
	for _, e := range evals {
		if f, ok := redFlag(e, l.name(e)); ok {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Severity == SeverityUrgent && out[j].Severity != SeverityUrgent })
	return out
}

func (l loc) redFlagsJSON(flags []RedFlag) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(flags))
	for _, f := range flags {
		out = append(out, jsonx.Obj("marker_id", f.MarkerID, "code", f.Code, "name", f.Name, "severity", f.Severity,
			"direction", f.Direction,
			"message", T("red_flags."+f.Severity+"_"+f.Direction, l.Locale, map[string]string{"name": f.Name})))
	}
	return out
}

// interpretationJSON is the stored summary plus what is derived on read (flags, questions, disclaimer).
func (l loc) interpretationJSON(lab store.LabReport, evals []Evaluated) any {
	if !lab.Interpretation.Valid {
		return nil
	}
	var in Interpretation
	if err := json.Unmarshal(lab.Interpretation.V, &in); err != nil {
		return nil
	}
	disclaimer := T("disclaimer_rules", l.Locale, nil)
	if in.Source == SummaryAI {
		disclaimer = T("disclaimer", l.Locale, nil)
	}
	return jsonx.Obj(
		"summary", in.Summary,
		"source", in.Source,
		"locale", in.Locale,
		"generated_at", iso(in.GeneratedAt),
		"stale", lab.Status != StatusReady,
		"red_flags", l.redFlagsJSON(redFlagsOf(evals, l)),
		"doctor_questions", questions(evals, l),
		"disclaimer", disclaimer,
	)
}

func lowConfidenceCount(evals []Evaluated) int {
	n := 0
	for _, e := range evals {
		if e.LowConfidence() {
			n++
		}
	}
	return n
}

// labJSON is GET /labs/{id}: the lab, its markers (attention first is the client's choice; sheet order here),
// files, interpretation and feedback.
func (l loc) labJSON(lab store.LabReport, evals []Evaluated, fs []store.LabFile) *jsonx.OrderedMap {
	markers := make([]*jsonx.OrderedMap, 0, len(evals))
	for _, e := range evals {
		markers = append(markers, l.markerJSON(e))
	}
	pages := make([]*jsonx.OrderedMap, 0, len(fs))
	for _, f := range fs {
		pages = append(pages, jsonx.Obj("id", f.ID, "page", f.Page, "mime", f.Mime, "size_bytes", f.SizeBytes,
			"url", "/api/v1/labs/"+strconv.FormatUint(lab.ID, 10)+"/files/"+strconv.FormatUint(f.ID, 10)))
	}
	var fasting any
	if lab.Fasting.Valid {
		fasting = lab.Fasting.Bool
	}
	var errMsg any
	if lab.ErrorCode.Valid {
		errMsg = T("errors."+lab.ErrorCode.String, l.Locale, nil)
	}
	var feedback any
	if lab.Feedback.Valid {
		feedback = jsonx.Obj("helpful", lab.Feedback.Int16 > 0, "note", strNull(lab.FeedbackNote), "at", isoNull(lab.FeedbackAt))
	}
	left := 0
	if lab.Source == SourceUpload {
		left = max(0, MaxInterpretations-int(lab.InterpretCount))
	}
	return jsonx.Obj(
		"id", lab.ID,
		"source", lab.Source,
		"category", lab.Category,
		"title", strNull(lab.Title),
		"display_title", l.title(lab),
		"taken_on", dateNull(lab.TakenOn),
		"date", labDate(lab.TakenOn, lab.CreatedAt).String(),
		"fasting", fasting,
		"lab_name", strNull(lab.LabName),
		"status", lab.Status,
		"stage", stage(lab),
		"progress", lab.Progress,
		"error_code", strNull(lab.ErrorCode),
		"error_message", errMsg,
		"editable", !busy(lab.Status) && lab.Status != StatusFailed,
		"ai_interpretations_left", left,
		"counts", countsJSON(countStates(evals)),
		"low_confidence_count", lowConfidenceCount(evals),
		"markers", markers,
		"files", pages,
		"interpretation", l.interpretationJSON(lab, evals),
		"feedback", feedback,
		"verified_at", isoNull(lab.VerifiedAt),
		"interpreted_at", isoNull(lab.InterpretedAt),
		"created_at", isoNull(lab.CreatedAt),
		"updated_at", isoNull(lab.UpdatedAt),
	)
}

// statusJSON is GET /labs/{id}/status: what the processing screen polls.
func statusJSON(lab store.LabReport, markers int, l loc) *jsonx.OrderedMap {
	var errMsg any
	if lab.ErrorCode.Valid {
		errMsg = T("errors."+lab.ErrorCode.String, l.Locale, nil)
	}
	return jsonx.Obj("id", lab.ID, "status", lab.Status, "stage", stage(lab), "progress", lab.Progress,
		"marker_count", markers, "error_code", strNull(lab.ErrorCode), "error_message", errMsg, "updated_at", isoNull(lab.UpdatedAt))
}

// listItemJSON is one row of GET /labs (nbl_Lab_Intro «تحلیل‌های قبلی»).
func (l loc) listItemJSON(lab store.LabReport, c Counts) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", lab.ID,
		"source", lab.Source,
		"category", lab.Category,
		"title", strNull(lab.Title),
		"display_title", l.title(lab),
		"taken_on", dateNull(lab.TakenOn),
		"date", labDate(lab.TakenOn, lab.CreatedAt).String(),
		"status", lab.Status,
		"stage", stage(lab),
		"progress", lab.Progress,
		"marker_count", c.Total,
		"attention_count", c.Attention,
		"all_normal", lab.Status == StatusReady && c.Total > 0 && c.Attention == 0,
		"created_at", isoNull(lab.CreatedAt),
	)
}

// catalogJSON is one marker of GET /labs/markers.
func (l loc) catalogJSON(m Marker) *jsonx.OrderedMap {
	var typical any
	if t := m.TypicalText(); t != "" {
		typical = t
	}
	var unit any
	if m.Meta.Unit != "" {
		unit = m.Meta.Unit
	}
	return jsonx.Obj("code", m.Code, "name", l.pick(m.Title), "subtitle", nilIfEmpty(l.pick(m.Meta.Subtitle)),
		"category", nilIfEmpty(m.Meta.Category), "unit", unit, "typical_range", typical, "range_varies", m.Meta.RangeVaries,
		"needs_review", m.NeedsReview)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// contextNotes are the personal lines of a marker's detail (nbl_Lab_Marker «پریودهای تو ۵ روزه ثبت شده»,
// «مخصوصاً چون در حالت اقدام به بارداری هستی»).
func (l loc) contextNotes(e Evaluated, uc UserContext) []string {
	notes := []string{}
	if e.Marker == nil {
		return notes
	}
	if e.Marker.Meta.RangeVaries {
		notes = append(notes, T("context.range_varies", l.Locale, nil))
	}
	if !Attention(e.State) {
		return notes
	}
	cycling := uc.Mode == enums.LifeModeCycle || uc.Mode == enums.LifeModeTTC || uc.Mode == enums.LifeModeTeen
	if lowSide(e.State) && uc.PeriodDays > 0 && cycling && (e.Marker.Meta.Category == "iron" || e.Marker.Code == "hemoglobin" || e.Marker.Code == "hematocrit") {
		notes = append(notes, T("context.period_days", l.Locale, map[string]string{"days": num(uc.PeriodDays, l.Locale)}))
	}
	switch uc.Mode {
	case enums.LifeModeTTC:
		notes = append(notes, T("context.mode_ttc", l.Locale, nil))
	case enums.LifeModePregnancy:
		notes = append(notes, T("context.mode_pregnancy", l.Locale, nil))
	case enums.LifeModePostpartum:
		notes = append(notes, T("context.mode_postpartum", l.Locale, nil))
	case enums.LifeModeMenopause:
		notes = append(notes, T("context.mode_menopause", l.Locale, nil))
	}
	return notes
}

// aboutJSON is the catalog part of a marker's detail: what it is, related factors and when to see a doctor.
func (l loc) aboutJSON(e Evaluated) (about, factors, seeDoctor any) {
	if e.Marker == nil {
		return nil, []string{}, nil
	}
	about = jsonx.Obj("code", e.Marker.Code, "name", l.pick(e.Marker.Title), "subtitle", nilIfEmpty(l.pick(e.Marker.Meta.Subtitle)),
		"body", nilIfEmpty(l.pick(e.Marker.Body)), "typical_range", nilIfEmpty(e.Marker.TypicalText()),
		"needs_review", e.Marker.NeedsReview)
	list := []string{}
	if adv := e.Marker.AdviceFor(e.State); adv != nil {
		for _, f := range adv.Factors {
			if s := l.pick(f); s != "" {
				list = append(list, s)
			}
		}
		seeDoctor = nilIfEmpty(l.pick(adv.SeeDoctor))
	}
	return about, list, seeDoctor
}
