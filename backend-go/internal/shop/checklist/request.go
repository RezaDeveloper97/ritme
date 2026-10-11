package checklist

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

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func pickKeys(body phpval.Map, keys ...string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now),
		validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

func str(data phpval.Map, key string) string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func has(data phpval.Map, key string) bool {
	v, ok := data.Get(key)
	return ok && v != nil
}

func intOf(data phpval.Map, key string) int {
	n, _ := strconv.Atoi(str(data, key))
	return n
}

var qtyRule = []any{"nullable", "integer", "min:0", "max:" + strconv.Itoa(MaxQuantity)}

// StateInput is PUT /shop/checklist/items/:id.
type StateInput struct {
	State    string
	Quantity int
}

// ValidateState validates {state, quantity?}.
func ValidateState(body phpval.Map, locale string, now time.Time) (StateInput, error) {
	data := pickKeys(body, "state", "quantity")
	rules := validation.Rules{
		validation.F("state", "required", "string", validation.In(States...)),
		validation.F("quantity", qtyRule...),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return StateInput{}, err
	}
	return StateInput{State: str(data, "state"), Quantity: intOf(data, "quantity")}, nil
}

// ValidateCustom validates a custom row: {title, group, state?, quantity?}; cur (update) makes every field optional.
func ValidateCustom(body phpval.Map, cur *CustomInput, locale string, now time.Time) (CustomInput, error) {
	data := pickKeys(body, "title", "group", "state", "quantity")
	presence := []any{"required"}
	if cur != nil {
		presence = []any{"sometimes", "required"}
	}
	rules := validation.Rules{
		validation.F("title", append(append([]any{}, presence...), "string", "max:"+strconv.Itoa(MaxTitleLen))...),
		validation.F("group", append(append([]any{}, presence...), "string", validation.In(Groups...))...),
		validation.F("state", "sometimes", "required", "string", validation.In(States...)),
		validation.F("quantity", qtyRule...),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return CustomInput{}, err
	}
	in := CustomInput{State: StateTodo}
	if cur != nil {
		in = *cur
	}
	if has(data, "title") {
		in.Title = str(data, "title")
	}
	if has(data, "group") {
		in.Group = str(data, "group")
	}
	if has(data, "state") {
		in.State = str(data, "state")
	}
	if _, ok := data.Get("quantity"); ok {
		in.Quantity = intOf(data, "quantity")
	}
	if in.Title == "" {
		return CustomInput{}, failValidation(locale, jsonx.Obj("title",
			[]string{lang.Default().Trans("validation.required", map[string]string{"attribute": attrName("title", locale)}, locale)}))
	}
	return in, nil
}

// attrName is the localized attribute name of key.
func attrName(key, locale string) string { return T("attributes."+key, locale) }

// ValidateSupply validates PUT /shop/supply-reminder: {enabled, lead_days?, product_id?}; omitted fields keep cur.
func ValidateSupply(body phpval.Map, cur Supply, locale string, now time.Time) (Supply, error) {
	data := pickKeys(body, "enabled", "lead_days", "product_id")
	rules := validation.Rules{
		validation.F("enabled", "required", "boolean"),
		validation.F("lead_days", "sometimes", "required", "integer",
			"min:"+strconv.Itoa(MinLeadDays), "max:"+strconv.Itoa(MaxLeadDays)),
		validation.F("product_id", "nullable", "integer", "min:1"),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return Supply{}, err
	}
	out := cur
	v, _ := data.Get("enabled")
	out.Enabled = phpval.Truthy(v)
	if has(data, "lead_days") {
		out.LeadDays = intOf(data, "lead_days")
	}
	if out.LeadDays < MinLeadDays || out.LeadDays > MaxLeadDays {
		out.LeadDays = DefaultLeadDays
	}
	if _, ok := data.Get("product_id"); ok {
		n, _ := strconv.ParseUint(str(data, "product_id"), 10, 64)
		out.ProductID = n
	}
	return out, nil
}
