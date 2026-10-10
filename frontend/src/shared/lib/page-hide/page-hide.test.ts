import { describe, expect, it } from 'vitest';

import { retryAfterSeconds } from './index';

describe('retryAfterSeconds', () => {
  it('prefers the body, then the header, then the fallback', () => {
    expect(retryAfterSeconds({ retry_after: 900 }, { 'retry-after': '30' })).toBe(900);
    expect(retryAfterSeconds({}, { 'retry-after': '30' })).toBe(30);
    expect(retryAfterSeconds(undefined, undefined)).toBe(60);
    expect(retryAfterSeconds({ retry_after: 'x' }, { 'retry-after': 'Wed, 21 Oct' }, 45)).toBe(45);
  });

  it('clamps', () => {
    expect(retryAfterSeconds({ retry_after: 99999 }, undefined)).toBe(3600);
    expect(retryAfterSeconds({ retry_after: 0.2 }, undefined)).toBe(1);
  });
});
