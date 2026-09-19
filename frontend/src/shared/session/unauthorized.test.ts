import { describe, expect, it } from 'vitest';

import { bearerOf, endsSession, type UnauthorizedResponse } from './unauthorized';

const TOKEN = 'header.payload.sig';

const base: UnauthorizedResponse = {
  status: 401,
  contentType: 'application/json',
  body: { error_code: 'token_revoked' },
  sentToken: TOKEN,
  currentToken: TOKEN,
};

describe('endsSession', () => {
  it.each(['token_expired', 'token_revoked', 'unauthenticated'])('ends the session on error_code %s', (code) => {
    expect(endsSession({ ...base, body: { success: false, error_code: code } })).toBe(true);
  });

  it('matches the T-M1-02 body shape exactly', () => {
    const body = (error_code: string) => ({ message: 'Unauthenticated.', error_code });
    expect(endsSession({ ...base, body: body('token_revoked') })).toBe(true);
    expect(endsSession({ ...base, body: body('token_expired') })).toBe(true);
    expect(endsSession({ ...base, body: body('unauthenticated') })).toBe(true);
  });

  it('keeps the session on unknown codes', () => {
    expect(endsSession({ ...base, body: { error_code: 'something_new' } })).toBe(false);
  });

  it("accepts Laravel's legacy JSON body from a backend without error_code", () => {
    expect(endsSession({ ...base, body: { message: 'Unauthenticated.' } })).toBe(true);
    expect(
      endsSession({ ...base, contentType: 'application/json; charset=UTF-8', body: { message: 'Unauthenticated.' } }),
    ).toBe(true);
  });

  it('ignores a JSON 401 with neither error_code nor the Laravel message', () => {
    expect(endsSession({ ...base, body: { message: 'Invalid code' } })).toBe(false);
    expect(endsSession({ ...base, body: null })).toBe(false);
    expect(endsSession({ ...base, body: '' })).toBe(false);
  });

  it('never ends the session on a non-JSON 401 (proxy / Basic-auth gate)', () => {
    expect(endsSession({ ...base, contentType: 'text/html', body: '<html>401</html>' })).toBe(false);
    expect(endsSession({ ...base, contentType: undefined })).toBe(false);
    // Even a gate that happens to echo a matching-looking body.
    expect(endsSession({ ...base, contentType: 'text/plain', body: { error_code: 'token_revoked' } })).toBe(false);
  });

  it('never ends the session on other statuses, network errors or timeouts', () => {
    for (const status of [400, 403, 404, 419, 422, 429, 500, 502, 503, 504]) {
      expect(endsSession({ ...base, status })).toBe(false);
    }
    // Network error / timeout: axios has no response at all.
    expect(endsSession({ ...base, status: undefined, contentType: undefined, body: undefined })).toBe(false);
  });

  it('ignores a 401 for a request that did not carry the token', () => {
    expect(endsSession({ ...base, sentToken: null })).toBe(false);
  });

  it('ignores a 401 for a token that was replaced while the request was in flight', () => {
    expect(endsSession({ ...base, currentToken: 'new.token.value' })).toBe(false);
    expect(endsSession({ ...base, currentToken: null })).toBe(false);
  });
});

describe('bearerOf', () => {
  it('extracts the token from a Bearer header', () => {
    expect(bearerOf(`Bearer ${TOKEN}`)).toBe(TOKEN);
    expect(bearerOf(`bearer ${TOKEN}`)).toBe(TOKEN);
  });

  it('returns null for anything else', () => {
    expect(bearerOf(undefined)).toBeNull();
    expect(bearerOf('Basic dXNlcjpwYXNz')).toBeNull();
    expect(bearerOf(42)).toBeNull();
  });
});
