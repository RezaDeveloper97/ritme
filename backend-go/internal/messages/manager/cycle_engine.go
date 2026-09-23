package manager

import (
	"context"
	"errors"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// ErrUndefinedOverrideCase reproduces CycleMessageEngine::determineOverrideType reaching
// OverrideType::LOW_ENERGY / BLOATING / ACNE — cases the PHP enum does not declare, so PHP
// throws "Undefined constant" and the request is a 500 (D-11, locked by the premium goldens).
var ErrUndefinedOverrideCase = errors.New("messages: undefined constant App\\Enums\\OverrideType (CycleMessageEngine.php:102)")

// cycleEngine is CycleMessageEngine (backend/app/Services/MessageSystem/Engines/CycleMessageEngine.php).
type cycleEngine struct {
	content Content
	locale  string
}

// base is getBaseMessage (Layer 1).
func (e cycleEngine) base(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
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
	t, err := determineOverrideType(mc)
	if err != nil || t == "" {
		return nil, err
	}
	return e.overrideFor(ctx, t)
}

// determineOverrideType keeps the PHP priority order; the last three arms name enum cases
// that do not exist (see ErrUndefinedOverrideCase).
func determineOverrideType(mc *Context) (enums.OverrideType, error) {
	switch {
	case mc.HasSymptom("heavy_flow"):
		return enums.OverrideTypeHeavyFlow, nil
	case mc.HasSymptom("cramps", "headache", "backache"):
		return enums.OverrideTypePain, nil
	case mc.HasSymptom("mood_sad", "mood_anxious"):
		return enums.OverrideTypeMoodSad, nil
	case mc.HasSymptom("mood_angry"):
		return enums.OverrideTypeMoodAngry, nil
	case mc.HasSymptom("low_energy", "fatigue"): // OverrideType::LOW_ENERGY
		return "", ErrUndefinedOverrideCase
	case mc.HasSymptom("bloating"): // OverrideType::BLOATING
		return "", ErrUndefinedOverrideCase
	case mc.HasSymptom("acne"): // OverrideType::ACNE
		return "", ErrUndefinedOverrideCase
	}
	return "", nil
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

// overrideFor is getOverrideMessageForType; nil (PHP []) for a type without copy.
func (e cycleEngine) overrideFor(ctx context.Context, t enums.OverrideType) (*jsonx.OrderedMap, error) {
	const group = "cycle_override"
	if !content.HasItem(group, string(t)) {
		return nil, nil
	}
	p, err := e.content.Resolve(ctx, group, string(t), e.locale)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"override_type", string(t),
		"override_label", t.Label(e.locale),
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
