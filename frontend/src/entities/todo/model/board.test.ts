import { describe, expect, it } from 'vitest';

import { boardSchema } from '../api/schema';
import { daysBetween, doneKind, filterGroups, groupsProgress, isFreshBoard, itemDueKind, opensList } from './board';
import type { TodoGroup, TodoTask } from './types';

const task = (over: Partial<TodoTask>): TodoTask => ({
  id: 1,
  title: 't',
  note: null,
  category: 'work',
  dueDate: null,
  dueTime: null,
  remind: false,
  done: false,
  doneAt: null,
  overdue: false,
  suggestionKey: null,
  list: null,
  ...over,
});

const groups: TodoGroup[] = [
  { key: 'today', date: '2026-10-06', tasks: [task({ id: 1 }), task({ id: 2, category: 'shopping', done: true })] },
  { key: 'tomorrow', date: '2026-10-07', tasks: [task({ id: 3, category: 'health' })] },
  { key: 'later', date: null, tasks: [] },
];

describe('todo board model', () => {
  it('filters by category and counts progress', () => {
    expect(groupsProgress(groups)).toEqual({ done: 1, total: 3 });
    const shopping = filterGroups(groups, 'shopping');
    expect(shopping.map((g) => g.tasks.length)).toEqual([1, 0, 0]);
    expect(groupsProgress(shopping)).toEqual({ done: 1, total: 1 });
    expect(filterGroups(groups, 'all')[0].tasks).toHaveLength(2);
  });

  it('knows a fresh board and which rows open a list', () => {
    expect(isFreshBoard({ date: '2026-10-06', suggestion: null, groups, done: [], remindersEnabled: true })).toBe(false);
    expect(isFreshBoard({ date: '2026-10-06', suggestion: null, groups: [{ key: 'today', date: null, tasks: [] }], done: [], remindersEnabled: true })).toBe(true);
    expect(opensList(task({ category: 'shopping' }))).toBe(true);
    expect(opensList(task({ list: { total: 1, open: 1 } }))).toBe(true);
    expect(opensList(task({}))).toBe(false);
  });

  it('labels item due dates and done dates', () => {
    expect(daysBetween('2026-10-06', '2026-10-09')).toBe(3);
    expect(daysBetween('2026-03-20', '2026-03-21')).toBe(1); // across the Nowruz / DST edge
    expect(itemDueKind(null, '2026-10-06')).toBeNull();
    expect(itemDueKind('2026-10-06', '2026-10-06')).toBe('today');
    expect(itemDueKind('2026-10-07', '2026-10-06')).toBe('tomorrow');
    expect(itemDueKind('2026-10-09', '2026-10-06')).toBe('before');
    expect(itemDueKind('2026-10-01', '2026-10-06')).toBe('past');
    expect(doneKind('2026-10-06T09:00:00+03:30', '2026-10-06')).toBe('today');
    expect(doneKind('2026-10-05T23:00:00+03:30', '2026-10-06')).toBe('yesterday');
    expect(doneKind('2026-10-01T09:00:00+03:30', '2026-10-06')).toBe('date');
  });

  it('parses the Go board golden shape and drops unknown categories', () => {
    const board = boardSchema.parse({
      date: '2026-09-23',
      categories: ['shopping', 'work', 'personal', 'health'],
      suggestion: { key: 'period_supplies', prompt: 'p', action: 'a', days: 3, period_start: '2026-09-26', category: 'shopping', task_title: 'x', item_title: 'y' },
      groups: [
        {
          key: 'today',
          date: '2026-09-23',
          tasks: [
            { id: 2, title: 'Fruit', note: null, category: 'shopping', due_date: '2026-09-23', due_time: null, remind: false, done: false, done_at: null, overdue: false, suggestion_key: null, list: { total: 2, open: 1 }, created_at: '2026-09-23T10:00:00+03:30' },
            { id: 9, title: 'Odd', category: 'gardening' },
          ],
        },
        { key: 'tomorrow', date: '2026-09-24', tasks: [] },
        { key: 'later', date: null, tasks: [] },
      ],
      done: [],
      notifications: { category: 'appointments', enabled: false },
    });
    expect(board.groups[0].tasks).toHaveLength(1);
    expect(board.groups[0].tasks[0].list).toEqual({ total: 2, open: 1 });
    expect(board.suggestion?.days).toBe(3);
    expect(board.remindersEnabled).toBe(false);
  });
});
