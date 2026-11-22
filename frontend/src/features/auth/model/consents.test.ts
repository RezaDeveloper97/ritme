import { describe, expect, it } from 'vitest';

import { SIGNUP_CONSENT_VERSIONS, verifyOtpBody } from './consents';

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
