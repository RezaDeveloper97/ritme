import { describe, expect, it } from 'vitest';

import { markDoneSheetArg, monthsSince } from './view';

describe('checkup detail view', () => {
  it('counts whole months since the last visit', () => {
    const now = new Date(2026, 8, 26);
    expect(monthsSince(null, now)).toBeNull();
    expect(monthsSince('2026-03-20', now)).toBe(6);
    expect(monthsSince('2026-09-30', now)).toBe(0);
  });
  it('builds the mark-done sheet arg', () => {
    expect(markDoneSheetArg(4)).toBe('4');
    expect(markDoneSheetArg(4, 9)).toBe('4-9');
  });
});
