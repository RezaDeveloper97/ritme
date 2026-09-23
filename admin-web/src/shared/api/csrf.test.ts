import { describe, expect, it } from 'vitest';

import { CSRF_COOKIE_NAMES } from '@/shared/config';

import { readCookie } from './csrf';

describe('readCookie', () => {
  it('finds the __Host- and the plain dev cookie', () => {
    expect(readCookie('a=1; __Host-ritme_admin_csrf=abc%3D; b=2', CSRF_COOKIE_NAMES)).toBe('abc=');
    expect(readCookie('ritme_admin_csrf=dev', CSRF_COOKIE_NAMES)).toBe('dev');
    expect(readCookie('other=1', CSRF_COOKIE_NAMES)).toBeNull();
  });
});
