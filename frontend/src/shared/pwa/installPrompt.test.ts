import { describe, expect, it } from 'vitest';

import { hidesInstallPrompt } from './InstallPrompt';

describe('install prompt visibility', () => {
  it('is hidden on the public doctor view only (B-N6-04b)', () => {
    expect(hidesInstallPrompt('/shared/report/abc')).toBe(true);
    expect(hidesInstallPrompt('/shared')).toBe(true);
    expect(hidesInstallPrompt('/sharedx')).toBe(false);
    expect(hidesInstallPrompt('/record/export')).toBe(false);
    expect(hidesInstallPrompt('/')).toBe(false);
  });
});
