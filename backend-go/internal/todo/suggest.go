package todo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/todo/store"
)

// Cycle suggestion (Todo_Home «پریودت ۵ روز دیگر است؛ «خرید نوار بهداشتی» را اضافه کنم؟ · افزودن»): offered while the
// cycle engine's predicted next period (§35 resolver, the same date /home/cycle-overview shows) is 1–SuggestionWindowDays
// days away, for a cycle / TTC / teen account (or one without a stored mode), until the user accepts or dismisses it
// for that predicted period. Accepting adds the item to the newest open shopping list, or a new shopping task due the
// day before the period. The timing is code; the copy is message_contents todo_suggestion (admin-editable, fa + en
// seeded by goose 00043) in the request language, else the default language, else lang/<code>/todo.json.

// Suggestion keys and copy group.
const (
	SuggestionGroup          = "todo_suggestion"
	SuggestionPeriodSupplies = "period_supplies"
	// SuggestionWindowDays: the suggestion shows from this many days before the predicted period to the day before.
	SuggestionWindowDays = 5
	ActionAccepted       = "accepted"
	ActionDismissed      = "dismissed"
)

// Suggestion is the offer due today.
type Suggestion struct {
	Key         string
	PeriodStart civildate.Date
	Days        int // days until PeriodStart (1 … SuggestionWindowDays)
}

// suggestionModes are the stored life modes that get the period suggestion ("" = no stored mode yet).
var suggestionModes = []enums.LifeMode{"", enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModeTeen}

// PeriodSuggestion is the period-supplies offer for a predicted next period start (zero = no prediction).
func PeriodSuggestion(next, today civildate.Date, mode enums.LifeMode) (Suggestion, bool) {
	if next.IsZero() || !modeAllowed(mode) {
		return Suggestion{}, false
	}
	days := today.DiffDays(next)
	if days < 1 || days > SuggestionWindowDays {
		return Suggestion{}, false
	}
	return Suggestion{Key: SuggestionPeriodSupplies, PeriodStart: next, Days: days}, true
}

func modeAllowed(mode enums.LifeMode) bool {
	for _, m := range suggestionModes {
		if m == mode {
			return true
		}
	}
	return false
}

// TaskDue is the due date of the task an accepted suggestion creates: the day before the period, not before today.
func (s Suggestion) TaskDue(today civildate.Date) civildate.Date {
	d := s.PeriodStart.AddDays(-1)
	if d.Before(today) {
		return today
	}
	return d
}

// SuggestionCopy is the live copy rows: item_key → locale → fields.
type SuggestionCopy map[string]map[string]map[string]string

// LoadSuggestionCopy reads the live todo_suggestion rows (string fields only).
func LoadSuggestionCopy(ctx context.Context, q store.Querier) (SuggestionCopy, error) {
	rows, err := q.ListLiveSuggestionCopy(ctx)
	if err != nil {
		return nil, fmt.Errorf("todo: suggestion copy: %w", err)
	}
	out := SuggestionCopy{}
	for _, r := range rows {
		var raw map[string]any
		if json.Unmarshal(r.Payload, &raw) != nil || raw == nil {
			continue
		}
		m := map[string]string{}
		for k, v := range raw {
			if s, ok := v.(string); ok {
				m[k] = s
			}
		}
		if out[r.ItemKey] == nil {
			out[r.ItemKey] = map[string]map[string]string{}
		}
		out[r.ItemKey][r.Locale] = m
	}
	return out, nil
}

// copyFields are the payload keys a suggestion needs.
var copyFields = []string{"prompt", "prompt_tomorrow", "action", "task_title", "item_title"}

// Pick is the copy of key in locale: each field from the request-locale row, else the default-language row, else the
// embedded fallback of locale (English when the locale has no file).
func (c SuggestionCopy) Pick(key, locale, defaultLocale string) map[string]string {
	out := map[string]string{}
	for _, f := range copyFields {
		switch {
		case c[key][locale][f] != "":
			out[f] = c[key][locale][f]
		case c[key][defaultLocale][f] != "":
			out[f] = c[key][defaultLocale][f]
		default:
			out[f] = fallbackField(key, f, locale)
		}
	}
	return out
}

func fallbackField(key, field, locale string) string {
	k := "todo.suggestions." + key + "." + field
	if s := translator().Trans(k, nil, locale); s != k { // Get falls back to lang.FallbackLocale itself
		return s
	}
	return ""
}

// Digits writes the ASCII digits of s in the locale's digits (lang `digits`).
func Digits(s, locale string) string {
	digits := []rune(T("digits", locale))
	if len(digits) != 10 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(digits[r-'0'])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SuggestionJSON is the offer as the board shows it.
func SuggestionJSON(s Suggestion, cp map[string]string, locale string) *jsonx.OrderedMap {
	prompt := cp["prompt"]
	if s.Days == 1 && cp["prompt_tomorrow"] != "" {
		prompt = cp["prompt_tomorrow"]
	}
	prompt = strings.NewReplacer("{days}", Digits(strconv.Itoa(s.Days), locale), "{title}", cp["task_title"]).Replace(prompt)
	return jsonx.Obj(
		"key", s.Key,
		"prompt", prompt,
		"action", cp["action"],
		"days", s.Days,
		"period_start", s.PeriodStart.String(),
		"category", CategoryShopping,
		"task_title", cp["task_title"],
		"item_title", cp["item_title"],
	)
}
