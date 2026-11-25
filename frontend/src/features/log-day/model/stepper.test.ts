import { describe, expect, it } from 'vitest';

import { stepOnGrid } from './stepper';

describe('stepOnGrid', () => {
  it('moves a full step from a grid value', () => {
    expect(stepOnGrid(23.5, 0.5, 1)).toBe(24);
    expect(stepOnGrid(24, 0.5, -1)).toBe(23.5);
    expect(stepOnGrid(36.5, 0.1, 1)).toBe(36.6);
    expect(stepOnGrid(36.6, 0.1, -1)).toBe(36.5);
  });

  it('snaps an off-grid value to the next grid line in the pressed direction', () => {
    expect(stepOnGrid(23.3, 0.5, 1)).toBe(23.5);
    expect(stepOnGrid(23.3, 0.5, -1)).toBe(23);
    expect(stepOnGrid(36.47, 0.1, 1)).toBe(36.5);
    expect(stepOnGrid(36.47, 0.1, -1)).toBe(36.4);
  });
});
