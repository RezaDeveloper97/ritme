// Package content is the weekly pregnancy content of GET /pregnancy/content/{week}
// (PregnancyWeeklyController::content + PregnancyWeeklyContent::getLocalizedContent).
package content

import (
	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// MinWeek and MaxWeek bound the {week} parameter (outside → 422).
const (
	MinWeek = 1
	MaxWeek = 40
)

// Locale is the controller's clamp: the resolved request locale when it is fa or en,
// otherwise en (not the default language).
func Locale(resolved string) string { return i18n.Clamp(resolved, "en", i18n.LegacyPair...) }

// Localize builds data.content: each section is the locale's value when present, else the
// whole column (e.g. the full {en, fa} object), NULL stays null.
func Localize(row store.PregnancyWeeklyContent, locale string) *jsonx.OrderedMap {
	pick := func(col db.NullRawJSON) any {
		if !col.Valid {
			return nil
		}
		return i18n.PickOrWhole(col.V, locale)
	}
	return jsonx.Obj(
		"week_number", row.WeekNumber,
		"fetal_development", pick(row.FetalDevelopment),
		"mother_body_changes", pick(row.MotherBodyChanges),
		"dos_and_donts", pick(row.DosAndDonts),
		"care_plan", pick(row.CarePlan),
		"body_adaptation", pick(row.BodyAdaptation),
		"emotional_status", pick(row.EmotionalStatus),
		"key_nutrition", pick(row.KeyNutrition),
		"physical_activity", pick(row.PhysicalActivity),
		"tests_and_checkups", pick(row.TestsAndCheckups),
		"faq", pick(row.Faq),
	)
}
