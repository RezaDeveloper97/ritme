package loss

import (
	"database/sql"
	"fmt"
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

// wallClock is the Tehran wall-clock datetime of reminders.scheduled_at; visitFormats are the accepted visit_at inputs.
const wallClock = "2006-01-02 15:04:05"

var visitFormats = []string{"Y-m-d H:i", "Y-m-d H:i:s"}

// RecordInput is POST /loss {type?, occurred_on?, notify_companion?}. HasType / HasDate tell a same-day correction
// which answers were given (an absent key keeps the earlier answer; an explicit null clears it).
type RecordInput struct {
	Type            string
	OccurredOn      civildate.NullDate
	NotifyCompanion bool
	HasType         bool
	HasDate         bool
}

// FollowupInput is PUT /loss/followup: nil = key absent (unchanged); a present null clears.
type FollowupInput struct {
	BleedingStopped *bool
	BetaNextOn      *civildate.NullDate
	BetaNegative    *bool
	VisitAt         *sql.NullTime
}

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

// fieldFail is a 422 on one field with a loss validation line.
func fieldFail(locale, field, key string) error {
	return failValidation(locale, jsonx.Obj(field, []string{T("validation."+key, locale)}))
}

// validate runs rules over the picked keys of body; unknown keys are never read.
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

// boolOf is a validated boolean field (true, 1, "1", "true", "on", "yes").
func boolOf(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	s := strings.ToLower(phpval.ToString(v))
	return s == "1" || s == "true" || s == "on" || s == "yes"
}

func parseDate(v any, field string) (civildate.Date, error) {
	d, err := civildate.Parse(phpval.ToString(v))
	if err != nil {
		return civildate.Date{}, fmt.Errorf("loss: %s: %w", field, err)
	}
	return d, nil
}

// validateRecord is POST /loss: type defaults to unspecified («ترجیح می‌دهم نگویم»), occurred_on is optional
// (approximate, not in the future, within the past year).
func validateRecord(body phpval.Map, locale string, now time.Time) (RecordInput, error) {
	F := validation.F
	data, err := validate(body, []string{"type", "occurred_on", "notify_companion"}, validation.Rules{
		F("type", "nullable", "string", validation.In(Types...)),
		F("occurred_on", "nullable", "date_format:Y-m-d"),
		F("notify_companion", "sometimes", "boolean"),
	}, locale, now)
	if err != nil {
		return RecordInput{}, err
	}
	in := RecordInput{Type: TypeUnspecified}
	_, in.HasType = data.Get("type")
	_, in.HasDate = data.Get("occurred_on")
	if v, _ := data.Get("type"); v != nil && phpval.ToString(v) != "" {
		in.Type = phpval.ToString(v)
	}
	if v, _ := data.Get("occurred_on"); v != nil {
		d, err := parseDate(v, "occurred_on")
		if err != nil {
			return RecordInput{}, err
		}
		today := civildate.InTehran(now)
		switch {
		case d.After(today):
			return RecordInput{}, fieldFail(locale, "occurred_on", "date_future")
		case d.Before(today.AddDays(-MaxPastDays)):
			return RecordInput{}, fieldFail(locale, "occurred_on", "date_too_old")
		}
		in.OccurredOn = civildate.NullDate{Date: d, Valid: true}
	}
	if v, ok := data.Get("notify_companion"); ok {
		in.NotifyCompanion = boolOf(v)
	}
	return in, nil
}

// validateFollowup is PUT /loss/followup: every key optional, null clears.
func validateFollowup(body phpval.Map, locale string, now time.Time) (FollowupInput, error) {
	F := validation.F
	data, err := validate(body, []string{"bleeding_stopped", "beta_next_on", "beta_negative", "visit_at"}, validation.Rules{
		F("bleeding_stopped", "sometimes", "boolean"),
		F("beta_next_on", "nullable", "date_format:Y-m-d"),
		F("beta_negative", "sometimes", "boolean"),
		F("visit_at", "nullable", "date_format:"+strings.Join(visitFormats, ",")),
	}, locale, now)
	if err != nil {
		return FollowupInput{}, err
	}
	var in FollowupInput
	if v, ok := data.Get("bleeding_stopped"); ok {
		b := boolOf(v)
		in.BleedingStopped = &b
	}
	if v, ok := data.Get("beta_negative"); ok {
		b := boolOf(v)
		in.BetaNegative = &b
	}
	if v, ok := data.Get("beta_next_on"); ok {
		nd := civildate.NullDate{}
		if v != nil {
			d, err := parseDate(v, "beta_next_on")
			if err != nil {
				return FollowupInput{}, err
			}
			if d.Before(civildate.InTehran(now)) {
				return FollowupInput{}, fieldFail(locale, "beta_next_on", "date_past")
			}
			nd = civildate.NullDate{Date: d, Valid: true}
		}
		in.BetaNextOn = &nd
	}
	if v, ok := data.Get("visit_at"); ok {
		nt := sql.NullTime{}
		if v != nil {
			at, err := parseWallClock(phpval.ToString(v))
			if err != nil {
				return FollowupInput{}, err
			}
			if at.Before(now) {
				return FollowupInput{}, fieldFail(locale, "visit_at", "datetime_past")
			}
			nt = sql.NullTime{Time: at, Valid: true}
		}
		in.VisitAt = &nt
	}
	return in, nil
}

// parseWallClock reads a validated visit_at as Tehran wall-clock.
func parseWallClock(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04", wallClock} {
		if t, err := time.ParseInLocation(layout, s, civildate.Tehran); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("loss: visit_at: unparsable")
}

// validateMood is POST /loss/moods {mood, date?} (default today; not in the future, within MoodDays).
func validateMood(body phpval.Map, locale string, now time.Time) (string, civildate.Date, error) {
	F := validation.F
	data, err := validate(body, []string{"mood", "date"}, validation.Rules{
		F("mood", "required", "string", validation.In(Moods...)),
		F("date", "nullable", "date_format:Y-m-d"),
	}, locale, now)
	if err != nil {
		return "", civildate.Date{}, err
	}
	m, _ := data.Get("mood")
	today := civildate.InTehran(now)
	day := today
	if v, _ := data.Get("date"); v != nil {
		if day, err = parseDate(v, "date"); err != nil {
			return "", civildate.Date{}, err
		}
		switch {
		case day.After(today):
			return "", civildate.Date{}, fieldFail(locale, "date", "date_future")
		case day.Before(today.AddDays(-(MoodDays - 1))):
			return "", civildate.Date{}, fieldFail(locale, "date", "date_too_old")
		}
	}
	return phpval.ToString(m), day, nil
}

// validateNote is PUT /loss/note {note}: a string up to MaxNoteLen; null or blank clears it.
func validateNote(body phpval.Map, locale string, now time.Time) (string, error) {
	F := validation.F
	data, err := validate(body, []string{"note"}, validation.Rules{
		F("note", "present", "nullable", "string", fmt.Sprintf("max:%d", MaxNoteLen)),
	}, locale, now)
	if err != nil {
		return "", err
	}
	v, _ := data.Get("note")
	if v == nil {
		return "", nil
	}
	return strings.TrimSpace(phpval.ToString(v)), nil
}

// validateNextStep is PUT /loss/next-step {choice}.
func validateNextStep(body phpval.Map, locale string, now time.Time) (string, error) {
	F := validation.F
	data, err := validate(body, []string{"choice"}, validation.Rules{
		F("choice", "required", "string", validation.In(NextSteps...)),
	}, locale, now)
	if err != nil {
		return "", err
	}
	v, _ := data.Get("choice")
	return phpval.ToString(v), nil
}
