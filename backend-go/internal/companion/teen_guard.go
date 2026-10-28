package companion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
)

// Teen suspension (B-N4-08b, CMP-M1, QUESTIONS #98). Since T-M11-05 family.Decide enforces it: a teen-mode owner's
// effective mode is outside every adult domain's mode scope, so every partner / spouse section denies
// (mode_inactive) while the teen card of a teen_parent link stays readable. CB-TEEN-01 refuses new partner / spouse invites and accepts for
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
