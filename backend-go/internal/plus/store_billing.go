package plus

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/plus/store"
)

// Store in-app billing (Café Bazaar, Myket): the Android store builds of the app buy through the store's billing
// library instead of the web gateway. The store charges its own price; the server only checks the store-signed
// purchase (payments.StoreVerifier) and grants what plus_store_products maps the SKU to — a plan period, or the one
// Plus trial. The client never names a plan, a duration or an amount.
//
// Binding: the purchase's developerPayload must be StorePayload(user) — the app passes it to the store before the
// purchase (POST /plus/store/intent), and the store signs it into the purchase. A purchase token belongs to the
// first account that claims it (plus_store_purchases UNIQUE(store, purchase_token)), so a leaked purchase cannot be
// redeemed by another account.
//
// Subscriptions: every distinct (token, purchase_time) is one paid period (receipt ref_id = token:time_ms), so a
// renewal the store reports with a newer purchase_time adds the next period, and a replay of the same signed
// purchase adds nothing. The app sends its active purchases on open (POST /plus/store/sync) to restore and renew.

// Store product kinds.
const (
	StoreKindPlan  = "plan"
	StoreKindTrial = "trial"
)

// Store billing errors.
var (
	ErrStoreSignature   = errors.New("plus: store purchase signature invalid")
	ErrStoreProduct     = errors.New("plus: store product unknown")
	ErrStorePayload     = errors.New("plus: store purchase belongs to another account")
	ErrStorePurchaseBad = errors.New("plus: store purchase rejected")
)

// WithStores sets the configured store verifiers (nil or empty = store billing unavailable).
func (s *Service) WithStores(v payments.StoreVerifiers) *Service {
	s.stores = v
	return s
}

// StoreChannels is the stores whose billing is configured.
func (s *Service) StoreChannels() []string {
	out := []string{}
	for _, name := range []string{payments.StoreBazaar, payments.StoreMyket} {
		if s.stores.Get(name) != nil {
			out = append(out, name)
		}
	}
	return out
}

// StorePayload is the developerPayload that binds a store purchase to userID. It is not a secret: the store signs
// it into the purchase, so nobody can change it afterwards; it only has to differ between accounts.
func StorePayload(userID uint64) string {
	sum := sha256.Sum256([]byte("ritme-plus-store:" + strconv.FormatUint(userID, 10)))
	return "rp1-" + hex.EncodeToString(sum[:12])
}

// StoreProducts maps store → plan id → SKU, for the plans view.
func (s *Service) StoreProducts(ctx context.Context) (map[string]map[uint64]string, string, error) {
	rows, err := s.q.ListStoreProducts(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("plus: store products: %w", err)
	}
	out := map[string]map[uint64]string{}
	trial := ""
	for _, r := range rows {
		if r.Kind == StoreKindTrial {
			if trial == "" {
				trial = r.Sku
			}
			continue
		}
		if !r.PlanID.Valid {
			continue
		}
		if out[r.Store] == nil {
			out[r.Store] = map[uint64]string{}
		}
		out[r.Store][uint64(r.PlanID.Int64)] = r.Sku //nolint:gosec // G115: ids are positive
	}
	return out, trial, nil
}

// StoreIntent is what the app passes to the store before a purchase.
type StoreIntent struct {
	Store   string
	SKU     string
	Kind    string
	Payload string
}

// Intent checks that sku is sellable on store to userID now and returns the payload to sign into the purchase.
func (s *Service) Intent(ctx context.Context, userID uint64, storeName, sku string, now time.Time) (StoreIntent, error) {
	if s.stores.Get(storeName) == nil {
		return StoreIntent{}, ErrPaymentUnavailable
	}
	p, err := s.q.GetStoreProduct(ctx, store.GetStoreProductParams{Store: storeName, Sku: sku})
	if errors.Is(err, sql.ErrNoRows) {
		return StoreIntent{}, ErrStoreProduct
	}
	if err != nil {
		return StoreIntent{}, fmt.Errorf("plus: store product: %w", err)
	}
	switch p.Kind {
	case StoreKindTrial:
		if err := s.trialAllowed(ctx, s.q, userID); err != nil {
			return StoreIntent{}, err
		}
	case StoreKindPlan:
		if !p.PlanID.Valid {
			return StoreIntent{}, ErrStoreProduct
		}
		if _, err := s.q.GetActivePlan(ctx, uint64(p.PlanID.Int64)); err != nil { //nolint:gosec // G115
			return StoreIntent{}, ErrStoreProduct
		}
	default:
		return StoreIntent{}, ErrStoreProduct
	}
	return StoreIntent{Store: storeName, SKU: sku, Kind: p.Kind, Payload: StorePayload(userID)}, nil
}

// trialAllowed is StartTrial's eligibility rule.
func (s *Service) trialAllowed(ctx context.Context, q *store.Queries, userID uint64) error {
	if _, err := q.GetTrial(ctx, userID); err == nil {
		return ErrTrialUsed
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("plus: trial: %w", err)
	}
	had, err := q.HasAnySubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("plus: subscriptions: %w", err)
	}
	if had {
		return ErrTrialUnavailable
	}
	return nil
}

// StoreSettled is the outcome of one store purchase.
type StoreSettled struct {
	Kind        string
	SKU         string
	Invoice     *store.PlusInvoice // the paid invoice of a plan purchase (nil for the trial)
	AlreadyDone bool               // this exact purchase was redeemed before (replay / re-sync)
}

// StoreVerify checks one signed store purchase and grants it to userID. Idempotent per (token, purchase_time).
func (s *Service) StoreVerify(ctx context.Context, userID uint64, storeName, signedData, signature string, now time.Time) (StoreSettled, error) {
	v := s.stores.Get(storeName)
	if v == nil {
		return StoreSettled{}, ErrPaymentUnavailable
	}
	p, err := v.Verify(signedData, signature)
	switch {
	case errors.Is(err, payments.ErrStoreSignature):
		s.logger.WarnContext(ctx, "plus: store purchase signature invalid", "store", storeName, "user_id", userID)
		return StoreSettled{}, ErrStoreSignature
	case err != nil:
		s.logger.WarnContext(ctx, "plus: store purchase rejected", "store", storeName, "user_id", userID, "error", err.Error())
		return StoreSettled{}, ErrStorePurchaseBad
	case p.DeveloperPayload != StorePayload(userID):
		s.logger.WarnContext(ctx, "plus: store purchase payload of another account", "store", storeName, "user_id", userID, "sku", p.ProductID)
		return StoreSettled{}, ErrStorePayload
	}
	ptime := p.PurchaseTime.In(now.Location()).Truncate(time.Second)
	refID := p.PurchaseToken + ":" + strconv.FormatInt(p.PurchaseTime.UnixMilli(), 10)
	if len(refID) > 191 {
		sum := sha256.Sum256([]byte(refID))
		refID = "h:" + hex.EncodeToString(sum[:])
	}

	out := StoreSettled{SKU: p.ProductID}
	err = s.inTx(ctx, func(q *store.Queries) error {
		prod, err := q.GetStoreProduct(ctx, store.GetStoreProductParams{Store: storeName, Sku: p.ProductID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrStoreProduct
		}
		if err != nil {
			return fmt.Errorf("plus: store product: %w", err)
		}
		out.Kind = prod.Kind
		stamp := sql.NullTime{Time: now, Valid: true}
		orderID := sql.NullString{String: p.OrderID, Valid: p.OrderID != ""}

		// Claim the token (first account wins), serialising concurrent verifies of it.
		row, err := q.LockStorePurchase(ctx, store.LockStorePurchaseParams{Store: storeName, PurchaseToken: p.PurchaseToken})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("plus: store purchase: %w", err)
		}
		// Then the user row, before any insert (see settle): the purchase row → user lock order of every settlement.
		if lerr := lockUser(ctx, q, userID); lerr != nil {
			return lerr
		}
		switch {
		case errors.Is(err, sql.ErrNoRows):
			err = q.InsertStorePurchase(ctx, store.InsertStorePurchaseParams{
				UserID: userID, Store: storeName, Sku: p.ProductID, PurchaseToken: p.PurchaseToken, OrderID: orderID,
				PurchaseTime: ptime, CreatedAt: stamp, UpdatedAt: stamp,
			})
			if isDuplicate(err) {
				return ErrStorePayload // a concurrent claim won; retrying re-reads it
			}
			if err != nil {
				return fmt.Errorf("plus: store purchase: %w", err)
			}
		case err != nil:
			return fmt.Errorf("plus: store purchase: %w", err)
		case row.UserID != userID:
			return ErrStorePayload
		case row.Sku != p.ProductID:
			return ErrStorePurchaseBad
		case ptime.After(row.PurchaseTime):
			if err := q.TouchStorePurchase(ctx, store.TouchStorePurchaseParams{PurchaseTime: ptime, OrderID: orderID, Now: stamp, ID: row.ID}); err != nil {
				return fmt.Errorf("plus: store purchase: %w", err)
			}
		}

		if prod.Kind == StoreKindTrial {
			if _, err := q.GetTrial(ctx, userID); err == nil {
				out.AlreadyDone = true // the trial was granted already (this token or the web)
				return nil
			}
			if err := s.trialAllowed(ctx, q, userID); err != nil {
				return err
			}
			if err := q.InsertTrial(ctx, store.InsertTrialParams{
				UserID: userID, StartedAt: now, EndsAt: now.AddDate(0, 0, s.cfg.TrialDays), CreatedAt: stamp, UpdatedAt: stamp,
			}); err != nil {
				if isDuplicate(err) {
					out.AlreadyDone = true
					return nil
				}
				return fmt.Errorf("plus: start trial: %w", err)
			}
			return nil
		}

		if prod.Kind != StoreKindPlan || !prod.PlanID.Valid {
			return ErrStoreProduct
		}
		found, err := q.ReceiptExists(ctx, store.ReceiptExistsParams{Gateway: storeName, RefID: refID})
		if err != nil {
			return fmt.Errorf("plus: receipt: %w", err)
		}
		if found {
			out.AlreadyDone = true
			return nil
		}
		plan, err := q.GetPlan(ctx, uint64(prod.PlanID.Int64)) //nolint:gosec // G115: ids are positive
		if err != nil {
			return fmt.Errorf("plus: plan: %w", err)
		}
		ref, err := newReference()
		if err != nil {
			return err
		}
		invoiceID, err := q.InsertInvoice(ctx, store.InsertInvoiceParams{
			Reference: ref, UserID: userID, PlanID: sql.NullInt64{Int64: int64(plan.ID), Valid: true}, //nolint:gosec // G115
			DurationMonths: plan.DurationMonths, DurationMinutes: plan.DurationMinutes,
			SubtotalRials: plan.PriceRials, TotalRials: plan.PriceRials, // the store charged its own price; VAT is the store's
			Gateway:   sql.NullString{String: storeName, Valid: true},
			ExpiresAt: now, CreatedAt: stamp, UpdatedAt: stamp,
		})
		if err != nil {
			return fmt.Errorf("plus: invoice: %w", err)
		}
		id := uint64(invoiceID) //nolint:gosec // G115: ids are positive
		err = q.InsertReceipt(ctx, store.InsertReceiptParams{
			InvoiceID: id, UserID: userID, Gateway: storeName, RefID: refID,
			AmountRials: plan.PriceRials, PaidAt: now, CreatedAt: stamp, UpdatedAt: stamp,
		})
		if isDuplicate(err) {
			return errReplayRace
		}
		if err != nil {
			return fmt.Errorf("plus: receipt: %w", err)
		}
		if err := activate(ctx, q, userID, plan.ID, id, periodOf(plan.DurationMonths, plan.DurationMinutes), now); err != nil {
			return err
		}
		if err := q.MarkInvoicePaid(ctx, store.MarkInvoicePaidParams{Now: stamp, ID: id}); err != nil {
			return fmt.Errorf("plus: invoice: %w", err)
		}
		inv, err := q.GetUserInvoiceByReference(ctx, store.GetUserInvoiceByReferenceParams{Reference: ref, UserID: userID})
		if err != nil {
			return fmt.Errorf("plus: invoice: %w", err)
		}
		out.Invoice = &inv
		return nil
	})
	if errors.Is(err, errReplayRace) {
		return StoreSettled{Kind: StoreKindPlan, SKU: p.ProductID, AlreadyDone: true}, nil
	}
	if err != nil {
		return StoreSettled{}, err
	}
	s.logger.InfoContext(ctx, "plus: store purchase", "store", storeName, "user_id", userID, "sku", p.ProductID,
		"kind", out.Kind, "already", out.AlreadyDone)
	if out.Invoice != nil {
		events.Goal(ctx, userID, "plus_purchased")
	}
	return out, nil
}

// errReplayRace is a concurrent verify of the same purchase that lost the receipt insert (rolled back, reported as
// already redeemed).
var errReplayRace = errors.New("plus: store purchase redeemed concurrently")

func (s *Service) storeConfigured(name string) bool { return s.stores.Get(name) != nil }
