package join

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Step inputs are read field by field from the allow-listed keys only; anything else in the body is ignored.
// Repeated fields use the `list` rule (not `array`): a JSON object would pass `array` but the parser walks them
// by numeric index, so a map would dodge the `.*` rules (and used to nil-deref on weekly_hours).

// digitFold maps Persian / Arabic digits to ASCII and drops separators (the mobile is validated normalised).
var digitFold = strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
	"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	" ", "", "-", "", "(", "", ")", "")

var stepKeys = map[int][]string{
	1: {"business_name", "category_slugs", "city_id", "district", "contact_name", "contact_phone", "owner_confirmed"},
	2: {"lat", "lng", "address", "about", "age_ranges", "amenities", "weekly_hours"},
	3: {"services"},
	4: {"booking_mode", "terms_accepted"},
}

func stepRules(step int) validation.Rules {
	F := validation.F
	switch step {
	case 1:
		return validation.Rules{
			F("business_name", "nullable", "string", "max:160"),
			F("category_slugs", "nullable", "list", "max:"+strconv.Itoa(MaxCategories)),
			F("category_slugs.*", "required", "string", "max:64"),
			F("city_id", "nullable", "integer", "min:1"),
			F("district", "nullable", "string", "max:120"),
			F("contact_name", "nullable", "string", "max:120"),
			F("contact_phone", "nullable", "string", "max:32"),
			F("owner_confirmed", "nullable", "boolean"),
		}
	case 2:
		return validation.Rules{
			F("lat", "nullable", "numeric", "between:-90,90"),
			F("lng", "nullable", "numeric", "between:-180,180"),
			F("address", "nullable", "string", "max:300"),
			F("about", "nullable", "string", "max:"+strconv.Itoa(MaxAbout)),
			F("age_ranges", "nullable", "list", "max:20"),
			F("age_ranges.*", "required", "string", "max:64"),
			F("amenities", "nullable", "list", "max:30"),
			F("amenities.*", "required", "string", "max:64"),
			F("weekly_hours", "nullable", "list", "max:7"),
			F("weekly_hours.*", "required", "array"),
			F("weekly_hours.*.day", "required", "integer", "min:0", "max:6"),
			F("weekly_hours.*.open", "required", "date_format:H:i"),
			F("weekly_hours.*.close", "required", "date_format:H:i"),
		}
	case 3:
		return validation.Rules{
			F("services", "nullable", "list", "max:"+strconv.Itoa(MaxServices)),
			F("services.*", "required", "array"),
			F("services.*.kind", "required", validation.In(ServiceKinds...)),
			F("services.*.name", "required", "string", "max:160"),
			F("services.*.description", "nullable", "string", "max:300"),
			F("services.*.format", "nullable", validation.In(ServiceFormats...)),
			F("services.*.duration_minutes", "nullable", "integer", "min:5", "max:600"),
			F("services.*.age_min_months", "nullable", "integer", "min:0", "max:240"),
			F("services.*.age_max_months", "nullable", "integer", "min:0", "max:240"),
			F("services.*.capacity", "nullable", "integer", "min:1", "max:500"),
			F("services.*.price_toman", "nullable", "integer", "min:0", "max:"+strconv.Itoa(MaxPrice)),
			F("services.*.package_sessions", "nullable", "integer", "min:1", "max:200"),
			F("services.*.package_validity_days", "nullable", "integer", "min:1", "max:730"),
		}
	default:
		return validation.Rules{
			F("booking_mode", "nullable", validation.In(BookingModes...)),
			F("terms_accepted", "nullable", "boolean"),
		}
	}
}

func text(data any, key string) string {
	v, ok := phpval.Get(data, key)
	if !ok || v == nil {
		return ""
	}
	return strings.Join(strings.Fields(phpval.ToString(v)), " ")
}

// longText keeps line breaks (the «درباره مجموعه» copy) but trims the ends.
func longText(data any, key string) string {
	v, ok := phpval.Get(data, key)
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(phpval.ToString(v))
}

func num(data any, key string) *int64 {
	v, ok := phpval.Get(data, key)
	if !ok || v == nil {
		return nil
	}
	n := int64(phpval.ToFloat(v))
	return &n
}

// numOr is num with a fallback for an absent / null value (never dereferences nil).
func numOr(data any, key string, fallback int64) int64 {
	if n := num(data, key); n != nil {
		return *n
	}
	return fallback
}

func truthy(data any, key string) bool {
	v, ok := phpval.Get(data, key)
	return ok && v != nil && phpval.Truthy(v)
}

func elems(data any, key string) []any {
	v, ok := phpval.Get(data, key)
	if !ok {
		return nil
	}
	_, vals := phpval.Entries(v)
	return vals
}

func strs(data any, key string) []string {
	var out []string
	for _, e := range elems(data, key) {
		out = append(out, strings.TrimSpace(phpval.ToString(e)))
	}
	return out
}

// fieldErrs collects semantic errors (after the rule validation passed) in insertion order.
type fieldErrs struct {
	bag    *jsonx.OrderedMap
	locale string
}

func (f *fieldErrs) add(field, key string) {
	if f.bag == nil {
		f.bag = jsonx.NewObject()
	}
	if _, ok := f.bag.Get(field); !ok {
		f.bag.Set(field, []string{T("validation."+key, f.locale, nil)})
	}
}

// codes checks a chip list against the allowed codes and duplicates.
func (f *fieldErrs) codes(field string, got, allowed []string, key string) {
	seen := map[string]bool{}
	for i, s := range got {
		item := field + "." + strconv.Itoa(i)
		switch {
		case !slices.Contains(allowed, s):
			f.add(item, key)
		case seen[s]:
			f.add(item, "distinct")
		}
		seen[s] = true
	}
}

// parseStep validates the step body and returns its Step1..Step4 input (a 422 error otherwise).
func (h *Handlers) parseStep(c fiber.Ctx, step int, locale string, now time.Time) (any, error) {
	body := validation.Input(c)
	data := phpval.NewMap()
	for _, k := range stepKeys[step] {
		if v, ok := body.Get(k); ok {
			if s, isStr := v.(string); isStr && k == "contact_phone" {
				v = digitFold.Replace(s)
			}
			data.Set(k, v)
		}
	}
	v := validation.Make(lang.Default(), locale, data, stepRules(step), validation.Now(now),
		validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return nil, failValidation(locale, v.ErrorBag())
	}
	errs := &fieldErrs{locale: locale}
	var out any
	switch step {
	case 1:
		in := Step1{BusinessName: text(data, "business_name"), CategorySlugs: strs(data, "category_slugs"),
			District: text(data, "district"), ContactName: text(data, "contact_name"),
			ContactPhone: text(data, "contact_phone"), OwnerConfirmed: truthy(data, "owner_confirmed")}
		if n := num(data, "city_id"); n != nil && *n > 0 {
			in.CityID = uint64(*n)
			ok, err := h.svc.CityActive(c, in.CityID)
			if err != nil {
				return nil, err
			}
			if !ok {
				errs.add("city_id", "city")
			}
		}
		if in.ContactPhone != "" && !mobile(in.ContactPhone) {
			errs.add("contact_phone", "phone")
		}
		if len(in.CategorySlugs) > 0 {
			refs, err := h.svc.LoadRefs(c)
			if err != nil {
				return nil, err
			}
			errs.codes("category_slugs", in.CategorySlugs, refs.Categories, "category")
		}
		out = in
	case 2:
		in := Step2{Lat: coordText(data, "lat"), Lng: coordText(data, "lng"), Address: text(data, "address"),
			About: longText(data, "about"), AgeRanges: strs(data, "age_ranges"), Amenities: strs(data, "amenities")}
		if (in.Lat == "") != (in.Lng == "") {
			errs.add("lat", "coordinates")
		}
		refs, err := h.svc.LoadRefs(c)
		if err != nil {
			return nil, err
		}
		errs.codes("age_ranges", in.AgeRanges, refs.AgeRanges, "code")
		errs.codes("amenities", in.Amenities, refs.Amenities, "code")
		days := map[int]bool{}
		for i := range elems(data, "weekly_hours") {
			p := "weekly_hours." + strconv.Itoa(i)
			hrs := Hours{Day: int(numOr(data, p+".day", -1)), Open: text(data, p+".open"), Close: text(data, p+".close")}
			if days[hrs.Day] {
				errs.add(p+".day", "day_distinct")
			}
			days[hrs.Day] = true
			if hrs.Close <= hrs.Open { // HH:MM compares as text
				errs.add(p+".close", "close_after_open")
			}
			in.WeeklyHours = append(in.WeeklyHours, hrs)
		}
		slices.SortFunc(in.WeeklyHours, func(a, b Hours) int { return a.Day - b.Day })
		out = in
	case 3:
		var in Step3
		for i := range elems(data, "services") {
			p := "services." + strconv.Itoa(i)
			sv := Offer{Kind: text(data, p+".kind"), Name: text(data, p+".name"), DurationMinutes: num(data, p+".duration_minutes"),
				AgeMinMonths: num(data, p+".age_min_months"), AgeMaxMonths: num(data, p+".age_max_months"),
				Capacity: num(data, p+".capacity"), PriceToman: num(data, p+".price_toman"),
				PackageSessions: num(data, p+".package_sessions"), PackageValidityDays: num(data, p+".package_validity_days")}
			if s := text(data, p+".description"); s != "" {
				sv.Description = &s
			}
			if s := text(data, p+".format"); s != "" {
				sv.Format = &s
			}
			if sv.Kind != "package" { // package fields only on packages
				sv.PackageSessions, sv.PackageValidityDays = nil, nil
			}
			if sv.AgeMinMonths != nil && sv.AgeMaxMonths != nil && *sv.AgeMaxMonths < *sv.AgeMinMonths {
				errs.add(p+".age_max_months", "age_order")
			}
			in.Services = append(in.Services, sv)
		}
		out = in
	default:
		out = Step4{BookingMode: text(data, "booking_mode"), TermsAccepted: truthy(data, "terms_accepted")}
	}
	if errs.bag != nil {
		return nil, failValidation(locale, errs.bag)
	}
	return out, nil
}

// mobile reports whether s is an Iranian mobile number 09xxxxxxxxx (the contact number shown to reviewers).
func mobile(s string) bool {
	if len(s) != 11 || !strings.HasPrefix(s, "09") {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// coordText is a validated coordinate as decimal(9,6) text ("" = none).
func coordText(data any, key string) string {
	v, ok := phpval.Get(data, key)
	if !ok || v == nil || phpval.ToString(v) == "" {
		return ""
	}
	return strconv.FormatFloat(phpval.ToFloat(v), 'f', 6, 64)
}
