package menomessages

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/inbox"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
)

// Copy is one rule's seed texts in one language.
type Copy struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	CTA        string `json:"cta,omitempty"`
	Seen       string `json:"seen,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

// The seed copy of the rules: copy/<code>.json, rule key → texts (the default language as the base, every other file
// as content.translations.<code>; the code-held rules read the fallback language's).
//
//go:embed copy/*.json
var copyFS embed.FS

var embedded = sync.OnceValue(func() map[string]map[string]Copy {
	out := map[string]map[string]Copy{}
	entries, err := copyFS.ReadDir("copy")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		b, err := copyFS.ReadFile("copy/" + e.Name())
		if err != nil {
			panic(err)
		}
		var m map[string]Copy
		if err := json.Unmarshal(b, &m); err != nil {
			panic(fmt.Errorf("menomessages: copy %s: %w", e.Name(), err))
		}
		out[strings.TrimSuffix(e.Name(), ".json")] = m
	}
	return out
})

// CopyOf is the seed copy of key in locale (the fallback language when the locale has none).
func CopyOf(key, locale string) (Copy, bool) {
	if c, ok := embedded()[locale][key]; ok {
		return c, true
	}
	c, ok := embedded()[lang.FallbackLocale][key]
	return c, ok
}

// CopyLanguages are the codes of the seed copy files, sorted.
func CopyLanguages() []string {
	out := make([]string, 0, len(embedded()))
	for code := range embedded() {
		out = append(out, code)
	}
	slices.Sort(out)
	return out
}

func (d Def) content(c Copy) rules.Content {
	v := rules.Variant{Title: c.Title, Body: c.Body}
	if c.CTA != "" && d.Link != "" {
		v.CTALabel, v.CTATarget = c.CTA, d.Link
	}
	return rules.Content{A: v, Seen: c.Seen, Confidence: c.Confidence}
}

// ContentJSON is the content column of d with base the default language's code.
func (d Def) ContentJSON(base string) (string, error) {
	b, ok := CopyOf(d.Key, base)
	if !ok {
		return "", fmt.Errorf("menomessages: no copy for %s", d.Key)
	}
	sc := struct {
		rules.Content
		Translations map[string]rules.Content `json:"translations,omitempty"`
	}{Content: d.content(b), Translations: map[string]rules.Content{}}
	for _, code := range CopyLanguages() {
		if t, ok := embedded()[code][d.Key]; ok && code != base {
			sc.Translations[code] = d.content(t)
		}
	}
	out, err := json.Marshal(sc)
	return string(out), err
}

// ConditionsJSON is the conditions column of d.
func (d Def) ConditionsJSON() string {
	b, _ := json.Marshal(d.Groups)
	return string(b)
}

// Rule is the code-held engine rule of d with id: the seeded row as it would be served (fallback-language copy).
func (d Def) Rule(id uint64) rules.Rule {
	cp, _ := CopyOf(d.Key, lang.FallbackLocale)
	return rules.Rule{ID: id, Level: d.Level, Kind: d.Kind, Priority: d.Prio, Placement: d.Slot,
		Content: d.content(cp), Groups: d.Groups, HoursFrom: -1, HoursTo: -1, AudiencePercent: 100,
		Status: rules.StatusApproved, IsActive: true, RuleKey: d.Key, Version: 1, ApprovedVersion: 1, Locked: d.Locked}
}

// CodeRuleIDBase is the first id of a code-held menopause rule without a row (no message_rules id is that large;
// the pregnancy rules use 9_000_000_000_000 + i, the condition nudges 9_000_000_002_000 + i).
const CodeRuleIDBase uint64 = 9_000_000_001_000

// CodeRuleID is the stable id of the code-held rule of key.
func CodeRuleID(key string) uint64 {
	return CodeRuleIDBase + uint64(max(0, slices.IndexFunc(Defs, func(d Def) bool { return d.Key == key }))) //nolint:gosec // small index
}

// ruleSet is the served meno.* rules of one message snapshot. A key whose row does not exist (a database the seed did
// not reach, e.g. the contract fixture) and a locked key whose row is off run the code-held rule; a non-locked key
// whose row exists but is not served is off (byKey nil).
type ruleSet struct {
	base  *compiled.Compiled
	byKey map[string]*rules.Rule
	inbox *compiled.Compiled // the alerts_inbox rules
	ids   map[uint64]bool    // ids of the alerts_inbox rules
}

type ruleCache struct {
	mu   sync.Mutex
	memo atomic.Pointer[ruleSet]
}

func (c *ruleCache) get(ctx context.Context, q store.Querier) (*ruleSet, error) {
	base, err := compiled.Load(ctx, q)
	if err != nil {
		return nil, err
	}
	if m := c.memo.Load(); m != nil && m.base == base {
		return m, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if m := c.memo.Load(); m != nil && m.base == base {
		return m, nil
	}
	rows, err := rules.ActiveRows(ctx, q, "")
	if err != nil {
		return nil, fmt.Errorf("menopause messages: rules: %w", err)
	}
	set := &ruleSet{base: base, byKey: map[string]*rules.Rule{}, ids: map[uint64]bool{}}
	for i := range rows {
		if !strings.HasPrefix(rows[i].RuleKey.String, KeyPrefix) {
			continue
		}
		if r := rules.FromRow(&rows[i]); r.Served() {
			set.byKey[r.RuleKey] = &r
		}
	}
	var box []rules.Rule
	for _, d := range Defs {
		r := set.byKey[d.Key]
		if r == nil {
			id, err := q.GetMessageRuleIDByKey(ctx, sql.NullString{String: d.Key, Valid: true})
			switch {
			case errors.Is(err, sql.ErrNoRows):
				id = CodeRuleID(d.Key)
			case err != nil:
				return nil, fmt.Errorf("menopause messages: rule id: %w", err)
			case !d.Locked:
				continue // switched off by the admin
			}
			cr := d.Rule(id)
			r = &cr
			set.byKey[d.Key] = r
		}
		if d.Slot == rules.SlotAlertsInbox {
			box = append(box, *r)
			set.ids[r.ID] = true
		}
	}
	set.inbox = compiled.CompileRules(box)
	c.memo.Store(set)
	return set, nil
}

// sync runs the inbox diff of the meno.* alerts for userID over f at now (user-local) and returns the open rule ids.
func (s *ruleSet) sync(ctx context.Context, q store.Querier, userID uint64, f rules.Facts, now time.Time) (map[uint64]bool, error) {
	box := inbox.New(q, nil, nil).WithLoader(func(context.Context) (*compiled.Compiled, error) { return s.inbox, nil })
	open, err := box.Evaluate(ctx, userID, f, now)
	if err != nil {
		return nil, fmt.Errorf("menopause messages: inbox: %w", err)
	}
	out := map[uint64]bool{}
	for _, o := range open {
		if o.Rule != nil && s.ids[o.Rule.ID] {
			out[o.Rule.ID] = true
		}
	}
	return out, nil
}

// holds is the adapter's Holds: an alerts inbox rule holds while its row is open and its condition matches (an HRT
// item is matched on its own days); a daily_advice tip rule while it matches, is in its audience and schedule.
func (s *ruleSet) holds(open map[uint64]bool, userID uint64, now time.Time) Holds {
	return func(d *Def, f rules.Facts) bool {
		r := s.byKey[d.Key]
		if r == nil || !rules.MatchRule(r, f) {
			return false
		}
		if d.Slot == rules.SlotAlertsInbox {
			return open[r.ID]
		}
		return r.InAudience(userID) && rules.InWindow(r, now)
	}
}
