package healthrecord

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func TestRecordsLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Document added", RT("messages.document_created", "en"))
	assert.Equal(t, "سند اضافه شد", RT("messages.document_created", "fa"))
	assert.Equal(t, "You can keep at most 500 documents.", RTp("validation.too_many_documents", map[string]string{"max": "500"}, "en"))
	assert.NotEmpty(t, recordAttributes("fa"))
	assert.NotEmpty(t, recordAttributes("en"))
}

func TestTimelineFilters(t *testing.T) {
	assert.Equal(t, []string{"all", "lab", "imaging", "visit", "prescription", "hospital", "other"}, TimelineFilters)
}

func TestMonths_GroupsByJalaliMonth(t *testing.T) {
	rows := []entry{
		{date: d("2026-10-06"), render: jsonx.Obj("id", 1)},
		{date: d("2026-09-23"), render: jsonx.Obj("id", 2)}, // 1 Mehr 1405
		{date: d("2026-09-22"), render: jsonx.Obj("id", 3)}, // 31 Shahrivar 1405
		{date: d("2025-03-04"), render: jsonx.Obj("id", 4)}, // 14 Esfand 1403
	}
	b, err := json.Marshal(months(rows))
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"key":"1405-07","jalali_year":1405,"jalali_month":7,"start":"2026-09-23","end":"2026-10-22","items":[{"id":1},{"id":2}]},
		{"key":"1405-06","jalali_year":1405,"jalali_month":6,"start":"2026-08-23","end":"2026-09-22","items":[{"id":3}]},
		{"key":"1403-12","jalali_year":1403,"jalali_month":12,"start":"2025-02-19","end":"2025-03-20","items":[{"id":4}]}
	]`, string(b))
	assert.Equal(t, []*jsonx.OrderedMap{}, months(nil))
}

func TestLabEntry(t *testing.T) {
	e, ok := labEntry(jsonx.Obj("id", uint64(9), "category", "blood", "title", "CBC", "date", "2026-09-09",
		"marker_count", 14, "attention_count", 2, "all_normal", false))
	require.True(t, ok)
	assert.True(t, e.isLab)
	assert.Equal(t, uint64(9), e.id)
	assert.Equal(t, d("2026-09-09"), e.date)
	b, err := json.Marshal(e.render)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"lab","id":9,"kind":"lab","title":"CBC","date":"2026-09-09","date_known":true,
		"ended_on":null,"centre":null,"doctor":null,"file_count":null,"review_state":null,
		"lab":{"category":"blood","marker_count":14,"attention_count":2,"all_normal":false},"links":[]}`, string(b))

	_, ok = labEntry(jsonx.Obj("id", uint64(1), "date", "soon"))
	assert.False(t, ok)
}

func TestExtrasJSONAndLists(t *testing.T) {
	assert.Nil(t, decodeList[Surgery](db.NullRawJSON{}))
	assert.Equal(t, []Surgery{}, decodeList[Surgery](db.NullRawJSON{V: []byte(`[]`), Valid: true}))
	assert.False(t, encodeList[Surgery](nil).Valid)
	assert.Equal(t, `[{"title":"C","date":""}]`, string(encodeList([]Surgery{{Title: "C"}}).V))

	b, err := json.Marshal(Extras{AllergiesOnCard: true, Surgeries: []Surgery{{Title: "C", Date: "2025-03-04"}},
		FamilyHistory: []FamilyItem{{Condition: "Diabetes"}}}.JSON())
	require.NoError(t, err)
	assert.JSONEq(t, `{"allergies":null,"allergies_on_emergency_card":true,
		"surgeries":[{"title":"C","date":"2025-03-04"}],"family_history":[{"condition":"Diabetes","relative":null}]}`, string(b))
}

func TestTruthy(t *testing.T) {
	for _, v := range []any{true, "1", "true", 1} {
		assert.True(t, truthy(v), v)
	}
	for _, v := range []any{false, "0", "false", 0, nil} {
		assert.False(t, truthy(v), v)
	}
}

func TestFileErrorMessages(t *testing.T) {
	for _, code := range []string{FileNotFound, FileTaken} {
		assert.NotEqual(t, "records.validation."+code, RT("validation."+code, "en"))
		assert.NotEqual(t, "records.validation."+code, RT("validation."+code, "fa"))
	}
}
