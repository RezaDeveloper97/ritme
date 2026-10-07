import { describe, expect, it } from 'vitest';

import { activeNav } from './nav';

describe('activeNav', () => {
  it.each([
    ['/', 'dashboard'],
    [null, 'dashboard'],
    ['/content', 'content'],
    ['/content/12', 'content'],
    ['/groups', 'groups'],
    ['/students', 'students'],
    ['/contentious', 'dashboard'],
  ])('%s → %s', (path, key) => {
    expect(activeNav(path)).toBe(key);
  });
});
