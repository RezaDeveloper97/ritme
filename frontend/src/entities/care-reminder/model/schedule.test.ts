import { describe, expect, it } from 'vitest';

import { medicationSchedule } from './schedule';

describe('medicationSchedule', () => {
  it('words the weekdays as every day, every other day or a count', () => {
    expect(medicationSchedule([0, 2, 4, 6])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([1, 3, 5])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([5, 1, 3, 3])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([0, 2])).toEqual({ kind: 'daysPerWeek', count: 2 });
    expect(medicationSchedule([0, 3])).toEqual({ kind: 'daysPerWeek', count: 2 });
    expect(medicationSchedule([0, 1, 2, 3, 4, 5, 6])).toEqual({ kind: 'everyDay' });
    expect(medicationSchedule([])).toEqual({ kind: 'everyDay' });
  });
});
