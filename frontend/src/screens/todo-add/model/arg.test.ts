import { describe, expect, it } from 'vitest';

import { parseTodoAddArg } from './arg';

describe('parseTodoAddArg', () => {
  it('reads a task id, a category or nothing', () => {
    expect(parseTodoAddArg('42')).toEqual({ taskId: 42, category: null });
    expect(parseTodoAddArg('shopping')).toEqual({ taskId: null, category: 'shopping' });
    expect(parseTodoAddArg(undefined)).toEqual({ taskId: null, category: null });
    expect(parseTodoAddArg('0')).toEqual({ taskId: null, category: null });
    expect(parseTodoAddArg('food')).toEqual({ taskId: null, category: null });
    expect(parseTodoAddArg('1;drop')).toEqual({ taskId: null, category: null });
  });
});
