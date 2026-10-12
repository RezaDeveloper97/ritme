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

// Sharing answers the record-sharing question (user decision 2026-10-12, CB-REC-06 L5): whether userID's stored
// life mode is teen and, if so, whether she has an active parent link. A teen may make a doctor code or a report link
// only with one (internal/sharelinks enforces it); her emergency card is always available.
func (p *Policy) Sharing(ctx context.Context, userID uint64) (isTeen, parentLinked bool, err error) {
	m, err := p.q.GetTeenLifeMode(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("teen: life mode: %w", err)
	}
	if enums.LifeMode(m.String) != enums.LifeModeTeen {
		return false, false, nil
	}
	linked, err := p.q.HasActiveParentLink(ctx, userID)
	if err != nil {
		return true, false, fmt.Errorf("teen: parent link: %w", err)
	}
	return true, linked, nil
}
