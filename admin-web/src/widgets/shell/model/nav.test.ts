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

describe('navFor with permissions (B-N9-02)', () => {
  it('shows only the capabilities the role grants', () => {
    const items = navFor('marketing', { dev: false, permissions: { dashboard: 'manage', content: 'manage' } }).flatMap(
      (g) => g.items.map((i) => i.key),
    );
    expect(items).toEqual(expect.arrayContaining(['dashboard', 'articles', 'password']));
    expect(items).not.toContain('users');
    expect(items).not.toContain('auditLog');
    expect(items).not.toContain('healthAccess');
    expect(items).not.toContain('roles');
  });
  it('health access only with the health capability', () => {
    const clinical = navFor('clinical_support', { dev: false, permissions: { health: 'manage', users: 'manage' } });
    expect(clinical.flatMap((g) => g.items.map((i) => i.key))).toContain('healthAccess');
    expect(keys('editor')).not.toContain('healthAccess');
    expect(keys('super')).toEqual(expect.arrayContaining(['roles', 'auditLog', 'healthAccess']));
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
