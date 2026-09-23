import { describe, expect, it } from 'vitest';

import { publicOrigin } from './public-origin';

const FALLBACK = 'https://localhost:3000';

describe('publicOrigin', () => {
  it('uses the proxy-forwarded host and scheme', () => {
    const h = new Headers({ host: 'localhost:3000', 'x-forwarded-host': 'stage.ritmeapp.ir', 'x-forwarded-proto': 'https' });
    expect(publicOrigin(h, FALLBACK)).toBe('https://stage.ritmeapp.ir');
  });
  it('falls back to Host, then to the server origin', () => {
    expect(publicOrigin(new Headers({ host: 'localhost:3001' }), 'http://localhost:3001')).toBe('http://localhost:3001');
    expect(publicOrigin(new Headers(), FALLBACK)).toBe(FALLBACK);
  });
  it('takes the first value of a forwarded chain', () => {
    const h = new Headers({ 'x-forwarded-host': 'adpanell.ritme.app, other', 'x-forwarded-proto': 'https,http' });
    expect(publicOrigin(h, FALLBACK)).toBe('https://adpanell.ritme.app');
  });
  it('rejects malformed hosts and schemes', () => {
    expect(publicOrigin(new Headers({ 'x-forwarded-host': 'evil.example/path' }), FALLBACK)).toBe(FALLBACK);
    expect(publicOrigin(new Headers({ host: 'a@b' }), FALLBACK)).toBe(FALLBACK);
    const h = new Headers({ host: 'adpanell.ritme.app', 'x-forwarded-proto': 'javascript' });
    expect(publicOrigin(h, FALLBACK)).toBe('https://adpanell.ritme.app');
  });
});
