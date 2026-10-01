package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// Fake payment states.
const (
	FakePending  = "pending"
	FakePaid     = "paid"
	FakeCanceled = "canceled"
)

// FakePayment is one payment held by the fake gateway.
type FakePayment struct {
	Authority     string    `json:"authority"`
	Reference     string    `json:"reference"`
	AmountRials   uint64    `json:"amount_rials"`
	Description   string    `json:"description"`
	CallbackURL   string    `json:"callback_url"` // the API's return URL (built by Gateway, never user input)
	Status        string    `json:"status"`
	RefID         string    `json:"ref_id,omitempty"`
	RefundedRials uint64    `json:"refunded_rials"`
	Refunds       int       `json:"refunds"`
	CreatedAt     time.Time `json:"created_at"`
}

// FakeStore keeps fake payments. Both the plus routes and the fake page build their own Gateway, so the state must
// live outside the process: Redis at runtime, memory in unit tests.
type FakeStore interface {
	Load(ctx context.Context, authority string) (FakePayment, bool, error)
	Save(ctx context.Context, p FakePayment) error
}

// fakeTTL bounds how long a fake payment is remembered.
const fakeTTL = 7 * 24 * time.Hour

// RedisFakeStore keeps fake payments under ritme-go:payments:fake:{authority} for a week.
type RedisFakeStore struct{ c *cache.Client }

// NewRedisFakeStore wraps the shared cache client.
func NewRedisFakeStore(c *cache.Client) *RedisFakeStore { return &RedisFakeStore{c: c} }

func fakeKey(authority string) string { return "payments:fake:" + authority }

// Load implements FakeStore.
func (s *RedisFakeStore) Load(ctx context.Context, authority string) (FakePayment, bool, error) {
	raw, err := s.c.Get(ctx, fakeKey(authority))
	if errors.Is(err, cache.ErrMiss) {
		return FakePayment{}, false, nil
	}
	if err != nil {
		return FakePayment{}, false, fmt.Errorf("payments: fake store: %w", err)
	}
	var p FakePayment
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return FakePayment{}, false, fmt.Errorf("payments: fake store: %w", err)
	}
	return p, true, nil
}

// Save implements FakeStore.
func (s *RedisFakeStore) Save(ctx context.Context, p FakePayment) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("payments: fake store: %w", err)
	}
	if err := s.c.Set(ctx, fakeKey(p.Authority), raw, fakeTTL); err != nil {
		return fmt.Errorf("payments: fake store: %w", err)
	}
	return nil
}

// MemoryFakeStore is an in-process FakeStore (tests).
type MemoryFakeStore struct {
	mu sync.Mutex
	m  map[string]FakePayment
}

// NewMemoryFakeStore returns an empty store.
func NewMemoryFakeStore() *MemoryFakeStore { return &MemoryFakeStore{m: map[string]FakePayment{}} }

// Load implements FakeStore.
func (s *MemoryFakeStore) Load(_ context.Context, authority string) (FakePayment, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.m[authority]
	return p, ok, nil
}

// Save implements FakeStore.
func (s *MemoryFakeStore) Save(_ context.Context, p FakePayment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[p.Authority] = p
	return nil
}

// Fake is the TEST provider. It never moves money: the user picks success or failure on a page served by this API.
// Authorities are "FAKE-" + the order reference (the reference is already 80 random bits), bank references
// "FAKE-REF-" + the reference.
type Fake struct {
	store   FakeStore
	pageURL string // {api origin}/api/v1/payments/fake/pay
	mu      sync.Mutex
	now     func() time.Time
}

// NewFake builds the fake provider; apiBase is the API's public origin (the pay page lives there).
func NewFake(store FakeStore, apiBase string) *Fake {
	return &Fake{store: store, pageURL: strings.TrimRight(apiBase, "/") + "/api/v1/payments/fake/pay", now: time.Now}
}

// Name implements Provider.
func (*Fake) Name() string { return FakeName }

// Create implements Provider: remembers the payment and points the user at the TEST page.
func (f *Fake) Create(ctx context.Context, req Request) (Session, error) {
	authority := "FAKE-" + req.Reference
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok, err := f.store.Load(ctx, authority); err != nil {
		return Session{}, err
	} else if ok {
		return Session{}, fmt.Errorf("%w: reference already used", ErrInvalidRequest)
	}
	p := FakePayment{
		Authority: authority, Reference: req.Reference, AmountRials: req.AmountRials, Description: req.Description,
		CallbackURL: req.CallbackURL, Status: FakePending, CreatedAt: f.now(),
	}
	if err := f.store.Save(ctx, p); err != nil {
		return Session{}, err
	}
	return Session{Authority: authority, RedirectURL: f.pageURL + "/" + url.PathEscape(authority)}, nil
}

// Verify implements Provider: the verdict is the fake's own state, never req.Callback. It reports the amount the
// payment was created with, so a caller verifying a different amount sees the mismatch. Idempotent.
func (f *Fake) Verify(ctx context.Context, req VerifyRequest) (Result, error) {
	p, ok, err := f.store.Load(ctx, req.Authority)
	if err != nil {
		return Result{}, err
	}
	if !ok || p.Status != FakePaid {
		return Result{}, nil
	}
	return Result{Paid: true, RefID: p.RefID, AmountRials: p.AmountRials}, nil
}

// Refund implements Provider: a paid payment can be refunded up to its amount, in parts.
func (f *Fake) Refund(ctx context.Context, req RefundRequest) (RefundResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok, err := f.store.Load(ctx, req.Authority)
	switch {
	case err != nil:
		return RefundResult{}, err
	case !ok || p.Status != FakePaid:
		return RefundResult{}, fmt.Errorf("%w: payment not paid", ErrRefundRejected)
	case req.RefID != "" && req.RefID != p.RefID:
		return RefundResult{}, fmt.Errorf("%w: reference mismatch", ErrRefundRejected)
	case req.AmountRials > p.AmountRials-p.RefundedRials:
		return RefundResult{}, fmt.Errorf("%w: more than the refundable amount", ErrRefundRejected)
	}
	p.RefundedRials += req.AmountRials
	p.Refunds++
	if err := f.store.Save(ctx, p); err != nil {
		return RefundResult{}, err
	}
	return RefundResult{RefundID: fmt.Sprintf("FAKE-RFD-%s-%d", p.Reference, p.Refunds), AmountRials: req.AmountRials}, nil
}

// ReturnParams implements Provider.
func (*Fake) ReturnParams(q map[string]string) (string, string) { return q["authority"], q["status"] }

// Payment returns a fake payment for the TEST page.
func (f *Fake) Payment(ctx context.Context, authority string) (FakePayment, bool, error) {
	return f.store.Load(ctx, authority)
}

// Decide records the tester's choice on the TEST page and returns where to send the browser (the API's return
// URL with authority and status). Only a pending payment changes; a repeated decision returns the recorded one.
func (f *Fake) Decide(ctx context.Context, authority string, success bool) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok, err := f.store.Load(ctx, authority)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%w: unknown payment", ErrInvalidRequest)
	}
	if p.Status == FakePending {
		p.Status = FakeCanceled
		if success {
			p.Status, p.RefID = FakePaid, "FAKE-REF-"+p.Reference
		}
		if err := f.store.Save(ctx, p); err != nil {
			return "", err
		}
	}
	status := "NOK"
	if p.Status == FakePaid {
		status = "OK"
	}
	u, err := url.Parse(p.CallbackURL)
	if err != nil {
		return "", fmt.Errorf("payments: fake callback: %w", err)
	}
	q := u.Query()
	q.Set("authority", p.Authority)
	q.Set("status", status)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
