package billing

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/content/form"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/plus"
)

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
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
	)), nil
}

// ShowSettings is GET /plus/settings: the trial-offer percent and the VAT rate checkout applies (the admin override
// when set, else PLUS_VAT_RATE_BPS — vat_source says which).
func (h *Handlers) ShowSettings(c fiber.Ctx) error {
	body, err := h.settingsJSON(c)
	if err != nil {
		return err
	}
	return httpadmin.OK(c, body)
}

// UpdateSettings is PUT /plus/settings (super): trial_offer_percent (0–100, 0 = no offer) and vat_rate_bps
// (0–10000 basis points; null removes the override so the env value applies again; absent = unchanged). New
// checkouts use the values at once; existing invoices keep their own VAT snapshot.
func (h *Handlers) UpdateSettings(c fiber.Ctx) error {
	data, err := form.Validate(c, validation.Rules{
		validation.F("trial_offer_percent", "required|integer|min:0|max:100"),
		validation.F("vat_rate_bps", "nullable|integer|min:0|max:"+strconv.Itoa(plus.MaxVATRateBps)),
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
