package fertility_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const insightsPath = "/api/v1/fertility/insights"

// lh logs an LH test result.
func (e *env) lh(t *testing.T, userID uint64, date, value string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO fertility_logs (user_id, log_date, lh_test, created_at, updated_at)
		VALUES (?, ?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, userID, date, value)
	require.NoError(t, err)
}

func TestInsights_TwoCyclesOverHTTP(t *testing.T) {
	e := setup(t)
	id, tok := bbtUser(t, e, "09120000701")
	e.lh(t, id, "2026-09-22", "positive")

	e.queries.n.Store(0)
	r := e.do(t, http.MethodGet, insightsPath, tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 4, e.queries.n.Load(), "profile, cycle history, readings, LH tests")
	assert.Regexp(t, `^\{"success":true,"data":\{"cycles_used":2,"window":\{"start":"2026-09-19","end":"2026-09-24","ovulation":"2026-09-24"\},"confidence":"high","evidence":\[`, r.raw)

	d := r.data()
	evidence := d["evidence"].([]any)
	require.Len(t, evidence, 3)
	assert.Equal(t, "۲ سیکل کامل ثبت شده", evidence[0].(map[string]any)["title"], "audit #18: the count is the title")
	var keys, strengths []any
	for _, row := range evidence {
		m := row.(map[string]any)
		keys, strengths = append(keys, m["key"]), append(strengths, m["strength"])
		assert.NotEmpty(t, m["title"])
		assert.NotEmpty(t, m["detail"])
	}
	assert.Equal(t, []any{"cycles", "bbt_shift", "lh"}, keys)
	assert.Equal(t, []any{"medium", "strong", "strong"}, strengths)

	history := d["history"].([]any)
	require.Len(t, history, 2)
	assert.Equal(t, map[string]any{
		"month_label": "شهریور", "ovulation_day": float64(15), "date": "2026-08-27", "source": "bbt", "cycle_start": "2026-08-13",
		"period_days": float64(5),
	}, history[0])
	assert.Len(t, d["tips"], 1)

	en := e.do(t, http.MethodGet, insightsPath, tok, "en", "")
	require.Equal(t, http.StatusOK, en.status, en.raw)
	assert.Equal(t, "August", en.data()["history"].([]any)[0].(map[string]any)["month_label"])
}

func TestInsights_NewUser(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000702", "")
	r := e.do(t, http.MethodGet, insightsPath, tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 0, d["cycles_used"])
	assert.Nil(t, d["window"])
	assert.Equal(t, "low", d["confidence"])
	assert.Equal(t, []any{}, d["history"])
	assert.Len(t, d["tips"], 2)
}

func TestInsights_OtherUsersDataIsInvisible(t *testing.T) {
	e := setup(t)
	idA, _ := bbtUser(t, e, "09120000703")
	e.lh(t, idA, "2026-09-22", "positive")
	_, tokB := e.user(t, "09120000704", "2026-09-10")

	r := e.do(t, http.MethodGet, insightsPath, tokB, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 0, d["cycles_used"])
	assert.Equal(t, "low", d["confidence"])
	assert.Equal(t, []any{}, d["history"])
	for _, row := range d["evidence"].([]any) {
		assert.Equal(t, "none", row.(map[string]any)["strength"])
	}
}

func TestInsights_Unauthenticated(t *testing.T) {
	e := setup(t)
	r := e.do(t, http.MethodGet, insightsPath, "", "fa", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw)
}
