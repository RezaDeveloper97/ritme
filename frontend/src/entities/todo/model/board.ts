import type { IconName, Tone } from '@/shared/ui';

import type { TodoBoard, TodoCategory, TodoFilter, TodoGroup, TodoTask } from './types';

/**
 * Look of each category (nbl_Todo_Home chips: خرید amber, کار turquoise,
 * شخصی bloom, سلامت rose) — tones only, never a hex (CLAUDE.md §10.2).
 */
export const CATEGORY_LOOK: Record<TodoCategory, { tone: Tone; icon: IconName }> = {
  shopping: { tone: 'warm', icon: 'bag' },
  work: { tone: 'data', icon: 'briefcase' },
  personal: { tone: 'bloom', icon: 'user' },
  health: { tone: 'danger', icon: 'heart' },
};

/** The board groups with only the tasks of `filter`. */
export function filterGroups(groups: readonly TodoGroup[], filter: TodoFilter): TodoGroup[] {
  if (filter === 'all') return groups.map((g) => ({ ...g, tasks: [...g.tasks] }));
  return groups.map((g) => ({ ...g, tasks: g.tasks.filter((t) => t.category === filter) }));
}

/** «۲/۶»: ticked vs all tasks of the visible groups. */
export function groupsProgress(groups: readonly TodoGroup[]): { done: number; total: number } {
  let done = 0;
  let total = 0;
  for (const g of groups) {
    for (const t of g.tasks) {
      total += 1;
      if (t.done) done += 1;
    }
  }
  return { done, total };
}

/** Whether a board has nothing at all (no task, nothing done recently) — the first-run empty state. */
export function isFreshBoard(board: TodoBoard): boolean {
  return board.done.length === 0 && board.groups.every((g) => g.tasks.length === 0);
}

/** A shopping task or any task with items opens its list (Todo_List); the others open the edit sheet. */
export function opensList(task: TodoTask): boolean {
  return task.category === 'shopping' || task.list !== null;
}

/** Calendar distance in days from `fromIso` to `toIso` (both `YYYY-MM-DD`). */
export function daysBetween(fromIso: string, toIso: string): number {
  const a = Date.UTC(+fromIso.slice(0, 4), +fromIso.slice(5, 7) - 1, +fromIso.slice(8, 10));
  const b = Date.UTC(+toIso.slice(0, 4), +toIso.slice(5, 7) - 1, +toIso.slice(8, 10));
  return Math.round((b - a) / 86_400_000);
}

/** How an item's due date reads: «امروز», «فردا», «قبل از ۱۷ مهر», or the past date. */
export type ItemDueKind = 'today' | 'tomorrow' | 'before' | 'past';

export function itemDueKind(dueDate: string | null, todayIso: string): ItemDueKind | null {
  if (!dueDate) return null;
  const d = daysBetween(todayIso, dueDate);
  if (d === 0) return 'today';
  if (d === 1) return 'tomorrow';
  return d > 1 ? 'before' : 'past';
}

/** When a done item was ticked: «امروز», «دیروز», or an older date. */
export function doneKind(doneAt: string | null, todayIso: string): 'today' | 'yesterday' | 'date' | null {
  if (!doneAt) return null;
  const d = daysBetween(doneAt.slice(0, 10), todayIso);
  if (d <= 0) return 'today';
  return d === 1 ? 'yesterday' : 'date';
}
