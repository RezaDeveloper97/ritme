package companion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Companion accounts (bloom B-N4-03). Men sign up only as companions: a male account (user_life_profiles.gender =
// male) has no cycle engine of his own and resolves to enums.LifeModeCompanion. The women's home and message
// endpoints answer him 409 companion_account (AccountConflict) instead of a cycle built from defaults;
// GET /companion/home is his home.

// Error codes of the account-kind refusals.
const (
	// ErrorCodeCompanionAccount: a women's cycle endpoint (home, daily messages) called by a companion account.
	ErrorCodeCompanionAccount = "companion_account"
	// ErrorCodeNotCompanionAccount: the companion panel home called by a woman's (or never-asked) account.
	ErrorCodeNotCompanionAccount = "not_companion_account"
)

// PathCompanionHome is the web route of the companion panel home (QUESTIONS #5: nav امروز · خدمات · من).
const PathCompanionHome = "/companion"

// Accounts answers whether an account is a companion (male) account.
type Accounts struct{ q *store.Queries }

// NewAccounts reads through conn (a *sql.DB or a transaction).
func NewAccounts(conn store.DBTX) *Accounts { return &Accounts{q: store.New(conn)} }

// IsCompanion reports whether userID's gender is male. No life-profile row or no gender = a woman's account (every
// user before onboarding v2).
func (a *Accounts) IsCompanion(ctx context.Context, userID uint64) (bool, error) {
	g, err := a.q.GetUserGender(ctx, userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("companion: gender: %w", err)
	}
	return g.Valid && g.String == string(enums.GenderMale), nil
}

// AccountConflict is the 409 a companion account gets from a women's cycle endpoint: the client routes him
// to his own home (`home`).
func AccountConflict(locale string) *httpx.FailError {
	return httpx.Fail(fiber.StatusConflict, T("messages.companion_account", locale),
		"error_code", ErrorCodeCompanionAccount, "mode", string(enums.LifeModeCompanion), "home", PathCompanionHome)
}

// NonCompanionForbidden is the 403 a woman's account gets from the companion panel home (she reads what others
// share with her through GET /companions/links and its sections).
func NonCompanionForbidden(locale string) *httpx.FailError {
	return httpx.Fail(fiber.StatusForbidden, T("messages.not_companion_account", locale),
		"error_code", ErrorCodeNotCompanionAccount)
}

// ViewerLinkJSON is a link as its companion sees it (the GET /companions/links item), for the panel home.
func ViewerLinkJSON(l Link, names map[uint64]string) *jsonx.OrderedMap {
	return companionLinkJSON(l, names)
}
