/*
 * The to-do list «کارهای من» (B-N6-08 on `/api/v1/todo*`, backend-go internal/todo).
 * Dates are `YYYY-MM-DD` (Tehran), times `HH:MM`; display goes through
 * `shared/lib/date` (Jalali for fa).
 */

export const TODO_CATEGORIES = ['shopping', 'work', 'personal', 'health'] as const;
export type TodoCategory = (typeof TODO_CATEGORIES)[number];

/** A chip of the board filter: every category or one. */
export type TodoFilter = 'all' | TodoCategory;

export const TODO_GROUPS = ['today', 'tomorrow', 'later'] as const;
export type TodoGroupKey = (typeof TODO_GROUPS)[number];

/** Server limits mirrored for the form (internal/todo/model.go). */
export const TODO_LIMITS = { title: 120, note: 500 } as const;

export interface TodoTask {
  id: number;
  title: string;
  note: string | null;
  category: TodoCategory;
  dueDate: string | null;
  dueTime: string | null;
  remind: boolean;
  done: boolean;
  doneAt: string | null;
  overdue: boolean;
  suggestionKey: string | null;
  /** Item counts of the list; null when the task has no items. */
  list: { total: number; open: number } | null;
}

export interface TodoItem {
  id: number;
  taskId: number;
  title: string;
  dueDate: string | null;
  done: boolean;
  doneAt: string | null;
}

export interface TodoGroup {
  key: TodoGroupKey;
  date: string | null;
  tasks: TodoTask[];
}

/** The cycle suggestion («پریودت ۵ روز دیگر است؛ …»); its copy is admin content, rendered as plain text. */
export interface TodoSuggestion {
  key: string;
  prompt: string;
  action: string;
  days: number;
}

export interface TodoBoard {
  date: string;
  suggestion: TodoSuggestion | null;
  groups: TodoGroup[];
  done: TodoTask[];
  /** Whether the reminder notification category is on. */
  remindersEnabled: boolean;
}

export interface TodoTaskDetail {
  task: TodoTask;
  items: TodoItem[];
}

/** Body of POST /todo/tasks (and the full form state of PUT). */
export interface TodoTaskInput {
  title: string;
  note: string | null;
  category: TodoCategory;
  dueDate: string | null;
  dueTime: string | null;
  remind: boolean;
}
