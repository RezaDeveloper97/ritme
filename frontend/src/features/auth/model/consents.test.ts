import { describe, expect, it } from 'vitest';

import { pendingConsentsSchema, shouldShowPending, SIGNUP_CONSENT_VERSIONS, verifyOtpBody } from './consents';

describe('signup consents (T-M10-23)', () => {
  it('sends the version of every mandatory text shown', () => {
    expect(SIGNUP_CONSENT_VERSIONS).toEqual({ terms: 1, health_data_processing: 1 });
    expect(verifyOtpBody('09120000000', '1234', { ...SIGNUP_CONSENT_VERSIONS })).toEqual({
      mobile: '09120000000',
      code: '1234',
      consents: { terms: 1, health_data_processing: 1 },
    });
  });

  it('omits consents when the phone screen recorded none (existing account, H-18)', () => {
    const body = verifyOtpBody('09120000000', '1234', null);
    expect(body).toEqual({ mobile: '09120000000', code: '1234' });
    expect('consents' in body).toBe(false);
  });

  it('copies the consents, never shares the frozen object', () => {
    const body = verifyOtpBody('m', 'c', SIGNUP_CONSENT_VERSIONS);
    expect(body.consents).not.toBe(SIGNUP_CONSENT_VERSIONS);
  });
});

describe('pending consents', () => {
  it('parses the server texts and drops anything already accepted', () => {
    const parsed = pendingConsentsSchema.parse({
      consents: [
        { code: 'terms', version: 1, title: 'T', body: 'B', points: ['p'], needs_consent: true },
        { code: 'health_data_processing', version: 1, title: null, body: null, points: null },
        { code: 'old', version: 1, needs_consent: false },
      ],
    });
    expect(parsed).toEqual([
      { code: 'terms', version: 1, title: 'T', body: 'B', points: ['p'] },
      { code: 'health_data_processing', version: 1, title: '', body: '', points: [] },
    ]);
  });

  it('rejects a version that is not a positive integer', () => {
    expect(() => pendingConsentsSchema.parse({ consents: [{ code: 'terms', version: 0 }] })).toThrow();
  });

  it('shows the sheet only while something is pending and not put off', () => {
    const one = [{ code: 'terms', version: 1, title: '', body: '', points: [] }];
    expect(shouldShowPending(undefined, false)).toBe(false);
    expect(shouldShowPending([], false)).toBe(false);
    expect(shouldShowPending(one, false)).toBe(true);
    expect(shouldShowPending(one, true)).toBe(false);
  });
});
