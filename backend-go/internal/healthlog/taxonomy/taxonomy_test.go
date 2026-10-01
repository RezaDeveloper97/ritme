package taxonomy_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/resources/translations"
)

func TestRegistry_CodesUniqueAndWellFormed(t *testing.T) {
	cats := map[string]bool{}
	for _, c := range taxonomy.Categories() {
		require.False(t, cats[c.Code], "duplicate category %s", c.Code)
		cats[c.Code] = true
		assert.NotEmpty(t, c.Modes, c.Code)
		for _, m := range c.Modes {
			assert.True(t, taxonomy.IsMode(m), "%s: mode %s", c.Code, m)
		}
		params := map[string]bool{}
		for i := range c.Params {
			p := &c.Params[i]
			key := c.Code + "." + p.Code
			require.False(t, params[p.Code], "duplicate param %s", key)
			params[p.Code] = true
			for _, m := range c.ParamModes(p) {
				assert.Contains(t, c.Modes, m, "%s: param mode outside the category", key)
			}
			opts := map[string]bool{}
			for _, o := range p.Options {
				require.False(t, opts[o.Code], "duplicate option %s.%s", key, o.Code)
				opts[o.Code] = true
				assert.LessOrEqual(t, len(o.Code), 64, key)
			}
			switch p.Type {
			case taxonomy.Single, taxonomy.Multi:
				assert.NotEmpty(t, p.Options, key)
			case taxonomy.Items:
				assert.NotEmpty(t, p.Levels, key)
				assert.True(t, p.Dynamic || len(p.Options) > 0, key)
			case taxonomy.Number, taxonomy.Integer:
				assert.NotNil(t, p.Range, key)
			case taxonomy.Text, taxonomy.TextItems:
				assert.Positive(t, p.MaxLen, key)
			case taxonomy.Link:
				assert.NotEmpty(t, p.Source, key)
			}
		}
	}
}

// Every data column of daily_health_logs has exactly one mapping, onto a slot the registry knows, with a
// param type that can hold it and injective renames into registry codes.
func TestLegacyMapping_CoversEveryColumn(t *testing.T) {
	mapped := map[string]bool{}
	for _, lc := range taxonomy.LegacyColumns() {
		require.False(t, mapped[lc.Column], "column %s mapped twice", lc.Column)
		mapped[lc.Column] = true
		_, ok := model.ColumnByName(lc.Column)
		require.True(t, ok, "%s is not a daily_health_logs column", lc.Column)

		cat, ok := taxonomy.CategoryByCode(lc.Category)
		require.True(t, ok, lc.Column)
		p, ok := cat.Param(lc.Param)
		require.True(t, ok, lc.Column)
		want := map[taxonomy.Kind][]taxonomy.Type{
			taxonomy.KindEnum: {taxonomy.Single}, taxonomy.KindBool: {taxonomy.Bool},
			taxonomy.KindLevel: {taxonomy.Items}, taxonomy.KindBoolItem: {taxonomy.Items},
			taxonomy.KindNumber: {taxonomy.Number, taxonomy.Integer}, taxonomy.KindText: {taxonomy.Text},
			taxonomy.KindArray: {taxonomy.Multi}, taxonomy.KindMeds: {taxonomy.TextItems},
		}[lc.Kind]
		assert.Contains(t, want, p.Type, lc.Column)
		if lc.Kind == taxonomy.KindLevel || lc.Kind == taxonomy.KindBoolItem {
			_, ok := p.Option(lc.Item)
			assert.True(t, ok, "%s: item %s", lc.Column, lc.Item)
		}
		seen := map[string]bool{}
		for _, v := range lc.LegacySet {
			code := v
			if r, ok := lc.Rename[v]; ok {
				code = r
			}
			require.False(t, seen[code], "%s: rename not injective at %s", lc.Column, code)
			seen[code] = true
			if lc.Kind == taxonomy.KindLevel {
				assert.Contains(t, p.Levels, code, lc.Column)
			} else {
				_, ok := p.Option(code)
				assert.True(t, ok, "%s: legacy value %s → %s is not an option", lc.Column, v, code)
			}
		}
	}
	for _, c := range model.Columns {
		assert.True(t, mapped[c.Name], "column %s has no v2 mapping", c.Name)
	}
}

func TestLabels_CompleteInEveryLanguage(t *testing.T) {
	store := i18n.NewTranslationStore(translations.FS, "")
	for _, code := range []string{"fa", "en"} {
		ns := store.RawNamespace(code, "log-taxonomy")
		has := func(path string) {
			v, ok := phpval.Get(ns, path)
			s, _ := v.(string)
			assert.True(t, ok && s != "", "%s: missing label %s", code, path)
		}
		units, levels, conds := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, c := range taxonomy.Categories() {
			has("categories." + c.Code + ".title")
			has("groups." + c.Group)
			for _, v := range c.Conditions {
				conds[v] = true
			}
			for _, p := range c.Params {
				base := "categories." + c.Code + ".params." + p.Code
				has(base + ".title")
				for _, o := range p.Options {
					has(base + ".options." + o.Code)
				}
				for _, l := range p.Levels {
					levels[l] = true
				}
				if p.Unit != "" {
					units[p.Unit] = true
				}
			}
		}
		for u := range units {
			has("units." + u)
		}
		for l := range levels {
			has("levels." + l)
		}
		for c := range conds {
			has("conditions." + c)
		}
	}
}

func TestLabels_FrontendCopyIsByteIdentical(t *testing.T) {
	for _, code := range []string{"fa", "en"} {
		seed, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", "translations", code, "log-taxonomy.json"))
		require.NoError(t, err)
		front, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "frontend", "messages", code, "log-taxonomy.json"))
		if os.IsNotExist(err) {
			t.Skip("frontend/ not checked out next to backend-go")
		}
		require.NoError(t, err)
		assert.Equal(t, string(front), string(seed), code)
	}
}

// The goose migration carries BackfillPrepared() and its Laravel twin BackfillSQL(), verbatim. A mapping
// change after 00020 shipped belongs in a new migration (and this test then pins the new text there).
func TestBackfillSQL_InMigrations(t *testing.T) {
	goose, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "migrations", "00020_health_log_entries.sql"))
	require.NoError(t, err)
	assert.Contains(t, string(goose), "-- backfill:begin\n"+taxonomy.BackfillPrepared()+"-- backfill:end\n")

	php, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "backend", "database", "migrations",
		"2026_10_01_000020_create_health_log_entries_table.php"))
	if os.IsNotExist(err) {
		t.Skip("backend/ not checked out next to backend-go")
	}
	require.NoError(t, err)
	assert.Contains(t, string(php), "<<<'SQL'\n"+taxonomy.BackfillSQL()+"SQL;\n")
	assert.Len(t, taxonomy.BackfillStatements(), strings.Count(taxonomy.BackfillSQL(), "INSERT IGNORE"))
}

func get(row map[string]any) func(string) taxonomy.LegacyValue {
	return func(c string) taxonomy.LegacyValue { return row[c] }
}

type flat struct{ Slot, Code, Num, Text string }

func flatten(es []taxonomy.Entry) []flat {
	out := make([]flat, 0, len(es))
	for _, e := range es {
		f := flat{Slot: e.Slot()}
		if e.Code.Valid {
			f.Code = "c:" + e.Code.String
		}
		if e.Num.Valid {
			f.Num = "n:" + e.Num.String
		}
		if e.Text.Valid {
			f.Text = "t:" + e.Text.String
		}
		out = append(out, f)
	}
	return out
}

func TestProject_TypicalRow(t *testing.T) {
	row := map[string]any{
		"bleeding_intensity": "low", "blood_color": "red", "has_clots": true, "clots_amount": "high", "spotting": false,
		"headache_intensity": "medium", "food_craving": true, "vaginal_burning": true, "vaginal_burning_intensity": "high",
		"vaginal_itching": false, "moods": json.RawMessage(`["calm","happy","calm"]`), "sleep_quality": "bad",
		"exercise_duration": int64(45), "weight": "58.40", "blood_sugar": "97.5",
		"medications": json.RawMessage(`{"painkillers":"ibuprofen","supplements":"iron"}`), "notes": "سلام",
	}
	got := flatten(taxonomy.ProjectRow(get(row)))
	assert.Equal(t, []flat{
		{Slot: "bleeding.flow.", Code: "c:light"},
		{Slot: "bleeding.color.", Code: "c:red"},
		{Slot: "bleeding.clots.", Code: "c:yes"},
		{Slot: "bleeding.clot_size.", Code: "c:large"},
		{Slot: "bleeding.spotting.", Code: "c:no"},
		{Slot: "pain.location.head", Code: "c:moderate"},
		{Slot: "appetite_energy.cravings.any", Code: "c:yes"},
		{Slot: "urogenital.symptoms.vaginal_burning", Code: "c:severe"}, // the intensity wins over the boolean
		{Slot: "urogenital.symptoms.vaginal_itching", Code: "c:no"},
		{Slot: "mood.moods.calm", Code: "c:yes"},
		{Slot: "mood.moods.happy", Code: "c:yes"},
		{Slot: "sleep.quality.", Code: "c:poor"},
		{Slot: "activity.duration.", Num: "n:45.00"},
		{Slot: "measurements.weight.", Num: "n:58.40"},
		{Slot: "measurements.blood_sugar.", Num: "n:97.50"},
		{Slot: "meds.other.painkillers", Text: "t:ibuprofen"},
		{Slot: "meds.other.supplements", Text: "t:iron"},
		{Slot: "note.text.", Text: "t:سلام"},
	}, got)
}

func TestProject_OddLegacyValuesAreKept(t *testing.T) {
	long := strings.Repeat("ب", 300)
	row := map[string]any{
		"bleeding_intensity": "extreme",                                        // not an enum value: verbatim
		"blood_color":        "",                                               // empty string: kept
		"moods":              json.RawMessage(`{"1":"calm","3":"sad"}`),        // PHP array with gaps
		"exercise_type":      json.RawMessage(`"yoga"`),                        // a bare string
		"sexual_activities":  json.RawMessage(`["x", {"a": 1}, null, 5, "x"]`), // nested, null, number, dup
		"medications":        json.RawMessage(`["a","b"]`),                     // not an object
		"notes":              long,
	}
	got := flatten(taxonomy.ProjectRow(get(row)))
	assert.Equal(t, []flat{
		{Slot: "bleeding.flow.", Code: "c:extreme"},
		{Slot: "bleeding.color.", Code: "c:"},
		{Slot: "mood.moods.calm", Code: "c:yes"},
		{Slot: "mood.moods.sad", Code: "c:yes"},
		{Slot: "activity.types.yoga", Code: "c:yes"},
		{Slot: "sex.symptoms.x", Code: "c:yes"},
		{Slot: `sex.symptoms.{"a": 1}`, Code: "c:yes"},
		{Slot: "sex.symptoms.5", Code: "c:yes"},
		{Slot: "meds.other._raw", Text: `t:["a","b"]`},
		{Slot: "note.text.", Text: "t:" + long},
	}, got)

	long2 := map[string]any{"moods": json.RawMessage(`["` + long + `"]`)}
	es := taxonomy.ProjectRow(get(long2))
	require.Len(t, es, 1)
	assert.Equal(t, strings.Repeat("ب", 191), es[0].Item, "array values are cut to the item width")
}

// Every legacy value round-trips: Reverse(Project(v)) == v.
func TestReverse_RoundTripsEveryLegacyValue(t *testing.T) {
	for _, lc := range taxonomy.LegacyColumns() {
		var values []any
		switch lc.Kind {
		case taxonomy.KindEnum, taxonomy.KindLevel:
			for _, v := range lc.LegacySet {
				values = append(values, v)
			}
			values = append(values, "odd_value")
		case taxonomy.KindBool, taxonomy.KindBoolItem:
			values = []any{true, false}
		}
		for _, v := range values {
			if lc.Column == "vaginal_burning" || lc.Column == "vaginal_itching" {
				continue // shares its item with the intensity column (covered below)
			}
			es := lc.Project(v)
			assert.Equal(t, v, lc.Reverse(es), "%s = %v", lc.Column, v)
		}
	}
	// arrays and medications
	moods, _ := taxonomy.LegacyColumnByName("moods")
	assert.Equal(t, []any{"calm", "sad"}, moods.Reverse(moods.Project(json.RawMessage(`["calm","sad"]`))))
	meds, _ := taxonomy.LegacyColumnByName("medications")
	m := meds.Reverse(meds.Project(json.RawMessage(`{"painkillers":"x","vitamin d":"5"}`))).(phpval.Map)
	assert.Equal(t, []string{"painkillers", "vitamin d"}, m.Keys())
	// a merged pair: true + high → severe → true + high
	burn, _ := taxonomy.LegacyColumnByName("vaginal_burning")
	burnI, _ := taxonomy.LegacyColumnByName("vaginal_burning_intensity")
	es := taxonomy.ProjectRow(get(map[string]any{"vaginal_burning": true, "vaginal_burning_intensity": "high"}))
	assert.Equal(t, true, burn.Reverse(es))
	assert.Equal(t, "high", burnI.Reverse(es))
}

func TestReverse_V2OnlyValues(t *testing.T) {
	e := func(cat, param, item, code string) taxonomy.Entry {
		x := taxonomy.Entry{Category: cat, Param: param, Item: item}
		x.Code.String, x.Code.Valid = code, true
		return x
	}
	color, _ := taxonomy.LegacyColumnByName("blood_color")
	assert.Nil(t, color.Reverse([]taxonomy.Entry{e("bleeding", "color", "", "pink")}), "no legacy pink")
	odor, _ := taxonomy.LegacyColumnByName("bleeding_smell")
	assert.Equal(t, "slightly_unusual", odor.Reverse([]taxonomy.Entry{e("bleeding", "odor", "", "changed")}))
	ex, _ := taxonomy.LegacyColumnByName("exercise_type")
	assert.Equal(t, []any{"yoga", "other"}, ex.Reverse([]taxonomy.Entry{
		e("activity", "types", "yoga", "yes"), e("activity", "types", "pilates", "yes"), e("activity", "types", "stretching", "yes"),
	}))
	moods, _ := taxonomy.LegacyColumnByName("moods")
	assert.Nil(t, moods.Reverse([]taxonomy.Entry{e("mood", "moods", "energetic", "yes")}), "v2-only moods are dropped")
	nausea, _ := taxonomy.LegacyColumnByName("nausea_intensity")
	assert.Nil(t, nausea.Reverse([]taxonomy.Entry{e("symptoms", "digestive", "nausea", "yes")}))
}

func TestSlotsOf(t *testing.T) {
	params, items := taxonomy.SlotsOf([]string{"moods", "headache_intensity", "vaginal_burning", "vaginal_burning_intensity"})
	assert.Equal(t, []string{"mood.moods"}, params)
	assert.Equal(t, []string{"pain.location.head", "urogenital.symptoms.vaginal_burning"}, items)
	all, _ := taxonomy.SlotsOf(nil)
	assert.True(t, slices.Contains(all, "meds.other"))
}
