import { describe, expect, it } from 'vitest';

import { safeNext } from './safe-next';

describe('safeNext', () => {
  it('keeps same-origin paths', () => {
    expect(safeNext('/users?page=2')).toBe('/users?page=2');
  });
  it('rejects open redirects and loops', () => {
    for (const bad of [null, '', 'https://evil.example', '//evil.example', '/\\evil', '/login', '/login?next=/x']) {
      expect(safeNext(bad)).toBe('/');
    }
  });
});
