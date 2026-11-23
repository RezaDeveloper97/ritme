import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import { stepUpErrorKey } from './stepUpErrors';

function failed(status: number): ApiError {
  return new ApiError(`Request failed with status code ${status}`, {
    response: { status, data: { success: false, message: 'x' }, headers: {} },
    sentToken: null,
  });
}

describe('stepUpErrorKey (T-M12-06)', () => {
  it('maps the verify failures to their own copy', () => {
    expect(stepUpErrorKey(failed(422), 'verify')).toBe('wrongCode');
    expect(stepUpErrorKey(failed(400), 'verify')).toBe('expired');
    expect(stepUpErrorKey(failed(429), 'verify')).toBe('tooMany');
    expect(stepUpErrorKey(failed(500), 'verify')).toBe('error');
    expect(stepUpErrorKey(new Error('network'), 'verify')).toBe('error');
  });

  it('a throttled send asks to wait; any other send failure is the generic error', () => {
    expect(stepUpErrorKey(failed(429), 'send')).toBe('wait');
    expect(stepUpErrorKey(failed(422), 'send')).toBe('error');
    expect(stepUpErrorKey(failed(401), 'send')).toBe('error');
  });
});
