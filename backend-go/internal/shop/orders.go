package shop

import (
	"context"
	"errors"
)

// ErrUnsupported is what the default Orders adapter answers: partner orders do not exist in N10 (DECISIONS #15;
// the internal COD shop of CB-SHOP-08 owns /profile/orders).
var ErrUnsupported = errors.New("shop: partner orders are not supported")

// PartnerOrder is a partner shipment as a future provider would report it. Never carries health data.
type PartnerOrder struct {
	Ref    string
	Status string
}

// Orders (the «ShopOrders» adapter) is the seam for partner order tracking. Only the `fake` provider exists and it answers
// ErrUnsupported; the Me «سفارش‌ها» partner row therefore stays hidden.
type Orders interface {
	ListOrders(ctx context.Context, userID uint64) ([]PartnerOrder, error)
}

// FakeOrders is the default Orders: unsupported.
type FakeOrders struct{}

// ListOrders implements Orders.
func (FakeOrders) ListOrders(context.Context, uint64) ([]PartnerOrder, error) {
	return nil, ErrUnsupported
}
