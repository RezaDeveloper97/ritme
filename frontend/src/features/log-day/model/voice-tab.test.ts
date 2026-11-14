import { describe, expect, it } from 'vitest';

import { voiceTabHidden } from './voice-tab';

describe('voiceTabHidden', () => {
  it('hides the locked voice tab for teens only (B-N3-14b)', () => {
    expect(voiceTabHidden('teen')).toBe(true);
        expect(voiceTabHidden('cycle')).toBe(false);
    expect(voiceTabHidden(undefined)).toBe(false);
  });
});
