// Package checklist is the layette checklist and the period-supply reminder of the internal shop (CB-SHOP-04, boards
// nbd_Shop_Checklist / nbd_Shop_Beauty): «her own list, not a sales funnel; the supply reminder is off by default».
//
//   - Checklist: the admin template (shop_layette_items, grouped) merged with the user's own state per item
//     (shop_user_checklist_items) plus her custom rows. Every item can be marked have / in cart / on the way /
//     not needed / later; «not needed» items leave the total. The due date comes from the active pregnancy through
//     DueDateSource. A companion with a live `pregnancy` grant may read the list (family share), never change it.
//   - Supply reminder: an opt-in (shop_supply_reminders, off by default). When on, the only health input is the
//     predicted next period start (PeriodPredictor); a one-off care reminder (`reminders`, type custom,
//     meta.source = shop_supply) is set lead_days before it at ReminderHour. Its title is discreet (no period
//     wording) and nothing reaches a seller. Off → no reminder row exists.
//
// Like internal/shop this package imports no health package and no query here joins a health table: the predictor,
// the due date and the companion check are injected (routes_shop_checklist.go). Nothing here logs health data.
package checklist

import (
	"slices"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Checklist groups (shop_layette_items.group_key / shop_user_checklist_items.group_key), in board order.
const (
	GroupClothing = "clothing"
	GroupSleep    = "sleep"
	GroupFeeding  = "feeding"
	GroupBath     = "bath"
	GroupOuting   = "outing"
)

// Groups are the known groups in display order. A template row in another group still lists (after these).
var Groups = []string{GroupClothing, GroupSleep, GroupFeeding, GroupBath, GroupOuting}

// Item states (shop_user_checklist_items.state).
const (
	StateTodo      = "todo"
	StateHave      = "have"
	StateInCart    = "in_cart"
	StateOnTheWay  = "on_the_way"
	StateNotNeeded = "not_needed"
	StateLater     = "later"
)

// States are every state, in display order.
var States = []string{StateTodo, StateHave, StateInCart, StateOnTheWay, StateNotNeeded, StateLater}

// Limits.
const (
	MaxCustomItems = 200
	MaxTitleLen    = 120
	MaxQuantity    = 99
)

// Item is one checklist line: a template item (with the user's state, todo when she has none) or a custom row.
type Item struct {
	// ID is the user's row id (0 for a template item she never touched).
	ID uint64
	// LayetteItemID is the template id (0 for a custom row).
	LayetteItemID uint64
	Group         string
	Title         string
	Note          string
	SuggestedQty  int
	IsEssential   bool
	IsFirstDays   bool
	CategorySlug  string
	State         string
	Quantity      int
	SortOrder     int32
}

// Custom reports whether the line is the user's own.
func (i Item) Custom() bool { return i.LayetteItemID == 0 }

// Group is one group of the list with its counts.
type Group struct {
	Key   string
	Items []Item
	// Ready is the number of items she has; Total excludes «not needed».
	Ready, Total int
}

// Summary is the whole list's counts.
type Summary struct {
	Ready, Total, NotNeeded, Percent int
}

// View is the assembled list.
type View struct {
	DueDate   civildate.Date // zero: no active pregnancy / no dating
	WeeksLeft int            // -1 when DueDate is zero
	Summary   Summary
	Groups    []Group
}

// Assemble groups items (template order kept) in Groups order, then unknown groups in key order, and counts them.
func Assemble(items []Item) ([]Group, Summary) {
	by := map[string][]Item{}
	var extra []string
	for _, it := range items {
		if _, seen := by[it.Group]; !seen && !slices.Contains(Groups, it.Group) {
			extra = append(extra, it.Group)
		}
		by[it.Group] = append(by[it.Group], it)
	}
	slices.Sort(extra)
	var sum Summary
	out := []Group{}
	for _, key := range append(slices.Clone(Groups), extra...) {
		list, ok := by[key]
		if !ok {
			continue
		}
		g := Group{Key: key, Items: list}
		for _, it := range list {
			switch it.State {
			case StateNotNeeded:
				sum.NotNeeded++
				continue
			case StateHave:
				g.Ready++
			}
			g.Total++
		}
		sum.Ready += g.Ready
		sum.Total += g.Total
		out = append(out, g)
	}
	if sum.Total > 0 {
		sum.Percent = sum.Ready * 100 / sum.Total
	}
	return out, sum
}

// WeeksLeft is the whole weeks from today to due (0 once due; -1 for an unknown due date).
func WeeksLeft(due, today civildate.Date) int {
	if due.IsZero() {
		return -1
	}
	return max(today.DiffDays(due), 0) / 7
}
