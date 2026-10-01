import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import { plusDenialOf, upgradeHelps } from './denial';

function failed(status: number, data: unknown): ApiError {
  return new ApiError('failed', { response: { status, data, headers: {} }, sentToken: null });
}

describe('plusDenialOf', () => {
  it('maps a 402 locked feature', () => {
    const d = plusDenialOf(
      failed(402, { success: false, message: 'پلاس لازم است', error_code: 'plus_required', feature: 'plus.pdf_share', reason: 'locked', limit: null, resets_at: null }),
    );
    expect(d).toEqual({ kind: 'locked', feature: 'plus.pdf_share', limit: null, resetsAt: null, message: 'پلاس لازم است' });
    expect(d && upgradeHelps(d)).toBe(true);
  });

  it('maps a 402 spent free quota and a 429 spent Plus quota', () => {
    const quota = plusDenialOf(
      failed(402, { error_code: 'plus_required', feature: 'plus.assistant_unlimited', reason: 'quota', limit: 5, resets_at: '2026-11-01T00:00:00+03:30' }),
    );
    expect(quota).toMatchObject({ kind: 'quota', limit: 5, resetsAt: '2026-11-01T00:00:00+03:30' });

    const exhausted = plusDenialOf(
      failed(429, { error_code: 'plus_quota_exceeded', feature: 'plus.lab_ai', reason: 'quota', limit: 10, resets_at: '2026-11-01T00:00:00+03:30' }),
    );
    expect(exhausted).toMatchObject({ kind: 'exhausted', feature: 'plus.lab_ai', limit: 10 });
    expect(exhausted && upgradeHelps(exhausted)).toBe(false);
  });

  it('ignores every other failure', () => {
    expect(plusDenialOf(failed(429, { error_code: 'too_many_requests' }))).toBeNull();
    expect(plusDenialOf(failed(402, { error_code: 'other' }))).toBeNull();
    expect(plusDenialOf(failed(402, '<html>'))).toBeNull();
    expect(plusDenialOf(new Error('x'))).toBeNull();
    expect(plusDenialOf(new ApiError('offline', { code: 'network', sentToken: null }))).toBeNull();
  });
});
