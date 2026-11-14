// Package credits is the AI credit ledger (goose 00125): what every AI feature spends.
//
// Rules (owner's decisions, 2026-11-12):
//
//   - AI needs a running subscription or trial. Without one nothing can be spent — not even extra credits
//     (ErrNoSubscription); they stay untouched for the next subscription.
//   - A subscription period (or the trial) carries its own credits. They are usable only while that period runs
//     and are gone with it; the next period starts from its plan's amount again.
//   - Extra credits (bought packs, admin gifts) never expire and are never zeroed.
//   - A use takes the period's credits first, then the extra ones (ErrInsufficient when both together are short).
//   - One use of a feature costs a fixed number of credits (ai_features.credit_cost, admin data).
//
// Spending is reserve-first: internal/ai/access calls Reserve before the handler runs and Refund when the handler
// failed, so parallel requests can never overspend and a provider failure costs the user nothing. Every write of
// one user's buckets runs under a row lock on all of them (LockUserGrants) in a READ COMMITTED transaction.
//
// Period buckets are created lazily (Ensure): the first balance read or reservation inside a period that has no
// bucket yet creates it from the plan's ai_credits (the trial's from the `trial_ai_credits` setting). Every way a
// period can start — gateway purchase, store billing, challenge reward, admin extend — is covered without a hook.
// A period without a plan (admin / challenge gift) carries no credits of its own.
package credits

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/ai/credits/store"
)

// Domain errors (internal/ai/access maps them to 402 / 403 responses).
var (
	// ErrNoSubscription: no subscription period or trial covers now.
	ErrNoSubscription = errors.New("credits: no active subscription")
	// ErrInsufficient: the period's and the extra credits together do not cover the feature's cost.
	ErrInsufficient = errors.New("credits: not enough credits")
	// ErrFeatureDisabled: the feature is switched off, unknown, or has no usable model.
	ErrFeatureDisabled = errors.New("credits: AI feature disabled")
)

// Grant kinds.
const (
	KindPlan  = "plan"
	KindTrial = "trial"
	KindExtra = "extra"
	KindAdmin = "admin"
)

// Period kinds of a Balance.
const (
	PeriodPlus  = "plus"
	PeriodTrial = "trial"
)

// Settings (plus_settings).
const (
	// SettingTrialCredits is how many credits the trial carries (0 = the trial has no AI of its own).
	SettingTrialCredits = "trial_ai_credits"
	DefaultTrialCredits = 20
	MaxTrialCredits     = 100000
)

// Balance is what a user has at one moment.
type Balance struct {
	// Active: a subscription period or the trial covers now — the precondition of every AI use.
	Active bool
	// Period is PeriodPlus / PeriodTrial while Active, "" otherwise.
	Period string
	// PeriodEndsAt is when the running period (and its credits) ends; zero when not Active.
	PeriodEndsAt time.Time
	// PlanTotal / PlanRemaining are the running period's credits (0 when not Active or the period has none).
	PlanTotal     uint32
	PlanRemaining uint32
	// Extra is the never-expiring balance. It is reported even when not Active (kept, but locked).
	Extra uint32
}

// Spendable is what a use can draw on right now.
func (b Balance) Spendable() uint32 {
	if !b.Active {
		return 0
	}
	return b.PlanRemaining + b.Extra
}

// Service is the ledger.
type Service struct {
	db *sql.DB
	q  *store.Queries
}

// NewService wires the ledger on db.
func NewService(db *sql.DB) *Service { return &Service{db: db, q: store.New(db)} }

func stampOf(now time.Time) sql.NullTime { return sql.NullTime{Time: now, Valid: true} }

// TrialCredits is the admin setting (code default when the row is missing or unreadable).
func (s *Service) TrialCredits(ctx context.Context) (uint32, error) {
	return trialCredits(ctx, s.q)
}

func trialCredits(ctx context.Context, q *store.Queries) (uint32, error) {
	v, err := q.GetSetting(ctx, SettingTrialCredits)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultTrialCredits, nil
	}
	if err != nil {
		return 0, fmt.Errorf("credits: setting: %w", err)
	}
	// An unreadable row falls back to the code default: a broken setting must never block the trial.
	if n, convErr := strconv.Atoi(strings.TrimSpace(v)); convErr == nil && n >= 0 && n <= MaxTrialCredits {
		return uint32(n), nil //nolint:gosec // G115: bounded above
	}
	return DefaultTrialCredits, nil
}

// period is the running subscription period or trial.
type period struct {
	kind   string
	endsAt time.Time
}

func currentPeriod(ctx context.Context, q *store.Queries, userID uint64, now time.Time) (*period, error) {
	sub, err := q.CurrentSubscription(ctx, store.CurrentSubscriptionParams{UserID: userID, Now: now})
	switch {
	case err == nil:
		return &period{kind: PeriodPlus, endsAt: sub.EndsAt}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("credits: subscription: %w", err)
	}
	tr, err := q.RunningTrial(ctx, store.RunningTrialParams{UserID: userID, Now: now})
	switch {
	case err == nil:
		return &period{kind: PeriodTrial, endsAt: tr.EndsAt}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("credits: trial: %w", err)
	}
	return nil, nil
}

// ensure creates the missing bucket of the running period (idempotent: UNIQUE on the source row).
func ensure(ctx context.Context, q *store.Queries, userID uint64, p *period, now time.Time) error {
	if p == nil {
		return nil
	}
	if p.kind == PeriodPlus {
		if _, err := q.EnsurePlanGrants(ctx, store.EnsurePlanGrantsParams{Stamp: stampOf(now), UserID: userID, Now: now}); err != nil {
			return fmt.Errorf("credits: plan grant: %w", err)
		}
		return nil
	}
	n, err := trialCredits(ctx, q)
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	if _, err := q.EnsureTrialGrant(ctx, store.EnsureTrialGrantParams{Amount: n, Stamp: stampOf(now), UserID: userID, Now: now}); err != nil {
		return fmt.Errorf("credits: trial grant: %w", err)
	}
	return nil
}

func isPeriodKind(kind string) bool { return kind == KindPlan || kind == KindTrial }

func balanceOf(p *period, grants []store.ValidGrantsRow) Balance {
	var b Balance
	if p != nil {
		b.Active, b.Period, b.PeriodEndsAt = true, p.kind, p.endsAt
	}
	for _, g := range grants {
		if isPeriodKind(g.Kind) {
			b.PlanTotal += g.Amount
			b.PlanRemaining += g.Remaining
			continue
		}
		b.Extra += g.Remaining
	}
	return b
}

// Balance is the user's credits at now. It creates the running period's bucket when it is missing.
func (s *Service) Balance(ctx context.Context, userID uint64, now time.Time) (Balance, error) {
	p, err := currentPeriod(ctx, s.q, userID, now)
	if err != nil {
		return Balance{}, err
	}
	if err := ensure(ctx, s.q, userID, p, now); err != nil {
		return Balance{}, err
	}
	grants, err := s.q.ValidGrants(ctx, store.ValidGrantsParams{UserID: userID, Now: now})
	if err != nil {
		return Balance{}, fmt.Errorf("credits: grants: %w", err)
	}
	return balanceOf(p, grants), nil
}

// allocation is how many credits one reservation took from one bucket.
type allocation struct {
	GrantID uint64 `json:"grant_id"`
	Credits uint32 `json:"credits"`
}

// Reservation is a counted use. Refund gives it back.
type Reservation struct {
	ID      uint64
	UserID  uint64
	Feature string
	Credits uint32
	// At is when the use was counted (the spend row's timestamp): what RefundAt finds it by.
	At time.Time
	// Balance is what the user has after the reservation.
	Balance Balance
}

// Reserve spends cost credits of the user for feature at now: the running period's credits first, then the extra
// ones. ErrNoSubscription without a running period (the Balance still reports the kept extra credits),
// ErrInsufficient when the credits do not cover cost (nothing is taken). A zero cost reserves nothing but still
// needs the running period.
func (s *Service) Reserve(ctx context.Context, userID uint64, feature string, cost uint32, now time.Time) (*Reservation, error) {
	res := &Reservation{UserID: userID, Feature: feature, Credits: cost, At: now}
	err := s.inTx(ctx, func(q *store.Queries) error {
		p, err := currentPeriod(ctx, q, userID, now)
		if err != nil {
			return err
		}
		if err := ensure(ctx, q, userID, p, now); err != nil {
			return err
		}
		if _, err := q.LockUserGrants(ctx, userID); err != nil {
			return fmt.Errorf("credits: lock: %w", err)
		}
		grants, err := q.ValidGrants(ctx, store.ValidGrantsParams{UserID: userID, Now: now})
		if err != nil {
			return fmt.Errorf("credits: grants: %w", err)
		}
		res.Balance = balanceOf(p, grants)
		if p == nil {
			return ErrNoSubscription
		}
		if res.Balance.Spendable() < cost {
			return ErrInsufficient
		}
		if cost == 0 {
			return nil
		}
		allocs, planPart, extraPart, err := take(ctx, q, userID, grants, cost, now)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(allocs)
		if err != nil {
			return fmt.Errorf("credits: allocations: %w", err)
		}
		id, err := q.InsertSpend(ctx, store.InsertSpendParams{
			UserID: userID, Feature: feature, Credits: cost, PlanCredits: planPart, ExtraCredits: extraPart,
			Allocations: raw, CreatedAt: stampOf(now), UpdatedAt: stampOf(now),
		})
		if err != nil {
			return fmt.Errorf("credits: spend: %w", err)
		}
		res.ID = uint64(id) //nolint:gosec // G115: ids are positive
		res.Balance.PlanRemaining -= planPart
		res.Balance.Extra -= extraPart
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrNoSubscription) || errors.Is(err, ErrInsufficient) {
			return res, err // the balance explains the refusal
		}
		return nil, err
	}
	return res, nil
}

// take draws cost from grants in order (they are locked and come in spending order).
func take(ctx context.Context, q *store.Queries, userID uint64, grants []store.ValidGrantsRow, cost uint32, now time.Time) (allocs []allocation, planPart, extraPart uint32, err error) {
	need := cost
	for _, g := range grants {
		if need == 0 {
			break
		}
		n := min(g.Remaining, need)
		if n == 0 {
			continue
		}
		rows, err := q.TakeFromGrant(ctx, store.TakeFromGrantParams{Credits: n, Stamp: stampOf(now), ID: g.ID, UserID: userID})
		if err != nil {
			return nil, 0, 0, fmt.Errorf("credits: take: %w", err)
		}
		if rows == 0 {
			return nil, 0, 0, fmt.Errorf("credits: grant %d changed under its lock", g.ID)
		}
		allocs = append(allocs, allocation{GrantID: g.ID, Credits: n})
		if isPeriodKind(g.Kind) {
			planPart += n
		} else {
			extraPart += n
		}
		need -= n
	}
	if need > 0 {
		return nil, 0, 0, ErrInsufficient
	}
	return allocs, planPart, extraPart, nil
}

// Refund gives a reservation back to the buckets it came from (idempotent; a nil or zero-credit reservation is a
// no-op). Credits return to their own bucket even when its period has ended meanwhile — they are then simply gone
// with it, which is what would have happened had they never been reserved.
func (s *Service) Refund(ctx context.Context, r *Reservation, now time.Time) error {
	if r == nil || r.ID == 0 {
		return nil
	}
	return s.inTx(ctx, func(q *store.Queries) error {
		if _, err := q.LockUserGrants(ctx, r.UserID); err != nil {
			return fmt.Errorf("credits: lock: %w", err)
		}
		sp, err := q.LockSpend(ctx, store.LockSpendParams{ID: r.ID, UserID: r.UserID})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("credits: spend: %w", err)
		}
		if sp.Status != "spent" {
			return nil
		}
		var allocs []allocation
		if err := json.Unmarshal(sp.Allocations, &allocs); err != nil {
			return fmt.Errorf("credits: allocations: %w", err)
		}
		for _, a := range allocs {
			if _, err := q.ReturnToGrant(ctx, store.ReturnToGrantParams{Credits: a.Credits, Stamp: stampOf(now), ID: a.GrantID, UserID: r.UserID}); err != nil {
				return fmt.Errorf("credits: return: %w", err)
			}
		}
		if _, err := q.MarkSpendRefunded(ctx, store.MarkSpendRefundedParams{Stamp: stampOf(now), ID: r.ID, UserID: r.UserID}); err != nil {
			return fmt.Errorf("credits: refund: %w", err)
		}
		return nil
	})
}

// RefundAt gives back the use of feature the user reserved at reservedAt (Reservation.At). For features whose work
// runs after the request (a queued extraction): they keep that moment on their own row and refund when the job
// fails. A no-op when no such reservation is (still) counted — refunding twice can never create credits.
func (s *Service) RefundAt(ctx context.Context, userID uint64, feature string, reservedAt, now time.Time) error {
	var id uint64
	err := s.inTx(ctx, func(q *store.Queries) error {
		sp, err := q.LockSpendAt(ctx, store.LockSpendAtParams{UserID: userID, Feature: feature, ReservedAt: stampOf(reservedAt)})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("credits: spend: %w", err)
		}
		id = sp.ID
		return nil
	})
	if err != nil || id == 0 {
		return err
	}
	return s.Refund(ctx, &Reservation{ID: id, UserID: userID}, now)
}

// ---------------------------------------------------------------------------
// Usage (the profile's «مصرف اعتبار» screen)

// Spend is one counted use.
type Spend struct {
	ID      uint64
	Feature string
	Credits uint32
	At      time.Time
}

// FeatureUsage is one feature's uses since a moment.
type FeatureUsage struct {
	Feature string
	Uses    int64
	Credits int64
}

// DayUsage is one Tehran day's credits.
type DayUsage struct {
	Day     string // YYYY-MM-DD
	Credits int64
}

// HistoryPageSize is the page of History.
const HistoryPageSize = 30

// History is a page of the user's uses, newest first; beforeID 0 = the first page. next is the cursor of the
// following page (0 = no more).
func (s *Service) History(ctx context.Context, userID, beforeID uint64) (items []Spend, next uint64, err error) {
	rows, err := s.q.ListUserSpends(ctx, store.ListUserSpendsParams{UserID: userID, BeforeID: beforeID, Limit: HistoryPageSize + 1})
	if err != nil {
		return nil, 0, fmt.Errorf("credits: history: %w", err)
	}
	if len(rows) > HistoryPageSize {
		rows = rows[:HistoryPageSize]
		next = rows[len(rows)-1].ID
	}
	items = make([]Spend, len(rows))
	for i, r := range rows {
		items[i] = Spend{ID: r.ID, Feature: r.Feature, Credits: r.Credits, At: r.CreatedAt.Time}
	}
	return items, next, nil
}

// Summary is the user's uses since `since`, by feature (most credits first) and by day (oldest first).
func (s *Service) Summary(ctx context.Context, userID uint64, since time.Time) ([]FeatureUsage, []DayUsage, error) {
	feats, err := s.q.UserSpendByFeature(ctx, store.UserSpendByFeatureParams{UserID: userID, Since: stampOf(since)})
	if err != nil {
		return nil, nil, fmt.Errorf("credits: by feature: %w", err)
	}
	days, err := s.q.UserSpendByDay(ctx, store.UserSpendByDayParams{UserID: userID, Since: stampOf(since)})
	if err != nil {
		return nil, nil, fmt.Errorf("credits: by day: %w", err)
	}
	fu := make([]FeatureUsage, len(feats))
	for i, f := range feats {
		fu[i] = FeatureUsage{Feature: f.Feature, Uses: f.Uses, Credits: f.Credits}
	}
	du := make([]DayUsage, len(days))
	for i, d := range days {
		du[i] = DayUsage{Day: dayString(d.Day), Credits: d.Credits}
	}
	return fu, du, nil
}

func dayString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	}
	return fmt.Sprint(v)
}

// Packs are the purchasable credit packs in display order.
func (s *Service) Packs(ctx context.Context) ([]store.AiCreditPack, error) {
	rows, err := s.q.ListActivePacks(ctx)
	if err != nil {
		return nil, fmt.Errorf("credits: packs: %w", err)
	}
	return rows, nil
}

// ---------------------------------------------------------------------------
// Admin adjustments

// MaxAdminGrant bounds one manual gift.
const MaxAdminGrant = 1_000_000

// AdminGrant gives the user extra credits (kind `admin`: never expiring, spent after the period's).
func (s *Service) AdminGrant(ctx context.Context, userID uint64, amount uint32, note string, now time.Time) (uint64, error) {
	if amount == 0 || amount > MaxAdminGrant {
		return 0, fmt.Errorf("credits: admin grant of %d", amount)
	}
	id, err := s.q.InsertAdminGrant(ctx, store.InsertAdminGrantParams{
		UserID: userID, Amount: amount, Note: sql.NullString{String: note, Valid: note != ""},
		CreatedAt: stampOf(now), UpdatedAt: stampOf(now),
	})
	if err != nil {
		return 0, fmt.Errorf("credits: admin grant: %w", err)
	}
	return uint64(id), nil //nolint:gosec // G115: ids are positive
}

// AdminDeduct takes up to amount from the user's extra credits (never from a period's) and returns how many were
// taken.
func (s *Service) AdminDeduct(ctx context.Context, userID uint64, amount uint32, now time.Time) (uint32, error) {
	var taken uint32
	err := s.inTx(ctx, func(q *store.Queries) error {
		if _, err := q.LockUserGrants(ctx, userID); err != nil {
			return fmt.Errorf("credits: lock: %w", err)
		}
		grants, err := q.ValidGrants(ctx, store.ValidGrantsParams{UserID: userID, Now: now})
		if err != nil {
			return fmt.Errorf("credits: grants: %w", err)
		}
		need := amount
		for _, g := range grants {
			if need == 0 {
				break
			}
			if isPeriodKind(g.Kind) {
				continue
			}
			n := min(g.Remaining, need)
			if n == 0 {
				continue
			}
			if _, err := q.TakeFromGrant(ctx, store.TakeFromGrantParams{Credits: n, Stamp: stampOf(now), ID: g.ID, UserID: userID}); err != nil {
				return fmt.Errorf("credits: deduct: %w", err)
			}
			taken += n
			need -= n
		}
		return nil
	})
	return taken, err
}

// inTx runs fn in a READ COMMITTED transaction: after the row locks are granted, the reads that follow must see
// what the previous holder committed (see internal/plus Service.inTx).
func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("credits: begin: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("credits: commit: %w", err)
	}
	return nil
}
