package voicelog

import (
	"context"
	"errors"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/conditions"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/pelvic"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Canvas items (CB-VOICE-01): what voice understands beyond the log taxonomy. Each is a field of another domain's
// own diary and is saved through that domain's service on POST /logs/voice/commit — never by SQL here:
//
//	hot_flash.*   menopause.Service.StartFlash   (hot_flashes; menopause mode)
//	pain_diary.*  conditions.Service.SavePain    (condition_pain_entries + the pain log; endometriosis program)
//	pill.status   contraception.Service.LogPill  (contraception_pill_logs; a pill method)
//	bladder.*     pelvic.Service.SaveDiary       (pelvic_bladder_logs; everyone)
//
// A suggestion of a canvas item carries target = its category; taxonomy suggestions carry target "log" and are
// still saved by the client through PUT /logs/days/{date} with voice_params.

// Targets of a suggestion.
const (
	TargetLog       = "log"
	TargetHotFlash  = "hot_flash"
	TargetPainDiary = "pain_diary"
	TargetPill      = "pill"
	TargetBladder   = "bladder"
)

// Voice-only limits of the canvas items.
const (
	// MaxVoiceFlashes caps the hot flashes one sentence may log («سه بار گرگرفتگی»).
	MaxVoiceFlashes = 10
	// MaxFlashMinutes is the longest single hot flash a sentence may state (menopause.MaxFlashSeconds).
	MaxFlashMinutes = menopause.MaxFlashSeconds / 60
	// DefaultFlashSeconds is the length given to a hot flash said without one (a flash in the diary always has a
	// length; a voice note is not a running timer). [needs clinical review]
	DefaultFlashSeconds = 3 * 60
	// MaxNightVoids mirrors the bladder diary's night-voids cap (PUT /pelvic/diary).
	MaxNightVoids = 20
	// MaxCommitDays: a voice note may be committed for today or up to this many days back (the hot-flash backfill
	// window, menopause.MaxBackfillDays).
	MaxCommitDays = menopause.MaxBackfillDays
)

// Night and day anchors of voice hot flashes without a clock time (Tehran): «دیشب» → 03:00 of the day, else noon.
const (
	nightFlashHour = 3
	dayFlashHour   = 12
)

// Canvas field types (ai.VocabEntry.Type).
const (
	fieldInteger = "integer"
	fieldSingle  = "single"
	fieldBool    = "bool"
	fieldText    = "text"
	fieldTime    = "time"
)

// canvasField is one canvas item a suggestion may fill.
type canvasField struct {
	target, param string
	typ           string
	options       []string // single
	min, max      float64  // integer
	maxLen        int      // text
}

func (f canvasField) key() string  { return f.target + "." + f.param }
func (f canvasField) code() string { return f.target + "_" + f.param }

// canvasFields are every canvas item, in review order.
var canvasFields = []canvasField{
	{target: TargetHotFlash, param: "count", typ: fieldInteger, min: 1, max: MaxVoiceFlashes},
	{target: TargetHotFlash, param: "severity", typ: fieldSingle, options: menopause.Severities},
	{target: TargetHotFlash, param: "night", typ: fieldBool},
	{target: TargetHotFlash, param: "sweat", typ: fieldBool},
	{target: TargetHotFlash, param: "duration", typ: fieldInteger, min: 1, max: MaxFlashMinutes},
	{target: TargetPainDiary, param: "score", typ: fieldInteger, min: 0, max: conditions.MaxScore},
	{target: TargetPainDiary, param: "missed_activity", typ: fieldBool},
	{target: TargetPainDiary, param: "analgesic", typ: fieldText, maxLen: conditions.MaxAnalgesicLen},
	{target: TargetPainDiary, param: "analgesic_time", typ: fieldTime},
	{target: TargetPainDiary, param: "analgesic_effect", typ: fieldSingle, options: conditions.AnalgesicEffects},
	{target: TargetPill, param: "status", typ: fieldSingle, options: contraception.PillStatuses},
	{target: TargetBladder, param: "leak", typ: fieldSingle, options: pelvic.Leaks},
	{target: TargetBladder, param: "night_voids", typ: fieldInteger, min: 0, max: MaxNightVoids},
}

func canvasFieldOf(target, param string) (canvasField, bool) {
	i := slices.IndexFunc(canvasFields, func(f canvasField) bool { return f.target == target && f.param == param })
	if i < 0 {
		return canvasField{}, false
	}
	return canvasFields[i], true
}

// IsCanvasTarget reports whether category names a canvas target.
func IsCanvasTarget(category string) bool {
	return slices.ContainsFunc(canvasFields, func(f canvasField) bool { return f.target == category })
}

// Writers are the services the canvas items are saved through (and read for who may use them). A nil writer
// switches its target off: it is neither offered to the parser nor accepted on commit.
type Writers struct {
	Flashes FlashWriter
	Pain    PainWriter
	Pills   PillWriter
	Bladder BladderWriter
}

// FlashWriter is the hot-flash diary (satisfied by *menopause.Service).
type FlashWriter interface {
	StartFlash(ctx context.Context, userID uint64, in menopause.FlashInput, now time.Time) (menopause.Flash, bool, error)
}

// PainWriter is the endometriosis pain diary (satisfied by *conditions.Service).
type PainWriter interface {
	Enrolments(ctx context.Context, userID uint64) ([]conditions.Enrolment, error)
	Pain(ctx context.Context, userID uint64, date civildate.Date) (conditions.PainDay, error)
	SavePain(ctx context.Context, userID uint64, in conditions.PainInput, locale string, now time.Time) (conditions.PainDay, error)
}

// PillWriter is the pill pack (satisfied by *contraception.Service).
type PillWriter interface {
	Overview(ctx context.Context, userID uint64, today civildate.Date) (contraception.Overview, error)
	LogPill(ctx context.Context, userID uint64, date civildate.Date, status string, now time.Time) error
}

// BladderWriter is the bladder diary (satisfied by *pelvic.Service).
type BladderWriter interface {
	SaveDiary(ctx context.Context, userID uint64, in pelvic.DiaryInput, now time.Time) (pelvic.Diary, error)
}

// eligibility is what the user may log by voice beyond the taxonomy now.
type eligibility struct {
	targets map[string]bool
	pill    *contraception.Method // the pill method (pill target only)
}

// eligible works out the canvas targets of the user: hot flashes in menopause mode, the pain diary while enrolled
// in the endometriosis program, the pill with a pill method, the bladder diary for everyone.
func (s *Service) eligible(ctx context.Context, userID uint64, mode string, now time.Time) (eligibility, error) {
	e := eligibility{targets: map[string]bool{}}
	w := s.writers
	if w.Flashes != nil && mode == taxonomy.ModeMenopause {
		e.targets[TargetHotFlash] = true
	}
	if w.Pain != nil {
		list, err := w.Pain.Enrolments(ctx, userID)
		if err != nil {
			return eligibility{}, err
		}
		e.targets[TargetPainDiary] = slices.ContainsFunc(list, func(en conditions.Enrolment) bool {
			return en.Program == conditions.ProgramEndo && en.Enrolled
		})
	}
	if w.Pills != nil {
		o, err := w.Pills.Overview(ctx, userID, civildate.InTehran(now))
		if err != nil {
			return eligibility{}, err
		}
		if o.Method != nil {
			if _, ok := o.Method.Pack(); ok {
				e.targets[TargetPill], e.pill = true, o.Method
			}
		}
	}
	if w.Bladder != nil {
		e.targets[TargetBladder] = true
	}
	return e, nil
}

// addCanvas offers the eligible canvas fields to the parser, with labels in locale.
func (v *vocabulary) addCanvas(targets map[string]bool, locale string) {
	if v.canvas == nil {
		v.canvas = map[string]canvasField{}
	}
	for _, f := range canvasFields {
		if !targets[f.target] {
			continue
		}
		e := ai.VocabEntry{Key: f.key(), Type: f.typ, Label: T("fields."+f.code(), locale, nil), MaxLen: f.maxLen}
		for _, o := range f.options {
			e.Values = append(e.Values, ai.VocabValue{Code: o, Label: T("options."+f.code()+"_"+o, locale, nil)})
		}
		if f.typ == fieldInteger {
			e.Min, e.Max = f.min, f.max
		}
		v.canvas[f.key()] = f
		v.entries = append(v.entries, e)
	}
}

var clockTime = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)

// normalizeCanvas checks a canvas value against its field and returns it in its wire form: integer → float64,
// single → option code, bool → bool, text → trimmed string, time → "HH:MM".
func normalizeCanvas(f canvasField, raw any) (any, bool) {
	switch f.typ {
	case fieldInteger:
		var n float64
		switch x := raw.(type) {
		case float64:
			n = x
		case int:
			n = float64(x)
		case int64:
			n = float64(x)
		case string:
			p, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
			if err != nil {
				return nil, false
			}
			n = p
		default:
			return nil, false
		}
		if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n < f.min || n > f.max {
			return nil, false
		}
		return n, true
	case fieldSingle:
		code, ok := raw.(string)
		return code, ok && slices.Contains(f.options, code)
	case fieldBool:
		b, ok := raw.(bool)
		return b, ok
	case fieldText:
		s, ok := raw.(string)
		s = strings.TrimSpace(s)
		return s, ok && s != "" && utf8.RuneCountInString(s) <= f.maxLen
	case fieldTime:
		s, ok := raw.(string)
		m := clockTime.FindStringSubmatch(strings.TrimSpace(s))
		if !ok || m == nil {
			return nil, false
		}
		h, _ := strconv.Atoi(m[1])
		return strconv.Itoa(h/10) + strconv.Itoa(h%10) + ":" + m[2], true
	}
	return nil, false
}

// canvasLabel is the chip text of a canvas value: «گرگرفتگی · 3 بار», «شدت درد · 6 از 10», «نشت ادرار · با سرفه».
func canvasLabel(f canvasField, value any, locale string) string {
	switch f.typ {
	case fieldInteger:
		n, _ := value.(float64)
		return T("chips."+f.code(), locale, map[string]string{"value": num(n, locale)})
	case fieldSingle:
		code, _ := value.(string)
		return T("chips."+f.code(), locale, map[string]string{"option": T("options."+f.code()+"_"+code, locale, nil)})
	case fieldBool:
		b, _ := value.(bool)
		return T("chips."+f.code()+"_"+strconv.FormatBool(b), locale, nil)
	default:
		s, _ := value.(string)
		if f.typ == fieldTime {
			s = digits(s, locale)
		}
		return T("chips."+f.code(), locale, map[string]string{"value": s})
	}
}

// CommitItem is one reviewed canvas suggestion sent back to be saved.
type CommitItem struct {
	Category, Param string
	Value           any
}

// ItemError is a 422 on one committed item: field "category" | "param" | "value" of items.<Index>, with its
// localized message.
type ItemError struct {
	Index   int
	Field   string
	Message string
}

func (e *ItemError) Error() string {
	return "voicelog: items." + strconv.Itoa(e.Index) + "." + e.Field + ": " + e.Message
}

// Saved is one committed item.
type Saved struct {
	Target, Param string
	Value         any
	Label         string
}

// commitPlan is the validated commit, grouped per target (field → index of the item that set it).
type commitPlan struct {
	values map[string]any // "target.param" → normalized value
	index  map[string]int
	order  []canvasField
}

func (p commitPlan) has(target string) bool {
	return slices.ContainsFunc(p.order, func(f canvasField) bool { return f.target == target })
}

func (p commitPlan) get(target, param string) (any, bool) {
	v, ok := p.values[target+"."+param]
	return v, ok
}

// at is the index of the first item of target that set one of params (any item of target when none did).
func (p commitPlan) at(target string, params ...string) int {
	for _, param := range params {
		if i, ok := p.index[target+"."+param]; ok {
			return i
		}
	}
	best := -1
	for k, i := range p.index {
		if strings.HasPrefix(k, target+".") && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

// Commit saves reviewed canvas items of date through their services (CB-VOICE-01): every item is validated and
// checked against what the user may log first, the writes that can still fail are pre-checked (a pain score
// needs a pain location that day; a pill day must be a pill day of the pack), then pain diary → pill → bladder
// diary → hot flashes are written. Returns the saved items in request order (duplicates: the first wins).
func (s *Service) Commit(ctx context.Context, userID uint64, date civildate.Date, items []CommitItem, locale string, now time.Time) ([]Saved, error) {
	mode, err := s.logs.LifeMode(ctx, userID)
	if err != nil {
		return nil, err
	}
	el, err := s.eligible(ctx, userID, mode, now)
	if err != nil {
		return nil, err
	}
	plan := commitPlan{values: map[string]any{}, index: map[string]int{}}
	for i, it := range items {
		f, ok := canvasFieldOf(it.Category, it.Param)
		switch {
		case !IsCanvasTarget(it.Category):
			return nil, &ItemError{Index: i, Field: "category", Message: T("messages.invalid_item", locale, nil)}
		case !el.targets[it.Category]:
			return nil, &ItemError{Index: i, Field: "category", Message: T("messages.unavailable_"+it.Category, locale, nil)}
		case !ok:
			return nil, &ItemError{Index: i, Field: "param", Message: T("messages.invalid_item", locale, nil)}
		}
		value, ok := normalizeCanvas(f, it.Value)
		if !ok {
			return nil, &ItemError{Index: i, Field: "value", Message: T("messages.invalid_value", locale, nil)}
		}
		if _, dup := plan.values[f.key()]; dup {
			continue
		}
		plan.values[f.key()], plan.index[f.key()] = value, i
		plan.order = append(plan.order, f)
	}

	pain, err := s.prePain(ctx, userID, date, plan, locale)
	if err != nil {
		return nil, err
	}
	if err := prePill(el, date, plan, locale); err != nil {
		return nil, err
	}
	if pain != nil {
		if _, err := s.writers.Pain.SavePain(ctx, userID, *pain, locale, now); err != nil {
			return nil, painErr(err, plan, locale)
		}
	}
	if v, ok := plan.get(TargetPill, "status"); ok {
		if err := s.writers.Pills.LogPill(ctx, userID, date, v.(string), now); err != nil {
			return nil, pillErr(err, plan.at(TargetPill), locale)
		}
	}
	if plan.has(TargetBladder) {
		if _, err := s.writers.Bladder.SaveDiary(ctx, userID, bladderInput(date, plan), now); err != nil {
			return nil, err
		}
	}
	if plan.has(TargetHotFlash) {
		if err := s.saveFlashes(ctx, userID, date, plan, now); err != nil {
			return nil, err
		}
	}
	out := make([]Saved, 0, len(plan.order))
	for _, f := range plan.order {
		v := plan.values[f.key()]
		out = append(out, Saved{Target: f.target, Param: f.param, Value: v, Label: canvasLabel(f, v, locale)})
	}
	return out, nil
}

// prePain builds the pain-diary input (nil when no pain item) and checks that a score above 0 has a location
// that day (the diary's own rule, pre-checked so nothing is half-saved).
func (s *Service) prePain(ctx context.Context, userID uint64, date civildate.Date, plan commitPlan, locale string) (*conditions.PainInput, error) {
	if !plan.has(TargetPainDiary) {
		return nil, nil
	}
	in := conditions.PainInput{Date: date, Set: map[string]bool{}}
	for _, f := range plan.order {
		if f.target != TargetPainDiary {
			continue
		}
		v := plan.values[f.key()]
		in.Set[f.param] = true
		switch f.param {
		case "score":
			n := int(v.(float64))
			in.Score = &n
		case "missed_activity":
			b := v.(bool)
			in.MissedActivity = &b
		case "analgesic":
			str := v.(string)
			in.Analgesic = &str
		case "analgesic_time":
			str := v.(string)
			in.AnalgesicTime = &str
		case "analgesic_effect":
			str := v.(string)
			in.AnalgesicEffect = &str
		}
	}
	if in.Score != nil && *in.Score > 0 {
		cur, err := s.writers.Pain.Pain(ctx, userID, date)
		if err != nil {
			return nil, err
		}
		if len(cur.Locations) == 0 {
			return nil, &ItemError{Index: plan.at(TargetPainDiary, "score"), Field: "value",
				Message: conditions.T("validation.locations_required", locale)}
		}
	}
	return &in, nil
}

// prePill checks the pill day against the pack (the pill diary's own rule, pre-checked so nothing is half-saved).
func prePill(el eligibility, date civildate.Date, plan commitPlan, locale string) error {
	if _, ok := plan.get(TargetPill, "status"); !ok || el.pill == nil {
		return nil
	}
	pack, _ := el.pill.Pack()
	switch {
	case date.Before(el.pill.PackStartedOn):
		return pillErr(contraception.ErrBeforePack, plan.at(TargetPill), locale)
	case pack.DayKind(el.pill.PackStartedOn, date) == contraception.KindBreak:
		return pillErr(contraception.ErrBreakDay, plan.at(TargetPill), locale)
	}
	return nil
}

func painErr(err error, plan commitPlan, locale string) error {
	var fe *conditions.FieldError
	switch {
	case errors.Is(err, conditions.ErrNotEnrolled):
		return &ItemError{Index: plan.at(TargetPainDiary), Field: "category", Message: T("messages.unavailable_"+TargetPainDiary, locale, nil)}
	case errors.As(err, &fe):
		return &ItemError{Index: plan.at(TargetPainDiary, fe.Field), Field: "value", Message: conditions.T("validation."+fe.Key, locale)}
	}
	return err
}

func pillErr(err error, index int, locale string) error {
	key := ""
	switch {
	case errors.Is(err, contraception.ErrNoPillMethod):
		return &ItemError{Index: index, Field: "category", Message: T("messages.unavailable_"+TargetPill, locale, nil)}
	case errors.Is(err, contraception.ErrBeforePack):
		key = "date_before_pack"
	case errors.Is(err, contraception.ErrBreakDay):
		key = "break_day"
	default:
		return err
	}
	return &ItemError{Index: index, Field: "value", Message: contraception.T("validation."+key, locale)}
}

func bladderInput(date civildate.Date, plan commitPlan) pelvic.DiaryInput {
	in := pelvic.DiaryInput{Date: date, Set: map[string]bool{}}
	if v, ok := plan.get(TargetBladder, "leak"); ok {
		s := v.(string)
		in.Set["leak"], in.Leak = true, &s
	}
	if v, ok := plan.get(TargetBladder, "night_voids"); ok {
		n := int(v.(float64))
		in.Set["night_voids"], in.NightVoids = true, &n
	}
	return in
}

// saveFlashes logs count finished hot flashes on date: each DefaultFlashSeconds long unless a length was said,
// the latest starting at 03:00 («دیشب», night) or noon of the day — or now when that is still ahead — and each
// earlier one a minute before the next began.
func (s *Service) saveFlashes(ctx context.Context, userID uint64, date civildate.Date, plan commitPlan, now time.Time) error {
	count := 1
	if v, ok := plan.get(TargetHotFlash, "count"); ok {
		count = int(v.(float64))
	}
	dur := DefaultFlashSeconds
	if v, ok := plan.get(TargetHotFlash, "duration"); ok {
		dur = int(v.(float64)) * 60
	}
	in := menopause.FlashInput{Set: map[string]bool{}, Duration: &dur}
	hour := dayFlashHour
	if v, ok := plan.get(TargetHotFlash, "night"); ok {
		b := v.(bool)
		in.Night = &b
		if b {
			hour = nightFlashHour
		}
	}
	if v, ok := plan.get(TargetHotFlash, "sweat"); ok {
		b := v.(bool)
		in.Sweat = &b
	}
	if v, ok := plan.get(TargetHotFlash, "severity"); ok {
		in.Severity = v.(string)
	}
	latest := date.TehranMidnight().Add(time.Duration(hour) * time.Hour)
	if limit := now.Add(-time.Duration(dur) * time.Second); latest.After(limit) {
		latest = limit
	}
	step := time.Duration(dur+60) * time.Second
	for i := count - 1; i >= 0; i-- { // oldest first, so the diary lists them newest first
		start := latest.Add(-time.Duration(i) * step)
		one := in
		one.StartedAt = &start
		if _, _, err := s.writers.Flashes.StartFlash(ctx, userID, one, now); err != nil {
			return err
		}
	}
	return nil
}

// CommitValue reads a committed value out of a request item (phpval: int64 → float64; float64, string and bool
// as they are; anything else — an array, an object, null — is not a value).
func CommitValue(v any) any {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case float64, string, bool:
		return x
	}
	return nil
}
