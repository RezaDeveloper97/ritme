package order

import (
	"context"
	"log/slog"
	"sync"
)

// Notice is the order-placed SMS («رسید … به پیامکت فرستاده شد»). The text is DISCREET by construction: the order code
// and «pay on delivery» only — never a product name, category or anything that hints at pregnancy or health. The
// recipient is the account's own mobile, resolved by the provider from UserID; no phone number or address travels in
// a Notice or reaches a log line.
type Notice struct {
	UserID  uint64
	OrderID uint64
	Code    string
	Text    string
}

// Notifier is the order SMS adapter. Sending is best effort and happens after the order is committed.
type Notifier interface {
	OrderPlaced(ctx context.Context, n Notice) error
}

// FakeNotifier is the default provider (no generic SMS gateway is wired for shop messages yet): it records the
// notices in memory and logs the order id only.
type FakeNotifier struct {
	logger *slog.Logger
	mu     sync.Mutex
	sent   []Notice
}

// NewFakeNotifier builds the fake provider; logger may be nil.
func NewFakeNotifier(logger *slog.Logger) *FakeNotifier { return &FakeNotifier{logger: logger} }

// OrderPlaced implements Notifier.
func (f *FakeNotifier) OrderPlaced(ctx context.Context, n Notice) error {
	f.mu.Lock()
	f.sent = append(f.sent, n)
	f.mu.Unlock()
	if f.logger != nil {
		f.logger.InfoContext(ctx, "shop: order sms (fake provider)", slog.Uint64("order_id", n.OrderID))
	}
	return nil
}

// Sent returns a copy of the recorded notices.
func (f *FakeNotifier) Sent() []Notice {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Notice(nil), f.sent...)
}
