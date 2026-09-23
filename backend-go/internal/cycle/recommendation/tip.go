package recommendation

import (
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Tip is one entry of the engine's bilingual `daily_tips`. Rows from the database carry a Title
// (Recommendation::toTip → {type, fa, en, title: {fa, en}}); the HealthDataEngine code fallback has
// none ({type, en, fa}). The fa/en pair is fixed by the PHP shapes, not a language list.
type Tip struct {
	Type  string
	FA    string
	EN    string
	Title *Title
}

// Title is the per-tip category title of a database tip.
type Title struct {
	FA string `json:"fa"`
	EN string `json:"en"`
}

// MarshalJSON writes the PHP array in its own key order.
func (t Tip) MarshalJSON() ([]byte, error) {
	if t.Title != nil {
		return jsonx.Obj("type", t.Type, "fa", t.FA, "en", t.EN, "title", t.Title).MarshalJSON()
	}
	return jsonx.Obj("type", t.Type, "en", t.EN, "fa", t.FA).MarshalJSON()
}

// text is `$tip[$locale] ?? $tip['en'] ?? null`: both keys always exist in either shape.
func (t Tip) text(locale string) string {
	if locale == "fa" {
		return t.FA
	}
	return t.EN
}

// LocalizedTip is DailyTipLocalizer's render-ready row.
type LocalizedTip struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Text  string `json:"text"`
}

// Localize is DailyTipLocalizer::localize: collapse bilingual tips to one locale, dropping tips
// without text, titling each from the tip itself or its category label.
// PHP: backend/app/Services/HealthEngine/DailyTipLocalizer.php:28.
func Localize(tips []Tip, locale string) []LocalizedTip {
	out := make([]LocalizedTip, 0, len(tips))
	for _, tip := range tips {
		text := tip.text(locale)
		if text == "" {
			continue
		}
		title := ""
		if tip.Title != nil {
			title = tip.Title.EN
			if locale == "fa" {
				title = tip.Title.FA
			}
		}
		if title == "" {
			title = enums.RecommendationTypeLabelFor(tip.Type, locale)
		}
		out = append(out, LocalizedTip{
			Type:  tip.Type,
			Title: title,
			Icon:  enums.RecommendationTypeIconFor(tip.Type),
			Text:  text,
		})
	}
	return out
}
