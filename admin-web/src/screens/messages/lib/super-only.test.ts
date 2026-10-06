import { describe, expect, it } from 'vitest';

import { canWriteGroup } from './super-only';

describe('canWriteGroup', () => {
  it('lets editors write everything but the clinical groups', () => {
    expect(canWriteGroup('pattern', 'editor')).toBe(true);
    expect(canWriteGroup('postpartum_alert', 'editor')).toBe(false);
    expect(canWriteGroup('postpartum_alert', 'super')).toBe(true);
    expect(canWriteGroup('pattern', 'editor', ['pattern'])).toBe(false);
    expect(canWriteGroup('postpartum_alert', undefined)).toBe(false);
  });
});
