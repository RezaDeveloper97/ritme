import { describe, expect, it } from 'vitest';

import { normalizeBasePath, stripBasePath, withBasePath } from './base-path';

describe('normalizeBasePath', () => {
  it('is empty when unset (production on its own host)', () => {
    expect(normalizeBasePath(undefined)).toBe('');
    expect(normalizeBasePath('')).toBe('');
    expect(normalizeBasePath('/')).toBe('');
  });
  it('adds the leading slash and drops trailing ones', () => {
    expect(normalizeBasePath('panel')).toBe('/panel');
    expect(normalizeBasePath('/panel/')).toBe('/panel');
    expect(normalizeBasePath(' /a/b// ')).toBe('/a/b');
  });
});

describe('withBasePath', () => {
  it('is the identity without a base path', () => {
    expect(withBasePath('/login?signed_out=1', '')).toBe('/login?signed_out=1');
    expect(withBasePath('/', '')).toBe('/');
  });
  it('prefixes app paths', () => {
    expect(withBasePath('/login?signed_out=1', '/panel')).toBe('/panel/login?signed_out=1');
    expect(withBasePath('/', '/panel')).toBe('/panel');
    expect(withBasePath('users', '/panel')).toBe('/panel/users');
  });
});

describe('stripBasePath', () => {
  it('is the identity without a base path', () => {
    expect(stripBasePath('/users', '')).toBe('/users');
  });
  it('removes the prefix only on a segment boundary', () => {
    expect(stripBasePath('/panel', '/panel')).toBe('/');
    expect(stripBasePath('/panel/users/3', '/panel')).toBe('/users/3');
    expect(stripBasePath('/panelists', '/panel')).toBe('/panelists');
    expect(stripBasePath('/login', '/panel')).toBe('/login');
  });
});
