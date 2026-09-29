import { describe, expect, it } from 'vitest';

import { showRemindChip } from './remind-chip';

describe('showRemindChip', () => {
  it('hides the reminder lead time once the bell is known to be off', () => {
    expect(showRemindChip({ isActive: false })).toBe(false);
    expect(showRemindChip({ isActive: true })).toBe(true);
    expect(showRemindChip(undefined)).toBe(true);
  });
});
