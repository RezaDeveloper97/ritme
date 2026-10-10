import type { Lesson } from '@/entities/course';

/**
 * Move one lesson up / down inside its group and return the sort_order updates to send (only the changed ones).
 * The group is renumbered from its smallest order so equal orders become distinct.
 */
export function moveLesson(group: Lesson[], id: number, dir: -1 | 1): { id: number; sort_order: number }[] {
  const i = group.findIndex((l) => l.id === id);
  const j = i + dir;
  if (i < 0 || j < 0 || j >= group.length) return [];
  const next = [...group];
  [next[i], next[j]] = [next[j] as Lesson, next[i] as Lesson];
  const base = Math.max(1, Math.min(...group.map((l) => l.sort_order)));
  return next
    .map((l, k) => ({ id: l.id, sort_order: base + k, before: l.sort_order }))
    .filter((u) => u.sort_order !== u.before)
    .map(({ id: lid, sort_order }) => ({ id: lid, sort_order }));
}
