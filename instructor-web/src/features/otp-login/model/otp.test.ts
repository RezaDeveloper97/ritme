import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import { retryAfterSeconds, sendErrorKey, verifyErrorKey } from './otp';
import { safeNext } from './safe-next';

const err = (status: number, extra: Partial<{ code: string; retryAfter: number }> = {}) =>
  new ApiError({ status, code: extra.code ?? 'http_error', retryAfter: extra.retryAfter });

describe('otp errors', () => {
  it('send-otp', () => {
    expect(sendErrorKey(err(422))).toBe('invalidMobile');
    expect(sendErrorKey(err(429))).toBe('wait');
    expect(sendErrorKey(err(0, { code: 'network_error' }))).toBe('network');
    expect(sendErrorKey(new Error('x'))).toBe('generic');
  });
  it('verify-otp', () => {
    expect(verifyErrorKey(err(422))).toBe('wrongCode');
    expect(verifyErrorKey(err(400))).toBe('expired');
    expect(verifyErrorKey(err(429))).toBe('tooManyAttempts');
    expect(verifyErrorKey(err(403))).toBe('blocked');
  });
  it('retry_after is clamped', () => {
    expect(retryAfterSeconds(err(429, { retryAfter: 42.2 }))).toBe(43);
    expect(retryAfterSeconds(err(429, { retryAfter: 500 }))).toBe(120);
    expect(retryAfterSeconds(err(429))).toBe(60);
  });
});

describe('safeNext', () => {
  it.each([
    [null, '/'],
    ['/groups', '/groups'],
    ['//evil.example', '/'],
    ['/\\evil.example', '/'],
    ['https://evil.example', '/'],
    ['/login?next=/x', '/'],
  ])('%s → %s', (raw, out) => {
    expect(safeNext(raw)).toBe(out);
  });
});
