package conditionnudges

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	cstore "github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/content"
	mfacts "github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err)
	}
	return t
})

// TextKeys are the texts of a nudge (= the payload keys of its message_contents rows).
var TextKeys = []string{"title", "body", "action", "doctor_action"}

// fallbackText is the embedded copy of rule/key in locale (the fallback locale's when missing).
func fallbackText(rule, key, locale string) string {
	k := "condition_nudges." + rule + "." + key
	if s := translator().Trans(k, nil, locale); s != k {
		return s
	}
	return ""
}

// DoctorDirectory links a nudge to the doctors who can see her.
//
// TODO(B-N7-02): the mother & child directory (bloom N7) does not exist yet. Once it does, implement this over its
// gynaecologist search and pass it to NewEngine; until then the engine has none and no doctor link is sent (never a
// dead link).
type DoctorDirectory interface {
	DoctorLink(ctx context.Context, rule string) (link string, ok bool, err error)
}

// CycleStates hands out a user's prepared v6 engine state for the user-local day (state.Source satisfies it). nil
// means CYCLE_ENGINE=v1: the window keeps the v1 rule (CycleWindow).
type CycleStates interface {
	ForUser(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error)
}

// Engine evaluates the nudges of one user. Reads only, all scoped to the user.
type Engine struct {
	mq      store.Querier
	cq      cstore.Querier
	logs    *healthlog.Service
	doctors DoctorDirectory
	states  CycleStates
	rules   ruleCache
}

// WithCycle makes the window and the mode come from the v6 engine (CYCLE_ENGINE shadow / v6, T-M9-37); nil keeps v1.
func (e *Engine) WithCycle(states CycleStates) *Engine {
	e.states = states
	return e
}

// NewEngine wires the engine on db; doctors may be nil (no directory yet).
func NewEngine(db store.DBTX, doctors DoctorDirectory) *Engine {
	return &Engine{mq: store.New(db), cq: cstore.New(db), logs: healthlog.NewService(db), doctors: doctors}
}

// Nudge is one raised nudge with its texts in the request locale.
type Nudge struct {
	Hit
	Texts       map[string]string // TextKeys → text ({days} filled)
	DoctorLink  *string
	NeedsReview bool // a text came from the embedded copy, not an approved admin row
}

// Result is the nudges of the current cycle window.
type Result struct {
	From, To civildate.Date
	Nudges   []Nudge
}

// Evaluate runs the rules for userID on today; locale picks the texts. The inbox clock is now when today is the
// Tehran day of now, else noon of today.
func (e *Engine) Evaluate(ctx context.Context, userID uint64, today civildate.Date, locale string) (Result, error) {
	now := time.Now().In(civildate.Tehran)
	if civildate.FromTime(now) != today {
		now = today.TehranMidnight().Add(12 * time.Hour)
	}
	return e.EvaluateAt(ctx, userID, now, locale)
}

// EvaluateAt runs the cond.* rules for userID at now (user-local; its day is the target day) through her alerts inbox
// — a new match opens a row, a new qualifying day merges into it, an ended pattern resolves it — and resolves the
// texts of the nudges; locale picks the texts.
func (e *Engine) EvaluateAt(ctx context.Context, userID uint64, now time.Time, locale string) (Result, error) {
	today := civildate.FromTime(now)
	f, err := e.facts(ctx, userID, today)
	if err != nil {
		return Result{}, err
	}
	res := Result{From: f.From, To: f.To, Nudges: []Nudge{}}
	set, err := e.rules.get(ctx, e.mq)
	if err != nil {
		return Result{}, err
	}
	open, err := set.sync(ctx, e.mq, userID, rules.Facts(mfacts.ConditionValues(Input(f))), now)
	if err != nil {
		return Result{}, err
	}
	hits := DetectWith(f, set.holds(open))
	if len(hits) == 0 {
		return res, nil
	}
	repo := content.NewRepository(e.mq)
	for _, h := range hits {
		n := Nudge{Hit: h, Texts: map[string]string{}}
		var admin content.Payload
		if v, ok, err := repo.Payload(ctx, Group, h.Rule, locale); err != nil {
			return Result{}, fmt.Errorf("condition nudges: texts: %w", err)
		} else if ok {
			admin = content.NewPayload(v)
		}
		days := strconv.Itoa(len(h.Dates))
		if locale == "fa" {
			days = phpround.PersianDigits(days) // «۲ روز», like the daily card (cycle/view)
		}
		for _, k := range TextKeys {
			s, _ := admin.Or(k, "").(string)
			if strings.TrimSpace(s) == "" {
				s, n.NeedsReview = fallbackText(h.Rule, k, locale), true
			}
			n.Texts[k] = strings.ReplaceAll(s, "{days}", days)
		}
		if e.doctors != nil {
			link, ok, err := e.doctors.DoctorLink(ctx, h.Rule)
			if err != nil {
				return Result{}, fmt.Errorf("condition nudges: doctors: %w", err)
			}
			if ok && link != "" {
				n.DoctorLink = &link
			}
		}
		res.Nudges = append(res.Nudges, n)
	}
	return res, nil
}

// facts loads the user's mode, enrolments, cycle window and day logs of the window.
func (e *Engine) facts(ctx context.Context, userID uint64, today civildate.Date) (Facts, error) {
	if e.states != nil {
		return e.factsV6(ctx, userID, today)
	}
	mode, err := e.logs.LifeMode(ctx, userID)
	if err != nil {
		return Facts{}, fmt.Errorf("condition nudges: %w", err)
	}
	f := Facts{Mode: mode, Enrolled: map[string]bool{}}
	if err := e.enrolments(ctx, userID, &f); err != nil {
		return Facts{}, err
	}

	var latest *civildate.Date
	cycleLen := 0
	profile, err := e.mq.GetMessageProfile(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return Facts{}, fmt.Errorf("condition nudges: profile: %w", err)
	default:
		if profile.CycleDuration.Valid {
			cycleLen = int(profile.CycleDuration.Int16)
		}
		if d := profile.LastPeriodStart; d.Valid && !d.Date.After(today) {
			latest = &d.Date
		}
	}
	histories, err := e.mq.ListMessageCycleHistories(ctx, userID)
	if err != nil {
		return Facts{}, fmt.Errorf("condition nudges: cycle histories: %w", err)
	}
	for _, h := range histories {
		if d := h.PeriodStartDate; !d.After(today) && (latest == nil || d.After(*latest)) {
			latest = &d
		}
	}
	f.From, f.To = CycleWindow(today, latest, cycleLen)

	if f.Days, err = e.logs.Range(ctx, userID, f.From, f.To); err != nil {
		return Facts{}, fmt.Errorf("condition nudges: %w", err)
	}
	return f, nil
}

// factsV6 is facts on the v6 engine: the engine's resolved mode and its current cycle (EngineCycleWindow); without a
// current cycle no day log is read and nothing is nudged.
func (e *Engine) factsV6(ctx context.Context, userID uint64, today civildate.Date) (Facts, error) {
	st, err := e.states.ForUser(ctx, userID, today)
	if err != nil {
		return Facts{}, fmt.Errorf("condition nudges: engine state: %w", err)
	}
	f := Facts{Mode: st.Mode().String(), Enrolled: map[string]bool{}}
	if !slices.Contains(nudgedModes, f.Mode) {
		return f, nil
	}
	from, to, ok := EngineCycleWindow(st, today)
	if !ok {
		return f, nil
	}
	f.From, f.To = from, to
	if err := e.enrolments(ctx, userID, &f); err != nil {
		return Facts{}, err
	}
	if f.Days, err = e.logs.Range(ctx, userID, f.From, f.To); err != nil {
		return Facts{}, fmt.Errorf("condition nudges: %w", err)
	}
	return f, nil
}

func (e *Engine) enrolments(ctx context.Context, userID uint64, f *Facts) error {
	rows, err := e.cq.ListEnrolments(ctx, userID)
	if err != nil {
		return fmt.Errorf("condition nudges: enrolments: %w", err)
	}
	for _, r := range rows {
		f.Enrolled[r.Program] = true
	}
	return nil
}

// JSON renders the result: {from, to, nudges: [{key, program, title, body, action, link, doctor, days, dates,
// needs_review}]}. doctor is {action, link} or null while no doctors directory exists.
func (r Result) JSON() *jsonx.OrderedMap {
	list := make([]any, 0, len(r.Nudges))
	for _, n := range r.Nudges {
		dates := make([]string, len(n.Dates))
		for i, d := range n.Dates {
			dates[i] = d.String()
		}
		var doctor any
		if n.DoctorLink != nil {
			doctor = jsonx.Obj("action", n.Texts["doctor_action"], "link", *n.DoctorLink)
		}
		list = append(list, jsonx.Obj(
			"key", n.Rule,
			"program", n.Program,
			"title", n.Texts["title"],
			"body", n.Texts["body"],
			"action", n.Texts["action"],
			"link", n.Link,
			"doctor", doctor,
			"days", len(n.Dates),
			"dates", dates,
			"needs_review", n.NeedsReview,
		))
	}
	if r.From.IsZero() { // v6 without a current cycle
		return jsonx.Obj("from", nil, "to", nil, "nudges", list)
	}
	return jsonx.Obj("from", r.From.String(), "to", r.To.String(), "nudges", list)
}
