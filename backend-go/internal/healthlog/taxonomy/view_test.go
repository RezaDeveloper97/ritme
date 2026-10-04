package taxonomy_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/resources/translations"
)

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := jsonx.Marshal(v, jsonx.UnescapedUnicode|jsonx.UnescapedSlashes)
	require.NoError(t, err)
	return string(b)
}

func TestDayJSON_RoundTripsThePutShape(t *testing.T) {
	body := `{"categories":{"note":{"text":"x"},"bleeding":{"spotting":true,"flow":"light"},` +
		`"pain":{"location":{"abdomen":{"level":"moderate","score":6},"head":"mild"},"relief":["heat"]},` +
		`"measurements":{"weight":58.4,"heart_rate":72},"meds":{"other":{"supplements":"iron"}}}}`
	changes, ve := parse(t, body, taxonomy.ModePregnancy)
	require.Nil(t, ve)
	var entries []taxonomy.Entry
	for _, ch := range changes {
		entries = append(entries, ch.Entries...)
	}
	assert.Equal(t,
		`{"bleeding":{"flow":"light","spotting":true},`+
			`"pain":{"location":{"abdomen":{"level":"moderate","score":6},"head":{"level":"mild","score":null}},"relief":["heat"]},`+
			`"measurements":{"weight":58.4,"heart_rate":72},"meds":{"other":{"supplements":"iron"}},"note":{"text":"x"}}`,
		marshal(t, taxonomy.DayJSON(entries)), "taxonomy order, normalized values")
	assert.Equal(t, `{}`, marshal(t, taxonomy.DayJSON(nil)))

	// a param the registry no longer knows still shows (after the known ones)
	odd := []taxonomy.Entry{{Category: "zzz", Param: "old", Code: sql.NullString{String: "v", Valid: true}}}
	assert.Equal(t, `{"zzz":{"old":"v"}}`, marshal(t, taxonomy.DayJSON(odd)))
}

func TestCategoriesJSON_ModeFilterAndLabels(t *testing.T) {
	store := i18n.NewTranslationStore(translations.FS, "")
	fa := taxonomy.NewLabels(store.NamespaceMessages("fa", "log-taxonomy", "fa"))

	post := marshal(t, taxonomy.CategoriesJSON(taxonomy.ModePostpartum, fa))
	assert.Contains(t, post, `"code":"lochia_amount"`)
	assert.NotContains(t, post, `"code":"flow"`)
	assert.Contains(t, post, `"code":"baby"`)
	assert.NotContains(t, post, `"code":"pregnancy"`)
	assert.Contains(t, post, "\"label\":\"پریود و لکه\u200cبینی\"")

	cycle := marshal(t, taxonomy.CategoriesJSON(taxonomy.ModeCycle, fa))
	assert.NotContains(t, cycle, `"legacy_only":true`, "legacy-only values are not offered")
	assert.NotContains(t, cycle, `"value":"stitches"`)

	all := marshal(t, taxonomy.CategoriesJSON("", fa))
	assert.Contains(t, all, `"value":"red","label":"قرمز","modes":null,"legacy_only":true`)
	assert.Equal(t, len(taxonomy.Categories()), strings.Count(all, `"group":{`))

	// missing labels fall back to the code
	bare := marshal(t, taxonomy.CategoriesJSON(taxonomy.ModeCycle, taxonomy.NewLabels(nil)))
	assert.Contains(t, bare, `{"code":"bleeding","label":"bleeding"`)
}

// CB-TEEN-04b: teen has no BBT param, so its measurements category is titled «وزن», not «وزن و دمای پایه»;
// every other mode (and the all-modes listing) keeps the category title.
func TestCategoriesJSON_ModeTitle(t *testing.T) {
	store := i18n.NewTranslationStore(translations.FS, "")
	for locale, want := range map[string][2]string{"fa": {"وزن", "وزن و دمای پایه"}, "en": {"Weight", "Weight & BBT"}} {
		l := taxonomy.NewLabels(store.NamespaceMessages(locale, "log-taxonomy", "fa"))
		teen := marshal(t, taxonomy.CategoriesJSON(taxonomy.ModeTeen, l))
		assert.Contains(t, teen, `{"code":"measurements","label":"`+want[0]+`"`, locale)
		assert.NotContains(t, teen, `"code":"bbt"`, locale)
		for _, mode := range []string{taxonomy.ModeCycle, taxonomy.ModeTTC, ""} {
			assert.Contains(t, marshal(t, taxonomy.CategoriesJSON(mode, l)), `{"code":"measurements","label":"`+want[1]+`"`, "%s %q", locale, mode)
		}
	}
}
