package taxonomy_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

func parse(t *testing.T, body, mode string, existing ...taxonomy.Entry) ([]taxonomy.Change, *httpx.ValidationError) {
	t.Helper()
	changes, err := taxonomy.Parse(validation.DecodeBody([]byte(body)), mode, "en", existing, testCustom)
	if err == nil {
		return changes, nil
	}
	var ve *httpx.ValidationError
	require.ErrorAs(t, err, &ve)
	return nil, ve
}

// testCustom: the user's active custom items in these tests.
var testCustom = taxonomy.CustomItems{"custom.items.coffee": true, "mood.moods.custom_7": true}

func fields(ve *httpx.ValidationError) []string {
	var out []string
	b := ve.Body()
	errs, _ := b.Get("errors")
	if m, ok := errs.(interface{ Keys() []string }); ok {
		out = m.Keys()
	}
	return out
}

func TestParse_AllTypes(t *testing.T) {
	changes, ve := parse(t, `{"categories": {
		"bleeding": {"flow": "medium", "spotting": false, "color": "dark_red"},
		"pain": {"location": {"abdomen": {"level": "moderate", "score": 6}, "head": "mild"}, "relief": ["heat", "rest"]},
		"measurements": {"weight": 58.456, "bbt": "36.42", "heart_rate": null},
		"meds": {"other": {"supplements": "آهن"}, "taken": {"12": "yes"}},
		"note": {"text": "سلام"},
		"custom": {"items": {"coffee": "yes"}}
	}}`, taxonomy.ModeCycle)
	require.Nil(t, ve)
	got := map[string][]flat{}
	for _, ch := range changes {
		got[ch.Key()] = flatten(ch.Entries)
	}
	assert.Equal(t, []flat{{Slot: "bleeding.flow.", Code: "c:medium"}}, got["bleeding.flow"])
	assert.Equal(t, []flat{{Slot: "bleeding.spotting.", Code: "c:no"}}, got["bleeding.spotting"])
	assert.Equal(t, []flat{
		{Slot: "pain.location.abdomen", Code: "c:moderate", Num: "n:6.00"},
		{Slot: "pain.location.head", Code: "c:mild"},
	}, got["pain.location"])
	assert.Equal(t, []flat{{Slot: "pain.relief.heat", Code: "c:yes"}, {Slot: "pain.relief.rest", Code: "c:yes"}}, got["pain.relief"])
	assert.Equal(t, []flat{{Slot: "measurements.weight.", Num: "n:58.46"}}, got["measurements.weight"])
	assert.Equal(t, []flat{{Slot: "measurements.bbt.", Num: "n:36.42"}}, got["measurements.bbt"])
	assert.Empty(t, got["measurements.heart_rate"], "null clears the param")
	assert.Contains(t, got, "measurements.heart_rate")
	assert.Equal(t, []flat{{Slot: "meds.other.supplements", Text: "t:آهن"}}, got["meds.other"])
	assert.Equal(t, []flat{{Slot: "meds.taken.12", Code: "c:yes"}}, got["meds.taken"])
	assert.Equal(t, []flat{{Slot: "custom.items.coffee", Code: "c:yes"}}, got["custom.items"])
}

func TestParse_NullCategoryClearsEveryStorableParam(t *testing.T) {
	changes, ve := parse(t, `{"categories": {"pregnancy": null, "sleep": null}}`, taxonomy.ModePregnancy)
	require.Nil(t, ve)
	var keys []string
	for _, ch := range changes {
		keys = append(keys, ch.Key())
		assert.Empty(t, ch.Entries)
	}
	assert.Equal(t, []string{"sleep.duration", "sleep.quality"}, keys, "link params are never stored")
}

func TestParse_RangesAndTypes(t *testing.T) {
	_, ve := parse(t, `{"categories": {
		"measurements": {"weight": 19, "bbt": 42.5, "heart_rate": 72.5},
		"pain": {"location": {"abdomen": {"level": "moderate", "score": 11}, "head": {"level": "awful"}}},
		"activity": {"duration": 601},
		"note": {"text": 5},
		"bleeding": {"spotting": "yes", "flow": ["medium"]},
		"mood": {"moods": ["calm", "calm", "ecstatic"]}
	}}`, taxonomy.ModeCycle)
	require.NotNil(t, ve)
	assert.ElementsMatch(t, []string{
		"categories.measurements.weight", "categories.measurements.bbt", "categories.measurements.heart_rate",
		"categories.pain.location.abdomen.score", "categories.pain.location.head.level",
		"categories.activity.duration", "categories.note.text",
		"categories.bleeding.spotting", "categories.bleeding.flow",
		"categories.mood.moods.1", "categories.mood.moods.2",
	}, fields(ve))
	assert.Equal(t, []string{"The categories.measurements.weight field must be between 20 and 300."}, ve.Messages("categories.measurements.weight"))
}

func TestParse_UnknownKeysAndReadOnly(t *testing.T) {
	_, ve := parse(t, `{"categories": {"nope": {}, "bleeding": {"nope": 1}, "pregnancy": {"kicks": 8}, "symptoms": {"general": {"telepathy": "yes"}}}}`, taxonomy.ModePregnancy)
	require.NotNil(t, ve)
	assert.ElementsMatch(t, []string{
		"categories.nope", "categories.bleeding.nope", "categories.pregnancy.kicks", "categories.symptoms.general.telepathy",
	}, fields(ve))

	_, ve = parse(t, `{}`, taxonomy.ModeCycle)
	require.NotNil(t, ve)
	assert.Equal(t, []string{"categories"}, fields(ve))
	_, ve = parse(t, `{"categories": "x"}`, taxonomy.ModeCycle)
	require.NotNil(t, ve)
}

func TestParse_ModeSpecificParams(t *testing.T) {
	cases := []struct {
		name, body, mode string
		ok               bool
	}{
		{"lochia in postpartum", `{"categories":{"bleeding":{"lochia_amount":"heavy"}}}`, taxonomy.ModePostpartum, true},
		{"lochia in cycle", `{"categories":{"bleeding":{"lochia_amount":"heavy"}}}`, taxonomy.ModeCycle, false},
		{"flow in postpartum", `{"categories":{"bleeding":{"flow":"heavy"}}}`, taxonomy.ModePostpartum, false},
		{"bbt in ttc", `{"categories":{"measurements":{"bbt":36.5}}}`, taxonomy.ModeTTC, true},
		{"bbt in pregnancy", `{"categories":{"measurements":{"bbt":36.5}}}`, taxonomy.ModePregnancy, false},
		{"vigorous activity in pregnancy", `{"categories":{"activity":{"intensity":"high"}}}`, taxonomy.ModePregnancy, false},
		{"light activity in pregnancy", `{"categories":{"activity":{"intensity":"low"}}}`, taxonomy.ModePregnancy, true},
		{"stitches pain postpartum", `{"categories":{"pain":{"location":{"stitches":"severe"}}}}`, taxonomy.ModePostpartum, true},
		{"stitches pain cycle", `{"categories":{"pain":{"location":{"stitches":"severe"}}}}`, taxonomy.ModeCycle, false},
		{"sex in teen", `{"categories":{"sex":{"desire":"normal"}}}`, taxonomy.ModeTeen, false},
		{"breasts in postpartum", `{"categories":{"breasts":{"symptoms":{"engorgement":"moderate"}}}}`, taxonomy.ModePostpartum, true},
		{"night sweats in menopause", `{"categories":{"symptoms":{"general":{"night_sweats":"yes"}}}}`, taxonomy.ModeMenopause, true},
		{"night sweats in cycle", `{"categories":{"symptoms":{"general":{"night_sweats":"yes"}}}}`, taxonomy.ModeCycle, false},
		{"legacy-only colour", `{"categories":{"bleeding":{"color":"red"}}}`, taxonomy.ModeCycle, false},
		{"clearing an out-of-mode param", `{"categories":{"bleeding":{"lochia_amount":null}}}`, taxonomy.ModeCycle, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ve := parse(t, tc.body, tc.mode)
			assert.Equal(t, tc.ok, ve == nil, "%v", ve)
		})
	}
}

// A stored value always passes again (legacy-only, verbatim legacy or logged in another mode), so a re-save
// of the day never fails.
func TestParse_StoredValuesPass(t *testing.T) {
	code := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
	existing := []taxonomy.Entry{
		{Category: "bleeding", Param: "color", Code: code("red")},
		{Category: "bleeding", Param: "flow", Code: code("extreme")},
		{Category: "mood", Param: "moods", Item: "weird", Code: code("yes")},
		{Category: "pain", Param: "location", Item: "head", Code: code("odd")},
		{Category: "measurements", Param: "bbt", Num: sql.NullString{String: "36.50", Valid: true}},
		{Category: "meds", Param: "other", Item: "vitamin d", Text: code("5")},
	}
	_, ve := parse(t, `{"categories":{
		"bleeding":{"color":"red","flow":"extreme"},
		"mood":{"moods":["weird","calm"]},
		"pain":{"location":{"head":"odd"}},
		"measurements":{"bbt":36.5},
		"meds":{"other":{"vitamin d":"5"}}
	}}`, taxonomy.ModePregnancy, existing...)
	assert.Nil(t, ve)

	_, ve = parse(t, `{"categories":{"measurements":{"bbt":36.6}}}`, taxonomy.ModePregnancy, existing...)
	assert.NotNil(t, ve, "a changed value must be available in the mode")
}

func TestParse_CustomItems(t *testing.T) {
	// the user's active custom items pass in their param (items and multi)
	changes, ve := parse(t, `{"categories": {"custom": {"items": {"coffee": "yes"}}, "mood": {"moods": ["calm", "custom_7"]}}}`, taxonomy.ModeCycle)
	require.Nil(t, ve)
	require.Len(t, changes, 2)
	assert.Equal(t, "custom_7", changes[1].Entries[1].Item)

	// anything else is refused: an unknown code, another user's item, an item of another param
	_, ve = parse(t, `{"categories": {"custom": {"items": {"custom_99": "yes"}}}}`, taxonomy.ModeCycle)
	require.NotNil(t, ve)
	assert.Equal(t, []string{"categories.custom.items.custom_99"}, fields(ve))
	_, ve = parse(t, `{"categories": {"mood": {"moods": ["coffee"]}}}`, taxonomy.ModeCycle)
	require.NotNil(t, ve)
	assert.Equal(t, []string{"categories.mood.moods.0"}, fields(ve))
	_, ve = parse(t, `{"categories": {"symptoms": {"general": {"custom_7": "yes"}}}}`, taxonomy.ModeCycle)
	require.NotNil(t, ve)

	// a deleted custom item (not in the active set) stays valid where it is stored
	stored := taxonomy.Entry{Category: "custom", Param: "items", Item: "custom_3", Code: sql.NullString{String: "yes", Valid: true}}
	_, ve = parse(t, `{"categories": {"custom": {"items": {"custom_3": "no"}}}}`, taxonomy.ModeCycle, stored)
	assert.Nil(t, ve)

	// care reminder ids in meds.taken stay free codes
	_, ve = parse(t, `{"categories": {"meds": {"taken": {"42": "yes"}}}}`, taxonomy.ModeCycle)
	assert.Nil(t, ve)
}
