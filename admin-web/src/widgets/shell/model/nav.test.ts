import { describe, expect, it } from 'vitest';

import { activeItem, navFor } from './nav';

const keys = (role: 'super' | 'editor', dev = false) =>
  navFor(role, { dev }).flatMap((g) => g.items.map((i) => i.key));

describe('navFor', () => {
  it('hides super-only sections from editors', () => {
    expect(keys('editor')).not.toContain('admins');
    expect(keys('editor')).not.toContain('languages');
    expect(keys('super')).toEqual(expect.arrayContaining(['admins', 'languages', 'dashboard']));
  });
  it('shows dev-only items in dev', () => {
    expect(keys('editor')).not.toContain('uiKit');
    expect(keys('editor', true)).toContain('uiKit');
  });
});

describe('activeItem', () => {
  it('matches the section, / only exactly', () => {
    expect(activeItem('/')?.key).toBe('dashboard');
    expect(activeItem('/users/12')?.key).toBe('users');
    expect(activeItem('/nowhere')).toBeNull();
  });
});

describe('NAV ↔ routes', () => {
  it('every item is ready and has a page under app/(panel)', async () => {
    const { existsSync } = await import('node:fs');
    const { fileURLToPath } = await import('node:url');
    const { NAV } = await import('./nav');
    const panel = fileURLToPath(new URL('../../../app/(panel)/', import.meta.url));
    for (const item of NAV.flatMap((g) => g.items)) {
      expect(item.ready, item.key).toBe(true);
      const dir = item.href === '/' ? '' : item.href.slice(1);
      expect(existsSync(`${panel}${dir}/page.tsx`), item.href).toBe(true);
    }
  });
});
