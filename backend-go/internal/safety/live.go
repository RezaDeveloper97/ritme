package safety

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/inbox"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/postpartum"
	"github.com/ritme/backend-go/internal/postpartum/guide"
	"github.com/ritme/backend-go/internal/safety/store"
	"github.com/ritme/backend-go/internal/vitals"
	vstore "github.com/ritme/backend-go/internal/vitals/store"
)

// Live is the live configuration a nightly run reads fresh from the database (no process cache): the preg.* rule
// rows, the served pregnancy alert rules compiled the way the inbox compiles them, and the vitals urgent copy.
type Live struct {
	rows   map[string][]store.ListSafetyRulesByKeysRow // every row of each locked key, active or not
	engine *liveEngine
	vitals vitals.AlertCopy
	err    error // the load failure (the config checks report it)
}

// EmergencyKeys are the clinical registry keys of the emergency paths (each must have a valid version today).
var EmergencyKeys = []string{
	clinicalconfig.KeyBPCrisis, clinicalconfig.KeyGlucoseUrgentLow, clinicalconfig.KeyEPDSUrgentTotal,
	clinicalconfig.KeyPostpartumRedFlags, clinicalconfig.KeyPregnancyBPHigh, clinicalconfig.KeyPregnancyBPSevere,
	clinicalconfig.KeyPregnancyGDMTargets, clinicalconfig.KeyPregnancyKicks, clinicalconfig.KeyPregnancyManualFetalWeek,
	clinicalconfig.KeyPregnancyContractions511,
}

// lockedDefs are the locked emergency (safety) pregnancy rules.
func lockedDefs() []inboxalerts.Def {
	var out []inboxalerts.Def
	for _, d := range inboxalerts.Defs {
		if d.Locked {
			out = append(out, d)
		}
	}
	return out
}

// LiveEnv builds the nightly run's environment on db: the active languages (default first), the live postpartum copy,
// the live pregnancy rules and the live configuration. A load failure is kept in Live (the config checks fail with
// it) and the path scenarios fall back to the code-held rules and the embedded copy.
func LiveEnv(ctx context.Context, q store.Querier, mq mstore.Querier, vq vstore.Querier, base *Env) *Env {
	e := *base
	live := &Live{rows: map[string][]store.ListSafetyRulesByKeysRow{}}
	e.Live = live
	langs, err := q.ListSafetyActiveLanguages(ctx)
	if err != nil {
		live.err = fmt.Errorf("languages: %w", err)
		return &e
	}
	if len(langs) > 0 {
		e.Langs = e.Langs[:0:0]
		for _, l := range langs {
			e.Langs = append(e.Langs, l.Code)
		}
		e.Default = langs[0].Code
	}
	e.Copy = content.NewRepository(mq)
	keys := make([]sql.NullString, 0, len(inboxalerts.Defs))
	for _, d := range lockedDefs() {
		keys = append(keys, sql.NullString{String: d.Key, Valid: true})
	}
	rows, err := q.ListSafetyRulesByKeys(ctx, keys)
	if err != nil {
		live.err = fmt.Errorf("rule rows: %w", err)
		return &e
	}
	for _, r := range rows {
		live.rows[r.RuleKey.String] = append(live.rows[r.RuleKey.String], r)
	}
	eng, err := loadLiveEngine(ctx, mq)
	if err != nil {
		live.err = fmt.Errorf("pregnancy rules: %w", err)
		return &e
	}
	live.engine = eng
	e.Preg = eng
	if live.vitals, err = vitals.LoadAlertCopy(ctx, vq); err != nil {
		live.err = err
	}
	return &e
}

// liveEngine is the served preg.* rules of the database compiled like the alerts inbox (inboxalerts snapshot): the
// served rows, plus the code-held definition of every locked rule that is not served (the emergency guarantee).
type liveEngine struct {
	list         []rules.Rule // owned: the compiled winners point into it
	comp         *compiled.Compiled
	byKey        map[string]*rules.Rule
	codeHeld     map[string]bool
	translations map[string]map[string]rules.Content
}

// storedContent is a message_rules content column: the base language's texts plus content.translations.<code>.
type storedContent struct {
	Translations map[string]rules.Content `json:"translations"`
}

func loadLiveEngine(ctx context.Context, mq mstore.Querier) (*liveEngine, error) {
	rows, err := mq.ListActiveMessageRules(ctx, mstore.ListActiveMessageRulesParams{Placement: rules.SlotAlertsInbox})
	if err != nil {
		return nil, err
	}
	if rows, err = rules.ServedRows(ctx, mq, rows); err != nil {
		return nil, err
	}
	e := &liveEngine{byKey: map[string]*rules.Rule{}, codeHeld: map[string]bool{}, translations: map[string]map[string]rules.Content{}}
	served := map[string]bool{}
	for i := range rows {
		row := &rows[i]
		if !strings.HasPrefix(row.RuleKey.String, inboxalerts.KeyPrefix) {
			continue
		}
		r := rules.FromRow(row)
		if !r.Served() {
			continue
		}
		e.list = append(e.list, r)
		served[r.RuleKey] = true
		var sc storedContent
		if json.Unmarshal([]byte(row.Content), &sc) == nil && len(sc.Translations) > 0 {
			e.translations[r.RuleKey] = sc.Translations
		}
	}
	for _, d := range lockedDefs() {
		if !served[d.Key] {
			e.list = append(e.list, d.Rule(inboxalerts.CodeRuleID(d.Key)))
			e.codeHeld[d.Key] = true
		}
	}
	for i := range e.list {
		e.byKey[e.list[i].RuleKey] = &e.list[i]
	}
	e.comp = compiled.CompileRules(e.list)
	return e, nil
}

// copyOf is the rule's texts in locale, as the inbox renders them (the row's base content with the locale's
// translation over it; the embedded copy for a code-held rule). ok is false when locale has no copy of its own (not
// the base language and no translation; base reports the default-language text).
func (e *liveEngine) copyOf(key, locale, def string) (inboxalerts.Copy, bool) {
	r := e.byKey[key]
	if r == nil || e.codeHeld[key] {
		return inboxalerts.CopyOf(key, locale)
	}
	cnt := r.Content
	t, has := e.translations[key][locale]
	if has {
		if t.A.Title != "" || t.A.Body != "" {
			cnt.A = t.A
		}
		for _, p := range []struct{ dst, src *string }{{&cnt.Seen, &t.Seen}, {&cnt.Confidence, &t.Confidence}, {&cnt.Contact, &t.Contact}} {
			if *p.src != "" {
				*p.dst = *p.src
			}
		}
	}
	return inboxalerts.Copy{Title: cnt.A.Title, Body: cnt.A.Body, Seen: cnt.Seen, Confidence: cnt.Confidence,
		Contact: cnt.Contact}, has || locale == def
}

// Fire runs the compiled alerts_inbox slot over the input's facts (comp.Select, as inbox.Sync does) and renders key's
// match with its evidence (inbox.EvidenceOf) in locale.
func (e *liveEngine) Fire(key string, in *facts.PregnancyAlertInput, locale string) (inboxalerts.Fired, bool) {
	f := inboxalerts.FactsOf(in)
	for _, ch := range e.comp.Select(inbox.Slot, f, nil, 0, in.Facts.Now) {
		r := ch.Rule
		if r.RuleKey != key {
			continue
		}
		if _, _, ok := inbox.EvidenceOf(r, f); !ok {
			return inboxalerts.Fired{}, false
		}
		cp, _ := e.copyOf(key, locale, "")
		return inboxalerts.Fired{Key: key, Level: r.Level, Title: inboxalerts.Render(cp.Title, f, locale),
			Seen: inboxalerts.Render(cp.Seen, f, locale), Confidence: inboxalerts.Render(cp.Confidence, f, locale),
			Ask: inboxalerts.Render(cp.Contact, f, locale)}, true
	}
	return inboxalerts.Fired{}, false
}

// ---- live configuration checks ----

// has115 reports whether text carries number in ASCII or the locale's digits.
func hasNumber(text, number, locale string) bool {
	return strings.Contains(text, number) || strings.Contains(text, vitals.Digits(number, locale))
}

func (e *Env) live(d *Detail) *Live {
	switch {
	case e.Live == nil:
		d.fail("no database")
		return nil
	case e.Live.err != nil:
		d.fail("load: %v", e.Live.err)
		return nil
	}
	return e.Live
}

// ConfigChecks read the live configuration.
var ConfigChecks = []Scenario{
	// Every locked emergency rule exists, is locked, active and served (approved), with its copy in every language.
	{"config.locked_rules", KindConfig, func(_ context.Context, e *Env) Detail {
		d := Detail{Threshold: "locked, active, approved, copy in every language"}
		l := e.live(&d)
		if l == nil {
			return d
		}
		for _, def := range lockedDefs() {
			rows := l.rows[def.Key]
			switch {
			case len(rows) == 0:
				d.fail("%s: no row", def.Key)
				continue
			case !rows[0].IsActive:
				d.fail("%s: inactive", def.Key)
			case l.engine.codeHeld[def.Key]:
				d.fail("%s: not approved", def.Key)
			}
			if !rows[0].Locked {
				d.fail("%s: not locked", def.Key)
			}
			if l.engine.codeHeld[def.Key] {
				continue
			}
			for _, loc := range e.Langs {
				cp, own := l.engine.copyOf(def.Key, loc, e.Default)
				if !own || cp.Title == "" || cp.Body == "" || cp.Seen == "" || cp.Contact == "" {
					d.fail("%s: no complete %s copy", def.Key, loc)
				}
			}
		}
		return d
	}},
	// Every emergency text calls 115 (EPDS item 10 also 123), in every language, as served today.
	{"config.emergency_copy", KindConfig, func(ctx context.Context, e *Env) Detail {
		d := Detail{Threshold: "115 in every emergency text; 123 for EPDS item 10"}
		l := e.live(&d)
		if l == nil {
			return d
		}
		need := func(where, loc, text string, numbers ...string) {
			for _, n := range numbers {
				if !hasNumber(text, n, loc) {
					d.fail("%s: %s text without %s", where, loc, n)
				}
			}
		}
		for _, loc := range e.Langs {
			for _, def := range lockedDefs() {
				cp, _ := l.engine.copyOf(def.Key, loc, e.Default)
				need(def.Key, loc, cp.Contact, postpartum.EmergencyNumber)
			}
			cp := e.copyFor(loc)
			for _, key := range append(append([]string{}, guide.RedFlagAlerts...), guide.AlertHeavyBleeding) {
				get := cp.Urgent
				if !postpartum.IsUrgentAlert(key) {
					get = cp.Text
				}
				t, err := get(ctx, guide.AlertGroup, key)
				if err != nil {
					d.fail("postpartum.%s: %v", key, err)
					continue
				}
				need("postpartum."+key, loc, t["title"]+" "+t["body"]+" "+t["action"], postpartum.EmergencyNumber)
			}
			t, err := cp.Text(ctx, guide.SafetyGroup, guide.SafetyUrgent)
			if err != nil {
				d.fail("epds.q10: %v", err)
			} else {
				need("epds.q10", loc, t["title"]+" "+t["body"]+" "+t["emergency_label"]+" "+t["social_label"],
					postpartum.EmergencyNumber, postpartum.SocialEmergencyNumber)
			}
			th := vitals.ThresholdsOn(e.Day)
			for _, r := range []vitals.Reading{
				{Type: vitals.TypeBP, Date: e.Day, Systolic: th.BPCrisisSystolic + 1, Diastolic: 80},
				{Type: vitals.TypeGlucose, Date: e.Day, MgDl: th.GlucoseUrgentBelow - 1},
			} {
				where := "vitals." + vitals.AlertRule(r)
				if !vitalsCalls(l.vitals.AlertJSON(r, loc, e.Default), postpartum.EmergencyNumber, loc) {
					d.fail("%s: %s text or call action without %s", where, loc, postpartum.EmergencyNumber)
				}
			}
		}
		return d
	}},
	// The registry has a valid version today for every emergency key (fields present and in range).
	{"config.registry", KindConfig, func(_ context.Context, e *Env) Detail {
		d := Detail{Threshold: "a valid version today for every emergency key"}
		for _, key := range EmergencyKeys {
			def, ok := clinicalconfig.Lookup(key)
			if !ok {
				d.fail("%s: unknown key", key)
				continue
			}
			v := clinicalconfig.At(key, e.Day)
			if v.Version < 1 {
				d.fail("%s: no version", key)
				continue
			}
			if p := def.Validate(v.Value); p != "" {
				d.fail("%s v%d: %s", key, v.Version, p)
			}
		}
		return d
	}},
}

// vitalsCalls reports whether a vitals urgent modal has a call action to number and says it in its texts.
func vitalsCalls(msg any, number, locale string) bool {
	m, ok := msg.(interface{ Get(string) (any, bool) })
	if !ok {
		return false
	}
	var text strings.Builder
	for _, k := range []string{"title", "what_we_saw", "advice", "contact"} {
		v, _ := m.Get(k)
		s, _ := v.(string)
		text.WriteString(s + " ")
	}
	raw, _ := m.Get("actions")
	list, _ := raw.([]any)
	call := false
	for _, a := range list {
		am, ok := a.(interface{ Get(string) (any, bool) })
		if !ok {
			continue
		}
		if p, _ := am.Get("phone"); p == number {
			call = true
		}
		l, _ := am.Get("label")
		s, _ := l.(string)
		text.WriteString(s + " ")
	}
	return call && hasNumber(text.String(), number, locale)
}
