import { describe, expect, it } from 'vitest';

import { findCatalogTarget, matchesAny, type PilotCatalog } from './catalog';
import { buildAutopilotManifest } from './manifest';
import { parseAutopilotScript } from './script';
import { describeTarget, isTargetId, matchTargetId, parseTargetId, pilot, pilotOption } from './target';

describe('target ids', () => {
  it('accepts area.type and area.type:key, rejects the rest', () => {
    for (const ok of ['log.save', 'log.param:bleeding.flow', 'nav.tab:calendar', 'log.date:2026-11-15']) expect(isTargetId(ok)).toBe(true);
    for (const bad of ['save', 'Log.save', 'log.param:', 'log param', '.log', 'log.param:a b', 'log.param:<x>']) expect(isTargetId(bad)).toBe(false);
  });

  it('splits head and key', () => {
    expect(parseTargetId('log.param:bleeding.flow')).toEqual({ head: 'log.param', key: 'bleeding.flow' });
    expect(parseTargetId('log.save')).toEqual({ head: 'log.save', key: null });
    expect(parseTargetId('nope')).toBeNull();
  });

  it('matches exact ids and families', () => {
    expect(matchTargetId('log.save', 'log.save')).toBe('');
    expect(matchTargetId('log.save', 'log.saved')).toBeNull();
    expect(matchTargetId('log.param:*', 'log.param:mood.moods')).toBe('mood.moods');
    expect(matchTargetId('log.param:*', 'log.param:')).toBeNull();
    expect(matchTargetId('log.param:*', 'log.cat:mood')).toBeNull();
    expect(matchesAny(['log.save', 'nav.tab:*'], 'nav.tab:me')).toBe(true);
  });

  it('builds the DOM attributes', () => {
    expect(pilot('log.save')).toEqual({ 'data-pilot': 'log.save' });
    expect(pilot('log.param:a.b', 'choice')).toEqual({ 'data-pilot': 'log.param:a.b', 'data-pilot-kind': 'choice' });
    expect(pilotOption('heavy')).toEqual({ 'data-pilot-option': 'heavy' });
  });

  it('describes a locator without its content', () => {
    expect(describeTarget('log.save')).toBe('log.save');
    expect(describeTarget({ role: 'button', name: 'ذخیره' })).toBe('role:button');
    expect(describeTarget({ css: '.x' })).toBe('css');
  });
});

describe('parseAutopilotScript', () => {
  it('accepts every action, as an object or as JSON text', () => {
    const script = {
      version: 1,
      id: 'a',
      options: { preset: 'brisk' },
      steps: [
        { do: 'open', route: '/calendar', query: { date: '2026-11-15' } },
        { do: 'open', sheet: 'log', arg: '2026-11-15' },
        { do: 'set', target: 'log.param:bleeding.flow', value: 'heavy' },
        { do: 'set', target: 'log.param:mood.moods', value: ['happy', 'calm'] },
        { do: 'set', target: 'log.param:sleep.hours', value: 7.5, say: 'خواب', optional: true },
        { do: 'set', target: 'log.param:note.text', value: null },
        { do: 'tap', target: { role: 'button', name: 'ذخیره' } },
        { do: 'expand', target: 'log.cat:pain' },
        { do: 'collapse', target: 'log.cat:pain' },
        { do: 'wait', ms: 200 },
        { do: 'wait', target: 'log.save', state: 'hidden', timeoutMs: 900 },
        { do: 'close' },
      ],
    };
    expect(parseAutopilotScript(script).ok).toBe(true);
    expect(parseAutopilotScript(JSON.stringify(script)).ok).toBe(true);
  });

  it.each([
    ['not JSON', '{nope'],
    ['wrong version', { version: 2, steps: [{ do: 'close' }] }],
    ['no steps', { version: 1, steps: [] }],
    ['unknown action', { version: 1, steps: [{ do: 'eval', code: 'x' }] }],
    ['unknown field', { version: 1, steps: [{ do: 'tap', target: 'log.save', js: 'x' }] }],
    ['bad target id', { version: 1, steps: [{ do: 'tap', target: 'javascript:alert(1)' }] }],
    ['open with both', { version: 1, steps: [{ do: 'open', route: '/a', sheet: 'log' }] }],
    ['open with neither', { version: 1, steps: [{ do: 'open' }] }],
    ['wait with nothing', { version: 1, steps: [{ do: 'wait' }] }],
    ['object value', { version: 1, steps: [{ do: 'set', target: 'log.save', value: { a: 1 } }] }],
  ])('rejects %s', (_name, input) => {
    const result = parseAutopilotScript(input);
    expect(result.ok).toBe(false);
  });

  it('reports where it broke, never the value', () => {
    const result = parseAutopilotScript({ version: 1, steps: [{ do: 'set', target: 'log.param:note.text', value: { secret: 'سردرد' } }] });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.issues.join(' ')).toContain('steps.0');
      expect(result.issues.join(' ')).not.toContain('سردرد');
    }
  });
});

const CATALOG: PilotCatalog = {
  routes: [
    { route: '/calendar', titleKey: 'r.calendar' },
    { route: '/todo', titleKey: 'r.todo', module: 'todo' },
    { route: '/dev/x', titleKey: 'r.dev' },
  ],
  sheets: [{ sheet: 'log', titleKey: 's.log', arg: 'YYYY-MM-DD' }],
  targets: [
    { id: 'log.save', value: 'action', titleKey: 't.save' },
    { id: 'log.param:*', value: 'single', reveal: (key) => [{ do: 'expand', target: `log.cat:${key.split('.')[0]}` }] },
    { id: 'log.param:bleeding.flow', value: 'single', titleKey: 't.flow' },
    { id: 'shop.buy', value: 'action', titleKey: 't.buy', module: 'shop' },
  ],
};

describe('catalog', () => {
  it('prefers an exact entry over its family and returns the family key', () => {
    expect(findCatalogTarget(CATALOG, 'log.param:bleeding.flow')?.entry.titleKey).toBe('t.flow');
    const family = findCatalogTarget(CATALOG, 'log.param:mood.moods');
    expect(family?.key).toBe('mood.moods');
    expect(family?.entry.reveal?.(family.key)).toEqual([{ do: 'expand', target: 'log.cat:mood' }]);
    expect(findCatalogTarget(CATALOG, 'nav.tab:me')).toBeNull();
  });
});

describe('buildAutopilotManifest', () => {
  const manifest = buildAutopilotManifest({
    catalog: CATALOG,
    dynamic: [
      { id: 'log.param:mood.moods', kind: 'multi', label: 'حال', options: [{ value: 'happy', label: 'خوشحال' }] },
      { id: 'log.param:secret.x', kind: 'text', label: 'x' },
      { id: 'other.thing:1', kind: 'text', label: 'not catalogued' },
    ],
    t: (key) => `«${key}»`,
    moduleEnabled: (module) => module !== 'todo' && module !== 'shop',
    blockedTargets: ['log.param:secret.x'],
    blockedRoutes: ['/dev'],
  });

  it('lists titled routes and sheets, minus disabled modules and blocked routes', () => {
    expect(manifest.routes).toEqual([{ route: '/calendar', title: '«r.calendar»' }]);
    expect(manifest.sheets).toEqual([{ sheet: 'log', title: '«s.log»', arg: 'YYYY-MM-DD' }]);
  });

  it('lists exact targets and the catalogued dynamic ones, minus blocked and module-off', () => {
    expect(manifest.targets.map((t) => t.id)).toEqual(['log.save', 'log.param:bleeding.flow', 'log.param:mood.moods']);
    expect(manifest.targets[0]).toEqual({ id: 'log.save', kind: 'action', label: '«t.save»' });
  });

  it('carries the contract', () => {
    expect(manifest.version).toBe(1);
    expect(manifest.actions).toContain('set');
    expect(manifest.guide).toContain('"version":1');
  });
});
