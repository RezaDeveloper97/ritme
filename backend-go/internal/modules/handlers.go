package modules

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers serve GET /api/v1/modules (public) and the admin GET / PUT /api/admin/v1/modules.
type Handlers struct {
	svc    *Service
	logger *slog.Logger
}

// NewHandlers wires the handlers.
func NewHandlers(svc *Service, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{svc: svc, logger: logger}
}

// Public is GET /api/v1/modules: {companion: bool, ttc: bool, …}. No auth — the onboarding reads it before the
// profile exists, and it holds nothing user-specific.
func (h *Handlers) Public(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "public, max-age=60")
	return httpx.OK(c, h.svc.States(c.Context()).JSON())
}

// AdminShow is GET /api/admin/v1/modules: the same map plus the key order for the admin list.
func (h *Handlers) AdminShow(c fiber.Ctx) error {
	return httpadmin.OK(c, fiber.Map{"modules": h.svc.States(c.Context()).JSON(), "keys": Keys})
}

// AdminUpdate is PUT /api/admin/v1/modules {companion?: bool, …}: a partial update; every change is audited.
func (h *Handlers) AdminUpdate(c fiber.Ctx) error {
	rules := make(validation.Rules, 0, len(Keys))
	for _, k := range Keys {
		rules = append(rules, validation.F(k, "sometimes", "boolean"))
	}
	data, err := httpadmin.Validate(c, rules)
	if err != nil {
		return err
	}
	before := h.svc.States(c.Context())
	changes := map[string]bool{}
	for _, k := range Keys {
		if _, sent := data.Get(k); sent {
			if on := httpadmin.Bool(data, k); on != before.Enabled(k) {
				changes[k] = on
			}
		}
	}
	if len(changes) > 0 {
		if err := h.svc.Set(c.Context(), changes, httpadmin.Now(c).In(civildate.Tehran).Truncate(1e9)); err != nil {
			return err
		}
		for k, on := range changes {
			httpadmin.Audit(c, h.logger, "modules.update", "module", 0, slog.String("module", k), slog.Bool("enabled", on))
		}
	}
	return httpadmin.OK(c, fiber.Map{"modules": h.svc.States(c.Context()).JSON(), "keys": Keys})
}
