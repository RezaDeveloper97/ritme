// Package safety is the safety & clinical content dashboard of the admin panel (bloom B-N9-10, D-90): aggregated
// urgent-signal counts (no user ids), the unified clinical review queue over the existing review states, the urgent-path
// health checks and the nightly safety self-test (T-M10-24, D-183; SPEC v6 §5.8).
package safety

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/pregnancy/tools"
	"github.com/ritme/backend-go/internal/vitals"
)

// ReducedKicks reports a kick session that ended without reaching tools.KickTarget within the counting window
// (nbl_Log_Kick: «اگر در ۲ ساعت کمتر از ۱۰ حرکت حس کردی …»). The signal count uses the same rule in SQL.
func ReducedKicks(kicks int, counted time.Duration) bool {
	return kicks < tools.KickTarget && counted >= tools.KickWindowMinutes*time.Minute
}

// Scenario kinds.
const (
	KindPath   = "path"   // synthetic inputs through the functions the API calls (no user data read or written)
	KindConfig = "config" // the live rule rows, emergency copy and clinical registry
)

// Probe is one synthetic input of a path scenario and the verdict it must give.
type Probe struct {
	Input string `json:"input"`
	Want  bool   `json:"want"`
	Got   bool   `json:"got"`
}

// Detail is what a scenario checked: the threshold in force, its probes and every problem found (all synthetic; never
// a user value).
type Detail struct {
	Threshold string   `json:"threshold,omitempty"`
	Probes    []Probe  `json:"probes,omitempty"`
	Problems  []string `json:"problems,omitempty"`
}

// Passed reports whether every probe gave its verdict and nothing was wrong.
func (d *Detail) Passed() bool {
	if len(d.Problems) > 0 {
		return false
	}
	for _, p := range d.Probes {
		if p.Want != p.Got {
			return false
		}
	}
	return true
}

// Failures are the failing probes and the problems, one line each.
func (d *Detail) Failures() []string {
	var out []string
	for _, p := range d.Probes {
		if p.Want != p.Got {
			out = append(out, fmt.Sprintf("%s: want %t, got %t", p.Input, p.Want, p.Got))
		}
	}
	return append(out, d.Problems...)
}

func (d *Detail) probe(input string, want, got bool) {
	d.Probes = append(d.Probes, Probe{input, want, got})
}

func (d *Detail) fail(format string, args ...any) {
	if p := fmt.Sprintf(format, args...); !slices.Contains(d.Problems, p) {
		d.Problems = append(d.Problems, p)
	}
}

// Scenario is one self-test scenario.
type Scenario struct {
	Name string
	Kind string
	Run  func(ctx context.Context, e *Env) Detail
}

// PregEngine evaluates one pregnancy alert rule (preg.*) on an input and renders it in a language.
type PregEngine interface {
	Fire(key string, in *facts.PregnancyAlertInput, locale string) (inboxalerts.Fired, bool)
}

// codeHeld is the engine of the code-held definitions (inboxalerts.Fires): the shipped rules, no database.
type codeHeld struct{}

func (codeHeld) Fire(key string, in *facts.PregnancyAlertInput, locale string) (inboxalerts.Fired, bool) {
	return inboxalerts.Fires(key, in, locale)
}

// Env is what one run reads: the day whose clinical config applies, the languages every emergency text must cover,
// the pregnancy rule engine and the live copy. PureEnv needs no database; the nightly run builds a live one.
type Env struct {
	Day     civildate.Date
	Langs   []string // languages to render (default first)
	Default string
	Preg    PregEngine
	Copy    guide.Payloader // live postpartum copy (nil = embedded)
	Live    *Live           // live configuration (nil = the config checks fail with «no database»)
}

// PureEnv runs the path scenarios over the code-held rules and the embedded copy in every embedded language.
func PureEnv(day civildate.Date) *Env {
	langs := inboxalerts.Languages()
	def := ""
	if len(langs) > 0 {
		def = langs[0]
	}
	return &Env{Day: day, Langs: langs, Default: def, Preg: codeHeld{}}
}

func (e *Env) copyFor(locale string) *guide.Copy { return guide.NewCopy(e.Copy, locale, e.Default) }

// ---- helpers ----

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func contractions(n int, every, last time.Duration) []labor.Contraction {
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	out := make([]labor.Contraction, n)
	for i := range out {
		s := start.Add(time.Duration(i) * every)
		out[i] = labor.Contraction{Start: s, End: s.Add(last)}
	}
	return out
}

// runOf is enough contractions every `every` for the tail to span p.Run.
func runOf(p labor.Params, every, last time.Duration) []labor.Contraction {
	n := 2
	if every > 0 {
		n = int(math.Ceil(float64(p.Run)/float64(every))) + 2
	}
	return contractions(n, every, last)
}

// epdsTotal is a full check with item 10 = 0 and the other nine items summing to total (≤ 27) on day.
func epdsTotal(day civildate.Date, total int) postpartum.Result {
	answers := map[string]int{postpartum.SelfHarmItem: 0}
	left := total
	for _, it := range postpartum.Items {
		if it.Code == postpartum.SelfHarmItem {
			continue
		}
		v := min(left, postpartum.ItemMax)
		answers[it.Code], left = v, left-v
	}
	return postpartum.ScoreOn(postpartum.KindFull, answers, day)
}

// hasCall reports whether a message carries a call action to number.
func hasCall(msg any, number string) bool {
	m, ok := msg.(*jsonx.OrderedMap)
	if !ok {
		return false
	}
	raw, _ := m.Get("actions")
	actions, _ := raw.([]*jsonx.OrderedMap)
	for _, a := range actions {
		if n, _ := a.Get("number"); n == number {
			return true
		}
	}
	return false
}

// recoveryAlert runs the real recovery alert builder on r in every language and reports whether it raised key with
// a call to 115 (urgent when the key is a red flag); a text missing in a language is a problem.
func (e *Env) recoveryAlert(ctx context.Context, d *Detail, r postpartum.Recovery, key string) bool {
	fired := false
	for i, loc := range e.Langs {
		alerts, err := postpartum.RecoveryAlerts(ctx, e.copyFor(loc), r, e.Day)
		if err != nil {
			d.fail("%s: %v", key, err)
			return false
		}
		var hit *jsonx.OrderedMap
		for _, raw := range alerts {
			if a, ok := raw.(*jsonx.OrderedMap); ok {
				if k, _ := a.Get("key"); k == key {
					hit = a
				}
			}
		}
		if i == 0 {
			fired = hit != nil
		} else if (hit != nil) != fired {
			d.fail("%s: raised in %s but not in %s", key, e.Langs[0], loc)
		}
		if hit == nil {
			continue
		}
		title, _ := hit.Get("title")
		level, _ := hit.Get("level")
		act, _ := hit.Get("action")
		am, _ := act.(*jsonx.OrderedMap)
		number := any(nil)
		if am != nil {
			number, _ = am.Get("number")
		}
		if title == "" || number != postpartum.EmergencyNumber {
			d.fail("%s: %s alert without a title or a call to %s", key, loc, postpartum.EmergencyNumber)
		}
		if postpartum.IsUrgentAlert(key) && level != postpartum.LevelUrgent {
			d.fail("%s: level %v, want %s", key, level, postpartum.LevelUrgent)
		}
	}
	return fired
}

// pregnant are the alert-engine facts of a pregnancy at weeks+0 on day, evaluated at 12:00.
func pregnant(day civildate.Date, weeks int) pregnancyalerts.Facts {
	return pregnancyalerts.Facts{Today: day, Week: weeks + 1, TotalDays: weeks * 7, Days: map[civildate.Date]pregnancyalerts.Day{},
		Now: day.TehranMidnight().Add(12 * time.Hour)}
}

// kickSession: kicks counted in a session started at 09:00 on day, ended after `after`, with the target-th kick at
// targetAt (0 = not reached).
func kickSession(day civildate.Date, kicks int, after, targetAt time.Duration) pregnancyalerts.Kick {
	start := day.TehranMidnight().Add(9 * time.Hour)
	k := pregnancyalerts.Kick{ID: 1, Start: start, End: start.Add(after), Count: kicks}
	if targetAt > 0 {
		k.TargetAt = start.Add(targetAt)
	}
	return k
}

// fire runs pregnancy rule key on f in every language and reports whether it fired (in the first language); a rule
// that fires in one language and not another, or renders without its title, evidence or (urgent rules) contact line
// is a problem.
func (e *Env) fire(d *Detail, key string, f pregnancyalerts.Facts) bool {
	def, ok := inboxalerts.DefOf(key)
	if !ok {
		d.fail("%s: unknown rule", key)
		return false
	}
	urgent := def.Level == rules.LevelFollowSoon
	in := &facts.PregnancyAlertInput{Facts: f}
	fired := false
	for i, loc := range e.Langs {
		m, ok := e.Preg.Fire(key, in, loc)
		if i == 0 {
			fired = ok
		} else if ok != fired {
			d.fail("%s: fires in %s but not in %s", key, e.Langs[0], loc)
		}
		if !ok {
			continue
		}
		if m.Title == "" || m.Seen == "" {
			d.fail("%s: %s text without a title or evidence", key, loc)
		}
		if urgent && (m.Level != rules.LevelFollowSoon || m.Ask == "") {
			d.fail("%s: %s not urgent or without a contact line", key, loc)
		}
	}
	if len(e.Langs) == 0 {
		d.fail("no language to render")
	}
	return fired
}

// boundary probes v-step, v and v+step with want(x) on got(x).
func boundary(d *Detail, v, step float64, label func(float64) string, want, got func(float64) bool) {
	for _, x := range []float64{v - step, v, v + step} {
		d.probe(label(x), want(x), got(x))
	}
}

// ---- path scenarios ----

func bpLabel(sys, dia float64) string { return num(sys) + "/" + num(dia) }

func vitalsBP(d civildate.Date, sys, dia float64) string {
	return vitals.AlertRule(vitals.Reading{Type: vitals.TypeBP, Date: d, Systolic: sys, Diastolic: dia})
}

// PathScenarios run synthetic inputs at threshold − 1, the threshold and threshold + 1 (the clinical config valid on
// the run's day) through the functions the API calls.
var PathScenarios = []Scenario{
	{"vitals.bp_crisis_systolic", KindPath, func(_ context.Context, e *Env) Detail {
		t := vitals.ThresholdsOn(e.Day)
		dia := t.BPCrisisDiastolic - 1
		d := Detail{Threshold: "systolic > " + num(t.BPCrisisSystolic)}
		boundary(&d, t.BPCrisisSystolic, 1, func(x float64) string { return bpLabel(x, dia) },
			func(x float64) bool { return x > t.BPCrisisSystolic },
			func(x float64) bool { return vitalsBP(e.Day, x, dia) == vitals.RuleBPCrisis })
		return d
	}},
	{"vitals.bp_crisis_diastolic", KindPath, func(_ context.Context, e *Env) Detail {
		t := vitals.ThresholdsOn(e.Day)
		sys := t.BPCrisisSystolic - 1
		d := Detail{Threshold: "diastolic > " + num(t.BPCrisisDiastolic)}
		boundary(&d, t.BPCrisisDiastolic, 1, func(x float64) string { return bpLabel(sys, x) },
			func(x float64) bool { return x > t.BPCrisisDiastolic },
			func(x float64) bool { return vitalsBP(e.Day, sys, x) == vitals.RuleBPCrisis })
		return d
	}},
	{"vitals.glucose_urgent_low", KindPath, func(_ context.Context, e *Env) Detail {
		at := vitals.ThresholdsOn(e.Day).GlucoseUrgentBelow
		d := Detail{Threshold: "mg/dL < " + num(at)}
		boundary(&d, at, 1, func(x float64) string { return num(x) + " mg/dL" },
			func(x float64) bool { return x < at },
			func(x float64) bool {
				return vitals.AlertRule(vitals.Reading{Type: vitals.TypeGlucose, Date: e.Day, MgDl: x}) == vitals.RuleGlucoseLow
			})
		return d
	}},
	{"epds.q10", KindPath, func(ctx context.Context, e *Env) Detail {
		d := Detail{Threshold: "item 10 ≥ 1"}
		boundary(&d, 1, 1, func(x float64) string { return "q10=" + num(x) },
			func(x float64) bool { return x >= 1 },
			func(x float64) bool {
				r := postpartum.ScoreOn(postpartum.KindFull, map[string]int{postpartum.SelfHarmItem: int(x)}, e.Day)
				return r.Urgent && slices.Contains(r.Reasons, postpartum.ReasonSelfHarm)
			})
		// The urgent message of a low total with item 10 > 0 calls 115 and 123 in every language.
		r := postpartum.ScoreOn(postpartum.KindFull, map[string]int{"q1": 1, "q2": 1, "q3": 1, postpartum.SelfHarmItem: 1}, e.Day)
		for _, loc := range e.Langs {
			msg, err := postpartum.SafetyJSON(ctx, e.copyFor(loc), r)
			if err != nil || !hasCall(msg, postpartum.EmergencyNumber) || !hasCall(msg, postpartum.SocialEmergencyNumber) {
				d.fail("%s: urgent message without calls to %s and %s", loc, postpartum.EmergencyNumber,
					postpartum.SocialEmergencyNumber)
			}
		}
		return d
	}},
	// T-M9-06 / H-01: the score path reads epds.urgent_total (31 = off: then no total without item 10 is urgent).
	{"epds.urgent_total", KindPath, func(_ context.Context, e *Env) Detail {
		at := postpartum.CutoffsOn(e.Day).UrgentTotal
		top := (len(postpartum.Items) - 1) * postpartum.ItemMax
		d := Detail{Threshold: "total ≥ " + strconv.Itoa(at)}
		for _, x := range []int{at - 1, at, at + 1} {
			if x < 0 || x > top {
				continue
			}
			d.probe("total="+strconv.Itoa(x), x >= at, epdsTotal(e.Day, x).Urgent)
		}
		if at > top {
			d.probe("total="+strconv.Itoa(top), false, epdsTotal(e.Day, top).Urgent)
		}
		return d
	}},
	{"epds.short_never_urgent", KindPath, func(_ context.Context, e *Env) Detail {
		d := Detail{Threshold: "short check never urgent"}
		for _, v := range []int{0, postpartum.ItemMax} {
			answers := map[string]int{}
			for _, it := range postpartum.ItemsOf(postpartum.KindShort) {
				answers[it.Code] = v
			}
			d.probe("short items="+strconv.Itoa(v), false, postpartum.ScoreOn(postpartum.KindShort, answers, e.Day).Urgent)
		}
		return d
	}},
	{"labor.511_interval", KindPath, func(_ context.Context, e *Env) Detail {
		p := labor.ParamsOn(e.Day)
		last := p.DurationMin + 15*time.Second
		d := Detail{Threshold: "average interval ≤ " + p.IntervalMax.String()}
		for _, every := range []time.Duration{p.IntervalMax - time.Minute, p.IntervalMax, p.IntervalMax + time.Minute} {
			d.probe("every "+every.String(), every <= p.IntervalMax, labor.FiveOneOne(runOf(p, every, last), p).Met)
		}
		return d
	}},
	{"labor.511_duration", KindPath, func(_ context.Context, e *Env) Detail {
		p := labor.ParamsOn(e.Day)
		d := Detail{Threshold: "average duration ≥ " + p.DurationMin.String()}
		for _, last := range []time.Duration{p.DurationMin - time.Second, p.DurationMin, p.DurationMin + time.Second} {
			d.probe("lasting "+last.String(), last >= p.DurationMin, labor.FiveOneOne(runOf(p, p.IntervalMax, last), p).Met)
		}
		return d
	}},
	// T-M9-06: the postpartum red flags are urgent with 115 (fever from postpartum.redflags.fever_c).
	{"postpartum.fever", KindPath, func(ctx context.Context, e *Env) Detail {
		fever := postpartum.FeverOn(e.Day)
		d := Detail{Threshold: "°C ≥ " + num(fever)}
		boundary(&d, fever, 0.1, func(x float64) string { return num(math.Round(x*10)/10) + " °C" },
			func(x float64) bool { return math.Round(x*10) >= math.Round(fever*10) },
			func(x float64) bool {
				return e.recoveryAlert(ctx, &d, postpartum.Recovery{TemperatureC: &x}, guide.AlertFever)
			})
		return d
	}},
	{"postpartum.yes_no_flags", KindPath, func(ctx context.Context, e *Env) Detail {
		d := Detail{Threshold: "yes = urgent"}
		for _, v := range []bool{false, true} {
			b := v
			d.probe("pad_per_hour="+strconv.FormatBool(v), v,
				e.recoveryAlert(ctx, &d, postpartum.Recovery{PadSoakedWithinHour: &b}, guide.AlertPadPerHour))
			d.probe("severe_pain="+strconv.FormatBool(v), v,
				e.recoveryAlert(ctx, &d, postpartum.Recovery{SeverePain: &b}, guide.AlertSeverePain))
			d.probe("breathless="+strconv.FormatBool(v), v,
				e.recoveryAlert(ctx, &d, postpartum.Recovery{Breathless: &b}, guide.AlertBreathless))
		}
		return d
	}},
	{"postpartum.heavy_bleeding", KindPath, func(ctx context.Context, e *Env) Detail {
		d := Detail{Threshold: "lochia = heavy"}
		for _, amount := range []string{"light", "medium", "heavy"} {
			d.probe("lochia="+amount, amount == "heavy",
				e.recoveryAlert(ctx, &d, postpartum.Recovery{LochiaAmount: amount}, guide.AlertHeavyBleeding))
		}
		return d
	}},
	// T-M9-05: the kick paths run the real alert rule, not a counter.
	{"preg.kicks_count", KindPath, func(_ context.Context, e *Env) Detail {
		p := pregnancyalerts.KickParamsOn(e.Day)
		d := Detail{Threshold: fmt.Sprintf("< %d kicks in %s from week %d", p.Target, p.Window, p.FromWeek)}
		for _, n := range []int{p.Target - 1, p.Target, p.Target + 1} {
			f := pregnant(e.Day, p.FromWeek)
			k := kickSession(e.Day, n, p.Window, 0)
			if n >= p.Target {
				k = kickSession(e.Day, n, p.Window, p.Window/2)
			}
			f.Kicks = []pregnancyalerts.Kick{k}
			d.probe("kicks="+strconv.Itoa(n), n < p.Target, e.fire(&d, "preg.fetal_movement_kicks", f))
		}
		return d
	}},
	{"preg.kicks_window", KindPath, func(_ context.Context, e *Env) Detail {
		p := pregnancyalerts.KickParamsOn(e.Day)
		d := Detail{Threshold: "target reached within " + p.Window.String()}
		for _, at := range []time.Duration{p.Window - time.Minute, p.Window, p.Window + time.Minute} {
			f := pregnant(e.Day, p.FromWeek)
			f.Kicks = []pregnancyalerts.Kick{kickSession(e.Day, p.Target, at, at)}
			d.probe("target at "+at.String(), at > p.Window, e.fire(&d, "preg.fetal_movement_kicks", f))
		}
		return d
	}},
	{"preg.kicks_from_week", KindPath, func(_ context.Context, e *Env) Detail {
		p := pregnancyalerts.KickParamsOn(e.Day)
		d := Detail{Threshold: "from week " + strconv.Itoa(p.FromWeek)}
		for _, w := range []int{p.FromWeek - 1, p.FromWeek, p.FromWeek + 1} {
			f := pregnant(e.Day, w)
			f.Kicks = []pregnancyalerts.Kick{kickSession(e.Day, p.Target-1, p.Window, 0)}
			d.probe("week "+strconv.Itoa(w), w >= p.FromWeek, e.fire(&d, "preg.fetal_movement_kicks", f))
		}
		return d
	}},
	{"preg.fetal_movement_manual", KindPath, func(_ context.Context, e *Env) Detail {
		from := clinicalconfig.Int(clinicalconfig.KeyPregnancyManualFetalWeek, "week", e.Day)
		d := Detail{Threshold: "reduced from week " + strconv.Itoa(from)}
		for _, w := range []int{from - 1, from, from + 1} {
			f := pregnant(e.Day, w)
			f.Fetal = []store.PregnancyFetalMovement{{LogDate: e.Day, PregnancyWeek: int32(w), MovementStatus: facts.FetalReduced}} //nolint:gosec // weeks
			d.probe("week "+strconv.Itoa(w), w >= from, e.fire(&d, "preg.fetal_movement", f))
		}
		return d
	}},
	{"preg.bp_severe_systolic", KindPath, func(_ context.Context, e *Env) Detail {
		at := pregnancyalerts.BPTierThreshold("severe", "systolic", e.Day)
		high := pregnancyalerts.BPTierThreshold("high", "diastolic", e.Day)
		d := Detail{Threshold: "systolic ≥ " + num(at)}
		boundary(&d, at, 1, func(x float64) string { return "weekly " + bpLabel(x, high-1) },
			func(x float64) bool { return x >= at },
			func(x float64) bool { return e.fire(&d, "preg.bp_severe", weeklyBP(e.Day, x, high-1)) })
		return d
	}},
	{"preg.bp_severe_diastolic", KindPath, func(_ context.Context, e *Env) Detail {
		at := pregnancyalerts.BPTierThreshold("severe", "diastolic", e.Day)
		high := pregnancyalerts.BPTierThreshold("high", "systolic", e.Day)
		d := Detail{Threshold: "diastolic ≥ " + num(at)}
		boundary(&d, at, 1, func(x float64) string { return "weekly " + bpLabel(high-1, x) },
			func(x float64) bool { return x >= at },
			func(x float64) bool { return e.fire(&d, "preg.bp_severe", weeklyBP(e.Day, high-1, x)) })
		return d
	}},
	// Pregnancy BP through vitals: a pregnant user's reading is handed to the pregnancy rules (not the vitals crisis
	// copy below 180/120), at the severe and the ≥ 140/90 tiers (urgent or follow-up per pregnancy.bp.high.urgent).
	{"preg.vitals_bp_severe", KindPath, func(_ context.Context, e *Env) Detail {
		at := pregnancyalerts.BPTierThreshold("severe", "systolic", e.Day)
		d := Detail{Threshold: "systolic ≥ " + num(at)}
		boundary(&d, at, 1, func(x float64) string { return "vitals " + bpLabel(x, 80) },
			func(x float64) bool { return x >= at },
			func(x float64) bool {
				r := vitals.Reading{Type: vitals.TypeBP, Date: e.Day, Systolic: x, Diastolic: 80, Pregnant: true}
				fired := e.fire(&d, "preg.bp_severe", vitalsFacts(e.Day, r))
				if fired && !r.Urgent() { // the severe tier is urgent in vitals too (the generic crisis may add more)
					d.fail("vitals %s: severe in pregnancy but the reading is not urgent", bpLabel(x, 80))
				}
				return fired
			})
		return d
	}},
	{"preg.vitals_bp_high", KindPath, func(_ context.Context, e *Env) Detail {
		at := pregnancyalerts.BPTierThreshold("high", "systolic", e.Day)
		key := "preg.bp_high"
		if pregnancyalerts.BPTierThreshold("high", "urgent", e.Day) >= 1 {
			key = "preg.bp_high_urgent"
		}
		d := Detail{Threshold: "systolic ≥ " + num(at) + " → " + key}
		boundary(&d, at, 1, func(x float64) string { return "vitals " + bpLabel(x, 70) },
			func(x float64) bool { return x >= at },
			func(x float64) bool {
				return e.fire(&d, key, vitalsFacts(e.Day, vitals.Reading{Type: vitals.TypeBP, Date: e.Day, Systolic: x,
					Diastolic: 70, Pregnant: true}))
			})
		return d
	}},
	{"preg.gdm_two_hour", KindPath, func(_ context.Context, e *Env) Detail {
		return gdm(e, vitals.ContextAfterMeal, pregnancyalerts.GlucoseTwoHour)
	}},
	{"preg.gdm_fasting", KindPath, func(_ context.Context, e *Env) Detail {
		return gdm(e, vitals.ContextFasting, pregnancyalerts.GlucoseFasting)
	}},
	{"preg.contractions_511", KindPath, func(_ context.Context, e *Env) Detail {
		p := labor.ParamsOn(e.Day)
		d := Detail{Threshold: "5-1-1 every ≤ " + p.IntervalMax.String()}
		for _, every := range []time.Duration{p.IntervalMax - time.Minute, p.IntervalMax, p.IntervalMax + time.Minute} {
			f := pregnant(e.Day, 38)
			f.Contractions = runOf(p, every, p.DurationMin+15*time.Second)
			f.ContractionSession = 1
			d.probe("every "+every.String(), every <= p.IntervalMax, e.fire(&d, "preg.contractions_511", f))
		}
		return d
	}},
	{"preg.critical_symptom", KindPath, func(_ context.Context, e *Env) Detail {
		d := Detail{Threshold: "bleeding / fluid leakage / severe sudden pain"}
		for _, sym := range []string{"", "bleeding", "fluid_leakage", "severe_sudden_pain"} {
			f := pregnant(e.Day, 30)
			if sym != "" {
				f.Days[e.Day] = pregnancyalerts.Day{sym: "mild"}
			}
			label := sym
			if label == "" {
				label = "nothing logged"
			}
			d.probe(label, sym != "", e.fire(&d, "preg.critical_symptom", f))
		}
		return d
	}},
}

func weeklyBP(day civildate.Date, sys, dia float64) pregnancyalerts.Facts {
	f := pregnant(day, 30)
	f.Weekly = []store.PregnancyWeeklyLog{{LogDate: day,
		SystolicPressure:  sql.NullInt32{Int32: int32(sys), Valid: true},  //nolint:gosec // mmHg
		DiastolicPressure: sql.NullInt32{Int32: int32(dia), Valid: true}}} //nolint:gosec // mmHg
	return f
}

// vitalsFacts are the pregnancy facts with r handed over the way vitals.Service.PregnancyAlerts does.
func vitalsFacts(day civildate.Date, r vitals.Reading) pregnancyalerts.Facts {
	f := pregnant(day, 30)
	if in, ok := vitals.PregnancyReading(r); ok {
		f.Readings = []pregnancyalerts.Reading{in}
	}
	return f
}

func gdm(e *Env, context, slot string) Detail {
	at := pregnancyalerts.GDMTarget(slot, e.Day)
	d := Detail{Threshold: slot + " ≥ " + num(at)}
	boundary(&d, at, 1, func(x float64) string { return slot + " " + num(x) + " mg/dL" },
		func(x float64) bool { return x >= at },
		func(x float64) bool {
			return e.fire(&d, "preg.gdm", vitalsFacts(e.Day, vitals.Reading{Type: vitals.TypeGlucose, Date: e.Day, MgDl: x,
				Context: context, Pregnant: true}))
		})
	return d
}
