// Package recommendation ports the admin-managed daily recommendations ("توصیه\u200cهای امروز"):
// backend/app/Models/Recommendation.php (appliesTo, toTip), backend/app/Services/HealthEngine/
// RecommendationRepository.php (request-scoped load + per-day matching) and DailyTipLocalizer.php.
//
// The database is reached only through Source, so the package stays pure; the sqlc adapter lives
// with the HTTP layer (T-M2-15).
package recommendation

import (
	"encoding/json"
	"slices"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Row is one `recommendations` row as the DB hands it over. JSON columns stay raw (the model's
// `array` casts are applied lazily, exactly where PHP reads them).
type Row struct {
	ID   int64
	Key  *string
	Type string
	// Title is the `title` JSON column (nil = NULL).
	Title json.RawMessage
	// Text is the `text` JSON column (NOT NULL).
	Text       json.RawMessage
	CyclePhase *string
	// CycleSubphases is the `cycle_subphases` JSON column (nil = NULL).
	CycleSubphases json.RawMessage
	SymptomTrigger *string
	IsActive       bool
	SortOrder      int
}

// AppliesTo is Recommendation::appliesTo: every target widens when blank (no phase = every phase,
// no sub-phase list = every sub-phase of the phase, no trigger = not symptom-gated).
// PHP: backend/app/Models/Recommendation.php:69.
func (r Row) AppliesTo(phase, subphase *string, triggers []string) bool {
	if r.CyclePhase != nil && (phase == nil || *r.CyclePhase != *phase) {
		return false
	}
	if r.SymptomTrigger != nil && !slices.Contains(triggers, *r.SymptomTrigger) {
		return false
	}

	// `$this->cycle_subphases ?: []` then in_array(..., true): only string entries can match.
	subphases := decodeList(r.CycleSubphases)
	if len(subphases) == 0 {
		return true
	}
	if subphase == nil {
		return false
	}
	for _, v := range subphases {
		if s, ok := v.(string); ok && s == *subphase {
			return true
		}
	}
	return false
}

// ToTip is Recommendation::toTip: bilingual text plus the resolved category title.
// PHP: backend/app/Models/Recommendation.php:91.
func (r Row) ToTip() Tip {
	typ := r.Type
	if typ == "" || typ == "0" { // `$this->type ?: GENERAL` — "0" is falsy in PHP
		typ = string(enums.RecommendationTypeGeneral)
	}
	text := decodeObject(r.Text)
	title := decodeObject(r.Title)

	return Tip{
		Type: typ,
		FA:   firstString(text, "fa", "en"),
		EN:   firstString(text, "en", "fa"),
		Title: &Title{
			FA: stringOr(title, "fa", enums.RecommendationTypeLabelFor(typ, "fa")),
			EN: stringOr(title, "en", enums.RecommendationTypeLabelFor(typ, "en")),
		},
	}
}

// decodeObject is the `array` cast read through `$attr['key']`: a JSON object, or nil for NULL,
// invalid JSON and non-objects (where every key read yields null).
func decodeObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// decodeList is the `array` cast of a JSON list column (object values count as entries too, as
// PHP arrays do; order is irrelevant for in_array).
func decodeList(raw json.RawMessage) []any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		out := make([]any, 0, len(t))
		for _, e := range t {
			out = append(out, e)
		}
		return out
	}
	return nil
}

// firstString is `(string) ($m[a] ?? $m[b] ?? "")`.
func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := phpString(m, k); ok {
			return s
		}
	}
	return ""
}

// stringOr is `(string) ($m[key] ?? $fallback)`.
func stringOr(m map[string]any, key, fallback string) string {
	if s, ok := phpString(m, key); ok {
		return s
	}
	return fallback
}

// phpString reads m[key] with `??` semantics (missing or null → not ok) and applies PHP's
// (string) cast to scalars.
func phpString(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return phpround.String(t), true // json_decode ints arrive as float64; "5" either way
	case bool:
		if t {
			return "1", true
		}
		return "", true
	}
	return "Array", true // (string) of an array: "Array" (with a warning)
}
