package vitals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// Urgent safety message (B-N6-01, pattern of the pregnancy alert engine B-N5-03): a saved reading over a safety
// threshold (thresholds.go: BP > 180 and/or > 120, glucose < 54 mg/dL) returns `alert` with the copy of a modal —
// title, what we saw, advice, emergency contact and the actions (call with the number, ack). The thresholds are code;
// the copy is message_contents group vitals_alert (admin-editable, fa + en seeded by goose 00035) in the request
// language, else the default language, else the embedded lang/<code>/vitals.json `alerts` fallback.

// Alert rules.
const (
	AlertGroup      = "vitals_alert"
	RuleBPCrisis    = "bp_crisis"
	RuleGlucoseLow  = "glucose_low"
	AlertLevel      = "urgent"
	alertValueToken = "{value}"
)

// AlertRule is the rule a reading trips ("" = none).
func AlertRule(r Reading) string {
	switch {
	case r.Type == TypeBP && UrgentBP(r.Systolic, r.Diastolic):
		return RuleBPCrisis
	case r.Type == TypeGlucose && UrgentGlucose(r.MgDl):
		return RuleGlucoseLow
	}
	return ""
}

// AlertCopy is the live copy rows: item_key → locale → payload.
type AlertCopy map[string]map[string]map[string]any

// LoadAlertCopy reads the live vitals_alert rows.
func LoadAlertCopy(ctx context.Context, q store.Querier) (AlertCopy, error) {
	rows, err := q.ListLiveAlertCopy(ctx)
	if err != nil {
		return nil, fmt.Errorf("vitals: alert copy: %w", err)
	}
	out := AlertCopy{}
	for _, r := range rows {
		var m map[string]any
		if json.Unmarshal(r.Payload, &m) != nil || m == nil {
			continue
		}
		if out[r.ItemKey] == nil {
			out[r.ItemKey] = map[string]map[string]any{}
		}
		out[r.ItemKey][r.Locale] = m
	}
	return out, nil
}

// fallbackCopy is the embedded alerts entry of locale (English when the locale has no file).
func fallbackCopy(rule, locale string) map[string]any {
	for _, l := range []string{locale, lang.FallbackLocale} {
		b, err := langFS.ReadFile("lang/" + l + "/vitals.json")
		if err != nil {
			continue
		}
		var f struct {
			Alerts map[string]map[string]any `json:"alerts"`
		}
		if json.Unmarshal(b, &f) == nil && f.Alerts[rule] != nil {
			return f.Alerts[rule]
		}
	}
	return map[string]any{}
}

// pick is the request-locale payload, else the default-language one, else the embedded fallback.
func (c AlertCopy) pick(rule, locale, defaultLocale string) map[string]any {
	if m, ok := c[rule][locale]; ok {
		return m
	}
	if m, ok := c[rule][defaultLocale]; ok {
		return m
	}
	return fallbackCopy(rule, locale)
}

// AlertValue is the reading as shown in the copy («۱۸۲/۱۲۱ mmHg», «۴۸ mg/dL» in the unit typed).
func AlertValue(r Reading, locale string) string {
	var s string
	switch r.Type {
	case TypeBP:
		s = fmt.Sprintf("%d/%d mmHg", int(r.Systolic), int(r.Diastolic))
	case TypeGlucose:
		if r.Unit == UnitMmolL {
			s = phpround.String(MmolL(r.MgDl)) + " mmol/L"
		} else {
			s = phpround.String(MgDlRounded(r.MgDl)) + " mg/dL"
		}
	}
	return Digits(s, locale)
}

// Digits writes the ASCII digits (and decimal point) of s in the locale's digits (lang `digits`,
// `decimal_separator`).
func Digits(s, locale string) string {
	digits := []rune(T("digits", locale))
	if len(digits) != 10 {
		return s
	}
	sep := T("decimal_separator", locale)
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(digits[r-'0'])
		case r == '.' && i > 0 && i+1 < len(rs) && rs[i-1] >= '0' && rs[i-1] <= '9' && rs[i+1] >= '0' && rs[i+1] <= '9':
			b.WriteString(sep)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func text(m map[string]any, key, value string) any {
	s, ok := m[key].(string)
	if !ok || s == "" {
		return nil
	}
	return strings.ReplaceAll(s, alertValueToken, value)
}

// AlertJSON is the urgent payload of a reading, or nil when the reading is under the thresholds.
func (c AlertCopy) AlertJSON(r Reading, locale, defaultLocale string) any {
	rule := AlertRule(r)
	if rule == "" {
		return nil
	}
	m := c.pick(rule, locale, defaultLocale)
	value := AlertValue(r, locale)
	actions := []any{}
	if list, ok := m["actions"].([]any); ok {
		for _, a := range list {
			am, ok := a.(map[string]any)
			if !ok {
				continue
			}
			key, _ := am["key"].(string)
			label, _ := am["label"].(string)
			if key == "" || label == "" {
				continue
			}
			act := jsonx.Obj("key", key, "label", label)
			if phone, ok := am["phone"].(string); ok && phone != "" {
				act.Set("phone", phone)
			}
			actions = append(actions, act)
		}
	}
	return jsonx.Obj(
		"rule", rule,
		"level", AlertLevel,
		"modal", true,
		"title", text(m, "title", value),
		"what_we_saw", text(m, "what_we_saw", value),
		"advice", text(m, "advice", value),
		"contact", text(m, "contact", value),
		"actions", actions,
	)
}
