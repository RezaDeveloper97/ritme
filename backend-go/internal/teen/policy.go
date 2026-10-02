package teen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/teen/store"
)

// Policy answers the commercial question for other domains (GET /banners, the home's Plus trial banner, and the
// shop / recommendations when they arrive): a user whose stored life-stage mode is teen gets none of them.
type Policy struct{ q *store.Queries }

// NewPolicy reads through db (a *sql.DB).
func NewPolicy(db store.DBTX) *Policy { return &Policy{q: store.New(db)} }

// AllowsCommercial reports whether userID may see commercial content (false for a teen-mode account).
func (p *Policy) AllowsCommercial(ctx context.Context, userID uint64) (bool, error) {
	m, err := p.q.GetTeenLifeMode(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("teen: life mode: %w", err)
	}
	return AllowsCommercial(enums.LifeMode(m.String)), nil
}
