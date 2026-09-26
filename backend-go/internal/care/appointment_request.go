package care

import (
	"fmt"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// appointmentFields are the request keys an appointment accepts (anything else is ignored;
// status changes only through /cancel).
var appointmentFields = []string{
	"kind", "with", "specialty", "topic", "title", "scheduled_at", "location",
	"remind_before", "add_to_calendar", "notes", "prep", "is_active",
	"care_item_key", "stage", "result_note",
}

// scheduledAtFormats are the accepted scheduled_at formats (Tehran wall-clock).
var scheduledAtFormats = []string{"Y-m-d H:i:s", "Y-m-d H:i"}

// maxPrepItems caps the prep checklist.
const maxPrepItems = 20

// appointmentRules validate the full appointment (a POST body, or a PUT body merged over
// the stored appointment).
func appointmentRules() validation.Rules {
	F := validation.F
	return validation.Rules{
		F("kind", "required", validation.In(AppointmentKinds...)),
		F("with", "nullable", "string", "max:255"),
		F("specialty", "nullable", "string", "max:255"),
		F("topic", "required", validation.In(AppointmentTopics...)),
		F("title", "nullable", "string", "max:255"),
		F("scheduled_at", "required", "date_format:"+strings.Join(scheduledAtFormats, ",")),
		F("location", "nullable", "string", "max:255"),
		F("remind_before", "sometimes", validation.In(RemindBefore...)),
		F("add_to_calendar", "sometimes", "boolean"),
		F("notes", "nullable", "string", "max:2000"),
		F("prep", "nullable", "array", fmt.Sprintf("max:%d", maxPrepItems)),
		F("prep.*", "array"),
		F("prep.*.text", "required", "string", "max:255"),
		F("prep.*.done", "sometimes", "boolean"),
		F("is_active", "sometimes", "boolean"),
		F("care_item_key", "nullable", "string", "max:64", "regex:/^[A-Za-z0-9_-]+$/"),
		F("stage", "nullable", validation.In(VisitStages...)),
		F("result_note", "nullable", "string", "max:2000"),
	}
}

// appointmentInput is a validated appointment.
type appointmentInput struct {
	Title       string
	Notes       *string
	ScheduledAt time.Time
	IsActive    bool
	Meta        AppointmentMeta
}

// pickAppointment copies the known appointment keys of in over base (a POST: an empty base;
// a PUT: the stored appointment).
func pickAppointment(base, in phpval.Map) phpval.Map {
	for _, k := range appointmentFields {
		if v, ok := in.Get(k); ok {
			base.Set(k, v)
		}
	}
	return base
}

// storedAppointment is the stored appointment as request data, the base a PUT merges over.
// The stored title is carried only when it is not the topic fallback, so changing the
// topic of an untitled appointment re-derives the title.
func storedAppointment(a Appointment, locale string) phpval.Map {
	data := phpval.NewMap()
	prep := make([]any, 0, len(a.Meta.Prep))
	for _, p := range a.Meta.Prep {
		item := phpval.NewMap()
		item.Set("id", p.ID)
		item.Set("text", p.Text)
		item.Set("done", p.Done)
		prep = append(prep, item)
	}
	data.Set("kind", a.Meta.Kind)
	data.Set("with", ptrValue(a.Meta.With))
	data.Set("specialty", ptrValue(a.Meta.Specialty))
	data.Set("topic", a.Meta.Topic)
	if label, ok := Label("topics", a.Meta.Topic, locale); !ok || label != a.Row.Title {
		data.Set("title", a.Row.Title)
	}
	if a.Row.ScheduledAt.Valid {
		data.Set("scheduled_at", a.Row.ScheduledAt.Time.In(civildate.Tehran).Format(wallClock))
	}
	data.Set("location", ptrValue(a.Meta.Location))
	data.Set("remind_before", a.Meta.RemindBefore)
	data.Set("add_to_calendar", a.Meta.AddToCalendar)
	data.Set("notes", nullString(a.Row.Notes))
	data.Set("prep", prep)
	data.Set("is_active", a.Row.IsActive)
	data.Set("care_item_key", ptrValue(a.Meta.CareItemKey))
	data.Set("stage", ptrValue(a.Meta.Stage))
	data.Set("result_note", ptrValue(a.Meta.ResultNote))
	return data
}

// validateAppointment runs the rules over data and builds the input. known is the stored
// prep list (nil on POST) whose ids are kept; status is carried over (new: scheduled).
func validateAppointment(locale string, data phpval.Map, now time.Time, known []PrepItem, status string) (appointmentInput, error) {
	v := validation.Make(lang.Default(), locale, data, appointmentRules(),
		validation.Now(now),
		validation.Attributes(attributesOf("appointment_attributes", locale)...))
	if v.Fails() {
		return appointmentInput{}, failValidation(locale, v.ErrorBag())
	}
	get := func(k string) any { x, _ := data.Get(k); return x }
	optional := func(k string) *string {
		x := get(k)
		if x == nil {
			return nil
		}
		s := strings.TrimSpace(phpval.ToString(x))
		if s == "" {
			return nil
		}
		return &s
	}

	in := appointmentInput{IsActive: true}
	in.Meta = AppointmentMeta{
		V: MetaVersion, Kind: phpval.ToString(get("kind")), With: optional("with"),
		Specialty: optional("specialty"), Topic: phpval.ToString(get("topic")), Location: optional("location"),
		RemindBefore: DefaultRemindBefore, Prep: []PrepItem{}, Status: status,
	}
	if x, ok := data.Get("remind_before"); ok {
		in.Meta.RemindBefore = phpval.ToString(x)
	}
	if x, ok := data.Get("add_to_calendar"); ok {
		in.Meta.AddToCalendar = phpval.Truthy(x)
	}
	if x, ok := data.Get("is_active"); ok {
		in.IsActive = phpval.Truthy(x)
	}
	in.Notes = optional("notes")
	in.Meta.CareItemKey, in.Meta.Stage, in.Meta.ResultNote = optional("care_item_key"), optional("stage"), optional("result_note")
	if t := optional("title"); t != nil {
		in.Title = *t
	} else if label, ok := Label("topics", in.Meta.Topic, locale); ok {
		in.Title = label
	} else {
		in.Title = in.Meta.Topic
	}

	_, items := phpval.Entries(get("prep"))
	prep := make([]PrepItem, 0, len(items))
	for _, it := range items {
		text := strings.TrimSpace(phpval.ToString(mustGet(it, "text")))
		if text == "" {
			continue
		}
		p := PrepItem{Text: text, Done: phpval.Truthy(mustGet(it, "done"))}
		if id := mustGet(it, "id"); id != nil {
			p.ID = phpval.ToString(id)
		}
		prep = append(prep, p)
	}
	in.Meta.Prep = AssignPrepIDs(prep, known)

	at, err := parseScheduledAt(phpval.ToString(get("scheduled_at")))
	if err != nil {
		return appointmentInput{}, err
	}
	in.ScheduledAt = at
	return in, nil
}

func mustGet(v any, key string) any {
	x, _ := phpval.Get(v, key)
	return x
}

// parseScheduledAt reads a validated scheduled_at as Tehran wall-clock.
func parseScheduledAt(s string) (time.Time, error) {
	for _, layout := range []string{wallClock, "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, s, civildate.Tehran); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("care: scheduled_at %q: unparseable", s)
}

// prepRules validate PATCH /prep/{itemId}.
func prepRules() validation.Rules {
	return validation.Rules{validation.F("done", "required", "boolean")}
}
