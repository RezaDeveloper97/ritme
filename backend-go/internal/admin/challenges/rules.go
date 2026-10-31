// Package challenges is the admin side of the v6 challenge rules (T-M10-10, SPEC §5.5, D-167): validation and storage
// of eligibility / contraindications in the {all_of, any_of, exclude} shape over the message fact registry, the
// on_join / on_phase_start / fixed_date start rules, and the day-by-day drop-off stats. The challenge CRUD itself
// stays in internal/admin/content (challenges_v2.go), which calls in here.
package challenges

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/admin/content/form"
	chs "github.com/ritme/backend-go/internal/challenges"
	chstore "github.com/ritme/backend-go/internal/challenges/store"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Field names of the v6 request keys.
const (
	FieldEligibility       = "eligibility"
	FieldContraindications = "contraindications"
	FieldStartRule         = "start_rule"
	FieldStartPhase        = "start_phase"
)

// StartRuleValues are the start_rule values the API accepts: the v6 ones plus the legacy anytime (stored as on_join)
// and menstrual (kept: any day of the menstrual phase, not only its first).
var StartRuleValues = append(slices.Clone(chs.StartRules), chs.StartAnytime, chs.StartMenstrual)

// MaxConditions bounds one rule.
const MaxConditions = 30

// Rules are the Laravel-style rules of the v6 fields (all optional). The rule objects are checked by Check.
func Rules() validation.Rules {
	return validation.Rules{
		validation.F(FieldStartRule, "nullable", validation.In(StartRuleValues...)),
		validation.F(FieldStartPhase, "nullable", validation.In(chs.StartPhases...)),
		validation.F(FieldEligibility, "nullable|array"),
		validation.F(FieldContraindications, "nullable|array"),
	}
}

// decodeRule reads a request value as a rule. An empty object or list ([] / {}) is no rule.
func decodeRule(v any) (chs.Rule, error) {
	if v == nil {
		return chs.Rule{}, nil
	}
	b := bytes.TrimSpace(form.JSON(v))
	if bytes.Equal(b, []byte("[]")) || bytes.Equal(b, []byte("{}")) || bytes.Equal(b, []byte("null")) {
		return chs.Rule{}, nil
	}
	if len(b) == 0 || b[0] != '{' {
		return chs.Rule{}, errors.New("a rule is an object {all_of, any_of, exclude}")
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		return chs.Rule{}, err
	}
	for k := range keys {
		if k != "all_of" && k != "any_of" && k != "exclude" {
			return chs.Rule{}, fmt.Errorf("unknown key «%s» (all_of, any_of, exclude)", k)
		}
	}
	var r chs.Rule
	// An empty section is sent as [] (a PHP array): decode each one alone so [] and {} both read as empty.
	for k, raw := range keys {
		if t := bytes.TrimSpace(raw); bytes.Equal(t, []byte("{}")) || bytes.Equal(t, []byte("null")) {
			continue
		}
		var err error
		switch k {
		case "all_of":
			err = json.Unmarshal(raw, &r.AllOf)
		case "any_of":
			err = json.Unmarshal(raw, &r.AnyOf)
		case "exclude":
			err = json.Unmarshal(raw, &r.Exclude)
		}
		if err != nil {
			return chs.Rule{}, fmt.Errorf("«%s» is a list of conditions {fact, op, value}", k)
		}
	}
	return r, nil
}

// checkRule validates every condition of a rule against the fact registry (rules.CheckFact).
func checkRule(field string, r chs.Rule, add form.Add) {
	if len(r.Conditions()) > MaxConditions {
		add(field, fmt.Sprintf("A rule has at most %d conditions.", MaxConditions))
		return
	}
	check := func(path string, c rules.Condition) {
		if c.Fact == "" || !slices.Contains(rules.Ops, c.Op) {
			add(path, "A condition needs a fact and a valid operator.")
			return
		}
		if msg, code := rules.CheckFact(c); code != "" {
			add(path, msg)
		}
	}
	for i, c := range r.AllOf {
		check(fmt.Sprintf("%s.all_of.%d", field, i), c)
	}
	for i, g := range r.AnyOf {
		for j, c := range g {
			check(fmt.Sprintf("%s.any_of.%d.%d", field, i, j), c)
		}
	}
	for i, c := range r.Exclude {
		check(fmt.Sprintf("%s.exclude.%d", field, i), c)
	}
}

// Check validates the rule objects and the start rule: on_phase_start needs start_phase.
func Check(_ fiber.Ctx) form.Check {
	return func(in phpval.Map, add form.Add) error {
		for _, f := range []string{FieldEligibility, FieldContraindications} {
			v, ok := in.Get(f)
			if !ok || v == nil {
				continue
			}
			r, err := decodeRule(v)
			if err != nil {
				add(f, "The "+f+" rule is invalid: "+err.Error()+".")
				continue
			}
			checkRule(f, r, add)
		}
		if rule, _ := in.Get(FieldStartRule); rule == chs.StartOnPhaseStart {
			if p, _ := in.Get(FieldStartPhase); p == nil || p == "" {
				add(FieldStartPhase, "An on_phase_start challenge needs the phase it starts on.")
			}
		}
		return nil
	}
}

// StartRule is the start rule a request stores: anytime and "" are on_join.
func StartRule(data phpval.Map) string {
	s, _ := data.Get(FieldStartRule)
	switch v := phpval.ToString(s); v {
	case "", chs.StartAnytime:
		return chs.StartOnJoin
	default:
		return v
	}
}

func list(data phpval.Map, key string) []string {
	v, _ := data.Get(key)
	_, vals := phpval.Entries(v)
	out := make([]string, 0, len(vals))
	for _, x := range vals {
		if s := phpval.ToString(x); !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func ruleColumn(r chs.Rule) db.NullRawJSON {
	if r.Empty() {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: form.JSON(r.Normalized()), Valid: true}
}

// Save writes the v6 rule columns of a validated request whose program row exists (call it after UpsertProgram, in
// the same transaction). The eligibility is the request's `eligibility` object, else its legacy include / exclude
// lists converted to the v6 shape; the contraindications are the request's object, else the stored ones are kept (a
// save from an older editor never drops them). start_phase is stored only for on_phase_start.
func Save(ctx context.Context, q *chstore.Queries, id uint64, data phpval.Map, now sql.NullTime) error {
	cur, err := q.GetProgram(ctx, id)
	if err != nil {
		return fmt.Errorf("admin challenges: program: %w", err)
	}
	var elig chs.Rule
	if v, ok := data.Get(FieldEligibility); ok {
		if elig, err = decodeRule(v); err != nil {
			return fmt.Errorf("admin challenges: eligibility: %w", err)
		}
	} else {
		elig = chs.FromLegacy(list(data, "modes_include"), list(data, "modes_exclude"),
			list(data, "phases_include"), list(data, "phases_exclude"))
	}
	contra := cur.Contraindications
	if v, ok := data.Get(FieldContraindications); ok {
		r, err := decodeRule(v)
		if err != nil {
			return fmt.Errorf("admin challenges: contraindications: %w", err)
		}
		contra = ruleColumn(r)
	}
	start := StartRule(data)
	var phase sql.NullString
	if p := phpval.ToString(mustGet(data, FieldStartPhase)); start == chs.StartOnPhaseStart && p != "" {
		phase = sql.NullString{String: p, Valid: true}
	}
	if err := q.SetProgramRules(ctx, chstore.SetProgramRulesParams{
		Eligibility: ruleColumn(elig), Contraindications: contra, StartRule: start, StartPhase: phase,
		Now: now, ChallengeID: id,
	}); err != nil {
		return fmt.Errorf("admin challenges: rules: %w", err)
	}
	return nil
}

func mustGet(data phpval.Map, key string) any {
	v, _ := data.Get(key)
	return v
}

// ruleJSON is a stored rule as the API echoes it (always the v6 shape; a legacy row converted); null = no rule. A
// stored value that does not parse is echoed as {"invalid": true} (a contraindication then refuses everyone).
func ruleJSON(raw db.NullRawJSON) any {
	r := chs.ParseRule(raw)
	if r.Broken {
		return jsonx.Obj("invalid", true)
	}
	if r.Empty() {
		return nil
	}
	n := r.Normalized()
	return jsonx.Obj("all_of", n.AllOf, "any_of", n.AnyOf, "exclude", n.Exclude)
}

// SetProgramJSON adds the v6 keys to a program object: eligibility and contraindications (v6 shape) and start_phase.
func SetProgramJSON(m *jsonx.OrderedMap, p chstore.ChallengeProgram) {
	m.Set(FieldEligibility, ruleJSON(p.Eligibility))
	m.Set(FieldContraindications, ruleJSON(p.Contraindications))
	var phase any
	if p.StartPhase.Valid && p.StartPhase.String != "" {
		phase = p.StartPhase.String
	}
	m.Set(FieldStartPhase, phase)
}

// DropOff is the day-by-day drop-off of a challenge (one query): [{day, reached, retained, dropped, retained_rate,
// drop_off_rate}], rates in percent of the runs that reached the day (null when none did).
func DropOff(ctx context.Context, q *chstore.Queries, id uint64, duration int, today civildate.Date) ([]*jsonx.OrderedMap, error) {
	runs, err := q.ProgramRuns(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("admin challenges: runs: %w", err)
	}
	return DropOffJSON(chs.DropOff(runs, duration, today)), nil
}

// DropOffJSON renders the retention rows.
func DropOffJSON(days []chs.DayRetention) []*jsonx.OrderedMap {
	rate := func(p int) any {
		if p < 0 {
			return nil
		}
		return p
	}
	out := make([]*jsonx.OrderedMap, 0, len(days))
	for _, d := range days {
		out = append(out, jsonx.Obj("day", d.Day, "reached", d.Reached, "retained", d.Retained, "dropped", d.Dropped,
			"retained_rate", rate(d.RetainedPct), "drop_off_rate", rate(d.DroppedPct)))
	}
	return out
}
