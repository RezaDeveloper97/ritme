package todo

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Error codes of the 404 / 422 bodies.
const (
	ErrorCodeNotFound           = "todo_not_found"
	ErrorCodeItemNotFound       = "todo_item_not_found"
	ErrorCodeSuggestionNotFound = "todo_suggestion_not_found"
)

// Handlers are the /api/v1/todo actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// fail maps the service errors to their bodies.
func fail(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", ErrorCodeNotFound)
	case errors.Is(err, ErrItemNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.item_not_found", locale), "error_code", ErrorCodeItemNotFound)
	case errors.Is(err, ErrSuggestionNotFound):
		return httpx.Fail(fiber.StatusNotFound, T("messages.suggestion_not_found", locale), "error_code",
			ErrorCodeSuggestionNotFound)
	case errors.Is(err, ErrTooManyTasks):
		return fieldFail(locale, "title", "too_many_tasks")
	case errors.Is(err, ErrTooManyItems):
		return fieldFail(locale, "title", "too_many_items")
	}
	return err
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

// Board is GET /todo (nbl_Todo_Home): today / tomorrow / later groups, the recently done, the cycle suggestion (null
// when none is due) and the reminder category state.
func (h *Handlers) Board(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	today := civildate.InTehran(now)
	tasks, err := h.svc.Board(c, userID, today)
	if err != nil {
		return err
	}
	b := BuildBoard(tasks, today)
	suggestion, err := h.suggestionJSON(c, userID, today, locale)
	if err != nil {
		return err
	}
	prefs, err := h.svc.Notifications(c, userID)
	if err != nil {
		return err
	}
	groups := []any{
		jsonx.Obj("key", GroupToday, "date", today.String(), "tasks", TasksJSON(b.Today, today)),
		jsonx.Obj("key", GroupTomorrow, "date", today.AddDays(1).String(), "tasks", TasksJSON(b.Tomorrow, today)),
		jsonx.Obj("key", GroupLater, "date", nil, "tasks", TasksJSON(b.Later, today)),
	}
	return httpx.OK(c, jsonx.Obj(
		"date", today.String(),
		"categories", Categories,
		"suggestion", suggestion,
		"groups", groups,
		"done", TasksJSON(b.Done, today),
		"notifications", jsonx.Obj("category", string(ReminderCategory), "enabled", prefs.Enabled(ReminderCategory)),
	))
}

func (h *Handlers) suggestionJSON(c fiber.Ctx, userID uint64, today civildate.Date, locale string) (any, error) {
	sg, ok, err := h.svc.Suggestion(c, userID, today)
	if err != nil || !ok {
		return nil, err
	}
	cp, err := h.svc.SuggestionCopy(c)
	if err != nil {
		return nil, err
	}
	return SuggestionJSON(sg, cp.Pick(sg.Key, locale, i18n.LanguagesOf(c).DefaultCode()), locale), nil
}

// Store is POST /todo/tasks (201 {task}).
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := ValidateTask(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	t, err := h.svc.Create(c, userID, in, "", now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.Created(c, jsonx.Obj("task", TaskJSON(t, civildate.InTehran(now))), T("messages.saved", locale))
}

// Show is GET /todo/tasks/{id}: the task with its list items (Todo_List).
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	t, err := h.svc.Task(c, userID, id)
	if err != nil {
		return fail(err, locale)
	}
	items, err := h.svc.Items(c, userID, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("task", TaskJSON(t, civildate.InTehran(now)), "items", ItemsJSON(items)))
}

// Update is PUT /todo/tasks/{id}: a partial update (missing fields keep their value); `done` ticks / unticks.
func (h *Handlers) Update(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	cur, err := h.svc.Task(c, userID, id)
	if err != nil {
		return fail(err, locale)
	}
	in, err := ValidateTask(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	t, err := h.svc.Update(c, userID, cur, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("task", TaskJSON(t, civildate.InTehran(now))), T("messages.updated", locale))
}

// Destroy is DELETE /todo/tasks/{id} (its items go with it).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	if err := h.svc.Delete(c, userID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// StoreItem is POST /todo/tasks/{id}/items (201 {item, task}).
func (h *Handlers) StoreItem(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := idParam(c, "id")
	if !ok {
		return fail(ErrNotFound, locale)
	}
	if _, err := h.svc.Task(c, userID, id); err != nil {
		return fail(err, locale)
	}
	in, err := ValidateItem(validation.Input(c), nil, locale, now)
	if err != nil {
		return err
	}
	it, err := h.svc.AddItem(c, userID, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return h.itemBody(c, userID, it, now, locale, fiber.StatusCreated, "messages.item_saved")
}

// itemBody is {item, task} (the task carries the fresh list counts).
func (h *Handlers) itemBody(c fiber.Ctx, userID uint64, it Item, now time.Time, locale string, status int, msg string) error {
	t, err := h.svc.Task(c, userID, it.TaskID)
	if err != nil {
		return fail(err, locale)
	}
	body := jsonx.Obj("item", ItemJSON(it), "task", TaskJSON(t, civildate.InTehran(now)))
	if status == fiber.StatusCreated {
		return httpx.Created(c, body, T(msg, locale))
	}
	return httpx.OK(c, body, T(msg, locale))
}

func (h *Handlers) currentItem(c fiber.Ctx, userID uint64, locale string) (Item, error) {
	taskID, ok1 := idParam(c, "id")
	id, ok2 := idParam(c, "item")
	if !ok1 || !ok2 {
		return Item{}, fail(ErrItemNotFound, locale)
	}
	it, err := h.svc.Item(c, userID, taskID, id)
	if err != nil {
		return Item{}, fail(err, locale)
	}
	return it, nil
}

// UpdateItem is PUT /todo/tasks/{id}/items/{item}: a partial update; `done` ticks / unticks ({item, task}).
func (h *Handlers) UpdateItem(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	cur, err := h.currentItem(c, userID, locale)
	if err != nil {
		return err
	}
	in, err := ValidateItem(validation.Input(c), &cur, locale, now)
	if err != nil {
		return err
	}
	it, err := h.svc.UpdateItem(c, userID, cur, in, now)
	if err != nil {
		return fail(err, locale)
	}
	return h.itemBody(c, userID, it, now, locale, fiber.StatusOK, "messages.updated")
}

// DestroyItem is DELETE /todo/tasks/{id}/items/{item} ({task} with the fresh counts).
func (h *Handlers) DestroyItem(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	cur, err := h.currentItem(c, userID, locale)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteItem(c, userID, cur.TaskID, cur.ID); err != nil {
		return fail(err, locale)
	}
	t, err := h.svc.Task(c, userID, cur.TaskID)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("task", TaskJSON(t, civildate.InTehran(now))), T("messages.deleted", locale))
}

// currentSuggestion is the suggestion of the {key} path parameter when it is the one due today, else 404.
func (h *Handlers) currentSuggestion(c fiber.Ctx, userID uint64, today civildate.Date, locale string) (Suggestion, error) {
	sg, ok, err := h.svc.Suggestion(c, userID, today)
	if err != nil {
		return Suggestion{}, err
	}
	if !ok || sg.Key != c.Params("key") {
		return Suggestion{}, fail(ErrSuggestionNotFound, locale)
	}
	return sg, nil
}

// AcceptSuggestion is POST /todo/suggestions/{key}/accept (201 {task, item}): the server recomputes the suggestion
// (the client never sends dates or titles), adds the item to the newest open shopping list or creates the task.
func (h *Handlers) AcceptSuggestion(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	today := civildate.InTehran(now)
	sg, err := h.currentSuggestion(c, userID, today, locale)
	if err != nil {
		return err
	}
	cp, err := h.svc.SuggestionCopy(c)
	if err != nil {
		return err
	}
	res, err := h.svc.Accept(c, userID, sg, cp.Pick(sg.Key, locale, i18n.LanguagesOf(c).DefaultCode()), today, now)
	if err != nil {
		return fail(err, locale)
	}
	var item any
	if res.Item != nil {
		item = ItemJSON(*res.Item)
	}
	return httpx.Created(c, jsonx.Obj("task", TaskJSON(res.Task, today), "item", item), T("messages.suggestion_added", locale))
}

// DismissSuggestion is POST /todo/suggestions/{key}/dismiss: not offered again for this predicted period.
func (h *Handlers) DismissSuggestion(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	sg, err := h.currentSuggestion(c, userID, civildate.InTehran(now), locale)
	if err != nil {
		return err
	}
	if err := h.svc.Dismiss(c, userID, sg, now); err != nil {
		return err
	}
	return httpx.OK(c, nil, T("messages.suggestion_dismissed", locale))
}
