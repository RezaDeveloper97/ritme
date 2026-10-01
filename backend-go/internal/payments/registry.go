package payments

import (
	"log/slog"
	"net/http"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Deps are what the registry needs to build a provider.
type Deps struct {
	App    config.App
	Config config.Payment
	Plus   config.Plus
	Cache  *cache.Client // the fake's state (Redis)
	Logger *slog.Logger
	// FakeStore overrides the fake's Redis store (tests). HTTPClient overrides the gateway client (tests).
	FakeStore  FakeStore
	HTTPClient *http.Client
}

// New builds the Gateway for the configured provider, or nil when payments are unavailable (provider none, or a
// real provider whose keys are missing — then checkout/verify answer 503 payment_unavailable). The fake is never
// built in production, whatever the config says (config.Load already refuses that combination).
func New(d Deps) *Gateway {
	cfg := d.Config
	provider := config.DefaultPaymentProvider(cfg.Provider, d.App)
	base := cfg.CallbackBaseURL
	if base == "" {
		base = d.App.URL
	}
	returns := cfg.ReturnURLs
	if len(returns) == 0 && d.Plus.CallbackURL != "" {
		returns = []string{d.Plus.CallbackURL}
	}
	client := d.HTTPClient
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		client = &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	var p Provider
	switch provider {
	case FakeName:
		if d.App.IsProduction() {
			d.Logger.Error("payments: the fake provider is disabled in production; payments unavailable")
			return nil
		}
		store := d.FakeStore
		if store == nil {
			if d.Cache == nil {
				d.Logger.Error("payments: the fake provider needs Redis; payments unavailable")
				return nil
			}
			store = NewRedisFakeStore(d.Cache)
		}
		p = NewFake(store, base)
	case ZarinpalName:
		if cfg.Zarinpal.MerchantID == "" {
			d.Logger.Error("payments: PAYMENT_PROVIDER=zarinpal but ZARINPAL_MERCHANT_ID is empty; payments unavailable")
			return nil
		}
		p = NewZarinpal(cfg.Zarinpal.MerchantID, cfg.Zarinpal.BaseURL, cfg.Zarinpal.Sandbox, client)
	default:
		return nil
	}
	return NewGateway(p, base, NewAllowList(returns), d.Logger)
}
