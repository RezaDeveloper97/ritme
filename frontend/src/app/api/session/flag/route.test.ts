import { NextRequest } from 'next/server';
import { describe, expect, it } from 'vitest';

import { POST } from './route';

describe('POST /api/session/flag', () => {
  it('sets a year-long, value-less flag cookie readable by the client', () => {
    const res = POST(new NextRequest('https://web.ritme.app/api/session/flag', { method: 'POST' }));
    expect(res.status).toBe(204);

    const cookie = res.headers.get('set-cookie') ?? '';
    expect(cookie).toMatch(/^ritme_auth=1;/);
    expect(cookie).toMatch(/Max-Age=31536000/i);
    expect(cookie).toMatch(/Path=\//i);
    expect(cookie).toMatch(/SameSite=lax/i);
    expect(cookie).toMatch(/Secure/i);
    expect(cookie).not.toMatch(/HttpOnly/i);
    expect(res.headers.get('cache-control')).toBe('no-store');
  });

  it('refuses cross-site requests', () => {
    const res = POST(
      new NextRequest('https://web.ritme.app/api/session/flag', {
        method: 'POST',
        headers: { 'sec-fetch-site': 'cross-site' },
      }),
    );
    expect(res.status).toBe(403);
    expect(res.headers.get('set-cookie')).toBeNull();
  });
});
