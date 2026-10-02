package fertility_test

// B-N3-14b (N3 stage smoke B-3): LH, BBT and mucus logged in the v2 log sheet (health_log_entries) reach
// /fertility/today, /fertility/days and the LH evidence of /fertility/insights, merged per day with the
// /fertility log like the TTC analysis (the stronger LH result, the more fertile mucus).

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// logSheet saves a day the way PUT /logs/days/{date} does.
func (e *env) logSheet(t *testing.T, userID uint64, date, body string) {
	t.Helper()
	changes, err := taxonomy.Parse(validation.DecodeBody([]byte(body)), "ttc", "fa", nil, nil)
	require.NoError(t, err)
	_, err = healthlog.NewService(e.db).SaveDay(context.Background(), userID, civildate.MustParse(date), changes, "fa",
		time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran))
	require.NoError(t, err)
}

func TestLogSheetValuesReachFertility(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000801", "2026-09-10")

	e.logSheet(t, id, today, `{"categories":{"measurements":{"lh_test":"faint","bbt":36.6},"discharge":{"consistency":"egg_white"}}}`)

	e.queries.n.Store(0)
	r := e.do(t, http.MethodGet, "/api/v1/fertility/today", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, int64(3), e.queries.n.Load(), "query budget unchanged: profile, histories, merged day")
	assert.Equal(t, map[string]any{"value": "faint", "label": "کم\u200cرنگ"}, r.data()["lh"])
	assert.Equal(t, map[string]any{"value": "36.60"}, r.data()["bbt"])

	r = e.do(t, http.MethodGet, dayPath(today), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "faint", r.data()["lh"])
	assert.Equal(t, "egg_white", r.data()["mucus"])

	// The /fertility log's weaker value does not hide the log sheet's; a stronger one wins.
	r = e.do(t, http.MethodPut, dayPath(today), tok, "fa", `{"lh":"negative","mucus":"sticky"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "faint", r.data()["lh"])
	assert.Equal(t, "egg_white", r.data()["mucus"])
	r = e.do(t, http.MethodPut, dayPath(today), tok, "fa", `{"lh":"positive"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "positive", r.data()["lh"])

	// watery has no /fertility mucus value; none reads as dry.
	e.logSheet(t, id, "2026-09-22", `{"categories":{"discharge":{"consistency":"watery"}}}`)
	assert.Nil(t, e.do(t, http.MethodGet, dayPath("2026-09-22"), tok, "fa", "").data()["mucus"])
	e.logSheet(t, id, "2026-09-21", `{"categories":{"discharge":{"consistency":"none"}}}`)
	assert.Equal(t, "dry", e.do(t, http.MethodGet, dayPath("2026-09-21"), tok, "fa", "").data()["mucus"])
}

func TestLogSheetLHCountsForInsights(t *testing.T) {
	e := setup(t)
	id, tok := bbtUser(t, e, "09120000802")
	e.logSheet(t, id, "2026-09-22", `{"categories":{"measurements":{"lh_test":"positive"}}}`)
	e.lh(t, id, "2026-09-22", "faint") // the same day in the /fertility log: one test, the stronger result

	r := e.do(t, http.MethodGet, insightsPath, tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var lh map[string]any
	for _, row := range r.data()["evidence"].([]any) {
		if m := row.(map[string]any); m["key"] == "lh" {
			lh = m
		}
	}
	require.NotNil(t, lh, r.raw)
	assert.Equal(t, "strong", lh["strength"], "a positive test from the log sheet")
}
