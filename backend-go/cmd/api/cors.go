package main

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// corsMiddleware reproduces Laravel's HandleCors + fruitcake/php-cors with
// backend/config/cors.php: paths api/*, allowed_methods/headers "*", the configured
// origin list, no exposed headers, max_age 3600, supports_credentials false.
//
// Header-for-header parity matters (contract diff), including the quirks:
//   - with exactly one configured origin it is always sent, and no Vary: Origin;
//   - otherwise the request Origin is echoed only when allowed, and Vary: Origin is
//     added to every api/* response, allowed or not;
//   - a preflight (OPTIONS + Access-Control-Request-Method) is answered 204 here,
//     before routing, and the method/headers are echoed back from the request.
func corsMiddleware(origins []string) fiber.Handler {
	allowAll := slices.Contains(origins, "*")
	single := !allowAll && len(origins) == 1

	allowOrigin := func(c fiber.Ctx) {
		switch {
		case allowAll:
			c.Set(fiber.HeaderAccessControlAllowOrigin, "*")
		case single:
			c.Set(fiber.HeaderAccessControlAllowOrigin, origins[0])
		default:
			if origin := c.Get(fiber.HeaderOrigin); origin != "" && slices.Contains(origins, origin) {
				c.Set(fiber.HeaderAccessControlAllowOrigin, origin)
			}
			addVary(c, fiber.HeaderOrigin)
		}
	}
	hasAllowOrigin := func(c fiber.Ctx) bool {
		return len(c.Response().Header.Peek(fiber.HeaderAccessControlAllowOrigin)) > 0
	}

	return func(c fiber.Ctx) error {
		// The admin API (internal/admin/httpadmin) owns its own CORS/cookie policy.
		if !strings.HasPrefix(c.Path(), "/api/") || strings.HasPrefix(c.Path(), "/api/admin/") {
			return c.Next()
		}

		if c.Method() == fiber.MethodOptions && c.Get(fiber.HeaderAccessControlRequestMethod) != "" {
			c.Status(fiber.StatusNoContent)
			allowOrigin(c)
			if hasAllowOrigin(c) {
				c.Set(fiber.HeaderAccessControlAllowMethods, strings.ToUpper(c.Get(fiber.HeaderAccessControlRequestMethod)))
				addVary(c, fiber.HeaderAccessControlRequestMethod)
				c.Set(fiber.HeaderAccessControlAllowHeaders, c.Get(fiber.HeaderAccessControlRequestHeaders))
				addVary(c, fiber.HeaderAccessControlRequestHeaders)
				c.Set(fiber.HeaderAccessControlMaxAge, "3600")
			}
			addVary(c, fiber.HeaderAccessControlRequestMethod)
			return nil
		}

		err := c.Next()
		if c.Method() == fiber.MethodOptions {
			addVary(c, fiber.HeaderAccessControlRequestMethod)
		}
		allowOrigin(c)
		return err
	}
}

// addVary appends name to the Vary header unless already listed (php-cors varyHeader).
func addVary(c fiber.Ctx, name string) {
	current := string(c.Response().Header.Peek(fiber.HeaderVary))
	if current == "" {
		c.Set(fiber.HeaderVary, name)
		return
	}
	for _, v := range strings.Split(current, ",") {
		if strings.TrimSpace(v) == name {
			return
		}
	}
	c.Set(fiber.HeaderVary, current+", "+name)
}
