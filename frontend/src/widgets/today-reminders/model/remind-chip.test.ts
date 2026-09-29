import { describe, expect, it } from 'vitest';

import { showRemindChip } from './remind-chip';

describe('showRemindChip', () => {
  it('hides the reminder lead time when the bell is off', () => {
    expect(showRemindChip({ isActive: false })).toBe(false);
    expect(showRemindChip({ isActive: true })).toBe(true);
  });
});
