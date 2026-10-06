package todo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
	"github.com/ritme/backend-go/internal/todo/store"
)

// Errors mapped to 404 / 422 by the handlers. A foreign id is the same 404 as a missing one.
var (
	ErrNotFound           = errors.New("todo: task not found")
	ErrItemNotFound       = errors.New("todo: item not found")
	ErrSuggestionNotFound = errors.New("todo: suggestion not available")
	ErrTooManyTasks       = errors.New("todo: task limit reached")
	ErrTooManyItems       = errors.New("todo: item limit reached")
)

// Service is the to-do data access. Every query is scoped by user id.
type Service struct {
	db    *sql.DB
	q     *store.Queries
	cycle *cycleservice.Service
	prefs notifications.Getter
}

// NewService wires the service on db.
func NewService(db *sql.DB) *Service {
	return &Service{db: db, q: store.New(db), cycle: cycleservice.New(db, nil), prefs: profilestore.New(db)}
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func nullDate(d civildate.Date) civildate.NullDate {
	return civildate.NullDate{Date: d, Valid: !d.IsZero()}
}

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: !t.IsZero()} }

func dbTime(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }

func timeOf(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func taskOf(r store.TodoTask, total, open int64) Task {
	var due civildate.Date
	if r.DueDate.Valid {
		due = r.DueDate.Date
	}
	return Task{
		ID: r.ID, Title: r.Title, Note: r.Note.String, Category: r.Category, DueDate: due, DueTime: r.DueTime.String,
		Remind: r.Remind, DoneAt: timeOf(r.DoneAt), SuggestionKey: r.SuggestionKey.String,
		ItemsTotal: int(total), ItemsOpen: int(open), CreatedAt: timeOf(r.CreatedAt),
	}
}

func itemOf(r store.TodoItem) Item {
	var due civildate.Date
	if r.DueDate.Valid {
		due = r.DueDate.Date
	}
	return Item{ID: r.ID, TaskID: r.TaskID, Title: r.Title, DueDate: due, DoneAt: timeOf(r.DoneAt)}
}

// Board reads the open tasks and the ones ticked in the last DoneDays.
func (s *Service) Board(ctx context.Context, userID uint64, today civildate.Date) ([]Task, error) {
	rows, err := s.q.ListBoardTasks(ctx, store.ListBoardTasksParams{
		UserID: userID, DoneSince: nullTime(today.AddDays(-DoneDays).TehranMidnight()), Limit: MaxBoardTasks,
	})
	if err != nil {
		return nil, fmt.Errorf("todo: board: %w", err)
	}
	out := make([]Task, 0, len(rows))
	for _, r := range rows {
		out = append(out, taskOf(store.TodoTask{
			ID: r.ID, UserID: r.UserID, Title: r.Title, Note: r.Note, Category: r.Category, DueDate: r.DueDate,
			DueTime: r.DueTime, Remind: r.Remind, DoneAt: r.DoneAt, SuggestionKey: r.SuggestionKey,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}, r.ItemsTotal, r.ItemsOpen))
	}
	return out, nil
}

// Task reads one task of the user with its item counts.
func (s *Service) Task(ctx context.Context, userID, id uint64) (Task, error) {
	return task(ctx, s.q, userID, id)
}

func task(ctx context.Context, q *store.Queries, userID, id uint64) (Task, error) {
	r, err := q.GetTask(ctx, store.GetTaskParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("todo: task: %w", err)
	}
	c, err := q.CountItems(ctx, store.CountItemsParams{TaskID: id, UserID: userID})
	if err != nil {
		return Task{}, fmt.Errorf("todo: item counts: %w", err)
	}
	return taskOf(r, c.Total, c.OpenCount), nil
}

// Items lists the items of one of the user's tasks.
func (s *Service) Items(ctx context.Context, userID, taskID uint64) ([]Item, error) {
	rows, err := s.q.ListItems(ctx, store.ListItemsParams{TaskID: taskID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("todo: items: %w", err)
	}
	out := make([]Item, 0, len(rows))
	for _, r := range rows {
		out = append(out, itemOf(r))
	}
	return out, nil
}

// Create stores a task.
func (s *Service) Create(ctx context.Context, userID uint64, in TaskInput, suggestionKey string, now time.Time) (Task, error) {
	return create(ctx, s.q, userID, in, suggestionKey, now)
}

func create(ctx context.Context, q *store.Queries, userID uint64, in TaskInput, suggestionKey string, now time.Time) (Task, error) {
	n, err := q.CountTasks(ctx, userID)
	if err != nil {
		return Task{}, fmt.Errorf("todo: count tasks: %w", err)
	}
	if n >= MaxTasks {
		return Task{}, ErrTooManyTasks
	}
	id, err := q.InsertTask(ctx, store.InsertTaskParams{
		UserID: userID, Title: in.Title, Note: nullStr(in.Note), Category: in.Category, DueDate: nullDate(in.DueDate),
		DueTime: nullStr(in.DueTime), Remind: in.Remind, SuggestionKey: nullStr(suggestionKey), Now: nullTime(dbTime(now)),
	})
	if err != nil {
		return Task{}, fmt.Errorf("todo: insert task: %w", err)
	}
	return task(ctx, q, userID, uint64(id)) //nolint:gosec // auto-increment id
}

// Update applies a validated (merged) input; ticking stamps done_at = now, unticking clears it.
func (s *Service) Update(ctx context.Context, userID uint64, cur Task, in TaskInput, now time.Time) (Task, error) {
	doneAt := cur.DoneAt
	if in.DoneSet {
		switch {
		case in.Done && doneAt.IsZero():
			doneAt = dbTime(now)
		case !in.Done:
			doneAt = time.Time{}
		}
	}
	n, err := s.q.UpdateTask(ctx, store.UpdateTaskParams{
		Title: in.Title, Note: nullStr(in.Note), Category: in.Category, DueDate: nullDate(in.DueDate),
		DueTime: nullStr(in.DueTime), Remind: in.Remind, DoneAt: nullTime(doneAt), Now: nullTime(dbTime(now)),
		ID: cur.ID, UserID: userID,
	})
	if err != nil {
		return Task{}, fmt.Errorf("todo: update task: %w", err)
	}
	if n == 0 {
		return Task{}, ErrNotFound
	}
	return s.Task(ctx, userID, cur.ID)
}

// Delete removes a task and its items.
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteTask(ctx, store.DeleteTaskParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("todo: delete task: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Item reads one item of one of the user's tasks.
func (s *Service) Item(ctx context.Context, userID, taskID, id uint64) (Item, error) {
	r, err := s.q.GetItem(ctx, store.GetItemParams{ID: id, TaskID: taskID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrItemNotFound
	}
	if err != nil {
		return Item{}, fmt.Errorf("todo: item: %w", err)
	}
	return itemOf(r), nil
}

// AddItem appends an item to one of the user's tasks (the task must exist: checked by the caller).
func (s *Service) AddItem(ctx context.Context, userID, taskID uint64, in ItemInput, now time.Time) (Item, error) {
	return addItem(ctx, s.q, userID, taskID, in, now)
}

func addItem(ctx context.Context, q *store.Queries, userID, taskID uint64, in ItemInput, now time.Time) (Item, error) {
	c, err := q.CountItems(ctx, store.CountItemsParams{TaskID: taskID, UserID: userID})
	if err != nil {
		return Item{}, fmt.Errorf("todo: item counts: %w", err)
	}
	if c.Total >= MaxItems {
		return Item{}, ErrTooManyItems
	}
	last, err := q.MaxItemSort(ctx, store.MaxItemSortParams{TaskID: taskID, UserID: userID})
	if err != nil {
		return Item{}, fmt.Errorf("todo: item order: %w", err)
	}
	id, err := q.InsertItem(ctx, store.InsertItemParams{
		TaskID: taskID, UserID: userID, Title: in.Title, DueDate: nullDate(in.DueDate),
		SortOrder: uint16(min(last+1, 65535)), Now: nullTime(dbTime(now)), //nolint:gosec // clamped
	})
	if err != nil {
		return Item{}, fmt.Errorf("todo: insert item: %w", err)
	}
	if err := q.TouchTask(ctx, store.TouchTaskParams{Now: nullTime(dbTime(now)), ID: taskID, UserID: userID}); err != nil {
		return Item{}, fmt.Errorf("todo: touch task: %w", err)
	}
	r, err := q.GetItem(ctx, store.GetItemParams{ID: uint64(id), TaskID: taskID, UserID: userID}) //nolint:gosec // auto-increment id
	if err != nil {
		return Item{}, fmt.Errorf("todo: read item: %w", err)
	}
	return itemOf(r), nil
}

// UpdateItem applies a validated (merged) item input.
func (s *Service) UpdateItem(ctx context.Context, userID uint64, cur Item, in ItemInput, now time.Time) (Item, error) {
	doneAt := cur.DoneAt
	if in.DoneSet {
		switch {
		case in.Done && doneAt.IsZero():
			doneAt = dbTime(now)
		case !in.Done:
			doneAt = time.Time{}
		}
	}
	n, err := s.q.UpdateItem(ctx, store.UpdateItemParams{
		Title: in.Title, DueDate: nullDate(in.DueDate), DoneAt: nullTime(doneAt), Now: nullTime(dbTime(now)),
		ID: cur.ID, TaskID: cur.TaskID, UserID: userID,
	})
	if err != nil {
		return Item{}, fmt.Errorf("todo: update item: %w", err)
	}
	if n == 0 {
		return Item{}, ErrItemNotFound
	}
	return s.Item(ctx, userID, cur.TaskID, cur.ID)
}

// DeleteItem removes an item.
func (s *Service) DeleteItem(ctx context.Context, userID, taskID, id uint64) error {
	n, err := s.q.DeleteItem(ctx, store.DeleteItemParams{ID: id, TaskID: taskID, UserID: userID})
	if err != nil {
		return fmt.Errorf("todo: delete item: %w", err)
	}
	if n == 0 {
		return ErrItemNotFound
	}
	return nil
}

// Suggestion is the cycle suggestion due today (ok=false: none), from the engine's predicted next period, unless the
// user already answered it for that period.
func (s *Service) Suggestion(ctx context.Context, userID uint64, today civildate.Date) (Suggestion, bool, error) {
	sn, err := s.cycle.Load(ctx, userID, today, today, today)
	if err != nil {
		return Suggestion{}, false, err
	}
	profile := sn.EngineProfile()
	m := metrics.Calculate(sn.Histories, profile)
	st := resolver.Resolve(sn.Histories, profile, today, today, m)
	sg, ok := PeriodSuggestion(st.PredictedNextPeriodStart, today, sn.LifeMode)
	if !ok {
		return Suggestion{}, false, nil
	}
	n, err := s.q.CountSuggestionEvents(ctx, store.CountSuggestionEventsParams{
		UserID: userID, SuggestionKey: sg.Key, RefDate: sg.PeriodStart,
	})
	if err != nil {
		return Suggestion{}, false, fmt.Errorf("todo: suggestion events: %w", err)
	}
	return sg, n == 0, nil
}

// SuggestionCopy reads the live suggestion copy.
func (s *Service) SuggestionCopy(ctx context.Context) (SuggestionCopy, error) {
	return LoadSuggestionCopy(ctx, s.q)
}

// Accepted is the result of accepting a suggestion: the task it created or the list it added the item to.
type Accepted struct {
	Task Task
	Item *Item // nil when a new task was created
}

// Accept records the answer and adds the suggestion: an item on the newest open shopping list (due the period start,
// «قبل از ۱۷ مهر»), else a new shopping task due the day before the period. One transaction; a second accept of the
// same period is ErrSuggestionNotFound (the answer row is the lock).
func (s *Service) Accept(ctx context.Context, userID uint64, sg Suggestion, cp map[string]string, today civildate.Date, now time.Time) (Accepted, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Accepted{}, fmt.Errorf("todo: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	n, err := q.InsertSuggestionEvent(ctx, store.InsertSuggestionEventParams{
		UserID: userID, SuggestionKey: sg.Key, RefDate: sg.PeriodStart, Action: ActionAccepted, Now: nullTime(dbTime(now)),
	})
	if err != nil {
		return Accepted{}, fmt.Errorf("todo: record suggestion: %w", err)
	}
	if n == 0 {
		return Accepted{}, ErrSuggestionNotFound
	}
	var out Accepted
	list, err := q.LatestOpenShoppingList(ctx, userID)
	switch {
	case err == nil:
		it, err := addItem(ctx, q, userID, list.ID, ItemInput{Title: cp["item_title"], DueDate: sg.PeriodStart}, now)
		if err != nil {
			return Accepted{}, err
		}
		out.Item = &it
		if out.Task, err = task(ctx, q, userID, list.ID); err != nil {
			return Accepted{}, err
		}
	case errors.Is(err, sql.ErrNoRows):
		in := TaskInput{Title: cp["task_title"], Category: CategoryShopping, DueDate: sg.TaskDue(today)}
		if out.Task, err = create(ctx, q, userID, in, sg.Key, now); err != nil {
			return Accepted{}, err
		}
	default:
		return Accepted{}, fmt.Errorf("todo: shopping list: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Accepted{}, fmt.Errorf("todo: commit: %w", err)
	}
	return out, nil
}

// Dismiss records that the user does not want this period's suggestion (idempotent).
func (s *Service) Dismiss(ctx context.Context, userID uint64, sg Suggestion, now time.Time) error {
	if _, err := s.q.InsertSuggestionEvent(ctx, store.InsertSuggestionEventParams{
		UserID: userID, SuggestionKey: sg.Key, RefDate: sg.PeriodStart, Action: ActionDismissed, Now: nullTime(dbTime(now)),
	}); err != nil {
		return fmt.Errorf("todo: record suggestion: %w", err)
	}
	return nil
}

// Notifications reads the user's notification preferences (reminder category state).
func (s *Service) Notifications(ctx context.Context, userID uint64) (notifications.Preferences, error) {
	return notifications.Load(ctx, s.prefs, userID)
}
