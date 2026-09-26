import { describe, expect, it } from 'vitest';

import { fillSample, levelClass } from './level';

describe('alert rule preview', () => {
  it('fills known placeholders and keeps unknown ones', () => {
    expect(fillSample('{days} روز، {x}')).toBe('3 روز، {x}');
    expect(fillSample('هفته {week}', { week: '۱۲' })).toBe('هفته ۱۲');
  });
  it('falls back to info colours', () => {
    expect(levelClass('nope', 'chip')).toBe(levelClass('info', 'chip'));
  });
});
