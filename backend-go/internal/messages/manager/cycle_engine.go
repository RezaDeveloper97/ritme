package manager

import (
	"context"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// cycleEngine is CycleMessageEngine (backend/app/Services/MessageSystem/Engines/CycleMessageEngine.php).
type cycleEngine struct {
	content Content
	locale  string
}

// base is getBaseMessage (Layer 1).
func (e cycleEngine) base(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if mc.neutral() && mc.V6Anchor {
		return e.neutralBase(ctx, mc)
	}
	if mc.CyclePhase == "" {
		return e.defaultMessage(), nil
	}
	phase := enums.CyclePhase(mc.CyclePhase)
	if mc.IsTTC() {
		return e.ttcBase(ctx, phase, mc)
	}
	return e.nonTTCBase(ctx, phase, mc)
}

// override is getOverrideMessage (Layer 2); nil when none applies.
func (e cycleEngine) override(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if len(mc.Symptoms) == 0 {
		return nil, nil
	}
	oc, ok := determineOverride(mc)
	if !ok {
		return nil, nil
	}
	return e.overrideFor(ctx, oc)
}

// overrideCase is one layer-2 symptom: its cycle_override item key and the label shown when the admin's row has none.
type overrideCase struct {
	key   string
	label func(locale string) string
}

func enumCase(t enums.OverrideType) overrideCase { return overrideCase{string(t), t.Label} }

// determineOverride keeps the PHP priority order. Laravel's last three arms named OverrideType cases that do not
// exist, so a low-energy, bloating or acne day was a 500 (D-11); Go serves their cycle_override copy instead
// (deviation D-214). Their labels reuse the symptom names the app already has.
func determineOverride(mc *Context) (overrideCase, bool) {
	switch {
	case mc.HasSymptom("heavy_flow"):
		return enumCase(enums.OverrideTypeHeavyFlow), true
	case mc.HasSymptom("cramps", "headache", "backache"):
		return enumCase(enums.OverrideTypePain), true
	case mc.HasSymptom("mood_sad", "mood_anxious"):
		return enumCase(enums.OverrideTypeMoodSad), true
	case mc.HasSymptom("mood_angry"):
		return enumCase(enums.OverrideTypeMoodAngry), true
	case mc.HasSymptom("low_energy", "fatigue"):
		return overrideCase{"low_energy", enums.RecommendationTriggerFatigue.Label}, true
	case mc.HasSymptom("bloating"):
		return overrideCase{"bloating", enums.RecommendationTriggerBloating.Label}, true
	case mc.HasSymptom("acne"):
		return overrideCase{"acne", enums.OverrideTypeSkinAcne.Label}, true
	}
	return overrideCase{}, false
}

// itemOr is `array_key_exists($key, <source>) ? $key : $fallbackKey`.
func itemOr(group, key, fallbackKey string) string {
	if content.HasItem(group, key) {
		return key
	}
	return fallbackKey
}

func (e cycleEngine) nonTTCBase(ctx context.Context, phase enums.CyclePhase, mc *Context) (*jsonx.OrderedMap, error) {
	const group = "cycle_base_non_ttc"
	p, err := e.content.Resolve(ctx, group, itemOr(group, string(phase), "default"), e.locale)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"phase", string(phase),
		"phase_label", phase.Label(e.locale),
		"cycle_day", intOrNil(mc.CycleDay),
		"short_message", p.Or("short", ""),
		"long_message", p.Or("long", ""),
		"action_suggestion", p.Or("action", ""),
		"dos", p.Or("dos", emptyList()),
		"donts", p.Or("donts", emptyList()),
	), nil
}

func (e cycleEngine) ttcBase(ctx context.Context, phase enums.CyclePhase, mc *Context) (*jsonx.OrderedMap, error) {
	const group = "cycle_base_ttc"
	p, err := e.content.Resolve(ctx, group, itemOr(group, string(phase), "default"), e.locale)
	if err != nil {
		return nil, err
	}
	var fertilityInfo any
	if mc.IsFertileWindow {
		if e.locale == "fa" {
			fertilityInfo = "شما در پنجره باروری هستید - بهترین زمان برای باردار شدن!"
		} else {
			fertilityInfo = "You are in your fertile window - best time to conceive!"
		}
	}
	return jsonx.Obj(
		"phase", string(phase),
		"phase_label", phase.Label(e.locale),
		"cycle_day", intOrNil(mc.CycleDay),
		"is_fertile_window", mc.IsFertileWindow,
		"fertility_info", fertilityInfo,
		"short_message", p.Or("short", ""),
		"long_message", p.Or("long", ""),
		"action_suggestion", p.Or("action", ""),
		"dos", p.Or("dos", emptyList()),
		"donts", p.Or("donts", emptyList()),
		"ttc_tips", p.Or("ttc_tips", emptyList()),
	), nil
}

// overrideFor is getOverrideMessageForType; nil (PHP []) for a type without copy. A `label` written in the admin's
// row wins over the built-in symptom name.
func (e cycleEngine) overrideFor(ctx context.Context, oc overrideCase) (*jsonx.OrderedMap, error) {
	const group = "cycle_override"
	if !content.HasItem(group, oc.key) {
		return nil, nil
	}
	p, err := e.content.Resolve(ctx, group, oc.key, e.locale)
	if err != nil {
		return nil, err
	}
	label := oc.label(e.locale)
	if l, _ := p.Field("label"); l != nil {
		if s, _ := l.(string); s != "" {
			label = s
		}
	}
	return jsonx.Obj(
		"override_type", oc.key,
		"override_label", label,
		"override_message", p.Or("message", ""),
		"override_action", p.Or("action", ""),
		"override_dos", p.Or("dos", emptyList()),
		"override_donts", p.Or("donts", emptyList()),
	), nil
}

// defaultMessage is getDefaultMessage (hard-coded fa / English).
func (e cycleEngine) defaultMessage() *jsonx.OrderedMap {
	fa := e.locale == "fa"
	pick := func(faText, enText string) string {
		if fa {
			return faText
		}
		return enText
	}
	return jsonx.Obj(
		"phase", nil,
		"phase_label", nil,
		"short_message", pick("به سلامت خود توجه کنید", "Take care of your health"),
		"long_message", pick("برای دریافت پیام‌های شخصی‌سازی شده، لطفاً تاریخ آخرین پریود خود را وارد کنید.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
			"Please enter your last period date to receive personalized messages."),
		"action_suggestion", pick("پروفایل خود را تکمیل کنید", "Complete your profile"),
		"dos", emptyList(),
		"donts", emptyList(),
	)
}

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// emptyList is PHP [] (encodes as []).
func emptyList() []any { return []any{} }
