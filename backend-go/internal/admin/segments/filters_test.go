package segments_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/admin/segments"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var now = time.Date(2026, 10, 31, 12, 0, 0, 0, civildate.Tehran)

func at(s string) sql.NullTime {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
	return sql.NullTime{Time: t, Valid: true}
}

func TestFilters_InvalidAndOver(t *testing.T) {
	assert.Equal(t, "", segments.Filters{Plan: "plus", Churn: "new", Mode: "pregnancy", DataQuality: "poor",
		MainWarning: "low_history", Cohort: "2026-10", Activity: "7d", Status: "blocked"}.Invalid())
	assert.Equal(t, "plan", segments.Filters{Plan: "gold"}.Invalid())
	assert.Equal(t, "cohort", segments.Filters{Cohort: "2026-13"}.Invalid())
	assert.Equal(t, "mode", segments.Filters{Mode: "x"}.Invalid())
	assert.Equal(t, "main_warning", segments.Filters{MainWarning: "headache"}.Invalid())
	f := segments.Filters{Plan: "plus", Churn: "high"}.Over(segments.Filters{Churn: "low", Mode: "ttc"})
	assert.Equal(t, segments.Filters{Plan: "plus", Churn: "low", Mode: "ttc"}, f)
}

func TestParse_RejectsQAndUnknown(t *testing.T) {
	f, ok := segments.Parse(json.RawMessage(`{"plan":"plus","churn":"high"}`))
	assert.True(t, ok)
	assert.Equal(t, "plus", f.Plan)
	_, ok = segments.Parse(json.RawMessage(`{"q":"sara"}`))
	assert.False(t, ok)
	_, ok = segments.Parse(json.RawMessage(`{"colour":"red"}`))
	assert.False(t, ok)
	_, ok = segments.Parse(json.RawMessage(`{"plan":"gold"}`))
	assert.False(t, ok)
}

func TestMatch_SameRulesAsDirectory(t *testing.T) {
	s := segments.Subject{CreatedAt: at("2026-10-02 09:00"), Projected: true, LastActivity: at("2026-10-28 00:00"),
		Churn: "high", Mode: "ttc", Plan: "trial", DataQuality: "partial", MainWarning: "low_history"}
	assert.True(t, segments.Filters{}.Match(s, now))
	assert.True(t, segments.Filters{Plan: "trial", Churn: "high", Mode: "ttc", Cohort: "2026-10", Activity: "7d",
		Status: "active", DataQuality: "partial", MainWarning: "low_history"}.Match(s, now))
	assert.False(t, segments.Filters{Plan: "plus"}.Match(s, now))
	assert.False(t, segments.Filters{Cohort: "2026-09"}.Match(s, now))
	assert.False(t, segments.Filters{Activity: "today"}.Match(s, now))
	assert.False(t, segments.Filters{Activity: "inactive"}.Match(s, now))
	assert.False(t, segments.Filters{Status: "blocked"}.Match(s, now))
	assert.False(t, segments.Filters{Q: "x"}.Match(s, now))
	// No projection row yet: projection filters never match, account filters still do.
	np := segments.Subject{CreatedAt: at("2026-10-30 09:00")}
	assert.False(t, segments.Filters{Plan: "free"}.Match(np, now))
	assert.True(t, segments.Filters{Cohort: "2026-10"}.Match(np, now))
}

func TestRefCodeQuery(t *testing.T) {
	c, ok := segments.RefCodeQuery(" rt-abc234 ")
	assert.True(t, ok)
	assert.Equal(t, "RT-ABC234", c)
	_, ok = segments.RefCodeQuery("RT-000A")
	assert.False(t, ok)
	_, ok = segments.RefCodeQuery("0912")
	assert.False(t, ok)
}
