package companion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
)

// Teen suspension (B-N4-08b, CMP-M1, QUESTIONS #98). CB-TEEN-01 refuses new partner / spouse invites and accepts for
// a teen-mode owner (checkTeenRule). A partner or spouse link that was already active when the owner switched to teen
// mode is neither revoked nor changed: while her stored life mode is teen it grants nothing — every section reads as
// none for the companion (section reads 403, the companion home shows no cards, «ثبت برای …» is refused) — and the
// stored grants come back unchanged when she leaves teen mode. The owner still sees and manages the link as it is.
// Parent links are not affected.

// ownerIsTeen reports whether ownerID's stored life mode is teen (the same source as checkTeenRule).
func ownerIsTeen(ctx context.Context, q *store.Queries, ownerID uint64) (bool, error) {
	mode, err := q.GetUserLifeMode(ctx, ownerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("companion: life mode: %w", err)
	}
	return mode.Valid && mode.String == string(enums.LifeModeTeen), nil
}

// teenSuspended reports whether a link of type t of ownerID grants nothing right now: a partner / spouse link of an
// owner in teen mode.
func teenSuspended(ctx context.Context, q *store.Queries, ownerID uint64, t Type) (bool, error) {
	if t == TypeParent {
		return false, nil
	}
	return ownerIsTeen(ctx, q, ownerID)
}

// suspendForTeens clears the effective grants of the links that teenSuspended (the companion's view of them). The
// owner's mode is looked up once per owner.
func suspendForTeens(ctx context.Context, q *store.Queries, links []Link) error {
	teen := map[uint64]bool{}
	for i := range links {
		l := &links[i]
		if l.Type == TypeParent {
			continue
		}
		is, seen := teen[l.OwnerID]
		if !seen {
			var err error
			if is, err = ownerIsTeen(ctx, q, l.OwnerID); err != nil {
				return err
			}
			teen[l.OwnerID] = is
		}
		if is {
			l.Grants = Grants{}
		}
	}
	return nil
}
