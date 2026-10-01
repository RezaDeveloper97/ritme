package plus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/plus/store"
)

// Admin-editable settings (table plus_settings, migration 00018). Each key has a code default used when the row is
// missing or unreadable, so a broken row can never block checkout. The admin module (B-N2-09) writes them through
// the Set* methods below.
const (
	// SettingTrialOfferPercent is the discount every plan gets while the user's trial runs (0 = offer off).
	SettingTrialOfferPercent = "trial_offer_percent"
	// DefaultTrialOfferPercent is the designed «۵۰٪ تخفیف» of nbl_Prem_TrialHome / nbl_Prem_TrialSheet.
	DefaultTrialOfferPercent = 50
	// SettingVATRateBps overrides PLUS_VAT_RATE_BPS (basis points, 0–10000; 1000 = 10 %). No row = the env value.
	// Checkout reads it at quote time and every invoice keeps its own snapshot (plus_invoices.vat_rate_bps).
	SettingVATRateBps = "vat_rate_bps"
	// MaxVATRateBps is 100 %.
	MaxVATRateBps = 10000
)

// ErrInvalidSetting rejects a setting value outside its range.
var ErrInvalidSetting = errors.New("plus: invalid setting value")

// TrialOfferPercent is the admin-set trial-offer discount, 0–100 (default 50).
func (s *Service) TrialOfferPercent(ctx context.Context) (int, error) {
	return trialOfferPercent(ctx, s.q, s)
}

func trialOfferPercent(ctx context.Context, q *store.Queries, s *Service) (int, error) {
	raw, err := q.GetSetting(ctx, SettingTrialOfferPercent)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultTrialOfferPercent, nil
	}
	if err != nil {
		return 0, fmt.Errorf("plus: setting %s: %w", SettingTrialOfferPercent, err)
	}
	n, ok := parsePercent(raw)
	if !ok {
		s.logger.WarnContext(ctx, "plus: unreadable setting, using the default", "key", SettingTrialOfferPercent)
		return DefaultTrialOfferPercent, nil
	}
	return n, nil
}

// parsePercent reads a stored 0–100 integer.
func parsePercent(raw string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	return n, err == nil && n >= 0 && n <= 100
}

// SetTrialOfferPercent stores the trial-offer discount (0–100; 0 turns the offer off). For the admin module.
func (s *Service) SetTrialOfferPercent(ctx context.Context, percent int, now time.Time) error {
	if percent < 0 || percent > 100 {
		return ErrInvalidSetting
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	if err := s.q.UpsertSetting(ctx, store.UpsertSettingParams{
		Key: SettingTrialOfferPercent, Value: strconv.Itoa(percent), CreatedAt: stamp, UpdatedAt: stamp,
	}); err != nil {
		return fmt.Errorf("plus: setting %s: %w", SettingTrialOfferPercent, err)
	}
	return nil
}

// VATRateBps is the VAT rate checkout applies now: the admin override (plus_settings.vat_rate_bps) when set and
// readable, else PLUS_VAT_RATE_BPS.
func (s *Service) VATRateBps(ctx context.Context) (int, error) {
	return vatRateBps(ctx, s.q, s)
}

// VATOverride is the admin's VAT override, nil when none is stored (the env value applies).
func (s *Service) VATOverride(ctx context.Context) (*int, error) {
	raw, err := s.q.GetSetting(ctx, SettingVATRateBps)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plus: setting %s: %w", SettingVATRateBps, err)
	}
	n, ok := parseBps(raw)
	if !ok {
		return nil, nil
	}
	return &n, nil
}

func vatRateBps(ctx context.Context, q *store.Queries, s *Service) (int, error) {
	raw, err := q.GetSetting(ctx, SettingVATRateBps)
	if errors.Is(err, sql.ErrNoRows) {
		return s.cfg.VATRateBps, nil
	}
	if err != nil {
		return 0, fmt.Errorf("plus: setting %s: %w", SettingVATRateBps, err)
	}
	n, ok := parseBps(raw)
	if !ok {
		s.logger.WarnContext(ctx, "plus: unreadable setting, using the env value", "key", SettingVATRateBps)
		return s.cfg.VATRateBps, nil
	}
	return n, nil
}

// parseBps reads a stored 0–10000 integer.
func parseBps(raw string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	return n, err == nil && n >= 0 && n <= MaxVATRateBps
}

// SetVATRateBps stores the VAT override (0–10000 basis points); nil removes it so PLUS_VAT_RATE_BPS applies again.
// Invoices already written keep the rate they were priced with. For the admin module.
func (s *Service) SetVATRateBps(ctx context.Context, bps *int, now time.Time) error {
	if bps == nil {
		if err := s.q.DeleteSetting(ctx, SettingVATRateBps); err != nil {
			return fmt.Errorf("plus: setting %s: %w", SettingVATRateBps, err)
		}
		return nil
	}
	if *bps < 0 || *bps > MaxVATRateBps {
		return ErrInvalidSetting
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	if err := s.q.UpsertSetting(ctx, store.UpsertSettingParams{
		Key: SettingVATRateBps, Value: strconv.Itoa(*bps), CreatedAt: stamp, UpdatedAt: stamp,
	}); err != nil {
		return fmt.Errorf("plus: setting %s: %w", SettingVATRateBps, err)
	}
	return nil
}
