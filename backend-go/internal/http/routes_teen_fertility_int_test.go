package http

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CB-TEEN-04b (CB-TEEN-04 QA #15): on her ovulation day a teen-mode account gets no fertility, ovulation,
// pregnancy-chance or "attractiveness" copy from /messages/daily or /cycle/today; a cycle account with the
// same period data still does (so the day really is the ovulation day).
func TestTeenRoutes_NoFertilityCopyInDailyMessageAndCycleToday(t *testing.T) {
	e := newCompanionEnv(t)
	ava := e.user("Ava", "09120003101")
	lily := e.user("Lily", "09120003102")
	e.teenMode("Ava")
	for _, name := range []string{"Ava", "Lily"} {
		// companionNow is 2026-09-23: cycle day 15 of a 28-day cycle → the ovulation phase.
		_, err := e.db.Exec(`INSERT INTO user_profiles (user_id, last_period_start, cycle_duration, period_duration, created_at, updated_at)
			VALUES (?, '2026-09-09', 28, 5, NOW(), NOW())`, e.ids[name])
		require.NoError(t, err)
	}

	words := []string{"attractive", "ovulat", "fertil", "pregnan", "conceive", "peak"}
	mentions := func(text string) []string {
		var hits []string
		for _, w := range words {
			if strings.Contains(strings.ToLower(text), w) {
				hits = append(hits, w)
			}
		}
		return hits
	}
	asText := func(v ...any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}

	// /messages/daily: the copy is the primary message, the supplements and the phase / sub-phase labels.
	daily := func(token string) (map[string]any, string) {
		r := e.do(fiber.MethodGet, "/api/v1/messages/daily", token, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		d := r.data()
		info, _ := d["context_info"].(map[string]any)
		return info, asText(d["primary_message"], d["supplements"], d["correlations"], info["phase_label"], info["subphase_label"])
	}
	info, text := daily(lily)
	assert.Equal(t, "ovulation", info["phase"], "the cycle account is on her ovulation day")
	assert.Contains(t, text, "attractiveness")
	info, text = daily(ava)
	assert.Empty(t, mentions(text), text)
	assert.Equal(t, "luteal", info["phase"], "the ovulation day reads as luteal")
	assert.Equal(t, false, info["is_fertile_window"])
	assert.Nil(t, info["estimated_ovulation_day"])
	assert.Nil(t, info["subphase"])

	// /cycle/today: the copy is the text flags, the daily tips and the daily card.
	today := func(token string) string {
		r := e.do(fiber.MethodGet, "/api/v1/cycle/today", token, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		calc, _ := r.data()["calculation"].(map[string]any)
		view, _ := r.data()["cycle_view"].(map[string]any)
		card, _ := view["daily_card"].(map[string]any)
		return asText(calc["text_flags"], calc["daily_tips"], card["title"], card["subtitle"], card["fertility_label"], card["badges"])
	}
	assert.NotEmpty(t, mentions(today(lily)), "the cycle account reads fertility copy")
	text = today(ava)
	assert.Empty(t, mentions(text), text)
	assert.Contains(t, text, "Current phase: Luteal")
}
