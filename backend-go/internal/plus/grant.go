package plus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/plus/store"
)

// SourceChallenge marks a free period earned by finishing a challenge (bloom B-N9-07).
const SourceChallenge = "challenge"

// GrantDays adds a free Plus period of days days to the user, on q (pass a transaction's queries to make the grant
// atomic with the caller's own write): it starts now, or right after the last valid period ends (a grant extends,
// never overlaps, under the user's row lock), with auto-renew off. It returns the subscription id. The caller owns idempotency.
func GrantDays(ctx context.Context, q *store.Queries, userID uint64, days int, source string, now time.Time) (int64, error) {
	if days <= 0 {
		return 0, fmt.Errorf("plus: grant of %d days", days)
	}
	if err := lockUser(ctx, q, userID); err != nil {
		return 0, err
	}
	start := now
	last, err := q.LatestValidSubscription(ctx, store.LatestValidSubscriptionParams{UserID: userID, Now: now})
	switch {
	case err == nil && last.EndsAt.After(now):
		start = last.EndsAt
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return 0, fmt.Errorf("plus: latest subscription: %w", err)
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	id, err := q.InsertGrantedSubscription(ctx, store.InsertGrantedSubscriptionParams{
		UserID: userID, Source: source, StartsAt: start, EndsAt: start.AddDate(0, 0, days), CreatedAt: stamp, UpdatedAt: stamp,
	})
	if err != nil {
		return 0, fmt.Errorf("plus: grant: %w", err)
	}
	return id, nil
}
