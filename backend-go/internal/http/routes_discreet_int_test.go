package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockedBuffer is a goroutine-safe log sink.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// inviteSMS returns the discreet attribute of every fake invite SMS logged so far, then clears the log.
func (b *lockedBuffer) inviteSMS(t *testing.T) []bool {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []bool
	sc := bufio.NewScanner(bytes.NewReader(b.buf.Bytes()))
	for sc.Scan() {
		var rec map[string]any
		require.NoError(t, json.Unmarshal(sc.Bytes(), &rec))
		if rec["msg"] != "Companion invite SMS (fake provider, not sent)" {
			continue
		}
		d, ok := rec["discreet"].(bool)
		require.True(t, ok, "the fake logs the discreet flag")
		out = append(out, d)
	}
	b.buf.Reset()
	return out
}

// CB-PRIV-01 (D-73): the app-lock screen reads GET /health-record/emergency-card/lock — no data at all while the
// lock-screen flag is off; with it on only the minimal card (first name, chronic illnesses, ongoing medications,
// emergency contact) — never the full name, gynaecological conditions, onboarding medications or insurance.
func TestEmergencyCardLock_MinimalAndOnlyWhenEnabled(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara Rezaei", "09120007101")
	const p = "/api/v1/health-record/emergency-card"
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, p+"/lock", "", nil).Status)

	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, "/api/v1/onboarding/steps/conditions", sara, map[string]any{
		"chronic_illnesses": []string{"thyroid"}, "gyn_conditions": []string{"pcos"}, "medications": []string{"iud"},
	}).Status)
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, "/api/v1/health-record/basics", sara,
		map[string]any{"blood_type": "O+", "allergies": []string{"penicillin"}}).Status)
	r := e.do(fiber.MethodPut, p, sara, map[string]any{
		"emergency_contact": map[string]any{"name": "Ali", "relation": "همسر", "phone": "09120000000"},
		"insurance":         map[string]any{"label": "تکمیلی", "last4": "4821"},
	})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	owner := r.data()["card"].(map[string]any)
	require.Equal(t, "Sara Rezaei", owner["name"], "the owner view holds what the lock view must not")
	require.NotEmpty(t, owner["profile_medications"])
	require.NotNil(t, owner["insurance"])

	// Flag off (the default): nothing but enabled=false.
	r = e.do(fiber.MethodGet, p+"/lock", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, map[string]any{"enabled": false}, r.data())
	assert.NotContains(t, r.Raw, "O+")
	assert.NotContains(t, r.Raw, "penicillin")

	// Flag on: the minimal card.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, p, sara, map[string]any{"show_on_lock_screen": true}).Status)
	r = e.do(fiber.MethodGet, p+"/lock", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["enabled"])
	card := r.data()["card"].(map[string]any)
	assert.Equal(t, "Sara", card["name"], "first name only")
	assert.Equal(t, "O+", card["blood_type"])
	assert.Equal(t, map[string]any{"chronic_illnesses": []any{"thyroid"}}, card["conditions"], "no gyn_conditions")
	assert.NotContains(t, card, "profile_medications")
	assert.NotContains(t, card, "insurance")
	assert.Equal(t, map[string]any{"name": "Ali", "relation": "همسر", "phone": "09120000000"}, card["emergency_contact"])
	for _, leak := range []string{"Rezaei", "pcos", "iud", "4821"} {
		assert.NotContains(t, r.Raw, leak)
	}

	// Another user's flag never shows her card.
	nina := e.user("Nina", "09120007102")
	assert.Equal(t, map[string]any{"enabled": false}, e.do(fiber.MethodGet, p+"/lock", nina, nil).data())
}

// CB-PRIV-01: the companion invite SMS (the only user-triggered SMS that is not the login OTP) follows the owner's
// «اعلان‌های محرمانه» — the B-N1-11 neutral_copy preference — with no second flag: discreet by default, plain once
// she switches it off on either screen, discreet again when she switches it back on.
func TestDiscreetNotifications_InviteSMSFollowsNeutralCopy(t *testing.T) {
	logs := &lockedBuffer{}
	e := newCompanionEnvWithLogger(t, slog.New(slog.NewJSONHandler(logs, nil)))
	sara := e.user("Sara", "09120007001")

	invite := func(phone string) {
		t.Helper()
		r := e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{
			"type": "partner", "display_name": "Ali", "phone": phone, "grants": map[string]any{"meds": "view"},
		})
		require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
		require.Equal(t, true, r.data()["invite"].(map[string]any)["sms_sent"])
	}
	setNeutral := func(on bool) {
		t.Helper()
		r := e.do(fiber.MethodPut, "/api/v1/profile/notification-settings", sara, map[string]any{"neutral_copy": on})
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		assert.Equal(t, on, r.data()["neutral_copy"])
	}

	invite("09120007002")
	assert.Equal(t, []bool{true}, logs.inviteSMS(t), "no preferences row: neutral copy is the default")

	setNeutral(false)
	invite("09120007003")
	assert.Equal(t, []bool{false}, logs.inviteSMS(t))

	setNeutral(true)
	invite("09120007004")
	assert.Equal(t, []bool{true}, logs.inviteSMS(t))

	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM notification_preferences WHERE user_id = ? AND neutral_copy = 1`, e.ids["Sara"]),
		"one stored flag, no discreet_notifications column")
}
