// Package todo is the to-do list «کارهای من» (bloom B-N6-08, D-66; artboards nbl_/nbd_Todo_* in f-tools): tasks with a
// category (shopping / work / personal / health), an optional due date / time and reminder; list items under a task
// (the shopping list «لیست خرید»); the board grouped into today / tomorrow / later plus the recently done; and one
// cycle suggestion derived from the cycle engine's predicted period («پریودت ۵ روز دیگر است؛ «خرید نوار بهداشتی» را
// اضافه کنم؟»), whose copy is admin-editable (message_contents todo_suggestion).
//
// Everything is scoped by the signed-in user (a foreign id is a uniform 404). Titles may carry health data: they are
// never logged.
package todo

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Categories, in chip order (Todo_Home «همه · خرید · کار · شخصی · سلامت»).
const (
	CategoryShopping = "shopping"
	CategoryWork     = "work"
	CategoryPersonal = "personal"
	CategoryHealth   = "health"
)

// Categories is every category in display order.
var Categories = []string{CategoryShopping, CategoryWork, CategoryPersonal, CategoryHealth}

// Board groups.
const (
	GroupToday    = "today"
	GroupTomorrow = "tomorrow"
	GroupLater    = "later"
)

// Limits.
const (
	MaxTitleLen = 120
	MaxNoteLen  = 500
	// MaxTasks caps the rows of one user (open + done); MaxItems the items of one list.
	MaxTasks = 1000
	MaxItems = 100
	// DoneDays is how long a ticked task stays on the board (in «انجام‌شده»); older ones stay stored, unlisted.
	DoneDays = 30
	// MaxBoardTasks caps one board read (open tasks + the done of the last DoneDays).
	MaxBoardTasks = 600
	// MaxDueYears bounds a due date: at most a year back, five years ahead.
	MaxDueYears = 5
)

// Task is one task with the counts of its list items.
type Task struct {
	ID            uint64
	Title         string
	Note          string
	Category      string
	DueDate       civildate.Date // zero = no date
	DueTime       string         // HH:MM, "" = no time
	Remind        bool
	DoneAt        time.Time // zero = open
	SuggestionKey string
	ItemsTotal    int
	ItemsOpen     int
	CreatedAt     time.Time
}

// Done reports whether the task is ticked.
func (t Task) Done() bool { return !t.DoneAt.IsZero() }

// Item is one list item.
type Item struct {
	ID      uint64
	TaskID  uint64
	Title   string
	DueDate civildate.Date // zero = no date
	DoneAt  time.Time      // zero = open
}

// Done reports whether the item is ticked.
func (it Item) Done() bool { return !it.DoneAt.IsZero() }

func iso(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return jsonx.ISO8601(t.In(civildate.Tehran))
}

func dateOrNil(d civildate.Date) any {
	if d.IsZero() {
		return nil
	}
	return d.String()
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// TaskJSON is one task. overdue = open with a due date before today; list = the item counts (null without items).
func TaskJSON(t Task, today civildate.Date) *jsonx.OrderedMap {
	var list any
	if t.ItemsTotal > 0 {
		list = jsonx.Obj("total", t.ItemsTotal, "open", t.ItemsOpen)
	}
	return jsonx.Obj(
		"id", t.ID,
		"title", t.Title,
		"note", strOrNil(t.Note),
		"category", t.Category,
		"due_date", dateOrNil(t.DueDate),
		"due_time", strOrNil(t.DueTime),
		"remind", t.Remind,
		"done", t.Done(),
		"done_at", iso(t.DoneAt),
		"overdue", !t.Done() && !t.DueDate.IsZero() && t.DueDate.Before(today),
		"suggestion_key", strOrNil(t.SuggestionKey),
		"list", list,
		"created_at", iso(t.CreatedAt),
	)
}

// TasksJSON is a list of tasks.
func TasksJSON(ts []Task, today civildate.Date) []any {
	out := make([]any, 0, len(ts))
	for _, t := range ts {
		out = append(out, TaskJSON(t, today))
	}
	return out
}

// ItemJSON is one list item.
func ItemJSON(it Item) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", it.ID,
		"task_id", it.TaskID,
		"title", it.Title,
		"due_date", dateOrNil(it.DueDate),
		"done", it.Done(),
		"done_at", iso(it.DoneAt),
	)
}

// ItemsJSON is a list of items.
func ItemsJSON(items []Item) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, ItemJSON(it))
	}
	return out
}
