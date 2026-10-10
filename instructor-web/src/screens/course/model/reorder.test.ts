import { describe, expect, it } from 'vitest';

import type { Lesson } from '@/entities/course';

import { moveLesson } from './reorder';

const l = (id: number, sort_order: number) => ({ id, sort_order }) as Lesson;

describe('moveLesson', () => {
  it('swaps neighbours and sends only changed orders', () => {
    expect(moveLesson([l(1, 1), l(2, 2), l(3, 3)], 3, -1)).toEqual([
      { id: 3, sort_order: 2 },
      { id: 2, sort_order: 3 },
    ]);
  });
  it('ignores moves past the edges', () => {
    expect(moveLesson([l(1, 1), l(2, 2)], 1, -1)).toEqual([]);
    expect(moveLesson([l(1, 1), l(2, 2)], 2, 1)).toEqual([]);
  });
  it('makes equal orders distinct', () => {
    expect(moveLesson([l(1, 4), l(2, 4)], 1, 1)).toEqual([{ id: 1, sort_order: 5 }]);
  });
});
