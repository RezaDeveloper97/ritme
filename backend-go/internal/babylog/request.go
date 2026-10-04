package babylog

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// dateTimeFormats are the accepted times (Tehran wall-clock), like /care/appointments and /ivf.
const dateTimeFormats = "date_format:Y-m-d H:i:s,Y-m-d H:i"

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// fieldFail is a 422 on one field with a babylog validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

func pick(body phpval.Map, keys []string) phpval.Map {
	data := phpval.NewMap()
	for _, k := range keys {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	return data
}

func validate(data phpval.Map, rules validation.Rules, locale string, now time.Time) error {
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
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

func optString(data phpval.Map, key string) sql.NullString {
	if s := str(data, key); s != "" {
		return sql.NullString{String: s, Valid: true}
	}
	return sql.NullString{}
}

func intOf(data phpval.Map, key string) (int, bool) {
	s := str(data, key)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return int(phpval.ToFloat(s)), true
	}
	return n, true
}

func dateTimeOf(data phpval.Map, key string) (time.Time, error) {
	s := str(data, key)
	for _, layout := range []string{time.DateTime, "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, s, civildate.Tehran); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("babylog: %s %q: unparseable", key, s)
}

func between(lo, hi int) string { return "between:" + strconv.Itoa(lo) + "," + strconv.Itoa(hi) }

var (
	noteRule   = validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxNoteLen))
	amountRule = validation.F("amount_ml", "nullable", "integer", between(0, MaxAmountMl))
)

// checkTime: not in the future (a minute of clock skew allowed), not before the birth day.
func checkTime(t time.Time, field string, birth civildate.Date, locale string, now time.Time) error {
	if t.After(now.Add(time.Minute)) {
		return fieldFail(locale, field, "future")
	}
	if t.Before(birth.TehranMidnight()) {
		return fieldFail(locale, field, "before_birth")
	}
	return nil
}

// StartFeedInput is POST /children/{id}/feeds/start.
type StartFeedInput struct{ Type, Side string }

func validateStartFeed(body phpval.Map, locale string, now time.Time) (StartFeedInput, error) {
	data := pick(body, []string{"type", "side"})
	if err := validate(data, validation.Rules{
		validation.F("type", "required", validation.In(FeedTypes...)),
		validation.F("side", "nullable", validation.In(Sides...)),
	}, locale, now); err != nil {
		return StartFeedInput{}, err
	}
	return StartFeedInput{Type: str(data, "type"), Side: str(data, "side")}, nil
}

// validateSide is POST /children/{id}/feeds/{fid}/side {side: left|right|null}.
func validateSide(body phpval.Map, locale string, now time.Time) (string, error) {
	data := pick(body, []string{"side"})
	if err := validate(data, validation.Rules{
		validation.F("side", "present", "nullable", validation.In(Sides...)),
	}, locale, now); err != nil {
		return "", err
	}
	return str(data, "side"), nil
}

// validateStopFeed is POST /children/{id}/feeds/{fid}/stop {amount_ml?, note?}.
func validateStopFeed(body phpval.Map, locale string, now time.Time) (StopInput, error) {
	data := pick(body, []string{"amount_ml", "note"})
	if err := validate(data, validation.Rules{amountRule, noteRule}, locale, now); err != nil {
		return StopInput{}, err
	}
	in := StopInput{Note: optString(data, "note")}
	if n, ok := intOf(data, "amount_ml"); ok {
		in.AmountMl = sql.NullInt16{Int16: int16(n), Valid: true} //nolint:gosec // ≤ MaxAmountMl
	}
	return in, nil
}

var feedKeys = []string{"type", "started_at", "left_minutes", "right_minutes", "duration_minutes", "side", "amount_ml", "note"}

// validateFeed is POST / PUT /children/{id}/feeds: an ended feed. Breast: left_minutes / right_minutes (at least one
// above 0) and the side it ended on (side); bottle / pump: duration_minutes and amount_ml.
func validateFeed(body phpval.Map, birth civildate.Date, locale string, now time.Time) (FeedInput, error) {
	F := validation.F
	minutes := between(0, MaxFeedMinutes)
	data := pick(body, feedKeys)
	if err := validate(data, validation.Rules{
		F("type", "required", validation.In(FeedTypes...)),
		F("started_at", "required", dateTimeFormats),
		F("left_minutes", "nullable", "integer", minutes),
		F("right_minutes", "nullable", "integer", minutes),
		F("duration_minutes", "nullable", "integer", minutes),
		F("side", "nullable", validation.In(Sides...)),
		amountRule, noteRule,
	}, locale, now); err != nil {
		return FeedInput{}, err
	}
	start, err := dateTimeOf(data, "started_at")
	if err != nil {
		return FeedInput{}, err
	}
	if err := checkTime(start, "started_at", birth, locale, now); err != nil {
		return FeedInput{}, err
	}
	in := FeedInput{Type: str(data, "type"), StartedAt: start, Note: optString(data, "note")}
	if in.Type == typeBreast {
		l, _ := intOf(data, "left_minutes")
		r, _ := intOf(data, "right_minutes")
		if l+r == 0 {
			return FeedInput{}, fieldFail(locale, "left_minutes", "minutes_required")
		}
		in.LeftSeconds, in.RightSeconds = uint32(l*60), uint32(r*60) //nolint:gosec // ≤ MaxFeedMinutes
		in.LastSide = optString(data, "side")
		if !in.LastSide.Valid {
			in.LastSide = sql.NullString{String: "left", Valid: true}
			if r > 0 {
				in.LastSide.String = "right"
			}
		}
		return in, nil
	}
	if d, ok := intOf(data, "duration_minutes"); ok {
		in.Duration = int64(d) * 60
	}
	if n, ok := intOf(data, "amount_ml"); ok {
		in.AmountMl = sql.NullInt16{Int16: int16(n), Valid: true} //nolint:gosec // ≤ MaxAmountMl
	}
	return in, nil
}

// validateNote is the optional {note} of a stop.
func validateNote(body phpval.Map, locale string, now time.Time) (sql.NullString, error) {
	data := pick(body, []string{"note"})
	if err := validate(data, validation.Rules{noteRule}, locale, now); err != nil {
		return sql.NullString{}, err
	}
	return optString(data, "note"), nil
}

// validateSleep is POST / PUT /children/{id}/sleeps: an ended sleep of at most MaxSleepHours.
func validateSleep(body phpval.Map, birth civildate.Date, locale string, now time.Time) (SleepInput, error) {
	F := validation.F
	data := pick(body, []string{"started_at", "ended_at", "note"})
	if err := validate(data, validation.Rules{
		F("started_at", "required", dateTimeFormats),
		F("ended_at", "required", dateTimeFormats),
		noteRule,
	}, locale, now); err != nil {
		return SleepInput{}, err
	}
	start, err := dateTimeOf(data, "started_at")
	if err != nil {
		return SleepInput{}, err
	}
	end, err := dateTimeOf(data, "ended_at")
	if err != nil {
		return SleepInput{}, err
	}
	if err := checkTime(start, "started_at", birth, locale, now); err != nil {
		return SleepInput{}, err
	}
	if err := checkTime(end, "ended_at", birth, locale, now); err != nil {
		return SleepInput{}, err
	}
	if !end.After(start) {
		return SleepInput{}, fieldFail(locale, "ended_at", "end_before_start")
	}
	if end.Sub(start) > MaxSleepHours*time.Hour {
		return SleepInput{}, fieldFail(locale, "ended_at", "too_long")
	}
	return SleepInput{StartedAt: start, EndedAt: end, Note: optString(data, "note")}, nil
}

// validateDiaper is POST / PUT /children/{id}/diapers {kind, changed_at? (default now), note?}.
func validateDiaper(body phpval.Map, birth civildate.Date, locale string, now time.Time) (DiaperInput, error) {
	F := validation.F
	data := pick(body, []string{"kind", "changed_at", "note"})
	if err := validate(data, validation.Rules{
		F("kind", "required", validation.In(DiaperKinds...)),
		F("changed_at", "nullable", dateTimeFormats),
		noteRule,
	}, locale, now); err != nil {
		return DiaperInput{}, err
	}
	in := DiaperInput{Kind: str(data, "kind"), ChangedAt: now, Note: optString(data, "note")}
	if str(data, "changed_at") != "" {
		t, err := dateTimeOf(data, "changed_at")
		if err != nil {
			return in, err
		}
		if err := checkTime(t, "changed_at", birth, locale, now); err != nil {
			return in, err
		}
		in.ChangedAt = t
	}
	return in, nil
}

// validateDay is ?date= (default today).
func validateDay(query phpval.Map, locale string, now time.Time) (civildate.Date, error) {
	data := pick(query, []string{"date"})
	if err := validate(data, validation.Rules{validation.F("date", "nullable", "date_format:Y-m-d")}, locale, now); err != nil {
		return civildate.Date{}, err
	}
	if s := str(data, "date"); s != "" {
		d, err := civildate.Parse(s)
		if err != nil {
			return civildate.Date{}, fmt.Errorf("babylog: date: %w", err)
		}
		return d, nil
	}
	return civildate.InTehran(now), nil
}

// validateDays is ?days= (1..MaxSummaryDays, default DefaultSummary).
func validateDays(query phpval.Map, locale string, now time.Time) (int, error) {
	data := pick(query, []string{"days"})
	if err := validate(data, validation.Rules{
		validation.F("days", "nullable", "integer", between(1, MaxSummaryDays)),
	}, locale, now); err != nil {
		return 0, err
	}
	if n, ok := intOf(data, "days"); ok {
		return n, nil
	}
	return DefaultSummary, nil
}
