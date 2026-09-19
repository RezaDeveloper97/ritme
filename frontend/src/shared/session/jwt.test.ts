import { describe, expect, it } from 'vitest';

import { tokenExpiresAt } from './jwt';

const b64url = (obj: unknown) =>
  Buffer.from(JSON.stringify(obj)).toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

describe('tokenExpiresAt', () => {
  it('reads exp as epoch milliseconds', () => {
    const token = `${b64url({ alg: 'RS256' })}.${b64url({ sub: '1', exp: 1_821_000_000 })}.sig`;
    expect(tokenExpiresAt(token)).toBe(1_821_000_000_000);
  });

  it('returns null for malformed tokens or a missing exp', () => {
    expect(tokenExpiresAt('not-a-jwt')).toBeNull();
    expect(tokenExpiresAt('a.%%%.c')).toBeNull();
    expect(tokenExpiresAt(`x.${b64url({ sub: '1' })}.y`)).toBeNull();
    expect(tokenExpiresAt(`x.${b64url({ exp: '2027' })}.y`)).toBeNull();
  });
});
