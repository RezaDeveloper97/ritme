import { describe, expect, it } from 'vitest';

import { voiceTabHidden } from './voice-tab';

describe('voiceTabHidden', () => {
  it('hides the locked voice tab for teens only (B-N3-14b)', () => {
    expect(voiceTabHidden('teen', true)).toBe(true);
    expect(voiceTabHidden('teen', false)).toBe(false);
    expect(voiceTabHidden('cycle', true)).toBe(false);
    expect(voiceTabHidden(undefined, true)).toBe(false);
  });
});
