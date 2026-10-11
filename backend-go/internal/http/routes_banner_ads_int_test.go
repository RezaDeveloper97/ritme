package http

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// B-N10-02 (D-93): sponsored slots on the banner system, through the production registry.

func (e *companionEnv) ad(position, modes string, active int) uint64 {
	e.t.Helper()
	var m any
	if modes != "" {
		m = modes
	}
	res, err := e.db.Exec(`INSERT INTO banners (title, image_path, position, link_url, link_type, is_active, sort_order, kind,
		advertiser, cta_label, modes, created_at, updated_at)
		VALUES ('{"fa":"آگهی","en":"Ad"}', 'banners/ad.webp', ?, 'https://example.com', 'external', ?, 1, 'ad', 'Acme Co',
		'{"fa":"بیشتر","en":"More"}', ?, NOW(), NOW())`, position, active, m)
	require.NoError(e.t, err)
	id, _ := res.LastInsertId()
	return uint64(id)
}

func TestBannerAds_TargetingSuppressionAndCounters(t *testing.T) {
	e := newCompanionEnv(t)
	tok := e.user("Mina", "09120004001")
	teenTok := e.user("Nila", "09120004002")
	e.teenMode("Nila")

	all := e.ad("ad_slider", "", 1)
	preg := e.ad("ad_square", `["pregnancy"]`, 1)
	e.ad("ad_strip", "", 0) // inactive

	// 401 without a token.
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, "/api/v1/banners/ads", "", nil).Status)
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/banners/%d/events", all), "", map[string]any{"event": "click"}).Status)

	// Mode targeting: ads for every mode always; the pregnancy ad only for pregnancy.
	r := e.do(fiber.MethodGet, "/api/v1/banners/ads?mode=cycle", tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	pos := r.data()["positions"].(map[string]any)
	assert.Len(t, pos["ad_slider"], 1)
	assert.Len(t, pos["ad_square"], 0)
	assert.Len(t, pos["ad_strip"], 0)
	assert.NotContains(t, r.Raw, "Acme", "advertiser is admin-only")
	r = e.do(fiber.MethodGet, "/api/v1/banners/ads?mode=pregnancy", tok, nil)
	assert.Len(t, r.data()["positions"].(map[string]any)["ad_square"], 1)

	// Teen: query mode or stored account mode gives every slot empty.
	for _, q := range []struct{ tok, path string }{{tok, "?mode=teen"}, {teenTok, "?mode=cycle"}} {
		r = e.do(fiber.MethodGet, "/api/v1/banners/ads"+q.path, q.tok, nil)
		for _, v := range r.data()["positions"].(map[string]any) {
			assert.Len(t, v, 0)
		}
	}

	// The legacy endpoint never carries ads.
	r = e.do(fiber.MethodGet, "/api/v1/banners", tok, nil)
	require.Equal(t, fiber.StatusOK, r.Status)
	assert.NotContains(t, r.Raw, "ad_slider")
	assert.NotContains(t, r.Raw, "banners/ad.webp")

	// Counters: per-day aggregate.
	url := fmt.Sprintf("/api/v1/banners/%d/events", all)
	for _, ev := range []string{"impression", "impression", "click"} {
		r = e.do(fiber.MethodPost, url, tok, map[string]any{"event": ev})
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	}
	var imp, clk int
	require.NoError(t, e.db.QueryRow(`SELECT impressions, clicks FROM banner_daily_stats WHERE banner_id = ?`, all).Scan(&imp, &clk))
	assert.Equal(t, 2, imp)
	assert.Equal(t, 1, clk)
	var rows int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM banner_daily_stats`).Scan(&rows))
	assert.Equal(t, 1, rows)

	// No user id (or any identity) column exists on the counters.
	var cols string
	require.NoError(t, e.db.QueryRow(`SELECT GROUP_CONCAT(column_name ORDER BY ordinal_position) FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'banner_daily_stats'`).Scan(&cols))
	assert.Equal(t, "banner_id,day,impressions,clicks", cols)

	// Validation, unknown / inactive / plain banners.
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, url, tok, map[string]any{"event": "view"}).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/banners/999999/events", tok, map[string]any{"event": "click"}).Status)
	_ = preg
}
