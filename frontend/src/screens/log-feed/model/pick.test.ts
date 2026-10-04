import { describe, expect, it } from 'vitest';

import { defaultChildId } from './pick';

describe('defaultChildId', () => {
  it('prefers the first own child', () => {
    expect(defaultChildId([{ id: 3, role: 'shared' }, { id: 7, role: 'owner' }, { id: 9, role: 'owner' }])).toBe(7);
  });
  it('falls back to a shared child, then null', () => {
    expect(defaultChildId([{ id: 3, role: 'shared' }])).toBe(3);
    expect(defaultChildId([])).toBeNull();
  });
});
