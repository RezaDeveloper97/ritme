package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ritme/backend-go/internal/shop/store"
)

// CB-SHOP-12 F-5: the user's own product review of a delivered order item. Verified purchase by construction (the
// row carries order_item_id, unique), one per order item, `pending` until the admin moderates it
// (/api/admin/v1/shop/reviews); only `published` reviews reach the catalog and its rating.

// Review limits.
const (
	MinRating     = 1
	MaxRating     = 5
	MaxReviewBody = 2000
)

// ReviewInput is a validated review.
type ReviewInput struct {
	Rating int
	Body   string
}

// ReviewStatus is the moderation state of her review of one order item ("" = not reviewed yet).
type ReviewStatus map[uint64]string

// authorName is what the catalog shows next to a review: the initial of her profile name ("S."), never the full name
// or phone; without a name the localized fallback.
func authorName(name sql.NullString, fallback string) string {
	n := strings.TrimSpace(name.String)
	if !name.Valid || n == "" {
		return fallback
	}
	r, _ := utf8.DecodeRuneInString(n)
	if r == utf8.RuneError || !unicode.IsLetter(r) {
		return fallback
	}
	return string(unicode.ToUpper(r)) + "."
}

// CreateReview files her review of one item of one of her orders: 404 when the order or the item is not hers, 422
// not_reviewable before the item's shipment is delivered (or when the product is gone), 409 already_reviewed for a
// second review of the same order item.
func (s *Service) CreateReview(ctx context.Context, userID, orderID, itemID uint64, in ReviewInput, fallbackAuthor string, now time.Time) (store.ShopProductReview, error) {
	var out store.ShopProductReview
	err := s.inTx(ctx, func(q *store.Queries) error {
		if _, err := q.LockOrderForUser(ctx, store.LockOrderForUserParams{ID: orderID, UserID: userID}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fail(404, "not_found", "")
			}
			return fmt.Errorf("order: lock: %w", err)
		}
		it, err := q.GetOrderItemForReview(ctx, store.GetOrderItemForReviewParams{ItemID: itemID, OrderID: orderID, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return fail(404, "not_found", "")
		}
		if err != nil {
			return fmt.Errorf("order: review item: %w", err)
		}
		if it.ShipmentStatus != StatusDelivered || !it.ProductID.Valid {
			return fail(422, "not_reviewable", "")
		}
		itemRef := sql.NullInt64{Int64: int64(it.ID), Valid: true} //nolint:gosec // G115
		if n, err := q.CountReviewsByOrderItem(ctx, itemRef); err != nil {
			return fmt.Errorf("order: reviews: %w", err)
		} else if n > 0 {
			return fail(409, "already_reviewed", "")
		}
		name, err := q.GetUserDisplayName(ctx, userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("order: user: %w", err)
		}
		last, err := q.CreateUserProductReview(ctx, store.CreateUserProductReviewParams{ProductID: id64(it.ProductID),
			UserID: userID, OrderItemID: itemRef, AuthorName: authorName(name, fallbackAuthor), VariantLabel: it.VariantLabel,
			Rating: uint8(in.Rating), Body: ns(strings.TrimSpace(in.Body)), CreatedAt: nt(now), UpdatedAt: nt(now)}) //nolint:gosec // G115: 1–5
		if err != nil {
			if isDuplicate(err) { // a twin request won the unique order_item_id
				return fail(409, "already_reviewed", "")
			}
			return fmt.Errorf("order: create review: %w", err)
		}
		out, err = q.GetProductReviewForUser(ctx, store.GetProductReviewForUserParams{ID: uint64(last), UserID: userID}) //nolint:gosec // G115
		if err != nil {
			return fmt.Errorf("order: review: %w", err)
		}
		return nil
	})
	return out, err
}

// reviewStatuses maps her order's item ids to the status of her review of each.
func (s *Service) reviewStatuses(ctx context.Context, userID, orderID uint64) (ReviewStatus, error) {
	rows, err := s.q.ListReviewsByOrderForUser(ctx, store.ListReviewsByOrderForUserParams{OrderID: orderID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("order: reviews: %w", err)
	}
	out := ReviewStatus{}
	for _, r := range rows {
		out[id64(r.OrderItemID)] = r.Status
	}
	return out, nil
}
