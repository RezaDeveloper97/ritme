import { describe, expect, it } from 'vitest';

import { relativeTime } from './relative';

describe('relativeTime', () => {
  const now = Date.parse('2026-10-08T12:00:00Z');
  it('uses hours, then days', () => {
    expect(relativeTime('2026-10-08T10:00:00Z', now)).toBe(new Intl.RelativeTimeFormat('fa', { numeric: 'auto' }).format(-2, 'hour'));
    expect(relativeTime('2026-10-07T12:00:00Z', now)).toBe(new Intl.RelativeTimeFormat('fa', { numeric: 'auto' }).format(-1, 'day'));
    expect(relativeTime(null, now)).toBe('');
  });
});
