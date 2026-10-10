package httpadmin

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/store"
)

// HealthAccessTTLMinutes is how long an approved health-data access request stays valid (B-N9-02).
const HealthAccessTTLMinutes = 30

// CodeHealthAccess is the 403 of a health-data read without a live approved request.
const CodeHealthAccess = "health_access_required"

// HealthGrantStore finds the admin's live grant (store.Queries satisfies it).
type HealthGrantStore interface {
	ActiveHealthAccess(ctx context.Context, arg store.ActiveHealthAccessParams) (store.ActiveHealthAccessRow, error)
}

// HealthGrant is the current admin's live access to userID's health data: their role holds the health capability
// AND a request for this user was approved and has not expired. nil = no access (no error).
func HealthGrant(c fiber.Ctx, q HealthGrantStore, userID uint64) (*store.ActiveHealthAccessRow, error) {
	a := CurrentAdmin(c)
	if a == nil || !Can(c, CapHealth, LevelManage) {
		return nil, nil
	}
	row, err := q.ActiveHealthAccess(c.Context(), store.ActiveHealthAccessParams{
		AdminID: a.ID, UserID: userID, Now: DBTime(Now(c)),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// HealthAccessRequired is the 403 for a health-data read without a grant.
func HealthAccessRequired() error {
	return Fail(fiber.StatusForbidden, CodeHealthAccess,
		"Health data needs an approved, unexpired access request for this user.")
}
