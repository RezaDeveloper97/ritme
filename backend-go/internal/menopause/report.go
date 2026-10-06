package menopause

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/vitals"
)

// The menopause doctor report (CB-MENO-03, nbl_Meno_Report «گزارش برای پزشک»): her own records over a window,
// summarised for a doctor — never a diagnosis. It is one section of bloom's report builder (B-N6-04): ReportSection
// plugs it into healthrecord.Service as the provider section `menopause`, so the PDF and the 7-day share link are the
// builder's; GET /menopause/report is the owner's preview for the board's 1 / 3 / 6 month chips. The patient's
// questions («سؤال‌هایم از پزشک») are the builder's `question` (share link) / drawn on the device for the PDF.
//
// Privacy: the section reads menopause tables, the day log, the BP readings and her treatment items only — no notes,
// no loss data, no ids; the share audience gets exactly the owner's content.

// Report windows («۱ ماه · ۳ ماه · ۶ ماه») and limits.
const (
	DefaultReportMonths = 3
	// TopSymptoms is the length of «شایع‌ترین علائم».
	TopSymptoms = 5
)

// ReportMonths are the accepted ?months= values.
var ReportMonths = []int{1, 3, 6}

// ReportWindow is the window of the last `months` months ending today: [today − months, today] (۱۲ تیر تا ۱۲ مهر).
func ReportWindow(today civildate.Date, months int) (civildate.Date, civildate.Date) {
	return engine.AddMonths(today, -months), today
}

// SymptomShare is one symptom of the top list: the logged days it appeared on.
type SymptomShare struct {
	Key  string // log taxonomy slot (category.param.item)
	Days int
}

// ItemAdherence is one treatment item over the window.
type ItemAdherence struct {
	Item store.TreatmentItem
	// DaysTaken of Days (scheduled active days of the window up to its end); hrt / supplement.
	DaysTaken, Days int
	// Amount is the window's lifestyle total; ActiveDays the days the goal ran.
	Amount, ActiveDays int
}

// AdherencePct is DaysTaken / Days in whole percent (nil for lifestyle goals or no scheduled day).
func (a ItemAdherence) AdherencePct() *int {
	if a.Item.Kind == KindLifestyle || a.Days == 0 {
		return nil
	}
	p := (a.DaysTaken*100 + a.Days/2) / a.Days
	return &p
}

// PerWeek is the lifestyle goal's weekly average over the days it ran (nil before it ran).
func (a ItemAdherence) PerWeek() *float64 {
	if a.Item.Kind != KindLifestyle || a.ActiveDays == 0 {
		return nil
	}
	v := phpround.Round(float64(a.Amount)*7/float64(a.ActiveDays), 1)
	return &v
}

// SideEffectCount is one side effect over the window.
type SideEffectCount struct {
	Code        string
	Days        int
	First, Last civildate.Date
}

// Report is the menopause report data over [From, To].
type Report struct {
	From, To civildate.Date
	Stage    Stage
	// First / Last are the window's oldest and newest questionnaires (nil without any).
	First, Last *Score
	ScoreMax    int
	// Flashes are the timer rows of the window; TrackedDays the days with a flash or a log entry.
	Flashes, TrackedDays int
	// SweatNights are the days with night sweats logged, or a sweaty night flash.
	SweatNights int
	// SleepHours are the logged nights' hours.
	SleepHours []float64
	// BleedingDays are the days with bleeding or spotting logged, oldest first.
	BleedingDays []civildate.Date
	BP           []vitals.Reading
	Symptoms     []SymptomShare // top TopSymptoms, most days first
	Items        []ItemAdherence
	SideEffects  []SideEffectCount
}

// Days is the window's length.
func (r Report) Days() int { return r.From.DiffDays(r.To) + 1 }

// Empty: nothing logged in the window and no treatment.
func (r Report) Empty() bool {
	return r.TrackedDays == 0 && r.First == nil && len(r.BP) == 0 && len(r.Items) == 0 && len(r.SideEffects) == 0
}

// BleedingEvents are the runs of consecutive bleeding days (each run's first day).
func (r Report) BleedingEvents() []civildate.Date {
	var out []civildate.Date
	for i, d := range r.BleedingDays {
		if i == 0 || r.BleedingDays[i-1].AddDays(1) != d {
			out = append(out, d)
		}
	}
	return out
}

// HRTAdherencePct pools the window's HRT items (taken / scheduled days).
func (r Report) HRTAdherencePct() *int {
	taken, days := 0, 0
	for _, a := range r.Items {
		if a.Item.Kind == KindHRT {
			taken, days = taken+a.DaysTaken, days+a.Days
		}
	}
	if days == 0 {
		return nil
	}
	p := (taken*100 + days/2) / days
	return &p
}

// overlaps: the item ran some day of [from, to].
func overlaps(it store.TreatmentItem, from, to civildate.Date) bool {
	if it.StartedOn.Valid && it.StartedOn.Date.After(to) {
		return false
	}
	return !it.StoppedOn.Valid || it.StoppedOn.Date.After(from)
}

// Report builds the report data of userID over [from, to].
func (s *Service) Report(ctx context.Context, userID uint64, from, to civildate.Date) (Report, error) {
	r := Report{From: from, To: to}
	var err error
	if r.Stage, err = s.Profile(ctx, userID, to); err != nil {
		return Report{}, err
	}
	if err := s.reportScores(ctx, userID, &r); err != nil {
		return Report{}, err
	}
	logs, err := s.logDays(ctx, userID, from, to)
	if err != nil {
		return Report{}, err
	}
	flashes, err := s.Flashes(ctx, userID, from, to)
	if err != nil {
		return Report{}, err
	}
	r.days(logs, flashes)
	if s.vitals != nil {
		if r.BP, err = s.vitals.Merged(ctx, userID, vitals.TypeBP, from, to); err != nil {
			return Report{}, err
		}
	}
	if err := s.reportTreatment(ctx, userID, &r); err != nil {
		return Report{}, err
	}
	return r, nil
}

func (s *Service) reportScores(ctx context.Context, userID uint64, r *Report) error {
	sc, err := s.Scale(ctx)
	if err != nil {
		return err
	}
	r.ScoreMax = sc.MaxTotal()
	scores, err := s.Scores(ctx, userID, MonthStart(r.From)) // newest first
	if err != nil {
		return err
	}
	var in []Score
	for _, x := range scores {
		if !x.Month.After(r.To) {
			in = append(in, x)
		}
	}
	if len(in) > 0 {
		r.Last, r.First = &in[0], &in[len(in)-1]
	}
	return nil
}

// days fills the per-day counts from the day log and the timer.
func (r *Report) days(logs map[civildate.Date]LogDay, flashes []Flash) {
	tracked := map[civildate.Date]bool{}
	sweaty := map[civildate.Date]bool{}
	flashDays := map[civildate.Date]bool{}
	for _, f := range flashes {
		d := f.Day()
		tracked[d], flashDays[d] = true, true
		if f.Sweat && f.Night {
			sweaty[d] = true
		}
	}
	counts := map[string]int{}
	symptoms := taxonomy.MenopauseSymptoms()
	for d, l := range logs {
		if len(l.Entries) == 0 {
			continue
		}
		tracked[d] = true
		if l.Has(SlotNightSweats) {
			sweaty[d] = true
		}
		if l.View != nil {
			if h, ok := l.View.SleepHours(); ok {
				r.SleepHours = append(r.SleepHours, h)
			}
		}
		if l.Bleeding() {
			r.BleedingDays = append(r.BleedingDays, d)
		}
		for _, ref := range symptoms {
			key := ref.Category + "." + ref.Param + "." + ref.Item
			if l.Has(key) || (key == SlotHotFlashes && flashDays[d]) {
				counts[key]++
			}
		}
	}
	for d := range flashDays { // a flash day without a log still shows hot flashes
		if l, ok := logs[d]; !ok || len(l.Entries) == 0 {
			counts[SlotHotFlashes]++
		}
	}
	r.Flashes, r.TrackedDays, r.SweatNights = len(flashes), len(tracked), len(sweaty)
	slices.SortFunc(r.BleedingDays, func(a, b civildate.Date) int { return a.Compare(b) })
	for _, ref := range symptoms {
		key := ref.Category + "." + ref.Param + "." + ref.Item
		if counts[key] > 0 {
			r.Symptoms = append(r.Symptoms, SymptomShare{Key: key, Days: counts[key]})
		}
	}
	slices.SortStableFunc(r.Symptoms, func(a, b SymptomShare) int { return cmp.Compare(b.Days, a.Days) })
	r.Symptoms = r.Symptoms[:min(len(r.Symptoms), TopSymptoms)]
}

func (s *Service) reportTreatment(ctx context.Context, userID uint64, r *Report) error {
	items, err := s.q.ListTreatmentItems(ctx, userID)
	if err != nil {
		return fmt.Errorf("menopause: list treatment: %w", err)
	}
	slices.SortStableFunc(items, func(a, b store.TreatmentItem) int {
		return slices.Index(Kinds, a.Kind) - slices.Index(Kinds, b.Kind)
	})
	intakes, err := s.itemIntakes(ctx, userID, items, r.From, r.To)
	if err != nil {
		return err
	}
	meds, err := s.medications(ctx, userID)
	if err != nil {
		return err
	}
	for _, it := range items {
		if !overlaps(it, r.From, r.To) {
			continue
		}
		var med *care.Medication
		if it.ReminderID.Valid {
			if m, ok := meds[uint64(it.ReminderID.Int64)]; ok { //nolint:gosec // FK id
				med = &m
			}
		}
		r.Items = append(r.Items, adherenceOf(it, med, intakes[it.ID], r.From, r.To))
	}
	logs, err := s.q.ListSideEffectLogsInRange(ctx, store.ListSideEffectLogsInRangeParams{
		UserID: userID, FromDate: r.From, ToDate: r.To,
	})
	if err != nil {
		return fmt.Errorf("menopause: list side effects: %w", err)
	}
	byCode := map[string]*SideEffectCount{}
	for _, l := range logs { // date order
		c := byCode[l.Code]
		if c == nil {
			c = &SideEffectCount{Code: l.Code, First: l.LogDate}
			byCode[l.Code] = c
		}
		c.Days++
		c.Last = l.LogDate
	}
	codes := make([]string, 0, len(byCode))
	for code := range byCode {
		codes = append(codes, code)
	}
	for _, code := range ordered(codes, SideEffectCodes) {
		r.SideEffects = append(r.SideEffects, *byCode[code])
	}
	return nil
}

// adherenceOf counts the item's scheduled active days of [from, to] and the taken ones (lifestyle: the amount over
// the days the goal ran).
func adherenceOf(it store.TreatmentItem, med *care.Medication, taken map[civildate.Date]sql.NullInt16,
	from, to civildate.Date,
) ItemAdherence {
	a := ItemAdherence{Item: it}
	for d := from; !d.After(to); d = d.AddDays(1) {
		if !activeOn(it, d) {
			continue
		}
		amount, ok := taken[d]
		if it.Kind == KindLifestyle {
			a.ActiveDays++
			if ok {
				a.Amount += amountOf(it, amount)
			}
			continue
		}
		if med != nil && !slices.Contains(med.Meta.Weekdays, care.SaturdayWeekday(d)) {
			continue
		}
		a.Days++
		if ok {
			a.DaysTaken++
		}
	}
	return a
}

// --- JSON -----------------------------------------------------------------------------------------------------------

// SymptomNamer names a taxonomy slot (analysis.Copy.SymptomLabel; nil = the key).
type SymptomNamer func(key string) string

func round1(x float64) float64 { return phpround.Round(x, 1) }

func percent(n, total int) int {
	if total == 0 {
		return 0
	}
	return int(phpround.Round(100*float64(n)/float64(total), 0))
}

func scorePoint(s *Score) any {
	if s == nil {
		return nil
	}
	return jsonx.Obj("month", s.Month, "total", s.Total)
}

// ReportJSON is the menopause section data (and GET /menopause/report's `report`).
func ReportJSON(r Report, name SymptomNamer) *jsonx.OrderedMap {
	var score, perDay, perWeek, sleepAvg, bp any
	if r.Last != nil {
		score = jsonx.Obj("first", scorePoint(r.First), "last", scorePoint(r.Last), "max", r.ScoreMax,
			"change", r.Last.Total-r.First.Total)
	}
	if r.TrackedDays > 0 {
		perDay = round1(float64(r.Flashes) / float64(r.TrackedDays))
		perWeek = round1(float64(r.SweatNights) * 7 / float64(r.TrackedDays))
	}
	if len(r.SleepHours) > 0 {
		sum := 0.0
		for _, h := range r.SleepHours {
			sum += h
		}
		sleepAvg = round1(sum / float64(len(r.SleepHours)))
	}
	if len(r.BP) > 0 {
		var sys, dia float64
		for _, x := range r.BP {
			sys, dia = sys+x.Systolic, dia+x.Diastolic
		}
		n := float64(len(r.BP))
		bp = jsonx.Obj("systolic", int(phpround.Round(sys/n, 0)), "diastolic", int(phpround.Round(dia/n, 0)),
			"readings", len(r.BP), "unit", "mmhg")
	}
	top := make([]*jsonx.OrderedMap, len(r.Symptoms))
	for i, s := range r.Symptoms {
		label := s.Key
		if name != nil {
			label = name(s.Key)
		}
		top[i] = jsonx.Obj("key", s.Key, "label", label, "days", s.Days, "percent", percent(s.Days, r.TrackedDays))
	}
	items := []*jsonx.OrderedMap{}
	lifestyle := []*jsonx.OrderedMap{}
	supplements := []string{}
	for _, a := range r.Items {
		it := a.Item
		if it.Kind == KindLifestyle {
			lifestyle = append(lifestyle, jsonx.Obj("name", it.Name, "weekly_goal", nullInt16(it.WeeklyGoal),
				"goal_unit", nullString(it.GoalUnit), "per_week", ptr(a.PerWeek())))
			continue
		}
		if it.Kind == KindSupplement {
			supplements = append(supplements, it.Name)
		}
		items = append(items, jsonx.Obj("kind", it.Kind, "name", it.Name, "dose", nullString(it.Dose),
			"started_on", nullDate(it.StartedOn), "stopped_on", nullDate(it.StoppedOn), "days_taken", a.DaysTaken,
			"days", a.Days, "adherence_pct", ptr(a.AdherencePct())))
	}
	effects := make([]*jsonx.OrderedMap, len(r.SideEffects))
	for i, e := range r.SideEffects {
		effects[i] = jsonx.Obj("code", e.Code, "days", e.Days, "first_on", e.First, "last_on", e.Last)
	}
	events := r.BleedingEvents()
	if events == nil {
		events = []civildate.Date{}
	}
	return jsonx.Obj(
		"window", jsonx.Obj("from", r.From, "to", r.To, "days", r.Days()),
		"stage", jsonx.Obj("code", strOrNil(r.Stage.Stage), "months_without_period", ptr(r.Stage.MonthsWithoutPeriod),
			"surgical", ptr(r.Stage.Surgical), "hrt", ptr(r.Stage.HRT)),
		"score", score,
		"hot_flashes", jsonx.Obj("total", r.Flashes, "per_day", perDay, "tracked_days", r.TrackedDays),
		"night_sweats", jsonx.Obj("nights", r.SweatNights, "per_week", perWeek),
		"sleep", jsonx.Obj("avg_hours", sleepAvg, "nights", len(r.SleepHours)),
		"bleeding", jsonx.Obj("events", len(events), "days", len(r.BleedingDays), "dates", events),
		"blood_pressure", bp,
		"symptoms", jsonx.Obj("tracked_days", r.TrackedDays, "top", top),
		"treatment", jsonx.Obj("hrt_adherence_pct", ptr(r.HRTAdherencePct()), "items", items, "lifestyle", lifestyle),
		"side_effects", effects,
		"supplements", supplements,
	)
}

func nullInt16(n sql.NullInt16) any {
	if !n.Valid {
		return nil
	}
	return int(n.Int16)
}

// --- builder section ------------------------------------------------------------------------------------------------

// ReportSection is the `menopause` section of bloom's report builder (healthrecord.SectionProvider). It is a report
// section only: the record summary (no section filter) leaves it out, and it is built for a user in menopause mode
// only. Both audiences get the same content (no ids, notes or free text other than her item names), never editable.
type ReportSection struct {
	svc    *Service
	labels Labels
}

// NewReportSection wires the section; labels names the symptoms (log-taxonomy namespace; nil = keys).
func NewReportSection(svc *Service, labels Labels) *ReportSection {
	return &ReportSection{svc: svc, labels: labels}
}

var _ healthrecord.SectionProvider = (*ReportSection)(nil)

// Key implements healthrecord.SectionProvider.
func (p *ReportSection) Key() string { return healthrecord.SectionMenopause }

// Section implements healthrecord.SectionProvider: the window is the builder's (custom From, else its range).
func (p *ReportSection) Section(ctx context.Context, userID uint64, _ healthrecord.Audience, opts healthrecord.Options,
) (healthrecord.Section, bool, error) {
	if opts.Sections == nil { // the summary screen, not a report
		return healthrecord.Section{}, false, nil
	}
	mode, err := p.svc.q.GetStoredLifeMode(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return healthrecord.Section{}, false, fmt.Errorf("menopause: life mode: %w", err)
	}
	if mode.String != string(enums.LifeModeMenopause) {
		return healthrecord.Section{}, false, nil
	}
	from := opts.From
	if from.IsZero() || from.After(opts.Today) {
		from = analysis.NewRange(opts.CycleRange, opts.Today).From
	}
	r, err := p.svc.Report(ctx, userID, from, opts.Today)
	if err != nil {
		return healthrecord.Section{}, false, err
	}
	return healthrecord.Section{
		Key: healthrecord.SectionMenopause, Empty: r.Empty(), Data: ReportJSON(r, p.namer(opts.Locale, opts.DefaultLocale)),
	}, true, nil
}

func (p *ReportSection) namer(locale, def string) SymptomNamer {
	return SymptomNamerFor(p.labels, locale, def)
}

// SymptomNamerFor names symptoms from the log-taxonomy namespace of locale (keys when labels is nil).
func SymptomNamerFor(labels Labels, locale, def string) SymptomNamer {
	if labels == nil {
		return nil
	}
	c := analysis.NewCopy(nil, labels.NamespaceMessages(locale, healthlog.TaxonomyNamespace, def))
	return c.SymptomLabel
}
