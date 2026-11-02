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

// Admin-editable settings (table plus_settings, migration 00018; the T-M10-29 quota keys need no migration). Each key has a code default used when the row is
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
	// SettingLabAIFreeLimit is the free lab-AI analyses per month (A-03; default DefaultLabAIFreeLimit, 0 = locked
	// for free users, at most MaxLabAIFreeLimit).
	SettingLabAIFreeLimit = "lab_ai_free_limit"
	// MaxLabAIFreeLimit keeps the free allowance below the Plus one (10).
	MaxLabAIFreeLimit = 10
	// SettingAssistantFairUse is the Plus assistant's fair-use cap per month (A-04; default DefaultAssistantFairUse,
	// 1–MaxAssistantFairUse).
	SettingAssistantFairUse = "assistant_fair_use_cap"
	// MaxAssistantFairUse bounds the cap.
	MaxAssistantFairUse = 100000
)

// tunable is one admin setting that overrides a code default of the entitlement rules (D-178).
type tunable struct {
	key      Key
	setting  string
	min, max int
	apply    func(d *Definition, n int)
}

// tunables are the admin-tunable allowances: the lab-AI free limit and the assistant's fair-use cap.
var tunables = []tunable{
	{key: LabAI, setting: SettingLabAIFreeLimit, min: 0, max: MaxLabAIFreeLimit, apply: func(d *Definition, n int) {
		d.Free = Quota{Enabled: n > 0, Limit: n}
	}},
	{key: AssistantUnlimited, setting: SettingAssistantFairUse, min: 1, max: MaxAssistantFairUse, apply: func(d *Definition, n int) {
		d.Plus.FairUse = n
	}},
}

func tunableOf(key Key) (tunable, bool) {
	for _, t := range tunables {
		if t.key == key {
			return t, true
		}
	}
	return tunable{}, false
}

// Definition is key's rule with the admin settings applied (one settings read only for a tunable key). A missing or
// unreadable row keeps the code default, so a broken setting can never lock or open a feature by surprise.
func (s *Service) Definition(ctx context.Context, key Key) (Definition, error) {
	d, ok := Lookup(key)
	if !ok {
		return Definition{}, ErrUnknownFeature
	}
	t, ok := tunableOf(key)
	if !ok {
		return d, nil
	}
	n, ok, err := s.intSetting(ctx, t.setting, t.min, t.max)
	if err != nil {
		return Definition{}, err
	}
	if ok {
		t.apply(&d, n)
	}
	return d, nil
}

// Definitions is every rule with the admin settings applied, in display order.
func (s *Service) Definitions(ctx context.Context) ([]Definition, error) {
	out := Definitions()
	for i := range out {
		if _, ok := tunableOf(out[i].Key); !ok {
			continue
		}
		d, err := s.Definition(ctx, out[i].Key)
		if err != nil {
			return nil, err
		}
		out[i] = d
	}
	return out, nil
}

// intSetting reads a stored integer in [lo, hi]; ok false = no row or an unreadable one (logged).
func (s *Service) intSetting(ctx context.Context, key string, lo, hi int) (int, bool, error) {
	raw, err := s.q.GetSetting(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("plus: setting %s: %w", key, err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < lo || n > hi {
		s.logger.WarnContext(ctx, "plus: unreadable setting, using the default", "key", key)
		return 0, false, nil //nolint:nilerr // logged; an unreadable setting falls back to the default
	}
	return n, true, nil
}

// LabAIFreeLimit is the effective free lab-AI allowance per month (for the admin module).
func (s *Service) LabAIFreeLimit(ctx context.Context) (int, error) {
	d, err := s.Definition(ctx, LabAI)
	if err != nil {
		return 0, err
	}
	return d.Free.Limit, nil
}

// AssistantFairUse is the effective Plus assistant fair-use cap per month (for the admin module).
func (s *Service) AssistantFairUse(ctx context.Context) (int, error) {
	d, err := s.Definition(ctx, AssistantUnlimited)
	if err != nil {
		return 0, err
	}
	return d.Plus.FairUse, nil
}

// SetLabAIFreeLimit stores the free lab-AI allowance (0–MaxLabAIFreeLimit; 0 locks it for free users); nil removes
// the row so DefaultLabAIFreeLimit applies again. For the admin module.
func (s *Service) SetLabAIFreeLimit(ctx context.Context, n *int, now time.Time) error {
	return s.setIntSetting(ctx, SettingLabAIFreeLimit, n, 0, MaxLabAIFreeLimit, now)
}

// SetAssistantFairUse stores the Plus assistant's fair-use cap (1–MaxAssistantFairUse); nil removes the row so
// DefaultAssistantFairUse applies again. For the admin module.
func (s *Service) SetAssistantFairUse(ctx context.Context, n *int, now time.Time) error {
	return s.setIntSetting(ctx, SettingAssistantFairUse, n, 1, MaxAssistantFairUse, now)
}

func (s *Service) setIntSetting(ctx context.Context, key string, n *int, lo, hi int, now time.Time) error {
	if n == nil {
		if err := s.q.DeleteSetting(ctx, key); err != nil {
			return fmt.Errorf("plus: setting %s: %w", key, err)
		}
		return nil
	}
	if *n < lo || *n > hi {
		return ErrInvalidSetting
	}
	stamp := sql.NullTime{Time: now, Valid: true}
	if err := s.q.UpsertSetting(ctx, store.UpsertSettingParams{
		Key: key, Value: strconv.Itoa(*n), CreatedAt: stamp, UpdatedAt: stamp,
	}); err != nil {
		return fmt.Errorf("plus: setting %s: %w", key, err)
	}
	return nil
}

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
