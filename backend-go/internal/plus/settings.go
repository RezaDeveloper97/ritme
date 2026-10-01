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
