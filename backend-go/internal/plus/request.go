package plus

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// validate runs rules over the keys of body (unknown keys are never read) and returns the validated data.
func validate(body phpval.Map, keys []string, rules validation.Rules, locale string, now time.Time) (phpval.Map, error) {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return nil, failValidation(locale, v.ErrorBag())
	}
	return data, nil
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any, extras ...any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), append([]any{"errors", errs}, extras...)...)
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

// CheckoutInput is a validated POST /plus/checkout. The body never carries an amount: the server prices the plan.
type CheckoutInput struct {
	PlanID       uint64
	DiscountCode string
	Preview      bool
}

func validateCheckout(body phpval.Map, locale string, now time.Time) (CheckoutInput, error) {
	F := validation.F
	data, err := validate(body, []string{"plan_id", "discount_code", "preview"}, validation.Rules{
		F("plan_id", "required", "integer", "min:1"),
		F("discount_code", "nullable", "string", "max:64"),
		F("preview", "nullable", "boolean"),
	}, locale, now)
	if err != nil {
		return CheckoutInput{}, err
	}
	id, _ := strconv.ParseUint(str(data, "plan_id"), 10, 64)
	preview, _ := data.Get("preview")
	return CheckoutInput{
		PlanID:       id,
		DiscountCode: strings.ToUpper(str(data, "discount_code")),
		Preview:      phpval.Truthy(preview),
	}, nil
}

// VerifyInput is a validated POST /plus/verify: the invoice reference, the gateway's payment id and the callback
// status the gateway returned the user with.
type VerifyInput struct {
	Reference string
	Authority string
	Status    string
}

func validateVerify(body phpval.Map, locale string, now time.Time) (VerifyInput, error) {
	F := validation.F
	data, err := validate(body, []string{"reference", "authority", "status"}, validation.Rules{
		F("reference", "required", "string", "max:32"),
		F("authority", "required", "string", "max:191"),
		F("status", "nullable", "string", "max:32"),
	}, locale, now)
	if err != nil {
		return VerifyInput{}, err
	}
	return VerifyInput{Reference: str(data, "reference"), Authority: str(data, "authority"), Status: str(data, "status")}, nil
}

// planInvalid is the 422 of an unknown or inactive plan.
func planInvalid(locale string) error {
	return failValidation(locale, jsonx.Obj("plan_id", []string{T("errors.plan_invalid", locale)}))
}

// discountInvalid is the 422 of a rejected discount code, error_code discount_<reason>.
func discountInvalid(locale, reason string) error {
	return failValidation(locale, jsonx.Obj("discount_code", []string{T("discount."+reason, locale)}), "error_code", "discount_"+reason)
}

// StoreIntentInput is a validated POST /plus/store/intent.
type StoreIntentInput struct {
	Store string
	SKU   string
}

func validateStoreIntent(body phpval.Map, locale string, now time.Time) (StoreIntentInput, error) {
	F := validation.F
	data, err := validate(body, []string{"store", "sku"}, validation.Rules{
		F("store", "required", "string", "in:bazaar,myket"),
		F("sku", "required", "string", "max:100"),
	}, locale, now)
	if err != nil {
		return StoreIntentInput{}, err
	}
	return StoreIntentInput{Store: str(data, "store"), SKU: str(data, "sku")}, nil
}

// StorePurchaseInput is one signed store purchase: the exact purchase JSON the store returned and its signature.
type StorePurchaseInput struct {
	SignedData string
	Signature  string
}

// StoreVerifyInput is a validated POST /plus/store/verify (one purchase) or /plus/store/sync (up to 20).
type StoreVerifyInput struct {
	Store     string
	Purchases []StorePurchaseInput
}

func validateStoreVerify(body phpval.Map, locale string, now time.Time) (StoreVerifyInput, error) {
	F := validation.F
	data, err := validate(body, []string{"store", "signed_data", "signature"}, validation.Rules{
		F("store", "required", "string", "in:bazaar,myket"),
		F("signed_data", "required", "string", "max:8192"),
		F("signature", "required", "string", "max:2048"),
	}, locale, now)
	if err != nil {
		return StoreVerifyInput{}, err
	}
	sd, _ := data.Get("signed_data") // untrimmed: the signature covers the exact bytes
	return StoreVerifyInput{Store: str(data, "store"), Purchases: []StorePurchaseInput{{
		SignedData: phpval.ToString(sd), Signature: str(data, "signature"),
	}}}, nil
}

func validateStoreSync(body phpval.Map, locale string, now time.Time) (StoreVerifyInput, error) {
	F := validation.F
	data, err := validate(body, []string{"store", "purchases"}, validation.Rules{
		F("store", "required", "string", "in:bazaar,myket"),
		F("purchases", "present", "array", "max:20"),
		F("purchases.*.signed_data", "required", "string", "max:8192"),
		F("purchases.*.signature", "required", "string", "max:2048"),
	}, locale, now)
	if err != nil {
		return StoreVerifyInput{}, err
	}
	out := StoreVerifyInput{Store: str(data, "store")}
	raw, _ := data.Get("purchases")
	list, _ := raw.([]any)
	for _, item := range list {
		m, ok := item.(phpval.Map)
		if !ok {
			continue
		}
		sd, _ := m.Get("signed_data")
		sig, _ := m.Get("signature")
		out.Purchases = append(out.Purchases, StorePurchaseInput{SignedData: phpval.ToString(sd), Signature: strings.TrimSpace(phpval.ToString(sig))})
	}
	return out, nil
}
