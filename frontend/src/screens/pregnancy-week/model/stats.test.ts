import { describe, expect, it } from 'vitest';

import { parseWeekStat } from './stats';

describe('parseWeekStat', () => {
  it('localizes averages, ranges and upper bounds', () => {
    expect(parseWeekStat('1.6', 'fa')).toEqual({ kind: 'approx', value: '۱٫۶' });
    expect(parseWeekStat('150-170', 'fa')).toEqual({ kind: 'range', from: '۱۵۰', to: '۱۷۰' });
    expect(parseWeekStat('<1', 'fa')).toEqual({ kind: 'less', value: '۱' });
    expect(parseWeekStat('2.3', 'en')).toEqual({ kind: 'approx', value: '2.3' });
    expect(parseWeekStat('about 3', 'en')).toEqual({ kind: 'text', value: 'about 3' });
  });
});
