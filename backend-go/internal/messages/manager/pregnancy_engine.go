package manager

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// pregnancyEngine is PregnancyMessageEngine
// (backend/app/Services/MessageSystem/Engines/PregnancyMessageEngine.php).
type pregnancyEngine struct {
	content Content
	locale  string
}

// weekMilestones are the keys of PregnancyMessageEngine::weekMilestones(), in source order
// (the ±2-week search keeps the first closest).
var weekMilestones = []int{4, 8, 12, 20, 28, 36, 40}

// base is getBaseMessage (Layer 1). The manager passes the 0-based gestational week, so week 0
// is falsy and gets the default message.
func (e pregnancyEngine) base(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if mc.PregnancyWeek == nil || *mc.PregnancyWeek == 0 {
		return e.defaultMessage(ctx)
	}
	week := *mc.PregnancyWeek
	trimester := mc.trimesterOr1()

	tm, err := e.trimesterMessage(ctx, trimester)
	if err != nil {
		return nil, err
	}
	ws, err := e.weekSpecific(ctx, week)
	if err != nil {
		return nil, err
	}
	// T-M7-04 layer 0: the admin's pregnancy_week_tip of the exact (1-based) week, the same row the
	// v2 Today card shows. Without a live row the payload is empty and the Laravel chain applies.
	tip, err := e.content.Resolve(ctx, WeekTipGroup, strconv.Itoa(week+1), e.locale)
	if err != nil {
		return nil, err
	}
	tipField := map[string]string{"short": "title", "long": "body"}
	chain := func(key string) any { // $tip ?? $weekSpecific[k] ?? $trimesterMessage[k] ?? ''
		if v, ok := tip.Field(tipField[key]); ok && v != "" {
			return v
		}
		if v, ok := ws.Field(key); ok {
			return v
		}
		return tm.Or(key, "")
	}
	out := jsonx.Obj(
		"week", week,
		"day", intOrNil(mc.PregnancyDay),
		"trimester", intOrNil(mc.Trimester),
		"trimester_label", e.trimesterLabel(trimester),
		"gestational_age", mc.GestationalAgeString(),
		"short_message", chain("short"),
		"long_message", chain("long"),
		"action_suggestion", chain("action"),
		"dos", tm.Or("dos", emptyList()),
		"donts", tm.Or("donts", emptyList()),
		"baby_development", ws.Or("baby", nil),
		"body_changes", ws.Or("body", nil),
	)
	if title, ok := tip.Field("title"); ok {
		out.Set("week_tip", jsonx.Obj(
			"week", week+1,
			"title", title,
			"body", tip.Or("body", nil),
			"read_minutes", tip.Or("read_minutes", nil),
			"article_url", tip.Or("article_url", nil),
		))
	}
	return out, nil
}

// WeekTipGroup is the admin-edited smart tip of each 1-based week (item_key 1..42).
const WeekTipGroup = "pregnancy_week_tip"

// override is getOverrideMessage (Layer 2); nil when none applies.
func (e pregnancyEngine) override(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	if len(mc.Symptoms) == 0 {
		return nil, nil
	}
	var t string
	switch {
	case mc.HasSymptom("nausea", "vomiting"):
		t = "nausea"
	case mc.HasSymptom("fatigue", "low_energy"):
		t = "fatigue"
	case mc.HasSymptom("backache"):
		t = "backache"
	case mc.HasSymptom("mood_anxious", "mood_sad"):
		t = "anxiety"
	default:
		return nil, nil
	}
	return e.overrideFor(ctx, t)
}

func (e pregnancyEngine) trimesterLabel(trimester int) string {
	fa := e.locale == "fa"
	switch trimester {
	case 1:
		if fa {
			return "سه‌ماهه اول" //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		}
		return "First Trimester"
	case 2:
		if fa {
			return "سه‌ماهه دوم" //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		}
		return "Second Trimester"
	case 3:
		if fa {
			return "سه‌ماهه سوم" //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		}
		return "Third Trimester"
	}
	if fa {
		return "نامشخص"
	}
	return "Unknown"
}

// defaultMessage is getDefaultMessage (group pregnancy_base / default).
func (e pregnancyEngine) defaultMessage(ctx context.Context) (*jsonx.OrderedMap, error) {
	p, err := e.content.Resolve(ctx, "pregnancy_base", "default", e.locale)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"week", nil,
		"trimester", nil,
		"short_message", p.Or("short", ""),
		"long_message", p.Or("long", ""),
		"action_suggestion", p.Or("action", ""),
		"dos", emptyList(),
		"donts", emptyList(),
	), nil
}

// trimesterMessage is getTrimesterMessage: the whole resolved array.
func (e pregnancyEngine) trimesterMessage(ctx context.Context, trimester int) (content.Payload, error) {
	const group = "pregnancy_trimester"
	return e.content.Resolve(ctx, group, itemOr(group, strconv.Itoa(trimester), "1"), e.locale)
}

// weekSpecific is getWeekSpecificMessage: the closest milestone within ±2 weeks, else a
// generated "Week N" short line (long / action / baby / body null).
func (e pregnancyEngine) weekSpecific(ctx context.Context, week int) (content.Payload, error) {
	closest, minDiff := -1, 100
	for _, m := range weekMilestones {
		diff := week - m
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff && diff <= 2 {
			minDiff, closest = diff, m
		}
	}
	if closest >= 0 {
		return e.content.Resolve(ctx, "pregnancy_week", strconv.Itoa(closest), e.locale)
	}
	short := fmt.Sprintf("Week %d of pregnancy", week)
	if e.locale == "fa" {
		short = fmt.Sprintf("هفته %d بارداری", week)
	}
	return content.NewPayload(jsonx.Obj("short", short, "long", nil, "action", nil, "baby", nil, "body", nil)), nil
}

// overrideFor is getOverrideForType; nil when the type has no copy.
func (e pregnancyEngine) overrideFor(ctx context.Context, t string) (*jsonx.OrderedMap, error) {
	const group = "pregnancy_override"
	if !content.HasItem(group, t) {
		return nil, nil
	}
	p, err := e.content.Resolve(ctx, group, t, e.locale)
	if err != nil {
		return nil, err
	}
	return jsonx.Obj(
		"override_type", t,
		"override_label", p.Or("label", ""),
		"override_message", p.Or("message", ""),
		"override_action", p.Or("action", ""),
		"override_dos", p.Or("dos", emptyList()),
		"override_donts", p.Or("donts", emptyList()),
	), nil
}
