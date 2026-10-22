package checkups_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M10-21 (D-172): the user API on screening.uspstf — test kinds on the cervical record, breast awareness
// read-only, the after-pregnancy label, the BMI condition.

func TestScreening_PapTestKind(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000301", "1992-06-15") // 34

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", papSmear), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, []any{"cytology", "hpv", "cotest"}, d["test_kinds"])
	assert.Equal(t, true, d["test_kind_required"], "from 30 the kind changes the interval")
	assert.Equal(t, true, d["registry"])
	assert.Contains(t, d["interval_source"], "USPSTF cervical 2018")
	assert.Equal(t, false, d["read_only"])

	// Required from 30, one of the registry kinds.
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", papSmear), tok, "fa", `{"done_on":"2026-09-01","result":"normal"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "test_kind")
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", papSmear), tok, "fa",
		`{"done_on":"2026-09-01","result":"normal","test_kind":"pcr"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "test_kind")

	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", papSmear), tok, "en",
		`{"done_on":"2026-09-01","result":"normal","test_kind":"hpv"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	rec := r.data()["record"].(map[string]any)
	assert.Equal(t, "hpv", rec["test_kind"])
	item := r.data()["item"].(map[string]any)
	assert.Equal(t, "2031-09-01", item["next_due_on"], "HPV: every 60 months")
	assert.Equal(t, "Every 5 years", item["interval_label"])
	recID := int(rec["id"].(float64))

	// An edit that does not send test_kind keeps it (the UPDATE writes the column, T-M10-17).
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", `{"note":"خوب بود"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "hpv", r.data()["record"].(map[string]any)["test_kind"])
	var stored string
	require.NoError(t, e.db.QueryRow(`SELECT test_kind FROM checkup_records WHERE id = ?`, recID).Scan(&stored))
	assert.Equal(t, "hpv", stored)

	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "en", `{"test_kind":"cytology"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2029-09-01", r.data()["item"].(map[string]any)["next_due_on"], "cytology: every 36 months")

	// Preview by test kind.
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/preview-next?type=%d&done_on=2026-09-20&test_kind=cotest", papSmear), tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2031-09-20", r.data()["next_due_on"])
	assert.Equal(t, "Every 5 years", r.data()["interval_label"])

	// A type without test kinds ignores the field.
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", dentist), tok, "fa",
		`{"done_on":"2026-09-01","result":"normal","test_kind":"hpv"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Nil(t, r.data()["record"].(map[string]any)["test_kind"])

	// Under 30 the kind is optional.
	_, young := e.user(t, "09120000302", "2001-01-01") // 25
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", papSmear), young, "fa", `{"done_on":"2026-09-01","result":"normal"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "2029-09-01", r.data()["item"].(map[string]any)["next_due_on"])
}

func TestScreening_BreastAwarenessReadOnly(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000303", "1992-06-15")
	e.record(t, id, selfExam, "2026-05-01") // logged before v6

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", selfExam), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "informational", d["status"])
	assert.Equal(t, true, d["read_only"])
	assert.Nil(t, d["next_due_on"])
	assert.Nil(t, d["timing_label"])
	assert.Nil(t, d["cycle_day_from"])
	assert.Equal(t, map[string]any{"enabled": true, "remind": false}, d["settings"], "no reminder")
	assert.Len(t, d["records"], 1, "the history stays readable")
	recID := int(d["records"].([]any)[0].(map[string]any)["id"].(float64))

	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", selfExam), tok, "fa", `{"done_on":"2026-09-01","result":"normal"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "این مورد برای آگاهی است و ثبت تازه ندارد.", r.body["message"])
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "en", `{"note":"x"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/checkups/records/%d", recID), tok, "fa", "")
	assert.Equal(t, http.StatusOK, r.status, "her data stays hers to delete")

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/preview-next?type=%d&done_on=2026-09-01", selfExam), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["next_due_on"])
}

func TestScreening_AfterPregnancyAndBMI(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000304", "1980-01-10")                                                  // 46
	_, err := e.db.Exec(`UPDATE user_profiles SET weight = 70.00, height = 170 WHERE user_id = ?`, id) // BMI 24.2
	require.NoError(t, err)

	items := itemsByID(t, e.do(t, http.MethodGet, "/api/v1/checkups", tok, "fa", "").data()["items"])
	assert.NotContains(t, items, glucoseScreening, "BMI under 25: no glucose screening")
	assert.Equal(t, "هر سال", items[bpScreening]["interval_label"], "40+: every 12 months")

	_, err = e.db.Exec(`UPDATE user_profiles SET weight = 72.25 WHERE user_id = ?`, id) // BMI 25.0
	require.NoError(t, err)
	items = itemsByID(t, e.do(t, http.MethodGet, "/api/v1/checkups", tok, "fa", "").data()["items"])
	assert.Contains(t, items, glucoseScreening, "BMI 25.0 lands in the higher band")

	_, err = e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, id)
	require.NoError(t, err)
	r := e.do(t, http.MethodGet, "/api/v1/checkups", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	items = itemsByID(t, r.data()["items"])
	mammo := items[mammography]
	require.NotNil(t, mammo, "not hidden any more")
	assert.Equal(t, "after_pregnancy", mammo["label"])
	assert.Equal(t, "After pregnancy", mammo["next_due_label"])
	assert.Equal(t, "age_based", mammo["section"])
	assert.Nil(t, items[papSmear]["label"])

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", mammography), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["settings"].(map[string]any)["remind"], "no reminders while pregnant")
	r = e.do(t, http.MethodGet, "/api/v1/checkups?filter=action", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.NotContains(t, itemsByID(t, r.data()["items"]), mammography)
}
