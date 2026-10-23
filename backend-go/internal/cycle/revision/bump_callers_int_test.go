package revision_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

// T-M9-30: writers outside the engine domains that run an engine-input helper on their own transaction bump on it —
// the condition programs' PBAC chart and pain diary (healthlog.NewService(tx)) and the pregnancy loss
// (pregnancy.EndForLoss on the loss transaction).
func TestBump_TransactionOwnersOutsideEngineDomains(t *testing.T) {
	e := newEnv(t)
	uid, tok := e.user()
	e.onboarded(tok, uid)

	e.bumps(0, uid, tok, 200, fiber.MethodPost, "/api/v1/conditions/enrolments", map[string]any{"program": "heavy_bleeding"})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/conditions/pbac/2026-09-22", map[string]any{"light": 1})
	e.bumps(0, uid, tok, 200, fiber.MethodPost, "/api/v1/conditions/enrolments", map[string]any{"program": "endo"})
	e.bumps(1, uid, tok, 200, fiber.MethodPut, "/api/v1/conditions/pain/2026-09-22", map[string]any{"score": 3, "locations": []string{"pelvis"}})

	e.bumps(1, uid, tok, 201, fiber.MethodPost, "/api/v1/pregnancy/onboarding", map[string]any{"age_source": "lmp", "lmp_date": "2026-08-02"})
	// Two transactions: the loss (pregnancy mode off) and the life-mode switch back to cycle each bump once.
	e.bumps(2, uid, tok, 201, fiber.MethodPost, "/api/v1/loss", map[string]any{"type": "early_miscarriage"})
}
