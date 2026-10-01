package conditionnudges

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	cstore "github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
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

// Engine evaluates the nudges of one user. Reads only, all scoped to the user.
type Engine struct {
	mq      store.Querier
	cq      cstore.Querier
	logs    *healthlog.Service
	doctors DoctorDirectory
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

// Evaluate runs the rules for userID on today; locale picks the texts.
func (e *Engine) Evaluate(ctx context.Context, userID uint64, today civildate.Date, locale string) (Result, error) {
	f, err := e.facts(ctx, userID, today)
	if err != nil {
		return Result{}, err
	}
	res := Result{From: f.From, To: f.To, Nudges: []Nudge{}}
	hits := Detect(f)
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
	mode, err := e.logs.LifeMode(ctx, userID)
	if err != nil {
		return Facts{}, fmt.Errorf("condition nudges: %w", err)
	}
	f := Facts{Mode: mode, Enrolled: map[string]bool{}}
	enrolments, err := e.cq.ListEnrolments(ctx, userID)
	if err != nil {
		return Facts{}, fmt.Errorf("condition nudges: enrolments: %w", err)
	}
	for _, r := range enrolments {
		f.Enrolled[r.Program] = true
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
	return jsonx.Obj("from", r.From.String(), "to", r.To.String(), "nudges", list)
}
