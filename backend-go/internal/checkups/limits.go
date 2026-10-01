package checkups

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// MaxCustomCheckups caps a user's own checkup types (security audit M3-M7 #5): the plan, the
// home card and every record write compute over all of them. Far above real use; the per-user
// write throttle (internal/http) bounds the rate.
const MaxCustomCheckups = 50

// ErrorCodeLimitReached is the 422's error_code when a cap is hit.
const ErrorCodeLimitReached = "limit_reached"

// checkCustomCap refuses a new custom checkup once the user has MaxCustomCheckups. q runs
// inside the insert's transaction.
func checkCustomCap(c fiber.Ctx, q *store.Queries, userID uint64, locale string) error {
	// Only her own rows are counted (audiences NULL), so no life mode is needed for the filter.
	types, err := q.ListActiveCheckupTypesForUser(c, store.ListActiveCheckupTypesForUserParams{
		UserID: int64(userID), //nolint:gosec // ids fit int64
	})
	if err != nil {
		return fmt.Errorf("checkups: count custom: %w", err)
	}
	n := 0
	for _, t := range types {
		if t.UserID.Valid {
			n++
		}
	}
	if n < MaxCustomCheckups {
		return nil
	}
	msg := T("messages.custom_limit", locale)
	return httpx.Fail(fiber.StatusUnprocessableEntity, msg,
		"errors", jsonx.Obj("limit", []string{msg}), "error_code", ErrorCodeLimitReached)
}
