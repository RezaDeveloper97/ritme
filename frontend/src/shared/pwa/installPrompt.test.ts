import { describe, expect, it } from 'vitest';

import { hidesInstallPrompt } from './InstallPrompt';

describe('install prompt visibility', () => {
  it('is hidden on the public doctor view only (B-N6-04b)', () => {
    expect(hidesInstallPrompt('/shared/report/abc')).toBe(true);
    expect(hidesInstallPrompt('/shared')).toBe(true);
    expect(hidesInstallPrompt('/sharedx')).toBe(false);
    expect(hidesInstallPrompt('/record/export')).toBe(false);
    expect(hidesInstallPrompt('/')).toBe(false);
    // CB-REC-05
    expect(hidesInstallPrompt('/d')).toBe(true);
    expect(hidesInstallPrompt('/d/abc')).toBe(true);
    expect(hidesInstallPrompt('/emergency/abc')).toBe(true);
    expect(hidesInstallPrompt('/places/12')).toBe(true); // CB-DIR-11
    expect(hidesInstallPrompt('/dashboard')).toBe(false);
    expect(hidesInstallPrompt('/record/emergency')).toBe(false);
  });
});
