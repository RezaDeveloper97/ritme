import type { TodoBoard } from '@/entities/todo';

/** Open tasks due today (overdue ones are grouped under today by the API) — the «N کار امروز» row. */
export function openTodayTasks(board: TodoBoard | undefined): number {
  const today = board?.groups.find((g) => g.key === 'today');
  return today ? today.tasks.filter((task) => !task.done).length : 0;
}
