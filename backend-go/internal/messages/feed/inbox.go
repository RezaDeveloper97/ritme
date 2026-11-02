package feed

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// The alerts inbox inside the feed (T-M10-07, D-164; SPEC §5.4.1): the feed runs the inbox evaluation on the facts it
// already built, takes the badge counts from the open rows after the diff, and fans the open rows out — an open
// safety row not yet acknowledged is the urgent_modal, an open engine row fills an empty status_banner.

// Inbox states that are open (open_flag = 1).
const (
	StateActive       = "active"
	StateAcknowledged = "acknowledged"
	StateNoted        = "added_to_visit_note"
)

// OpenAlert is one open inbox row after the evaluation. Rule is the served rule it was opened for (nil when the rule
// is no longer served: the row still counts in the badge but is not fanned out).
type OpenAlert struct {
	ID          uint64
	Rule        *rules.Rule
	Kind, Level string // stored level (follow_soon = urgent)
	State       string
	LastAt      time.Time
}

// Inbox is the evaluation the feed runs (inbox.Service).
type Inbox interface {
	// Need are the facts the alerts_inbox slot reads (conditions and text variables).
	Need(comp *compiled.Compiled) []string
	// Sync runs the alerts_inbox slot over f and diffs it with the user's open rows; open is the known open count
	// (0 = there are none to read). It returns the open rows after the diff.
	Sync(ctx context.Context, comp *compiled.Compiled, userID uint64, f rules.Facts, now time.Time, open int64) ([]OpenAlert, error)
}

var inboxErrors = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "messages_feed_inbox_errors_total",
	Help: "Feed builds whose alerts inbox evaluation failed (the feed answered with the counts read before it).",
})

func init() { metrics.Registry().MustRegister(inboxErrors) }

// WithInbox makes the feed (Request.Alerts) evaluate the alerts inbox and fan its open rows out.
func (s *Service) WithInbox(in Inbox) *Service {
	s.inbox = in
	return s
}

type inboxBox struct{ in Inbox }

var installedInbox atomic.Pointer[inboxBox]

// InstallInbox is the inbox every feed service without its own (WithInbox) evaluates — the route mount wires it once,
// as events.Install does for goals; nil uninstalls.
func InstallInbox(in Inbox) {
	if in == nil {
		installedInbox.Store(nil)
		return
	}
	installedInbox.Store(&inboxBox{in: in})
}

// inboxOf is the service's inbox, else the installed one (nil = counts only).
func (s *Service) inboxOf() Inbox {
	if s.inbox != nil {
		return s.inbox
	}
	if b := installedInbox.Load(); b != nil {
		return b.in
	}
	return nil
}

// countOpen are the badge counts of the open rows (urgent = the urgent level, as CountUserOpenInbox).
func countOpen(open []OpenAlert) Alerts {
	a := Alerts{Open: int64(len(open))}
	for _, o := range open {
		if o.Level == rules.LevelFollowSoon {
			a.Urgent++
		}
	}
	return a
}

// fanOut adds the open inbox rows to the slots SPEC §5.4.1 mirrors them in: an unacknowledged open safety row goes
// first in urgent_modal (it stays there until acknowledged); an open engine row fills an empty status_banner. A row
// whose text cannot be rendered is skipped; the slot keeps its capacity.
func fanOut(slot string, winners []rules.Choice, texts []rules.Variant, open []OpenAlert, ev openEvidence, x *extras,
	f rules.Facts, locale string, userID uint64,
) ([]rules.Choice, []rules.Variant, map[uint64]uint64) {
	var pick func(o *OpenAlert) bool
	first := false
	switch slot {
	case "urgent_modal":
		pick = func(o *OpenAlert) bool { return o.Kind == rules.KindSafety && o.State == StateActive }
		first = true
	case "status_banner":
		if len(winners) > 0 {
			return winners, texts, nil
		}
		pick = func(o *OpenAlert) bool { return o.Kind == rules.KindEngine }
	default:
		return winners, texts, nil
	}
	var add []rules.Choice
	var addTexts []rules.Variant
	ids := map[uint64]uint64{} // rule id → inbox row id of the fanned-out items (feed alert_id)
	for i := range open {      // open is newest first
		o := &open[i]
		if o.Rule == nil || !pick(o) || containsRule(winners, o.Rule.ID) || containsRule(add, o.Rule.ID) {
			continue
		}
		ch := rules.Choice{Rule: o.Rule, Variant: o.Rule.Assign(userID)}
		v, ok := renderOpen(x.text(ch.Rule, ch.Variant, locale), o, ev, f, locale)
		if !ok {
			continue
		}
		add, addTexts = append(add, ch), append(addTexts, v)
		ids[o.Rule.ID] = o.ID
	}
	if len(add) == 0 {
		return winners, texts, nil
	}
	var outW []rules.Choice
	var outT []rules.Variant
	if first {
		outW, outT = slices.Concat(add, winners), slices.Concat(addTexts, texts)
	} else {
		outW, outT = slices.Concat(winners, add), slices.Concat(texts, addTexts)
	}
	n := capacity(slot, len(outW))
	return outW[:min(n, len(outW))], outT[:min(n, len(outT))], ids
}

// EvidenceRenderer renders text with an inbox row's variables in locale (a missing variable stays as written).
type EvidenceRenderer func(text string, vars map[string]any, locale string) string

var (
	renderersMu sync.Mutex
	renderers   atomic.Pointer[map[string]EvidenceRenderer] // rule_key prefix → renderer
)

// InstallEvidenceRenderer makes fanOut render the open rows of the rules whose rule_key starts with prefix through fn,
// with the row's stored evidence over the request's facts (the pregnancy alerts: inboxalerts.Render, T-M10-08b) — the
// text says what the alert was raised for, in that domain's localized terms. nil removes the prefix.
func InstallEvidenceRenderer(prefix string, fn EvidenceRenderer) {
	renderersMu.Lock()
	defer renderersMu.Unlock()
	next := map[string]EvidenceRenderer{}
	if cur := renderers.Load(); cur != nil {
		maps.Copy(next, *cur)
	}
	if fn == nil {
		delete(next, prefix)
	} else {
		next[prefix] = fn
	}
	renderers.Store(&next)
}

// rendererOf is the evidence renderer of a rule key (nil = the plain fact render).
func rendererOf(key string) EvidenceRenderer {
	cur := renderers.Load()
	if cur == nil || key == "" {
		return nil
	}
	for prefix, fn := range *cur {
		if strings.HasPrefix(key, prefix) {
			return fn
		}
	}
	return nil
}

// HeldRule is the code-held definition of a locked rule (rule_key key, the open row's rule_id id) with its texts in
// locale, or nil when the domain holds none for key.
type HeldRule func(key string, id uint64, locale string) *rules.Rule

var (
	heldMu sync.Mutex
	held   atomic.Pointer[map[string]HeldRule] // rule_key prefix → code-held rules
)

// InstallHeldRules makes fanOut fan out an open row whose rule is not served (its row missing, inactive or not
// approved: no rule in the snapshot) through fn's code-held rule for the rule keys starting with prefix — the locked
// pregnancy safety rules (inboxalerts' guarantee, T-M10-08c): such an alert is raised by the domain's own evaluation
// and must still reach the urgent modal. nil removes the prefix.
func InstallHeldRules(prefix string, fn HeldRule) {
	heldMu.Lock()
	defer heldMu.Unlock()
	next := map[string]HeldRule{}
	if cur := held.Load(); cur != nil {
		maps.Copy(next, *cur)
	}
	if fn == nil {
		delete(next, prefix)
	} else {
		next[prefix] = fn
	}
	held.Store(&next)
}

// heldOf is the code-held rule source of a rule key (nil = none).
func heldOf(key string) HeldRule {
	cur := held.Load()
	if cur == nil || key == "" {
		return nil
	}
	for prefix, fn := range *cur {
		if strings.HasPrefix(key, prefix) {
			return fn
		}
	}
	return nil
}

// openEvidence are the stored evidence variables of open inbox rows, by row id.
type openEvidence map[uint64]map[string]any

// evidenceOf reads the stored evidence of the open rows an evidence renderer renders (one query, only when one of
// them was not written by this evaluation, or an open row's rule is not served; nil otherwise). An open row whose
// rule is not served gets its code-held rule (InstallHeldRules) in locale, so fanOut can show it.
func (s *Service) evidenceOf(ctx context.Context, userID uint64, open []OpenAlert, locale string) (openEvidence, error) {
	need := false
	for i := range open {
		if open[i].ID != 0 && (open[i].Rule == nil || rendererOf(open[i].Rule.RuleKey) != nil) {
			need = true
			break
		}
	}
	if !need {
		return nil, nil
	}
	rows, err := s.q.ListUserOpenInbox(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range open {
		if open[i].Rule != nil || open[i].ID == 0 {
			continue
		}
		for _, r := range rows {
			if r.ID != open[i].ID {
				continue
			}
			if fn := heldOf(r.RuleKey.String); fn != nil {
				open[i].Rule = fn(r.RuleKey.String, r.RuleID, locale)
			}
			break
		}
	}
	out := openEvidence{}
	for _, r := range rows {
		var d struct {
			Vars map[string]any `json:"vars"`
		}
		if r.Evidence.Valid && json.Unmarshal(r.Evidence.V, &d) == nil && d.Vars != nil {
			out[r.ID] = d.Vars
		}
	}
	return out, nil
}

// renderOpen renders a fanned-out open row: through its rule's evidence renderer with the stored evidence over the
// facts when it has one and every variable fills, else the plain fact render.
func renderOpen(v rules.Variant, o *OpenAlert, ev openEvidence, f rules.Facts, locale string) (rules.Variant, bool) {
	if fn := rendererOf(o.Rule.RuleKey); fn != nil {
		vars := make(map[string]any, len(f)+len(ev[o.ID]))
		maps.Copy(vars, f)
		maps.Copy(vars, ev[o.ID])
		out := v
		out.Title, out.Body, out.CTALabel = fn(v.Title, vars, locale), fn(v.Body, vars, locale), fn(v.CTALabel, vars, locale)
		if len(rules.Placeholders(out.Title, out.Body, out.CTALabel)) == 0 {
			return out, true
		}
	}
	return rules.RenderVariant(v, f)
}

func containsRule(chs []rules.Choice, id uint64) bool {
	for _, c := range chs {
		if c.Rule.ID == id {
			return true
		}
	}
	return false
}

// Renderer renders rules' texts in a locale with the snapshot's translations (the alerts inbox list, T-M10-07).
type Renderer struct{ x *extras }

// RendererOf is the renderer of comp (built once per snapshot, shared with the feed).
func RendererOf(ctx context.Context, q store.Querier, comp *compiled.Compiled) Renderer {
	return Renderer{x: extrasOf(ctx, q, comp)}
}

// Text is variant v of r in locale (the default language's text when locale has no translation).
func (r Renderer) Text(rule *rules.Rule, v, locale string) rules.Variant {
	if r.x == nil {
		return rule.VariantContent(v)
	}
	return r.x.text(rule, v, locale)
}

// Evidence is the «what we saw / how sure / contact» of rule in locale.
func (r Renderer) Evidence(rule *rules.Rule, locale string) rules.Content {
	if r.x == nil {
		return rule.Content
	}
	return r.x.evidence(rule, locale)
}

// Translations are the stored translations of a rule (language code → content), for the variables of every language.
func (r Renderer) Translations(ruleID uint64) map[string]rules.Content {
	if r.x == nil {
		return nil
	}
	return r.x.translations[ruleID]
}
