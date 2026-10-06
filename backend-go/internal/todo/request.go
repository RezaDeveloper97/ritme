package todo

import (
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

// failValidation is the controller-style 422 {success:false, message, errors}.
func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

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

func has(data phpval.Map, key string) bool {
	_, ok := data.Get(key)
	return ok
}

func truthy(data phpval.Map, key string) bool {
	v, _ := data.Get(key)
	return phpval.Truthy(v)
}

func dateOf(s string) civildate.Date {
	if s == "" {
		return civildate.Date{}
	}
	d, err := civildate.Parse(s)
	if err != nil {
		return civildate.Date{}
	}
	return d
}

// presence is the first rules of a field: required on create, sometimes + required on a partial update.
func presence(partial bool) []any {
	if partial {
		return []any{"sometimes", "required"}
	}
	return []any{"required"}
}

func rule(base []any, more ...any) []any { return append(append([]any(nil), base...), more...) }

// TaskInput is a validated task (the full state after a create or a partial update).
type TaskInput struct {
	Title    string
	Note     string
	Category string
	DueDate  civildate.Date
	DueTime  string
	Remind   bool
	// Done is the requested state (only meaningful when DoneSet).
	Done, DoneSet bool
}

var taskKeys = []string{"title", "note", "category", "due_date", "due_time", "remind", "done"}

// ValidateTask is the body of POST /todo/tasks {title, category, note?, due_date?, due_time?, remind?} and of
// PUT /todo/tasks/{id} (every field optional, `done` too; the missing ones keep cur). A reminder needs a due date and
// time; a due date is at most a year back and MaxDueYears ahead.
func ValidateTask(body phpval.Map, cur *Task, locale string, now time.Time) (TaskInput, error) {
	data := pick(body, taskKeys)
	partial := cur != nil
	rules := validation.Rules{
		validation.F("title", rule(presence(partial), "string", "max:"+strconv.Itoa(MaxTitleLen))...),
		validation.F("note", "nullable", "string", "max:"+strconv.Itoa(MaxNoteLen)),
		validation.F("category", rule(presence(partial), validation.In(Categories...))...),
		validation.F("due_date", "nullable", "date_format:Y-m-d"),
		validation.F("due_time", "nullable", "date_format:H:i"),
		validation.F("remind", "nullable", "boolean"),
	}
	if partial {
		rules = append(rules, validation.F("done", "sometimes", "required", "boolean"))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return TaskInput{}, err
	}
	in := TaskInput{}
	if cur != nil {
		in = TaskInput{Title: cur.Title, Note: cur.Note, Category: cur.Category, DueDate: cur.DueDate, DueTime: cur.DueTime,
			Remind: cur.Remind}
	}
	if has(data, "title") {
		in.Title = str(data, "title")
	}
	if has(data, "note") {
		in.Note = str(data, "note")
	}
	if has(data, "category") {
		in.Category = str(data, "category")
	}
	if has(data, "due_date") {
		in.DueDate = dateOf(str(data, "due_date"))
	}
	if has(data, "due_time") {
		in.DueTime = str(data, "due_time")
	}
	if has(data, "remind") {
		in.Remind = truthy(data, "remind")
	}
	if partial && has(data, "done") {
		in.Done, in.DoneSet = truthy(data, "done"), true
	}
	if err := checkDue(in.DueDate, now, locale); err != nil {
		return TaskInput{}, err
	}
	if in.Remind && (in.DueDate.IsZero() || in.DueTime == "") {
		return TaskInput{}, fieldFail(locale, "remind", "remind_needs_time")
	}
	return in, nil
}

func checkDue(d civildate.Date, now time.Time, locale string) error {
	if d.IsZero() {
		return nil
	}
	today := civildate.InTehran(now)
	if d.Before(today.AddDays(-366)) || d.After(today.AddDays(366*MaxDueYears)) {
		return fieldFail(locale, "due_date", "due_out_of_range")
	}
	return nil
}

// ItemInput is a validated list item.
type ItemInput struct {
	Title         string
	DueDate       civildate.Date
	Done, DoneSet bool
}

// ValidateItem is the body of POST /todo/tasks/{id}/items {title, due_date?} and of PUT …/items/{item} (every field
// optional, `done` too; the missing ones keep cur).
func ValidateItem(body phpval.Map, cur *Item, locale string, now time.Time) (ItemInput, error) {
	data := pick(body, []string{"title", "due_date", "done"})
	partial := cur != nil
	rules := validation.Rules{
		validation.F("title", rule(presence(partial), "string", "max:"+strconv.Itoa(MaxTitleLen))...),
		validation.F("due_date", "nullable", "date_format:Y-m-d"),
	}
	if partial {
		rules = append(rules, validation.F("done", "sometimes", "required", "boolean"))
	}
	if err := validate(data, rules, locale, now); err != nil {
		return ItemInput{}, err
	}
	in := ItemInput{}
	if cur != nil {
		in = ItemInput{Title: cur.Title, DueDate: cur.DueDate}
	}
	if has(data, "title") {
		in.Title = str(data, "title")
	}
	if has(data, "due_date") {
		in.DueDate = dateOf(str(data, "due_date"))
	}
	if partial && has(data, "done") {
		in.Done, in.DoneSet = truthy(data, "done"), true
	}
	if err := checkDue(in.DueDate, now, locale); err != nil {
		return ItemInput{}, err
	}
	return in, nil
}
