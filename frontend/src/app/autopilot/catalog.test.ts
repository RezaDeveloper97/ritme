import { existsSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

import { isPattern, isTargetId, matchesAny } from '@/shared/pilot';

import en from '../../../messages/en/autopilot-catalog.json';
import fa from '../../../messages/fa/autopilot-catalog.json';
import { AUTOPILOT_CONFIG } from '@/shared/autopilot/config';

import { PILOT_CATALOG } from './catalog';

/**
 * Guards the autopilot catalog (frontend/CLAUDE.md §11.6): what the DOM is marked with, what the catalog
 * declares and what the docs list must be the same set. A control marked but not catalogued can't be
 * reached or offered to the agent; a catalogued one nobody marks is a promise the screen doesn't keep.
 *
 * Marks are found in source: `pilot('id')`, `pilot(`family:${…}`)`, `data-pilot="id"`, an id handed down as a
 * `pilot…=` prop (`pilotId="med.name"`, `pilotKey={`ck.row:${id}`}`), and the id builders of any
 * `pilot-targets.ts` (a slice's own id table). An id assembled some other way is still accepted when its text
 * appears in a file that imports `@/shared/pilot` (see `mentioned`).
 */
const SRC = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
const DOC = resolve(SRC, '..', '..', 'docs', 'autopilot-catalog.md');
/** The vocabulary, the engine, this folder and the dev lab talk ABOUT ids; they don't mark controls. */
const NOT_MARKS = ['shared/pilot', 'shared/autopilot', 'app/autopilot', 'screens/autopilot-lab'];

function sourceFiles(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) sourceFiles(path, out);
    else if (/\.tsx?$/.test(name) && !/\.test\.tsx?$/.test(name)) out.push(path);
  }
  return out;
}

/** Sources that deal in target ids: where an id built by a helper or passed through a variable still appears as text. */
const pilotSources: string[] = [];

/** Is this catalog id (or the prefix of this family) written anywhere in the marking code? */
function mentioned(id: string): boolean {
  const needle = isPattern(id) ? id.slice(0, -1) : id;
  return pilotSources.some((source) => source.includes(needle));
}

interface Marks {
  /** Whole ids written in source: `pilot('log.save')`, `pilotId="med.name"`. */
  exact: Map<string, string>;
  /** The fixed start of an id finished at run time: `pilot(`care.tab:${tab}`)` → `care.tab:`. */
  prefixes: Map<string, string>;
}

function marksInSource(): Marks {
  const exact = new Map<string, string>();
  const prefixes = new Map<string, string>();
  for (const file of sourceFiles(SRC)) {
    const rel = relative(SRC, file);
    if (NOT_MARKS.some((skip) => rel.startsWith(skip))) continue;
    const source = readFileSync(file, 'utf8')
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/^\s*\/\/.*$/gm, '');
    const whole = [
      ...source.matchAll(/\bpilot\(\s*'([^']+)'/g),
      ...source.matchAll(/data-pilot="([^"]+)"/g),
      ...source.matchAll(/\bpilot[A-Za-z]*=\{?\s*["']([^"']+)["']/g),
    ].map((m) => m[1]);
    const starts = [...source.matchAll(/\bpilot\(\s*`([^`$]+)\$\{/g), ...source.matchAll(/\bpilot[A-Za-z]*=\{\s*`([^`$]+)\$\{/g)].map((m) => m[1]);
    if (/(^|\/)pilot-targets\.ts$/.test(rel)) {
      whole.push(...[...source.matchAll(/'([a-z][a-z0-9-]*(?:\.[a-z][a-z0-9-]*)+)'/g)].map((m) => m[1]));
      starts.push(...[...source.matchAll(/`([a-z][a-z0-9.-]*:)\$\{/g)].map((m) => m[1]));
    }
    for (const id of whole) if (isTargetId(id)) exact.set(id, rel);
    for (const prefix of starts) if (/^[a-z][a-z0-9-]*\./.test(prefix)) prefixes.set(prefix, rel);
    if (source.includes('@/shared/pilot') || /\bpilot[A-Za-z]*[=:?]/.test(source)) pilotSources.push(source);
  }
  return { exact, prefixes };
}

const lookup = (messages: unknown, key: string): unknown =>
  key.split('.').reduce<unknown>((node, part) => (node && typeof node === 'object' ? (node as Record<string, unknown>)[part] : undefined), messages);

function catalogDoc(): string {
  const title = (key?: string) => (key ? String(lookup(fa, key)) : '—');
  const lines = [
    '# Autopilot catalog',
    '',
    '> Generated from `frontend/src/app/autopilot/catalog.ts` — do not edit. Regenerate with `npm run autopilot:catalog` (in `frontend/`).',
    '> Contract and rules: `docs/autopilot.md`. Members of a `family:*` come from data (the log taxonomy) and are listed in the live manifest (`/dev/autopilot`).',
    '',
    '## Routes (`open` → `route`)',
    '',
    '| Route | Title (fa) | Module |',
    '|---|---|---|',
    ...PILOT_CATALOG.routes.map((r) => `| \`${r.route}\` | ${title(r.titleKey)} | ${r.module ?? '—'} |`),
    '',
    '## Sheets (`open` → `sheet`)',
    '',
    '| Sheet | Title (fa) | `arg` | Module |',
    '|---|---|---|---|',
    ...PILOT_CATALOG.sheets.map((s) => `| \`${s.sheet}\` | ${title(s.titleKey)} | ${s.arg ?? '—'} | ${s.module ?? '—'} |`),
    '',
    '## Targets',
    '',
    '| Id | Value | Title (fa) | Surface | Reveal | Options |',
    '|---|---|---|---|---|---|',
    ...PILOT_CATALOG.targets.map((t) => {
      const query = t.surface?.query ? `?${new URLSearchParams(t.surface.query).toString()}` : '';
      const surface = t.surface?.sheet ? `sheet \`${t.surface.sheet}\`` : t.surface?.route ? `route \`${t.surface.route}${query}\`` : '—';
      const options = t.options ? t.options.map((o) => `\`${o.value}\``).join(' ') : '—';
      return `| \`${t.id}\` | ${t.value} | ${title(t.titleKey)} | ${surface} | ${t.reveal ? 'yes' : '—'} | ${options} |`;
    }),
    '',
  ];
  return lines.join('\n');
}

/** Ids whose name says share/invite/consent/send/export/join/delete/pay. */
const SENSITIVE = /share|invite|consent|send|export|join|delete|pay/;
/**
 * Reviewed, inert matches: they only fill a field or move through a form (nothing leaves the app until the
 * confirmed submit/create id). `city-join.` is the whole business-registration form; its submit is a confirm target.
 */
const INERT = new Set([
  'teen.invite-open', 'teen.invite-phone', 'teen.invite-name',
  'invite.next', 'invite.contact-toggle', 'invite.name', 'invite.phone', 'invite.preview', 'invite.done',
]);
const isInert = (id: string) => INERT.has(id) || (id.startsWith('city-join.') && id !== 'city-join.submit');

/** Sensitive ids no safety list covers. Uses the engine's own matcher (wildcards included). */
export function uncoveredSensitive(ids: readonly string[], confirm: readonly string[], blocked: readonly string[]): string[] {
  return ids.filter((id) => SENSITIVE.test(id) && !isInert(id) && !matchesAny(confirm, id) && !matchesAny(blocked, id));
}

describe('autopilot safety guard', () => {
  const { confirmTargets, blockedTargets } = AUTOPILOT_CONFIG.safety;
  const ids = PILOT_CATALOG.targets.map((t) => t.id);

  it('covers every share/invite/consent/send/export/join/delete/pay id by confirmTargets or blockedTargets', () => {
    expect(uncoveredSensitive(ids, confirmTargets, blockedTargets)).toEqual([]);
  });

  it('keeps the named sensitive ids in the lists', () => {
    for (const id of ['invite.create', 'invite.consent', 'companion.join-submit', 'teen.share:*', 'report.send', 'privacy.export']) {
      expect(matchesAny([...confirmTargets, ...blockedTargets], id), id).toBe(true);
    }
  });

  it('reports a new uncovered id (negative case) and accepts wildcard coverage', () => {
    expect(uncoveredSensitive([...ids, 'fake.share-everything'], confirmTargets, blockedTargets)).toEqual(['fake.share-everything']);
    expect(uncoveredSensitive(['teen.share:cycle'], ['teen.share:*'], [])).toEqual([]);
    expect(uncoveredSensitive(['teen.share:cycle'], [], [])).toEqual(['teen.share:cycle']);
  });
});

describe('autopilot catalog', () => {
  const marks = marksInSource();
  const declared = PILOT_CATALOG.targets.map((t) => t.id);

  /** A catalog id without the family star: what an id in the DOM starts with. */
  const stem = (id: string) => (isPattern(id) ? id.slice(0, -1) : id);

  it('declares every id the screens are marked with', () => {
    // A whole id is declared as itself, as a member of a family, or as the base its component extends
    // (`quick.last-period` → `quick.last-period:day`). A run-time id needs something declared under its start.
    const undeclaredExact = [...marks.exact].filter(
      ([id]) => !matchesAny(declared, id) && !declared.some((d) => d.startsWith(`${id}:`)),
    );
    const undeclaredPrefix = [...marks.prefixes].filter(([prefix]) => !declared.some((d) => stem(d).startsWith(prefix) || prefix.startsWith(stem(d))));
    expect([...undeclaredExact, ...undeclaredPrefix].map(([id, file]) => `${id} (${file})`)).toEqual([]);
  });

  it('marks every id it declares', () => {
    const prefixes = [...marks.prefixes.keys()];
    const exact = [...marks.exact.keys()];
    const unmarked = declared.filter((id) => {
      const head = stem(id);
      if (marks.exact.has(id) || prefixes.some((p) => head.startsWith(p))) return false;
      // A component that extends the id it was given: `quick.last-period` → `:day`, `rec.doc-date` → `-cal`.
      if (exact.some((e) => head.startsWith(`${e}:`) || head.startsWith(`${e}-`))) return false;
      if (prefixes.some((p) => p.endsWith(':') && head.startsWith(`${p.slice(0, -1)}-`))) return false;
      return !mentioned(id);
    });
    expect(unmarked).toEqual([]);
  });

  it('uses well-formed, unique ids', () => {
    for (const id of declared) expect(isTargetId(isPattern(id) ? `${id.slice(0, -1)}x` : id), id).toBe(true);
    expect(new Set(declared).size).toBe(declared.length);
  });

  it('titles every route, sheet and exact target in both bundled languages', () => {
    const keys = [
      ...PILOT_CATALOG.routes.map((r) => r.titleKey),
      ...PILOT_CATALOG.sheets.map((s) => s.titleKey),
      ...PILOT_CATALOG.targets.filter((t) => !isPattern(t.id)).map((t) => t.titleKey ?? `MISSING titleKey for ${t.id}`),
      ...PILOT_CATALOG.targets.flatMap((t) => [...(t.options ?? []).map((o) => o.titleKey), ...(t.unitKey ? [t.unitKey] : [])]),
    ];
    for (const key of keys) {
      expect(typeof lookup(fa, key), `fa ${key}`).toBe('string');
      expect(typeof lookup(en, key), `en ${key}`).toBe('string');
    }
  });

  it('only lists routes and sheets that exist', () => {
    for (const { route } of PILOT_CATALOG.routes) {
      expect(existsSync(join(SRC, 'app', '[locale]', ...route.split('/').filter(Boolean), 'page.tsx')), route).toBe(true);
    }
    const registry = readFileSync(join(SRC, 'app', 'sheets', 'registry.tsx'), 'utf8');
    for (const { sheet } of PILOT_CATALOG.sheets) {
      expect(new RegExp(`^  '?${sheet}'?: \\{`, 'm').test(registry), sheet).toBe(true);
    }
  });

  it('points every surface at a catalogued sheet or route', () => {
    for (const target of PILOT_CATALOG.targets) {
      if (target.surface?.sheet) expect(PILOT_CATALOG.sheets.some((s) => s.sheet === target.surface?.sheet), target.id).toBe(true);
      if (target.surface?.route) expect(PILOT_CATALOG.routes.some((r) => r.route === target.surface?.route), target.id).toBe(true);
    }
  });

  it('reveals through catalogued targets only', () => {
    for (const target of PILOT_CATALOG.targets) {
      for (const step of target.reveal?.('sample.key') ?? []) {
        const id = 'target' in step && typeof step.target === 'string' ? step.target : null;
        if (id) expect(matchesAny(declared, id), `${target.id} → ${id}`).toBe(true);
      }
    }
  });

  it('matches docs/autopilot-catalog.md (npm run autopilot:catalog)', () => {
    const doc = catalogDoc();
    if (process.env.UPDATE_AUTOPILOT_CATALOG) writeFileSync(DOC, doc);
    expect(existsSync(DOC) ? readFileSync(DOC, 'utf8') : '').toBe(doc);
  });
});
