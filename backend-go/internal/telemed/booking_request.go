package telemed

import (
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// StartsAtFormat is the wall-clock format of starts_at in requests (Tehran time, the slot's date + time).
const StartsAtFormat = "2006-01-02 15:04"

func parseStart(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(StartsAtFormat, s, civildate.Tehran)
	return t, err == nil
}

func uintOf(data phpval.Map, key string) uint64 {
	n, _ := strconv.ParseUint(str(data, key), 10, 64)
	return n
}

var bookKeys = []string{"doctor_id", "mode", "starts_at", "for_whom", "child_id", "patient_name", "reason", "note", "share"}

// ValidateBook is POST /telemed/bookings {doctor_id, mode, starts_at: "Y-m-d H:i", for_whom: self|child|other,
// child_id (child), patient_name (other), reason?: catalog code, note?: ≤ 1000, share?: [scope…]}. reasons are the
// active telemed_visit_reasons codes.
func ValidateBook(body phpval.Map, locale string, now time.Time, reasons []string) (BookInput, error) {
	data := pick(body, bookKeys)
	rules := validation.Rules{
		validation.F("doctor_id", "required", "integer", "min:1"),
		validation.F("mode", "required", "string", validation.In(Modes...)),
		validation.F("starts_at", "required", "date_format:Y-m-d H:i"),
		validation.F("for_whom", "required", "string", validation.In(ForWhom...)),
		validation.F("child_id", "required_if:for_whom,"+ForChild, "nullable", "integer", "min:1"),
		validation.F("patient_name", "required_if:for_whom,"+ForOther, "nullable", "string", "max:"+strconv.Itoa(MaxPatientNameLen)),
		validation.F("reason", "nullable", "string", validation.In(reasons...)),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxBookingNoteLen)),
		validation.F("share", "nullable", "array"),
		validation.F("share.*", "required", "string", validation.In(Scopes...)),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return BookInput{}, err
	}
	in := BookInput{
		DoctorID: uintOf(data, "doctor_id"), Mode: str(data, "mode"), ForWhom: str(data, "for_whom"),
		Reason: str(data, "reason"), Note: str(data, "note"),
	}
	in.StartsAt, _ = parseStart(str(data, "starts_at"))
	switch in.ForWhom {
	case ForChild:
		in.ChildID = uintOf(data, "child_id")
	case ForOther:
		in.PatientName = str(data, "patient_name")
	}
	seen := map[string]bool{}
	if v, ok := data.Get("share"); ok {
		_, items := phpval.Entries(v)
		for _, it := range items {
			s := phpval.ToString(it)
			if !seen[s] {
				seen[s] = true
			}
		}
	}
	for _, s := range Scopes { // Scopes order, no duplicates
		if seen[s] {
			in.Share = append(in.Share, s)
		}
	}
	return in, nil
}

var verifyKeys = []string{"reference", "authority"}

// ValidateVerify is POST /telemed/bookings/verify {reference, authority}.
func ValidateVerify(body phpval.Map, locale string, now time.Time) (reference, authority string, err error) {
	data := pick(body, verifyKeys)
	rules := validation.Rules{
		validation.F("reference", "required", "string", "max:24"),
		validation.F("authority", "required", "string", "max:191"),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return "", "", err
	}
	return str(data, "reference"), str(data, "authority"), nil
}

// ValidateReschedule is POST /telemed/bookings/{id}/reschedule {starts_at: "Y-m-d H:i"}.
func ValidateReschedule(body phpval.Map, locale string, now time.Time) (time.Time, error) {
	data := pick(body, []string{"starts_at"})
	rules := validation.Rules{validation.F("starts_at", "required", "date_format:Y-m-d H:i")}
	if err := validate(data, rules, locale, now); err != nil {
		return time.Time{}, err
	}
	t, _ := parseStart(str(data, "starts_at"))
	return t, nil
}

// ValidateConsent is PUT /telemed/bookings/{id}/consent {cycle_summary?, bbt_lh?, assistant_summaries?: boolean}:
// only the scopes sent change (an empty body changes nothing).
func ValidateConsent(body phpval.Map, locale string, now time.Time) (map[string]bool, error) {
	data := pick(body, Scopes)
	rules := validation.Rules{}
	for _, s := range Scopes {
		rules = append(rules, validation.F(s, "nullable", "boolean"))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, s := range Scopes {
		if v, ok := data.Get(s); ok && v != nil {
			out[s] = truthy(data, s)
		}
	}
	return out, nil
}

// ValidateQuote is GET /telemed/bookings/quote?doctor_id=&mode=.
func ValidateQuote(query phpval.Map, locale string, now time.Time) (uint64, string, error) {
	data := pick(query, []string{"doctor_id", "mode"})
	rules := validation.Rules{
		validation.F("doctor_id", "required", "integer", "min:1"),
		validation.F("mode", "nullable", "string", validation.In(Modes...)),
	}
	if err := validate(data, rules, locale, now); err != nil {
		return 0, "", err
	}
	return uintOf(data, "doctor_id"), str(data, "mode"), nil
}

// ValidateListScope is GET /telemed/bookings?scope=upcoming|past|all (default upcoming).
func ValidateListScope(query phpval.Map, locale string, now time.Time) (string, error) {
	data := pick(query, []string{"scope"})
	rules := validation.Rules{validation.F("scope", "nullable", "string", validation.In(ListScopes...))}
	if err := validate(data, rules, locale, now); err != nil {
		return "", err
	}
	if s := str(data, "scope"); s != "" {
		return s, nil
	}
	return ScopeUpcoming, nil
}
