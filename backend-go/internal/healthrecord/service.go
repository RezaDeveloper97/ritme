package healthrecord

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/profile"
	"github.com/ritme/backend-go/internal/vitals"
)

// ErrNotFound: no such manual pregnancy of this user (a foreign id is the same 404).
var ErrNotFound = errors.New("healthrecord: pregnancy entry not found")

// ErrTooMany: the user already has MaxManualPregnancy manual entries.
var ErrTooMany = errors.New("healthrecord: too many pregnancy entries")

// LabsSource gives the record's lab rows (internal/labs.Service.RecordLabs).
type LabsSource interface {
	RecordLabs(ctx context.Context, userID uint64, locale, def string, limit int) ([]*jsonx.OrderedMap, error)
}

// SectionProvider adds a section the record does not build itself (roadmap CB-REC documents, CB-MENO report data).
// Section returns ok=false to leave the section out. A provider must honour the audience like the built-in sections.
type SectionProvider interface {
	Key() string
	Section(ctx context.Context, userID uint64, aud Audience, opts Options) (Section, bool, error)
}

// Service builds and edits the record. Every query is scoped by the user id it is given.
type Service struct {
	q         *store.Queries
	care      carestore.Querier
	cycle     *cycleservice.Service
	logs      *healthlog.Service
	vitals    *vitals.Service
	labs      LabsSource
	bundles   *i18n.TranslationStore
	providers []SectionProvider
}

// WithBundles wires the translation bundles the cycle section labels its symptoms from (the `analysis` and log
// taxonomy namespaces, like /analysis/symptoms); without them a label is its key.
func (s *Service) WithBundles(b *i18n.TranslationStore) *Service {
	s.bundles = b
	return s
}

func (s *Service) copy(opts Options) *analysis.Copy {
	if s.bundles == nil {
		return nil
	}
	return analysis.NewCopy(s.bundles.NamespaceMessages(opts.Locale, analysis.Namespace, opts.DefaultLocale),
		s.bundles.NamespaceMessages(opts.Locale, healthlog.TaxonomyNamespace, opts.DefaultLocale))
}

// NewService wires the service on dbtx. labs may be nil (the labs section stays empty).
func NewService(dbtx *sql.DB, labs LabsSource) *Service {
	return &Service{
		q: store.New(dbtx), care: carestore.New(dbtx), cycle: cycleservice.New(dbtx, nil),
		logs: healthlog.NewService(dbtx), vitals: vitals.NewService(dbtx), labs: labs,
	}
}

// WithProviders appends extra sections (built after the built-in ones, in order).
func (s *Service) WithProviders(p ...SectionProvider) *Service {
	s.providers = append(s.providers, p...)
	return s
}

// Options steer one Build. Zero values take the screen defaults.
type Options struct {
	Today                 civildate.Date
	Locale, DefaultLocale string
	// Sections limits the built sections (nil = every section, providers included).
	Sections []string
	// VitalsDays is the vitals window ending Today (default 30); CycleRange an analysis range key (default 6m).
	VitalsDays int
	CycleRange string
	// Checkups / Labs cap the listed rows (default 5 each).
	Checkups, Labs int
	// From, when set, is a custom report window [From, Today] (B-N6-04): it replaces CycleRange and VitalsDays.
	From civildate.Date
}

func (o Options) withDefaults() Options {
	if !o.From.IsZero() && !o.From.After(o.Today) {
		o.VitalsDays = o.From.DiffDays(o.Today) + 1
	}
	if o.VitalsDays <= 0 {
		o.VitalsDays = DefaultVitalsDays
	}
	if o.CycleRange == "" {
		o.CycleRange = DefaultCycleRange
	}
	if o.Checkups <= 0 {
		o.Checkups = DefaultCheckupsList
	}
	if o.Labs <= 0 {
		o.Labs = DefaultLabsList
	}
	return o
}

func (o Options) wants(key string) bool { return o.Sections == nil || slices.Contains(o.Sections, key) }

// Section is one card of the record.
type Section struct {
	Key      string
	Editable bool
	Empty    bool
	Data     *jsonx.OrderedMap
}

// JSON is {key, editable, empty, data}.
func (s Section) JSON() *jsonx.OrderedMap {
	return jsonx.Obj("key", s.Key, "editable", s.Editable, "empty", s.Empty, "data", s.Data)
}

// Record is a built record.
type Record struct {
	Audience  Audience
	Today     civildate.Date
	UpdatedAt time.Time
	Person    *jsonx.OrderedMap
	Sections  []Section
}

// JSON is the GET /health-record body.
func (r *Record) JSON() *jsonx.OrderedMap {
	var updated any
	if !r.UpdatedAt.IsZero() {
		updated = jsonx.DateTime(r.UpdatedAt)
	}
	sections := make([]*jsonx.OrderedMap, 0, len(r.Sections))
	for _, s := range r.Sections {
		sections = append(sections, s.JSON())
	}
	return jsonx.Obj("audience", string(r.Audience), "date", r.Today.String(), "updated_at", updated,
		"person", r.Person, "sections", sections)
}

// Section returns the built section key (ok=false when it was not built).
func (r *Record) Section(key string) (Section, bool) {
	for _, s := range r.Sections {
		if s.Key == key {
			return s, true
		}
	}
	return Section{}, false
}

// facts are the rows several sections read.
type facts struct {
	name      sql.NullString
	profile   *store.GetRecordProfileRow
	life      *store.GetRecordLifeProfileRow
	record    *store.HealthRecord
	pregnancy *store.GetRecordPregnancyRow
}

func optional[T any](v T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) facts(ctx context.Context, userID uint64) (facts, error) {
	var f facts
	var err error
	if f.name, err = s.q.GetRecordUser(ctx, userID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return f, fmt.Errorf("healthrecord: user: %w", err)
	}
	if f.profile, err = optional(s.q.GetRecordProfile(ctx, userID)); err != nil {
		return f, fmt.Errorf("healthrecord: profile: %w", err)
	}
	if f.life, err = optional(s.q.GetRecordLifeProfile(ctx, userID)); err != nil {
		return f, fmt.Errorf("healthrecord: life profile: %w", err)
	}
	if f.record, err = optional(s.q.GetHealthRecord(ctx, userID)); err != nil {
		return f, fmt.Errorf("healthrecord: record: %w", err)
	}
	if f.pregnancy, err = optional(s.q.GetRecordPregnancy(ctx, userID)); err != nil {
		return f, fmt.Errorf("healthrecord: pregnancy: %w", err)
	}
	return f, nil
}

func (f facts) mode() enums.LifeMode {
	var gender, stored, goal string
	if f.life != nil {
		gender, stored = f.life.Gender.String, f.life.LifeMode.String
	}
	if f.profile != nil {
		goal = f.profile.UserGoal
	}
	return enums.ResolveAccountMode(gender, stored, f.pregnancy != nil && f.pregnancy.PregnancyMode, goal)
}

func (f facts) person(today civildate.Date) *jsonx.OrderedMap {
	var name, age, gender any
	if f.name.Valid && f.name.String != "" {
		name = f.name.String
	}
	if f.profile != nil && f.profile.Birthday.Valid {
		age = f.profile.Birthday.Date.AgeOn(today)
	}
	if f.life != nil && f.life.Gender.Valid {
		gender = f.life.Gender.String
	}
	return jsonx.Obj("name", name, "age", age, "gender", gender, "life_mode", string(f.mode()))
}

func newest(ts ...sql.NullTime) time.Time {
	var out time.Time
	for _, t := range ts {
		if t.Valid && t.Time.After(out) {
			out = t.Time
		}
	}
	return out
}

// Build builds the record of userID for aud.
func (s *Service) Build(ctx context.Context, userID uint64, aud Audience, opts Options) (*Record, error) {
	opts = opts.withDefaults()
	f, err := s.facts(ctx, userID)
	if err != nil {
		return nil, err
	}
	rec := &Record{Audience: aud, Today: opts.Today, Person: f.person(opts.Today)}
	stamps := []sql.NullTime{}
	if f.profile != nil {
		stamps = append(stamps, f.profile.UpdatedAt)
	}
	if f.life != nil {
		stamps = append(stamps, f.life.UpdatedAt)
	}
	if f.record != nil {
		stamps = append(stamps, f.record.UpdatedAt)
	}
	owner := aud == AudienceOwner
	builders := []struct {
		key   string
		build func() (Section, error)
	}{
		{SectionBasics, func() (Section, error) { return basicsSection(f, owner), nil }},
		{SectionConditions, func() (Section, error) { return conditionsSection(f, owner), nil }},
		{SectionMedications, func() (Section, error) {
			sec, ts, err := s.medicationsSection(ctx, userID, f, aud, opts)
			stamps = append(stamps, ts...)
			return sec, err
		}},
		{SectionAllergies, func() (Section, error) { return allergiesSection(f, owner), nil }},
		{SectionCycle, func() (Section, error) { return s.cycleSection(ctx, userID, f, opts) }},
		{SectionVitals, func() (Section, error) { return s.vitalsSection(ctx, userID, opts) }},
		{SectionPregnancies, func() (Section, error) {
			sec, ts, err := s.pregnanciesSection(ctx, userID, f, aud)
			stamps = append(stamps, ts...)
			return sec, err
		}},
		{SectionCheckups, func() (Section, error) { return s.checkupsSection(ctx, userID, opts) }},
		{SectionLabs, func() (Section, error) { return s.labsSection(ctx, userID, opts) }},
	}
	for _, b := range builders {
		if !opts.wants(b.key) {
			continue
		}
		sec, err := b.build()
		if err != nil {
			return nil, err
		}
		rec.Sections = append(rec.Sections, sec)
	}
	for _, p := range s.providers {
		if !opts.wants(p.Key()) {
			continue
		}
		sec, ok, err := p.Section(ctx, userID, aud, opts)
		if err != nil {
			return nil, err
		}
		if ok {
			if !owner {
				sec.Editable = false
			}
			rec.Sections = append(rec.Sections, sec)
		}
	}
	rec.UpdatedAt = newest(stamps...)
	return rec, nil
}

// --- sections -------------------------------------------------------------------------------------------------------

func decimalOf(s sql.NullString) (float64, bool) {
	if !s.Valid {
		return 0, false
	}
	f, err := strconv.ParseFloat(s.String, 64)
	return f, err == nil && f > 0
}

// bloodType is the user's blood type and where it came from (record | pregnancy), "" when unknown.
func (f facts) bloodType() (string, string) {
	if f.record != nil && f.record.BloodType.Valid && slices.Contains(BloodTypes, f.record.BloodType.String) {
		return f.record.BloodType.String, "record"
	}
	if p := f.pregnancy; p != nil && p.BloodType.Valid && p.RhFactor.Valid {
		sign := ""
		switch enums.RhFactor(p.RhFactor.String) {
		case enums.RhFactorPositive:
			sign = "+"
		case enums.RhFactorNegative:
			sign = "-"
		}
		if v := p.BloodType.String + sign; sign != "" && slices.Contains(BloodTypes, v) {
			return v, "pregnancy"
		}
	}
	return "", ""
}

func basicsSection(f facts, owner bool) Section {
	var height, weight, bmi, blood, source any
	h, w := 0, 0.0
	if p := f.profile; p != nil {
		if p.Height.Valid && p.Height.Int16 > 0 {
			h = int(p.Height.Int16)
			height = h
		}
		if v, ok := decimalOf(p.Weight); ok {
			w = v
			weight = phpround.Round(v, 1)
		}
	}
	if raw, ok := profile.RawBmi(w, h); ok {
		bmi = jsonx.Obj("value", phpround.Round(raw, 1), "category", string(profile.BmiCategoryOf(raw)))
	}
	if v, src := f.bloodType(); v != "" {
		blood, source = v, src
	}
	return Section{
		Key: SectionBasics, Editable: owner, Empty: height == nil && weight == nil && blood == nil,
		Data: jsonx.Obj("height_cm", height, "weight_kg", weight, "bmi", bmi, "blood_type", blood,
			"blood_type_source", source),
	}
}

// codes decodes a JSON list column: nil when NULL (never answered), [] when answered «none».
func codes(raw db.NullRawJSON) []string {
	if !raw.Valid {
		return nil
	}
	out := []string{}
	_ = json.Unmarshal(raw.V, &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func listOrNil(xs []string) any {
	if xs == nil {
		return nil
	}
	return xs
}

func conditionsSection(f facts, owner bool) Section {
	var chronic, gyn []string
	if f.life != nil {
		chronic, gyn = codes(f.life.ChronicIllnesses), codes(f.life.GynConditions)
	}
	return Section{
		Key: SectionConditions, Editable: owner, Empty: len(chronic) == 0 && len(gyn) == 0,
		Data: jsonx.Obj("chronic_illnesses", listOrNil(chronic), "gyn_conditions", listOrNil(gyn),
			"answered", chronic != nil || gyn != nil),
	}
}

func allergiesSection(f facts, owner bool) Section {
	var items []string
	if f.record != nil {
		items = codes(f.record.Allergies)
	}
	return Section{
		Key: SectionAllergies, Editable: owner, Empty: len(items) == 0,
		Data: jsonx.Obj("items", listOrNil(items), "answered", items != nil),
	}
}

func (s *Service) medicationsSection(ctx context.Context, userID uint64, f facts, aud Audience, opts Options,
) (Section, []sql.NullTime, error) {
	rows, err := s.care.ListActiveMedications(ctx, userID)
	if err != nil {
		return Section{}, nil, fmt.Errorf("healthrecord: medications: %w", err)
	}
	var stamps []sql.NullTime
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		m := care.ParseMedication(r)
		stamps = append(stamps, r.UpdatedAt)
		var dose, notes any
		if sub := m.DisplaySubtitle(opts.Locale); sub.Valid {
			dose = sub.String
		}
		if aud == AudienceOwner && r.Notes.Valid && r.Notes.String != "" {
			notes = r.Notes.String
		}
		items = append(items, jsonx.Obj("id", r.ID, "title", r.Title, "dose", dose, "form", m.Meta.Form,
			"recurrence", m.Meta.Recurrence(), "weekdays", m.Meta.Weekdays, "times", m.Meta.Times, "notes", notes))
	}
	var listed []string
	if f.life != nil {
		listed = codes(f.life.Medications)
	}
	return Section{
		Key: SectionMedications, Editable: aud == AudienceOwner, Empty: len(items) == 0 && len(listed) == 0,
		Data: jsonx.Obj("items", items, "profile_medications", listOrNil(listed)),
	}, stamps, nil
}

func (s *Service) cycleSection(ctx context.Context, userID uint64, f facts, opts Options) (Section, error) {
	today := opts.Today
	sn, err := s.cycle.Load(ctx, userID, today, today, today)
	if err != nil {
		return Section{}, err
	}
	in := &analysis.Input{Today: today, Range: analysis.NewRange(opts.CycleRange, today), Histories: sn.Histories,
		Profile: sn.EngineProfile(), Days: map[civildate.Date]*analysis.Day{}}
	if !opts.From.IsZero() && !opts.From.After(today) {
		in.Range = analysis.Range{Key: ReportRangeCustom, From: opts.From, To: today}
	}
	if f.profile != nil {
		if f.profile.Height.Valid {
			in.HeightCM = int(f.profile.Height.Int16)
		}
		if f.profile.Birthday.Valid {
			in.Birthday = f.profile.Birthday.Date
		}
	}
	rows, err := s.logs.Range(ctx, userID, in.Range.From, today)
	if err != nil {
		return Section{}, err
	}
	dayRows := make([]analysis.DayEntries, len(rows))
	for i, r := range rows {
		dayRows[i] = analysis.DayEntries{Date: r.Date, Entries: r.Entries}
	}
	in.Days = analysis.BuildDays(dayRows)
	c := analysis.BuildCycle(in)
	syms := analysis.BuildSymptoms(in)

	var last civildate.Date
	if c.Current != nil {
		last = c.Current.Start
	}
	for _, h := range sn.Histories {
		if !h.PeriodStart.After(today) && h.PeriodStart.After(last) {
			last = h.PeriodStart
		}
	}
	var lastStart, currentDay any
	if !last.IsZero() {
		lastStart = last.String()
	}
	if c.Current != nil {
		currentDay = c.CurrentDay
	}
	cp := s.copy(opts)
	top := make([]*jsonx.OrderedMap, 0, TopSymptoms)
	for _, t := range syms.Top {
		if len(top) >= TopSymptoms {
			break
		}
		if analysis.IsMoodKey(t.Key) {
			continue
		}
		top = append(top, jsonx.Obj("key", t.Key, "label", cp.SymptomLabel(t.Key), "days", t.Days))
	}
	return Section{
		Key: SectionCycle, Empty: c.BasedOn == 0 && last.IsZero(),
		Data: jsonx.Obj(
			"range", in.Range.JSON(),
			"based_on", c.BasedOn,
			"median_cycle", intOrNil(c.MedianCycle),
			"variation", intOrNil(c.Variation),
			"median_period", intOrNil(c.MedianPeriod),
			"regularity", c.Regularity,
			"cycle_status", strOrNil(c.CycleStatus),
			"period_status", strOrNil(c.PeriodStatus),
			"last_period_start", lastStart,
			"current_day", currentDay,
			"top_symptoms", top,
		),
	}, nil
}

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Service) vitalsSection(ctx context.Context, userID uint64, opts Options) (Section, error) {
	from := opts.Today.AddDays(1 - opts.VitalsDays)
	rs, err := s.vitals.Merged(ctx, userID, "", from, opts.Today)
	if err != nil {
		return Section{}, err
	}
	data, has := VitalsSummary(rs, from, opts.Today)
	return Section{Key: SectionVitals, Empty: !has, Data: data}, nil
}

func (s *Service) pregnanciesSection(ctx context.Context, userID uint64, f facts, aud Audience,
) (Section, []sql.NullTime, error) {
	t := TrackedPregnancies{}
	if p := f.pregnancy; p != nil && p.PregnancyMode {
		t.Active = true
		if p.EstimatedDueDate.Valid {
			t.Due = p.EstimatedDueDate.Date
		}
	}
	birth, err := optional(s.q.GetRecordBirth(ctx, userID))
	if err != nil {
		return Section{}, nil, fmt.Errorf("healthrecord: birth: %w", err)
	}
	t.Birth = birth
	losses, err := s.q.CountRecordLosses(ctx, userID)
	if err != nil {
		return Section{}, nil, fmt.Errorf("healthrecord: losses: %w", err)
	}
	t.LossCount = int(losses)
	if t.ManualRows, err = s.q.ListManualPregnancies(ctx, userID); err != nil {
		return Section{}, nil, fmt.Errorf("healthrecord: pregnancies: %w", err)
	}
	stamps := make([]sql.NullTime, 0, len(t.ManualRows))
	for _, m := range t.ManualRows {
		stamps = append(stamps, m.UpdatedAt)
	}
	entries := t.Entries()
	return Section{
		Key: SectionPregnancies, Editable: aud == AudienceOwner, Empty: len(entries) == 0,
		Data: pregnanciesSection(entries, aud),
	}, stamps, nil
}

func (s *Service) checkupsSection(ctx context.Context, userID uint64, opts Options) (Section, error) {
	rows, err := s.q.ListRecordCheckups(ctx, store.ListRecordCheckupsParams{UserID: userID, Limit: int32(opts.Checkups)}) //nolint:gosec // small cap
	if err != nil {
		return Section{}, fmt.Errorf("healthrecord: checkups: %w", err)
	}
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		var key, icon any
		if r.CheckupKey.Valid && r.CheckupKey.String != "" {
			key = r.CheckupKey.String
		}
		if r.CheckupIcon.Valid && r.CheckupIcon.String != "" {
			icon = r.CheckupIcon.String
		}
		items = append(items, jsonx.Obj("id", r.ID, "title", i18n.PickString(r.CheckupTitle, opts.Locale, opts.DefaultLocale),
			"key", key, "icon", icon, "done_on", r.DoneOn.String(), "result", r.Result))
	}
	return Section{Key: SectionCheckups, Empty: len(items) == 0, Data: jsonx.Obj("items", items)}, nil
}

func (s *Service) labsSection(ctx context.Context, userID uint64, opts Options) (Section, error) {
	items := []*jsonx.OrderedMap{}
	if s.labs != nil {
		var err error
		if items, err = s.labs.RecordLabs(ctx, userID, opts.Locale, opts.DefaultLocale, opts.Labs); err != nil {
			return Section{}, err
		}
	}
	return Section{Key: SectionLabs, Empty: len(items) == 0, Data: jsonx.Obj("items", items)}, nil
}

// --- writes ---------------------------------------------------------------------------------------------------------

// BasicsInput is a validated PUT /health-record/basics: only the Set* fields change.
type BasicsInput struct {
	SetBloodType bool
	BloodType    string // "" = clear
	SetAllergies bool
	Allergies    []string // nil = clear (never answered), [] = «ندارم»
}

// SaveBasics applies in to the user's health_records row (created on first save).
func (s *Service) SaveBasics(ctx context.Context, userID uint64, in BasicsInput, now time.Time) error {
	cur, err := optional(s.q.GetHealthRecord(ctx, userID))
	if err != nil {
		return fmt.Errorf("healthrecord: record: %w", err)
	}
	p := store.UpsertHealthRecordParams{UserID: userID, Now: sql.NullTime{Time: now, Valid: true}}
	if cur != nil {
		p.BloodType, p.Allergies = cur.BloodType, cur.Allergies
	}
	if in.SetBloodType {
		p.BloodType = sql.NullString{String: in.BloodType, Valid: in.BloodType != ""}
	}
	if in.SetAllergies {
		p.Allergies = db.NullRawJSON{}
		if in.Allergies != nil {
			b, _ := json.Marshal(in.Allergies)
			p.Allergies = db.NullRawJSON{V: b, Valid: true}
		}
	}
	if err := s.q.UpsertHealthRecord(ctx, p); err != nil {
		return fmt.Errorf("healthrecord: save record: %w", err)
	}
	return nil
}

// PregnancyInput is a validated manual pregnancy.
type PregnancyInput struct {
	Outcome   string
	EndedOn   civildate.Date // zero = unknown
	BabyCount int            // 0 = not told (always 0 for «ended»)
}

func (in PregnancyInput) columns() (civildate.NullDate, sql.NullInt16) {
	var d civildate.NullDate
	if !in.EndedOn.IsZero() {
		d = civildate.NullDate{Date: in.EndedOn, Valid: true}
	}
	var n sql.NullInt16
	if in.BabyCount > 0 && in.Outcome != OutcomeEnded {
		n = sql.NullInt16{Int16: int16(in.BabyCount), Valid: true} //nolint:gosec // validated ≤ 4
	}
	return d, n
}

// GetPregnancy is one manual entry of the user.
func (s *Service) GetPregnancy(ctx context.Context, userID, id uint64) (store.HealthRecordPregnancy, error) {
	row, err := s.q.GetManualPregnancy(ctx, store.GetManualPregnancyParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return row, ErrNotFound
	}
	if err != nil {
		return row, fmt.Errorf("healthrecord: get pregnancy: %w", err)
	}
	return row, nil
}

// CreatePregnancy adds a manual entry (ErrTooMany past MaxManualPregnancy).
func (s *Service) CreatePregnancy(ctx context.Context, userID uint64, in PregnancyInput, now time.Time,
) (store.HealthRecordPregnancy, error) {
	n, err := s.q.CountManualPregnancies(ctx, userID)
	if err != nil {
		return store.HealthRecordPregnancy{}, fmt.Errorf("healthrecord: count pregnancies: %w", err)
	}
	if n >= MaxManualPregnancy {
		return store.HealthRecordPregnancy{}, ErrTooMany
	}
	d, babies := in.columns()
	id, err := s.q.InsertManualPregnancy(ctx, store.InsertManualPregnancyParams{
		UserID: userID, Outcome: in.Outcome, EndedOn: d, BabyCount: babies, Now: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return store.HealthRecordPregnancy{}, fmt.Errorf("healthrecord: insert pregnancy: %w", err)
	}
	return s.GetPregnancy(ctx, userID, uint64(id)) //nolint:gosec // auto-increment id
}

// UpdatePregnancy replaces a manual entry of the user.
func (s *Service) UpdatePregnancy(ctx context.Context, userID, id uint64, in PregnancyInput, now time.Time,
) (store.HealthRecordPregnancy, error) {
	if _, err := s.GetPregnancy(ctx, userID, id); err != nil {
		return store.HealthRecordPregnancy{}, err
	}
	d, babies := in.columns()
	if _, err := s.q.UpdateManualPregnancy(ctx, store.UpdateManualPregnancyParams{
		ID: id, UserID: userID, Outcome: in.Outcome, EndedOn: d, BabyCount: babies,
		Now: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return store.HealthRecordPregnancy{}, fmt.Errorf("healthrecord: update pregnancy: %w", err)
	}
	return s.GetPregnancy(ctx, userID, id)
}

// DeletePregnancy removes a manual entry of the user.
func (s *Service) DeletePregnancy(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteManualPregnancy(ctx, store.DeleteManualPregnancyParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("healthrecord: delete pregnancy: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
