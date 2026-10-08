import { describe, expect, it } from 'vitest';

import type { TodoBoard, TodoTask } from '@/entities/todo';

import { openTodayTasks } from './todo-count';

const task = (id: number, done = false) => ({ id, done }) as unknown as TodoTask;
const board = (groups: TodoBoard['groups']) =>
  ({ date: '2026-10-07', suggestion: null, groups, done: [], remindersEnabled: true }) as TodoBoard;

describe('openTodayTasks (CB-NAV-03 «برای امروز»)', () => {
  it('counts the open tasks of the today group only', () => {
    const b = board([
      { key: 'today', date: '2026-10-07', tasks: [task(1), task(2, true), task(3)] },
      { key: 'tomorrow', date: '2026-10-08', tasks: [task(4)] },
    ]);
    expect(openTodayTasks(b)).toBe(2);
  });

  it('is 0 without a board or a today group', () => {
    expect(openTodayTasks(undefined)).toBe(0);
    expect(openTodayTasks(board([{ key: 'later', date: null, tasks: [task(1)] }]))).toBe(0);
  });
});
