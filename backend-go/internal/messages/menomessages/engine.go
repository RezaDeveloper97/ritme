package menomessages

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/store"
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

// fallbackText is the embedded copy of rule/key in locale (the fallback locale's when missing).
func fallbackText(rule, key, locale string) string {
	k := "menopause_messages." + rule + "." + key
	if s := translator().Trans(k, nil, locale); s != k {
		return s
	}
	return ""
}

func uitoa(n uint64) string { return strconv.FormatUint(n, 10) }

// SignalSource is where the facts come from (*menopause.Service).
type SignalSource interface {
	Signals(ctx context.Context, userID uint64, now time.Time) (menopause.Signals, error)
}

// Engine evaluates the menopause messages of one user. Reads only, all scoped to the user.
type Engine struct {
	mq  store.Querier
	src SignalSource
}

// NewEngine wires the engine: db for the message_contents texts, src for the menopause facts.
func NewEngine(db store.DBTX, src SignalSource) *Engine {
	return &Engine{mq: store.New(db), src: src}
}

// Message is one raised message with its texts in the request locale.
type Message struct {
	Hit
	Texts       map[string]string // TextKeys → text (placeholders filled)
	NeedsReview bool
}

// Result is the user's menopause messages of today.
type Result struct {
	Signals  menopause.Signals
	Messages []Message
}

// Evaluate runs the rules for userID at now; loc picks the texts (request locale, then the default language).
func (e *Engine) Evaluate(ctx context.Context, userID uint64, now time.Time, loc catalog.Localizer) (Result, error) {
	sig, err := e.src.Signals(ctx, userID, now)
	if err != nil {
		return Result{}, err
	}
	res := Result{Signals: sig, Messages: []Message{}}
	repo := content.NewRepository(e.mq)
	for _, h := range Detect(sig) {
		m := Message{Hit: h, Texts: map[string]string{}}
		switch {
		case h.Kind == KindTip:
			m.Texts = itemTexts(h.Item, loc)
			m.NeedsReview = h.Item.NeedsReview
		case h.Rule == RuleBleeding:
			m.Texts = itemTexts(h.Item, loc)
			m.NeedsReview = h.Item == nil || h.Item.NeedsReview
			for _, k := range TextKeys {
				if strings.TrimSpace(m.Texts[k]) == "" {
					m.Texts[k], m.NeedsReview = fallbackText(h.Rule, k, loc.Locale), true
				}
			}
		default:
			var admin content.Payload
			if v, ok, err := repo.Payload(ctx, Group, h.Rule, loc.Locale); err != nil {
				return Result{}, fmt.Errorf("menopause messages: texts: %w", err)
			} else if ok {
				admin = content.NewPayload(v)
			}
			vars := placeholders(h, sig, loc)
			for _, k := range TextKeys {
				s, _ := admin.Or(k, "").(string)
				if strings.TrimSpace(s) == "" {
					s, m.NeedsReview = fallbackText(h.Rule, k, loc.Locale), true
				}
				m.Texts[k] = vars.Replace(s)
			}
		}
		res.Messages = append(res.Messages, m)
	}
	return res, nil
}

// itemTexts are a catalog item's title, body and meta.cta (action) in the request locale.
func itemTexts(it *catalog.Item, loc catalog.Localizer) map[string]string {
	out := map[string]string{"title": "", "body": "", "action": ""}
	if it == nil {
		return out
	}
	out["title"], _ = loc.Text(it.Title).(string)
	out["body"], _ = loc.Text(it.Body).(string)
	if meta, ok := loc.Meta(it.Meta).(*jsonx.OrderedMap); ok {
		if v, ok := meta.Get("cta"); ok {
			out["action"], _ = v.(string)
		}
	}
	return out
}

func cl(loc catalog.Localizer) checkups.Lang {
	return checkups.Lang{Locale: loc.Locale, Default: loc.Default}
}

// checkupTitle is the checkup's title in the request locale ("" when the plan has no row for it).
func checkupTitle(h Hit, sig menopause.Signals, loc catalog.Localizer) string {
	if h.Checkup == nil || sig.Plan == nil {
		return ""
	}
	t, _ := sig.Plan.ItemJSON(*h.Checkup, cl(loc)).Get("title")
	s, _ := t.(string)
	return s
}

// placeholders are a rule's {name} values, numbers in the locale's digits and dates in its calendar.
func placeholders(h Hit, sig menopause.Signals, loc catalog.Localizer) *strings.Replacer {
	num := func(n int) string { return checkups.LocalizeDigits(strconv.Itoa(n), loc.Locale) }
	var kv []string
	switch h.Rule {
	case RuleCheckupOverdue, RuleCheckupDue:
		kv = []string{"{checkup}", checkupTitle(h, sig, loc), "{count}", num(h.CheckupCount)}
	case RuleScoreWorsened:
		prev, delta := 0, 0
		if h.Score.Previous != nil {
			prev = h.Score.Previous.Total
		}
		if d := h.Score.Delta(); d != nil {
			delta = *d
		}
		kv = []string{"{total}", num(h.Score.Total), "{previous}", num(prev), "{delta}", num(delta)}
	case RuleHRTReview:
		d := h.Treatment.ReviewOn.Date
		kv = []string{
			"{name}", h.Treatment.Name, "{date}", checkups.FullDate(d, loc.Locale),
			"{when}", checkups.RelativeDue(d, sig.Date, loc.Locale), "{days}", num(h.DaysLeft),
		}
	}
	return strings.NewReplacer(kv...)
}

// JSON renders the result: {mode, stage, messages: [{key, kind, priority, title, body, action, link, needs_review,
// data}]}; data is the rule's facts (null for tips).
func (r Result) JSON(loc catalog.Localizer) *jsonx.OrderedMap {
	list := make([]any, 0, len(r.Messages))
	for _, m := range r.Messages {
		var link any
		if m.Link != "" {
			link = m.Link
		}
		list = append(list, jsonx.Obj(
			"key", m.Rule,
			"kind", m.Kind,
			"priority", m.Priority,
			"title", emptyToNil(m.Texts["title"]),
			"body", emptyToNil(m.Texts["body"]),
			"action", emptyToNil(m.Texts["action"]),
			"link", link,
			"needs_review", m.NeedsReview,
			"data", m.data(r.Signals, loc),
		))
	}
	var stage any
	if s := r.Signals.Stage.Stage; s != "" {
		stage = s
	}
	return jsonx.Obj("mode", string(r.Signals.Mode), "stage", stage, "messages", list)
}

func emptyToNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (m Message) data(sig menopause.Signals, loc catalog.Localizer) any {
	if m.Kind == KindTip { // a tip's key is its catalog code, which may repeat a rule key (meno_tips hrt_review)
		return nil
	}
	switch m.Rule {
	case RuleBleeding:
		var last any
		if m.BleedingLastOn != nil {
			last = *m.BleedingLastOn
		}
		return jsonx.Obj("last_on", last)
	case RuleCheckupOverdue, RuleCheckupDue:
		return jsonx.Obj("count", m.CheckupCount, "checkup", sig.Plan.ItemJSON(*m.Checkup, cl(loc)))
	case RuleScoreWorsened:
		var prev, prevMonth any
		if p := m.Score.Previous; p != nil {
			prev, prevMonth = p.Total, p.Month
		}
		return jsonx.Obj(
			"month", m.Score.Month, "total", m.Score.Total,
			"previous_month", prevMonth, "previous", prev, "delta", *m.Score.Delta(),
		)
	case RuleHRTReview:
		return jsonx.Obj(
			"treatment_item_id", m.Treatment.ID, "name", m.Treatment.Name,
			"review_on", m.Treatment.ReviewOn.Date, "days_left", m.DaysLeft,
		)
	}
	return nil
}
