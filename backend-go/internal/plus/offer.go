package plus

import (
	"context"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/plus/store"
)

// The trial offer (B-N2-06): while the user's free trial runs and she has no subscription, every plan costs
// trial_offer_percent less (admin setting, default 50 %). It ends with the trial. Checkout applies it server-side;
// the home banner (nbl_Prem_TrialHome) and the trial sheet (nbl_Prem_TrialSheet) show it with a countdown.

// Offer is a running trial offer.
type Offer struct {
	Now      time.Time
	Percent  int
	EndsAt   time.Time // the trial's end = the offer's end
	Featured Plan      // the banner/sheet plan: the highlighted one, else the first active plan
	Plans    []Plan
}

// OfferPrice is price less percent, the same rounding checkout uses (the discount is floored, so the buyer pays
// the ceiling).
func OfferPrice(price uint64, percent int) uint64 {
	return price - price*uint64(min(max(percent, 0), 100))/100 //nolint:gosec // G115: clamped 0–100
}

// offerPercent is the discount the user gets at now through q: the admin percent while her trial runs, else 0.
func offerPercent(ctx context.Context, q *store.Queries, s *Service, userID uint64, now time.Time) (int, *store.PlusTrial, error) {
	tier, _, trial, err := tierOf(ctx, q, userID, now)
	if err != nil || tier != TierTrial {
		return 0, nil, err
	}
	pct, err := trialOfferPercent(ctx, q, s)
	if err != nil {
		return 0, nil, err
	}
	return pct, trial, nil
}

// TrialOffer is the user's running offer at now, nil when there is none (no running trial, subscribed, offer off
// or no active plan).
func (s *Service) TrialOffer(ctx context.Context, userID uint64, now time.Time) (*Offer, error) {
	pct, trial, err := offerPercent(ctx, s.q, s, userID, now)
	if err != nil || pct == 0 {
		return nil, err
	}
	return s.offer(ctx, pct, trial.EndsAt, now)
}

// offer prices the active plans at pct (nil when there is no active plan).
func (s *Service) offer(ctx context.Context, pct int, endsAt, now time.Time) (*Offer, error) {
	plans, err := s.Plans(ctx)
	if err != nil || len(plans) == 0 {
		return nil, err
	}
	featured := plans[0]
	for _, p := range plans {
		if p.IsHighlighted {
			featured = p
			break
		}
	}
	return &Offer{Now: now, Percent: pct, EndsAt: endsAt, Featured: featured, Plans: plans}, nil
}

// TrialSheet is the trial sheet: status, the offer and what the user used of Plus since the trial began.
type TrialSheet struct {
	State      State
	Offer      *Offer
	Plans      []Plan
	UsageSince civildate.Date      // first day counted (the trial's start month, else this month)
	Used       []TrialFeatureUsage // in display order
}

// TrialFeatureUsage is one «این روزها از پلاس استفاده کردی» line.
type TrialFeatureUsage struct {
	Definition Definition
	Used       int
}

// TrialSheet builds the trial sheet at now. Uses are summed per feature over the quota months since the trial's
// start month (a 7-day trial can straddle two months); without a trial it is this month's usage.
func (s *Service) TrialSheet(ctx context.Context, userID uint64, now time.Time) (TrialSheet, error) {
	st, err := s.Status(ctx, userID, now)
	if err != nil {
		return TrialSheet{}, err
	}
	sheet := TrialSheet{State: st, Offer: st.Offer, UsageSince: st.Usage.PeriodStart}
	if st.Offer != nil {
		sheet.Plans = st.Offer.Plans
	} else if sheet.Plans, err = s.Plans(ctx); err != nil {
		return TrialSheet{}, err
	}
	if st.Trial != nil {
		sheet.UsageSince = PeriodStart(st.Trial.StartedAt)
	}
	rows, err := s.q.SumUsageSince(ctx, store.SumUsageSinceParams{UserID: userID, Since: sheet.UsageSince})
	if err != nil {
		return TrialSheet{}, fmt.Errorf("plus: trial usage: %w", err)
	}
	used := make(map[Key]int, len(rows))
	for _, r := range rows {
		used[Key(r.Feature)] = int(r.Used)
	}
	for _, d := range definitions {
		sheet.Used = append(sheet.Used, TrialFeatureUsage{Definition: d, Used: used[d.Key]})
	}
	return sheet, nil
}

// TrialBannerJSON is the home trial banner (TrialOfferJSON, null when no offer runs) for the request locale and
// the default language — the home.TrialBanner port of GET /home/cycle-overview.
func (s *Service) TrialBannerJSON(ctx context.Context, userID uint64, now time.Time, locale, defaultLocale string) (any, error) {
	o, err := s.TrialOffer(ctx, userID, now.In(civildate.Tehran).Truncate(time.Second))
	if err != nil {
		return nil, err
	}
	return TrialOfferJSON(o, Localizer{Locale: locale, Default: defaultLocale}), nil
}
