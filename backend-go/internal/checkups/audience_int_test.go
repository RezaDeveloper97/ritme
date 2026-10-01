package checkups_test

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/store"
)

// menoKeys are the nine audience-scoped rows of 00022_menopause, activated by 00024 (CB-MENO-01b).
var menoKeys = []string{
	"meno_blood_pressure", "meno_blood_sugar", "meno_bone_density", "meno_colon_screening", "meno_eye_exam",
	"meno_lipids", "meno_thyroid", "meno_vitamin_d_calcium", "meno_weight_waist",
}

// setMode stores the user's life mode (bloom user_life_profiles, B-N2-01).
func (e *env) setMode(t *testing.T, userID uint64, mode string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE life_mode = VALUES(life_mode)`, userID, mode)
	require.NoError(t, err)
}

func (e *env) typeID(t *testing.T, key string) uint64 {
	t.Helper()
	var id uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM checkup_types WHERE `key` = ? AND user_id IS NULL", key).Scan(&id))
	return id
}

// itemsByKey lists GET /checkups and indexes the items by key (custom ones, which have no key, by id).
func (e *env) itemsByKey(t *testing.T, token string) map[string]map[string]any {
	t.Helper()
	r := e.do(t, http.MethodGet, "/api/v1/checkups", token, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	out := map[string]map[string]any{}
	for _, x := range r.data()["items"].([]any) {
		m := x.(map[string]any)
		key, _ := m["key"].(string)
		if key == "" {
			key = fmt.Sprintf("custom:%v", m["id"])
		}
		out[key] = m
	}
	return out
}

func menoOf(items map[string]map[string]any) []string {
	var out []string
	for k := range items {
		if strings.HasPrefix(k, "meno_") {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// A cycle-mode (or ttc / no life profile) user never sees the menopause rows; a menopause-mode user sees them next to
// the shared catalog, whose items are the same for both.
func TestAudience_ListByLifeMode(t *testing.T) {
	e := setup(t)
	cycleID, cycleTok := e.fixture(t, "09120000101")
	menoID, menoTok := e.fixture(t, "09120000102")
	ttcID, ttcTok := e.fixture(t, "09120000103")
	_, legacyTok := e.fixture(t, "09120000104") // no user_life_profiles row
	e.setMode(t, cycleID, "cycle")
	e.setMode(t, menoID, "menopause")
	e.setMode(t, ttcID, "ttc")

	cycle := e.itemsByKey(t, cycleTok)
	assert.Empty(t, menoOf(cycle))
	assert.Len(t, cycle, catalogLength)
	assert.Empty(t, menoOf(e.itemsByKey(t, ttcTok)))
	assert.Len(t, e.itemsByKey(t, legacyTok), catalogLength)

	meno := e.itemsByKey(t, menoTok)
	assert.Equal(t, menoKeys, menoOf(meno))
	assert.Len(t, meno, catalogLength+len(menoKeys))
	for key, it := range cycle {
		mi, ok := meno[key]
		require.True(t, ok, key)
		if key == "breast_self_exam" || key == "pap_smear" { // B-N2-11b: no cycle-day timing in menopause
			assert.NotNil(t, it["timing_label"], key)
			it, mi = maps.Clone(it), maps.Clone(mi)
			delete(it, "timing_label")
			delete(mi, "timing_label")
		}
		assert.Equal(t, it, mi, "shared row %s unchanged", key)
	}
	assert.Equal(t, "A set day of the month", meno["breast_self_exam"]["timing_label"])
	assert.Nil(t, meno["pap_smear"]["timing_label"])
	selfExam := e.typeID(t, "breast_self_exam")
	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", selfExam), menoTok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "یک روز ثابت از ماه", r.data()["timing_label"])
	assert.Nil(t, r.data()["cycle_day_from"], "no cycle-day hint without a cycle")
	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", selfExam), cycleTok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "روز ۷ تا ۱۰ سیکل", r.data()["timing_label"])
	assert.EqualValues(t, 7, r.data()["cycle_day_from"])

	// The home card keeps its three-query budget and counts the menopause rows for her only.
	r = e.home(t, menoTok, "fa")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	menoTotal := r.data()["summary"].(map[string]any)["total"].(float64)
	r = e.home(t, cycleTok, "fa")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 6, r.data()["summary"].(map[string]any)["total"])
	assert.Greater(t, menoTotal, 6.0)

	// An active pregnancy wins over a stored menopause mode.
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, created_at, updated_at) VALUES (?, 1, NOW(), NOW())`, menoID)
	require.NoError(t, err)
	assert.Empty(t, menoOf(e.itemsByKey(t, menoTok)))
}

// The detail, record and settings endpoints answer 404 for a type outside the user's audience.
func TestAudience_OtherModeTypeIsNotFound(t *testing.T) {
	e := setup(t)
	cycleID, cycleTok := e.user(t, "09120000105", "1970-03-01")
	menoID, menoTok := e.user(t, "09120000106", "1970-03-01")
	e.setMode(t, cycleID, "cycle")
	e.setMode(t, menoID, "menopause")
	bp := e.typeID(t, "meno_blood_pressure")

	r := e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", bp), cycleTok, "fa", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", bp), cycleTok, "fa", `{"done_on":"2026-09-01","result":"normal"}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", bp), cycleTok, "fa", `{"enabled":false}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)

	r = e.do(t, http.MethodGet, fmt.Sprintf("/api/v1/checkups/%d", bp), menoTok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/checkups/%d/records", bp), menoTok, "fa", `{"done_on":"2026-09-01","result":"normal"}`)
	assert.Contains(t, []int{http.StatusOK, http.StatusCreated}, r.status, r.raw)
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/checkups/%d/settings", bp), menoTok, "fa", `{"enabled":false}`)
	assert.Equal(t, http.StatusOK, r.status, r.raw)
}

// EngineInputs and the store queries apply the same audience filter.
func TestAudience_EngineInputs(t *testing.T) {
	e := setup(t)
	cycleID, _ := e.user(t, "09120000107", "")
	menoID, _ := e.user(t, "09120000108", "")
	e.setMode(t, menoID, "menopause")
	ctx := context.Background()
	q := store.New(e.db)

	mode, err := checkups.UserLifeMode(ctx, q, menoID)
	require.NoError(t, err)
	assert.EqualValues(t, "menopause", mode)
	mode, err = checkups.UserLifeMode(ctx, q, cycleID)
	require.NoError(t, err)
	assert.EqualValues(t, "cycle", mode)

	in, err := checkups.EngineInputs(ctx, q, cycleID)
	require.NoError(t, err)
	assert.Len(t, in.Types, catalogLength)
	in, err = checkups.EngineInputs(ctx, q, menoID)
	require.NoError(t, err)
	assert.Len(t, in.Types, catalogLength+len(menoKeys))
}
