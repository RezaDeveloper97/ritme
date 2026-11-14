package companion

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// preferenceRules are the Laravel-style rules of the optional «ریتمی همراه» v2 keys (POST /companions and PUT
// /companions/{id}/preferences). null means "not sent". Unknown signal keys are refused by parsePreferences.
func preferenceRules() validation.Rules {
	return validation.Rules{
		validation.F("goal", "nullable", "string", validation.In(goalStrings()...)),
		validation.F("signals", "nullable", "array"),
		validation.F("signals.pain", "nullable", "boolean"),
		validation.F("signals.mood", "nullable", "boolean"),
		validation.F("signals.energy", "nullable", "boolean"),
		validation.F("signals.sleep", "nullable", "boolean"),
		validation.F("action_prefs", "nullable", "array", "max:"+strconv.Itoa(MaxActionPrefs)),
		validation.F("action_prefs.*", "string", validation.In(actionStrings()...)),
	}
}

// parsePreferences reads the validated optional keys as a patch (absent / null keys stay nil). An unknown key inside
// signals is a 422 on signals.<key>; repeated action kinds are deduped keeping the first one's place.
func parsePreferences(locale string, body phpval.Map) (PreferencesPatch, error) {
	var p PreferencesPatch
	if raw, ok := body.Get("goal"); ok && raw != nil {
		g := Goal(phpval.ToString(raw))
		p.Goal = &g
	}
	if raw, ok := body.Get("signals"); ok && raw != nil {
		keys, vals := phpval.Entries(raw)
		for i, k := range keys {
			s := Signal(k)
			if !s.Valid() {
				return p, fieldError(locale, "signals."+k, T("validation.signal_invalid", locale))
			}
			if vals[i] == nil {
				continue
			}
			if p.Signals == nil {
				p.Signals = SignalScope{}
			}
			p.Signals[s] = phpval.Truthy(vals[i])
		}
	}
	if raw, ok := body.Get("action_prefs"); ok && raw != nil {
		_, vals := phpval.Entries(raw)
		acts := make([]ActionPref, 0, len(vals))
		for _, v := range vals {
			acts = append(acts, ActionPref(phpval.ToString(v)))
		}
		acts, err := NormalizeActions(acts)
		if err != nil {
			return p, fieldError(locale, "action_prefs", T("validation.action_prefs_invalid", locale))
		}
		p.Actions = &acts
	}
	return p, nil
}

// UpdatePreferences is PUT /companions/{id}/preferences {goal?, signals?, action_prefs?}: a partial update of the
// owner's «ریتمی همراه» v2 settings of one of her links (only the sent keys change). 404 for anyone else's link.
func (h *Handlers) UpdatePreferences(c fiber.Ctx) error {
	ownerID, err := currentUser(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := linkID(c)
	if !ok {
		return notFoundErr(locale)
	}
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, preferenceRules(), validation.Now(h.now(c)),
		validation.Messages("goal.in", T("validation.goal_invalid", locale),
			"action_prefs.*.in", T("validation.action_prefs_invalid", locale)))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	patch, err := parsePreferences(locale, body)
	if err != nil {
		return err
	}
	svc := h.svc(c)
	if _, err := svc.UpdatePreferences(c.Context(), ownerID, id, patch); err != nil {
		return mapError(locale, err)
	}
	l, err := svc.OwnerLink(c.Context(), ownerID, id)
	if err != nil {
		return mapError(locale, err)
	}
	return h.ownerLinkOK(c, svc, l, T("messages.preferences_updated", locale))
}

// signalsScopeJSON is the owner's consent per signal, every key present (card order).
func signalsScopeJSON(sc SignalScope) *jsonx.OrderedMap {
	out := jsonx.NewObject()
	for _, s := range SignalOrder {
		out.Set(string(s), sc.On(s))
	}
	return out
}

func actionsJSON(acts []ActionPref) []string {
	out := make([]string, 0, len(acts))
	for _, a := range acts {
		out = append(out, string(a))
	}
	return out
}

// SettingsJSON is what a member sees of the owner's preferences: the goal and the action kinds, never the signal
// consents themselves (the home shows a signal's value or null instead).
func SettingsJSON(p Preferences) *jsonx.OrderedMap {
	return jsonx.Obj("goal", string(p.Goal.orUnknown()), "action_prefs", actionsJSON(p.Actions))
}

func goalStrings() []string {
	out := make([]string, len(Goals))
	for i, g := range Goals {
		out[i] = string(g)
	}
	return out
}

func actionStrings() []string {
	out := make([]string, len(ActionPrefs))
	for i, a := range ActionPrefs {
		out[i] = string(a)
	}
	return out
}
