package companionhome

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n/lang"
)

// «امروز چه کار کنی؟» — the companion tips. Admin copy: message_contents group TipGroup (registered in
// internal/admin/messages/registry as CompanionTipGroup, typed items `<phase>_note` {body} and `<phase>_tip_<n>`
// {title, body}, placeholder `{name}`). A live row of the request locale wins, else one of the default language,
// else the embedded copy (lang/<code>/companion_home.json, English for a language without it). A written row is
// used as is: an empty title hides the tip, an empty body hides the note.

// TipGroup is the message_contents group of the tips.
const TipGroup = "companion_tip"

// Tip phases. The partner's cycle phase picks one; her active pregnancy (when shared) wins; general covers no cycle
// grant, no data or an unknown phase.
const (
	PhaseMenstrual  = "menstrual"
	PhaseFollicular = "follicular"
	PhaseFertile    = "fertile"
	PhaseLuteal     = "luteal"
	PhasePregnancy  = "pregnancy"
	PhaseGeneral    = "general"
)

// TipPhases are the phases in display order (= registry.CompanionTipPhases).
var TipPhases = []string{PhaseMenstrual, PhaseFollicular, PhaseFertile, PhaseLuteal, PhasePregnancy, PhaseGeneral}

// TipsPerPhase is how many tip slots each phase has (= registry.CompanionTipsPerPhase).
const TipsPerPhase = 3

// TipKeys are the item keys of TipGroup in registry order.
func TipKeys() []string {
	out := make([]string, 0, len(TipPhases)*(TipsPerPhase+1))
	for _, p := range TipPhases {
		out = append(out, noteKey(p))
		for i := 1; i <= TipsPerPhase; i++ {
			out = append(out, tipKey(p, i))
		}
	}
	return out
}

func noteKey(phase string) string       { return phase + "_note" }
func tipKey(phase string, n int) string { return phase + "_tip_" + strconv.Itoa(n) }

// PhaseOf maps the cycle engine's main phase to a tip phase (period expected = late luteal; unknown → general).
func PhaseOf(main enums.MainPhase) string {
	switch main {
	case enums.MainPhaseMenstrual:
		return PhaseMenstrual
	case enums.MainPhaseFollicular:
		return PhaseFollicular
	case enums.MainPhaseFertile:
		return PhaseFertile
	case enums.MainPhaseLuteal, enums.MainPhasePeriodExpected:
		return PhaseLuteal
	}
	return PhaseGeneral
}

//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the embedded companion-home line for key ("empty.title") in locale.
func T(key, locale string) string {
	return translator().Trans("companion_home."+key, nil, locale)
}

// Tip is one «امروز چه کار کنی؟» card.
type Tip struct {
	Key   string
	Title string
	Body  string
}

// tipCopy is the admin rows of TipGroup for one request (item key → locale → payload).
type tipCopy struct {
	rows          map[string]map[string]map[string]any
	locale, deflt string
}

func loadTipCopy(ctx context.Context, q *store.Queries, locale, deflt string) (*tipCopy, error) {
	locales := []string{locale}
	if deflt != "" && deflt != locale {
		locales = append(locales, deflt)
	}
	rows, err := q.ListCompanionTipContents(ctx, locales)
	if err != nil {
		return nil, fmt.Errorf("companion home: tips: %w", err)
	}
	tc := &tipCopy{rows: map[string]map[string]map[string]any{}, locale: locale, deflt: deflt}
	for _, r := range rows {
		var p map[string]any
		if json.Unmarshal(r.Payload, &p) != nil || p == nil {
			continue
		}
		if tc.rows[r.ItemKey] == nil {
			tc.rows[r.ItemKey] = map[string]map[string]any{}
		}
		tc.rows[r.ItemKey][r.Locale] = p
	}
	return tc, nil
}

// field is the admin row's text (ok=true when a row of the locale or the default language exists), else the
// embedded copy.
func (tc *tipCopy) field(key, field string) string {
	if byLocale := tc.rows[key]; byLocale != nil {
		for _, l := range []string{tc.locale, tc.deflt} {
			if p, ok := byLocale[l]; ok {
				s, _ := p[field].(string)
				return strings.TrimSpace(s)
			}
		}
	}
	k := "tips." + key // a note is one string, a tip an object {title, body}
	if !strings.HasSuffix(key, "_note") {
		k += "." + field
	}
	line, ok := translator().Get("companion_home."+k, tc.locale)
	if !ok {
		return ""
	}
	s, _ := line.(string)
	return s
}

// note is the phase line under the cycle card ("" = none), with {name} filled.
func (tc *tipCopy) note(phase, name string) string {
	return fill(tc.field(noteKey(phase), "body"), name)
}

// tips are the phase's visible tips, {name} filled.
func (tc *tipCopy) tips(phase, name string) []Tip {
	out := make([]Tip, 0, TipsPerPhase)
	for i := 1; i <= TipsPerPhase; i++ {
		k := tipKey(phase, i)
		title := fill(tc.field(k, "title"), name)
		if title == "" {
			continue
		}
		out = append(out, Tip{Key: k, Title: title, Body: fill(tc.field(k, "body"), name)})
	}
	return out
}

func fill(s, name string) string { return strings.ReplaceAll(s, "{name}", name) }
