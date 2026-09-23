package checkups

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Enum values of the API.
var (
	Results       = []string{"normal", "follow_up", "pending"}
	Performers    = []string{"self", "doctor", "lab", "dentist"}
	ListFilters   = []string{"all", "action", "done"}
	RecordFilters = []string{"all", "this_year", "with_attachment"}
)

// Custom checkup bounds.
const (
	customTitleMax    = 120
	customNoteMax     = 500
	customIntervalMax = 120
	recordNoteMax     = 2000
	// customSortOrder places custom checkups after the catalog.
	customSortOrder = 1000
)

// recordFields / customFields are the body keys each write accepts (anything else is ignored).
var (
	recordFields = []string{"done_on", "result", "findings", "note", "has_attachment", "next_due_on"}
	customFields = []string{"title", "interval_months", "performed_by", "note", "last_done_on"}
)

func pastDate() []any { return []any{"date_format:Y-m-d", "before_or_equal:today"} }

func recordRules(findingKeys []string) validation.Rules {
	F, in := validation.F, validation.In
	return validation.Rules{
		F("done_on", append([]any{"required"}, pastDate()...)...),
		F("result", "required", in(Results...)),
		F("findings", "nullable", "array"),
		F("findings.*", "required", "string", in(findingKeys...)),
		F("note", "nullable", "string", "max:"+strconv.Itoa(recordNoteMax)),
		F("has_attachment", "sometimes", "boolean"),
		F("next_due_on", "nullable", "date_format:Y-m-d", "after:done_on"),
	}
}

func customRules() validation.Rules {
	F, in := validation.F, validation.In
	return validation.Rules{
		F("title", "required", "string", "max:"+strconv.Itoa(customTitleMax)),
		F("interval_months", "required", "integer", "min:1", "max:"+strconv.Itoa(customIntervalMax)),
		F("performed_by", "required", in(Performers...)),
		F("note", "nullable", "string", "max:"+strconv.Itoa(customNoteMax)),
		F("last_done_on", append([]any{"nullable"}, pastDate()...)...),
	}
}

func messages(locale string) validation.Option {
	future := T("validation.date_future", locale)
	return validation.Messages(
		"done_on.before_or_equal", future,
		"last_done_on.before_or_equal", future,
		"findings.*.in", T("validation.findings_invalid", locale),
	)
}

// validate runs rules over data; a failure is the controller-style 422.
func validate(locale string, data phpval.Map, rules validation.Rules, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules,
		validation.Now(now), validation.Attributes(attributes(locale)...), messages(locale))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	return nil
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// pick copies the known keys of in over base.
func pick(base, in phpval.Map, keys []string) phpval.Map {
	for _, k := range keys {
		if v, ok := in.Get(k); ok {
			base.Set(k, v)
		}
	}
	return base
}

// str is the string value of key (nil when missing, null or blank).
func str(data phpval.Map, key string) *string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return nil
	}
	s := phpval.ToString(v)
	if s == "" {
		return nil
	}
	return &s
}

// date is the Y-m-d value of key (validated before), zero when missing or null.
func date(data phpval.Map, key string) civildate.Date {
	s := str(data, key)
	if s == nil {
		return civildate.Date{}
	}
	d, err := civildate.Parse(*s)
	if err != nil {
		return civildate.Date{}
	}
	return d
}

// stringList is the list value of key, de-duplicated in request order.
func stringList(data phpval.Map, key string) []string {
	v, ok := data.Get(key)
	if !ok || v == nil {
		return nil
	}
	_, vals := phpval.Entries(v)
	out := []string{}
	seen := map[string]bool{}
	for _, x := range vals {
		s := phpval.ToString(x)
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// parseID is a positive integer route segment.
func parseID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return id, err == nil && id > 0
}
