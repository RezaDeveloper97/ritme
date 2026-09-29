import { describe, expect, it } from 'vitest';

import { fillSample, levelClass } from './level';

describe('alert rule preview', () => {
  it('fills known placeholders and keeps unknown ones', () => {
    expect(fillSample('{days} days, {x}', 'en')).toBe('3 days, {x}');
    expect(fillSample('هفته {week}', 'fa', { week: 'دوازده' })).toBe('هفته دوازده');
  });
  it('writes the sample numbers in the digits of the text language', () => {
    expect(fillSample('در هفتهٔ {week}، {days} روز', 'fa')).toBe('در هفتهٔ ۱۲، ۳ روز');
    expect(fillSample('Week {week}', 'en')).toBe('Week 12');
  });
  it('falls back to info colours', () => {
    expect(levelClass('nope', 'chip')).toBe(levelClass('info', 'chip'));
  });
});
