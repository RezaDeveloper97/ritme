package billing

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/ai/credits"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/plus"
)

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// aiCostFXKey is the plus_settings key the KPI job reads (metrics.FXRateSetting): IRR per USD for the AI cost share.
const aiCostFXKey = "ai_cost_usd_irr_rate"

// aiCostFXRate is the stored rate as a number (nil when unset or unparsable — the KPI treats both as unset).
func (h *Handlers) aiCostFXRate(c fiber.Ctx) (any, error) {
	var v string
	err := h.db.QueryRowContext(c, "SELECT `value` FROM plus_settings WHERE `key` = ?", aiCostFXKey).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, perr := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if perr != nil || f <= 0 {
		return nil, nil //nolint:nilerr // an unreadable or non-positive setting means "unset"
	}
	return f, nil
}

func (h *Handlers) settingsJSON(c fiber.Ctx) (*jsonx.OrderedMap, error) {
	pct, err := h.svc.TrialOfferPercent(c)
	if err != nil {
		return nil, err
	}
	override, err := h.svc.VATOverride(c)
	if err != nil {
		return nil, err
	}
	vat, err := h.svc.VATRateBps(c)
	if err != nil {
		return nil, err
	}
	fx, err := h.aiCostFXRate(c)
	if err != nil {
		return nil, err
	}
	trialCredits, err := h.credits.TrialCredits(c)
	if err != nil {
		return nil, err
	}
	cfg := h.svc.Config()
	source := "env"
	if override != nil {
		source = "admin"
	}
	return jsonx.Obj("settings", jsonx.Obj(
		"trial_offer_percent", pct,
		"vat_rate_bps", vat,
		"vat_override_bps", intOrNil(override),
		"vat_env_rate_bps", cfg.VATRateBps,
		"vat_source", source,
		"trial_days", cfg.TrialDays,
		"ai_cost_usd_irr_rate", fx,
		"trial_ai_credits", trialCredits,
		"max_trial_ai_credits", credits.MaxTrialCredits,
	)), nil
}

// ShowSettings is GET /plus/settings: the trial-offer percent, the VAT rate checkout applies (the admin override
// when set, else PLUS_VAT_RATE_BPS — vat_source says which) and the trial's AI credits (trial_ai_credits).
func (h *Handlers) ShowSettings(c fiber.Ctx) error {
	body, err := h.settingsJSON(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body)
}

// UpdateSettings is PUT /plus/settings (super): trial_offer_percent (0–100, 0 = no offer) and vat_rate_bps
// (0–10000 basis points; null removes the override so the env value applies again; absent = unchanged). New
// checkouts use the values at once; ai_cost_usd_irr_rate (IRR per USD, 1–100 000 000; null removes it so the KPI
// ai_cost.share is null again; absent = unchanged) feeds the KPI snapshot job; existing invoices keep their own VAT snapshot.
// trial_ai_credits (0–100000; absent or null = unchanged) is what a trial that starts from now on carries — a trial
// whose credit bucket already exists keeps its amount.
func (h *Handlers) UpdateSettings(c fiber.Ctx) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("trial_offer_percent", "required|integer|min:0|max:100"),
		validation.F("vat_rate_bps", "nullable|integer|min:0|max:"+strconv.Itoa(plus.MaxVATRateBps)),
		validation.F("ai_cost_usd_irr_rate", "nullable|numeric|min:1|max:100000000"),
		validation.F("trial_ai_credits", "nullable|integer|min:0|max:"+strconv.Itoa(credits.MaxTrialCredits)),
	})
	if err != nil {
		return err
	}
	at := now(c)
	details := jsonx.Obj()
	oldPct, err := h.svc.TrialOfferPercent(c)
	if err != nil {
		return err
	}
	pct := int(u64(data, "trial_offer_percent")) //nolint:gosec // G115: validated 0–100
	if pct != oldPct {
		if err := h.svc.SetTrialOfferPercent(c, pct, at); err != nil {
			return err
		}
		details.Set("trial_offer_percent", jsonx.Obj("from", oldPct, "to", pct))
	}
	if _, sent := data.Get("vat_rate_bps"); sent {
		oldVAT, err := h.svc.VATOverride(c)
		if err != nil {
			return err
		}
		var next *int
		if form.Has(data, "vat_rate_bps") {
			n := int(u64(data, "vat_rate_bps")) //nolint:gosec // G115: validated 0–10000
			next = &n
		}
		if !sameInt(oldVAT, next) {
			if err := h.svc.SetVATRateBps(c, next, at); err != nil {
				return err
			}
			details.Set("vat_override_bps", jsonx.Obj("from", intOrNil(oldVAT), "to", intOrNil(next)))
		}
	}
	if v, sent := data.Get("ai_cost_usd_irr_rate"); sent {
		old, err := h.aiCostFXRate(c)
		if err != nil {
			return err
		}
		var next any
		if v != nil {
			next = phpval.ToFloat(v)
		}
		if old != next {
			if next == nil {
				_, err = h.db.ExecContext(c, "DELETE FROM plus_settings WHERE `key` = ?", aiCostFXKey)
			} else {
				_, err = h.db.ExecContext(c, "INSERT INTO plus_settings (`key`, `value`, created_at, updated_at) VALUES (?, ?, ?, ?) "+
					"ON DUPLICATE KEY UPDATE `value` = VALUES(`value`), updated_at = VALUES(updated_at)",
					aiCostFXKey, strconv.FormatFloat(next.(float64), 'f', -1, 64), at, at)
			}
			if err != nil {
				return err
			}
			details.Set("ai_cost_usd_irr_rate", jsonx.Obj("from", old, "to", next))
		}
	}
	if form.Has(data, "trial_ai_credits") {
		old, err := h.credits.TrialCredits(c)
		if err != nil {
			return err
		}
		next := uint32(u64(data, "trial_ai_credits")) //nolint:gosec // G115: validated ≤ MaxTrialCredits
		if next != old {
			if err := h.q.AdminUpsertPlusSetting(c, store.AdminUpsertPlusSettingParams{SettingKey: credits.SettingTrialCredits,
				SettingValue: strconv.FormatUint(uint64(next), 10), Now: sql.NullTime{Time: at, Valid: true}}); err != nil {
				return err
			}
			details.Set("trial_ai_credits", jsonx.Obj("from", old, "to", next))
		}
	}
	if details.Len() > 0 {
		if err := h.record(c, h.q, action{name: "settings.update", targetType: "settings", details: details}); err != nil {
			return err
		}
	}
	body, err := h.settingsJSON(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body, "Settings saved.")
}

func sameInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
