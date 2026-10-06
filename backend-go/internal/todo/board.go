package todo

import (
	"cmp"
	"slices"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Board is GET /todo's grouping (Todo_Home): «امروز» (due today, overdue, or ticked today while due earlier),
// «فردا», «بعداً» (later or no date), and «انجام‌شده» — tasks ticked before today (within DoneDays), newest first.
// Inside a group open tasks come first, then by due date / time (none last) and id; a task ticked today stays in
// its group, struck through, so the day's progress «۲/۶» reads off the groups.
type Board struct {
	Today, Tomorrow, Later, Done []Task
}

// GroupOf is the board group of a task that is open or ticked today.
func GroupOf(t Task, today civildate.Date) string {
	switch {
	case t.DueDate.IsZero():
		return GroupLater
	case !t.DueDate.After(today):
		return GroupToday
	case t.DueDate == today.AddDays(1):
		return GroupTomorrow
	}
	return GroupLater
}

// BuildBoard groups tasks (open + recently done) for today.
func BuildBoard(tasks []Task, today civildate.Date) Board {
	var b Board
	for _, t := range tasks {
		if t.Done() && civildate.InTehran(t.DoneAt).Before(today) {
			b.Done = append(b.Done, t)
			continue
		}
		switch GroupOf(t, today) {
		case GroupToday:
			b.Today = append(b.Today, t)
		case GroupTomorrow:
			b.Tomorrow = append(b.Tomorrow, t)
		default:
			b.Later = append(b.Later, t)
		}
	}
	for _, g := range [][]Task{b.Today, b.Tomorrow, b.Later} {
		slices.SortStableFunc(g, compareOpenFirst)
	}
	slices.SortStableFunc(b.Done, func(a, c Task) int {
		if x := c.DoneAt.Compare(a.DoneAt); x != 0 {
			return x
		}
		return cmp.Compare(c.ID, a.ID)
	})
	return b
}

func compareOpenFirst(a, b Task) int {
	if a.Done() != b.Done() {
		if a.Done() {
			return 1
		}
		return -1
	}
	if x := compareNoneLast(a.DueDate.IsZero(), b.DueDate.IsZero(), a.DueDate.Compare(b.DueDate)); x != 0 {
		return x
	}
	if x := compareNoneLast(a.DueTime == "", b.DueTime == "", cmp.Compare(a.DueTime, b.DueTime)); x != 0 {
		return x
	}
	return cmp.Compare(a.ID, b.ID)
}

// compareNoneLast orders a missing value after any present one, else by cmpPresent.
func compareNoneLast(aNone, bNone bool, cmpPresent int) int {
	switch {
	case aNone && bNone:
		return 0
	case aNone:
		return 1
	case bNone:
		return -1
	}
	return cmpPresent
}
