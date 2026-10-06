import { TODO_CATEGORIES, type TodoCategory } from '@/entities/todo';

/** What `?sheet=todo-add&arg=` asks for: a task id to edit, a category to preset, or nothing. Ids only (§11). */
export interface TodoAddTarget {
  taskId: number | null;
  category: TodoCategory | null;
}

export function parseTodoAddArg(arg: string | undefined): TodoAddTarget {
  if (arg && /^\d{1,12}$/.test(arg) && Number(arg) > 0) return { taskId: Number(arg), category: null };
  const category = (TODO_CATEGORIES as readonly string[]).includes(arg ?? '') ? (arg as TodoCategory) : null;
  return { taskId: null, category };
}
