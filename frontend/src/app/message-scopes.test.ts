import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

import { scopeNamespace } from '@/shared/i18n';

import { ROUTE_NAMESPACES, SHELL_NAMESPACES } from './message-scopes';

/**
 * Guards the per-route message lists (`message-scopes.ts`). Each list must equal
 * the namespaces its subtree translates: a missing one renders raw keys in
 * production, an extra one ships strings nobody reads.
 *
 * "Its subtree" is the static import graph (including `import()`s, so every
 * lazily-loaded sheet counts) from the layout or from a `page.tsx`, and a
 * namespace is used where a file calls `useTranslations('ns…')` or
 * `getTranslations('ns…')`. Importing a slice's `index.ts` counts the whole
 * slice, which errs on the side of shipping a namespace.
 */
const SRC = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const LOCALE_DIR = join(SRC, 'app', '[locale]');

function resolveImport(from: string, spec: string): string | null {
  let base: string;
  if (spec.startsWith('@/')) base = join(SRC, spec.slice(2));
  else if (spec.startsWith('.')) base = resolve(dirname(from), spec);
  else return null; // a package
  if (existsSync(base) && statSync(base).isFile()) return base;
  for (const suffix of ['.ts', '.tsx', '/index.ts', '/index.tsx']) {
    if (existsSync(base + suffix)) return base + suffix;
  }
  return null; // css, images, fonts
}

/**
 * One thing a file needs from the bundle, as a dotted path from the root: `home.nb.hero.started`
 * (a literal key) or `menopause.home` (a dynamic key: the whole subtree it can reach). A scope
 * entry serves a need when it is that path or an ancestor of it.
 */
export type KeyNeed = string;

interface FileInfo {
  deps: string[];
  namespaces: string[];
  needs: KeyNeed[];
}

const join2 = (prefix: string, key: string) => [prefix, key].filter(Boolean).join('.');

/**
 * Every message path a source file reads. For each `const t = useTranslations('a.b')` /
 * `getTranslations('a.b')` it collects `t('k')`, `t.rich('k')`, `t.raw('k')`, `t.has('k')` and
 * `t.markup('k')`:
 *  - a literal key → `a.b.k`;
 *  - a template literal → the literal part before the first `${`, cut at its last dot
 *    (`\`items.${x}.title\`` → `a.b.items`): the whole subtree it can reach;
 *  - any other expression, or `t` used without being called (passed to a helper, returned,
 *    stored) → the whole `a.b` subtree.
 * The same variable name declared twice in a file is treated as every declaration at once, which
 * can only widen the result.
 */
/** How many `name={name}` props are handed to JSX tags of components declared in `localComponents`. */
function countHandedToLocal(source: string, name: string, localComponents: Set<string>): number {
  let count = 0;
  for (const open of source.matchAll(/<([A-Z]\w*)\s/g)) {
    if (!localComponents.has(open[1] as string)) continue;
    // Walk the opening tag to its closing `>`, skipping `{…}` expressions (they hold `=>` and `>`).
    let depth = 0;
    let end = (open.index ?? 0) + open[0].length;
    for (; end < source.length; end += 1) {
      const ch = source[end];
      if (ch === '{') depth += 1;
      else if (ch === '}') depth -= 1;
      else if (ch === '>' && depth === 0) break;
    }
    const tag = source.slice((open.index ?? 0) + open[0].length, end);
    count += tag.split(`={${name}}`).length - 1;
  }
  return count;
}

/** The two literals of a `t(cond ? 'a' : 'b')` argument starting at `from`, or null for anything else. */
function ternaryLiterals(source: string, from: number): string[] | null {
  let depth = 0;
  let end = from;
  for (; end < source.length; end += 1) {
    const ch = source[end];
    if (ch === '(' || ch === '[' || ch === '{') depth += 1;
    else if (ch === ')' || ch === ']' || ch === '}') {
      if (depth === 0) break;
      depth -= 1;
    } else if (ch === ',' && depth === 0) break;
  }
  const match = /\?\s*(['"])([^'"]*)\1\s*:\s*(['"])([^'"]*)\3\s*$/.exec(source.slice(from, end));
  return match ? [match[2] as string, match[4] as string] : null;
}

export function messageNeedsOf(source: string): KeyNeed[] {
  const needs = new Set<KeyNeed>();
  const decl =
    /\b(?:const|let)\s+(\w+)\s*=\s*(?:await\s+)?(?:useTranslations|getTranslations)\(\s*(?:\{[^}]*namespace:\s*)?(?:(['"])([^'"]*)\2)?[^)]*\)/g;
  const prefixesByVar = new Map<string, Set<string>>();
  for (const m of source.matchAll(decl)) {
    const set = prefixesByVar.get(m[1] as string) ?? new Set<string>();
    set.add(m[3] ?? '');
    prefixesByVar.set(m[1] as string, set);
  }
  for (const [name, prefixes] of prefixesByVar) {
    const callRe = new RegExp(
      `(?<![\\w.$])${name}(?:\\.(?:rich|raw|has|markup))?\\(\\s*(?:(['"])([^'"]*)\\1\\s*[,)]|\`([^\`]*)\`|)`,
      'g',
    );
    let wholeSubtree = false;
    for (const call of source.matchAll(callRe)) {
      const literal = call[2];
      const template = call[3];
      for (const prefix of prefixes) {
        if (literal !== undefined) needs.add(join2(prefix, literal));
        else if (template !== undefined) {
          const hole = template.indexOf('${');
          const head = template.slice(0, hole);
          needs.add(join2(prefix, hole === -1 ? template : head.slice(0, Math.max(0, head.lastIndexOf('.')))));
        } else {
          // `t(cond ? 'a' : 'b')`: both literals are needed; any other expression can reach the whole subtree.
          const branches = ternaryLiterals(source, (call.index ?? 0) + call[0].length);
          if (branches) branches.forEach((key) => needs.add(join2(prefix, key)));
          else wholeSubtree = true;
        }
      }
    }
    // `t` used as a value (handed to a helper, stored, returned): it can read anything below its
    // prefix. Not counted: a destructured component prop (`function X({ t }: …)`) and `t={t}` handed
    // to a component of this same file — that component's own `t('…')` calls are collected above.
    const bare = new RegExp(`(?<![\\w.$'"\`])${name}(?![\\w$'"\`:])(?!\\s*=(?!=))(?!\\s*(?:\\(|\\.(?:rich|raw|has|markup)\\())`, 'g');
    const withoutProps = source.replace(/\(\s*\{[^{}]*\}\s*:/g, '(:');
    const localComponents = new Set(
      [...source.matchAll(/\b(?:function|const)\s+([A-Z]\w*)/g)].map((m) => m[1] as string),
    );
    const handedLocal = countHandedToLocal(withoutProps, name, localComponents);
    const uses = [...withoutProps.matchAll(bare)].length - handedLocal;
    if (uses > 0) wholeSubtree = true;
    if (wholeSubtree) for (const prefix of prefixes) needs.add(prefix);
  }
  return [...needs].filter(Boolean);
}

/**
 * Removes `//` and block comments but leaves string and template literals alone, so a `/*` inside a
 * line comment (« … `/fertility/*` … ») or a string never opens a "block comment" that swallows real code.
 */
export function stripComments(source: string): string {
  return source.replace(
    /("(?:\\.|[^"\\\n])*"|'(?:\\.|[^'\\\n])*'|`(?:\\.|[^`\\])*`)|\/\/[^\n]*|\/\*[\s\S]*?\*\//g,
    (match, literal: string | undefined) => (literal === undefined ? '' : match),
  );
}

const infoCache = new Map<string, FileInfo>();

function infoOf(file: string): FileInfo {
  const cached = infoCache.get(file);
  if (cached) return cached;
  // Commented-out code (e.g. a temporarily hidden widget) must not count.
  const source = stripComments(readFileSync(file, 'utf8'));
  const deps = [...source.matchAll(/(?:from|import)\s*\(?\s*['"]([^'"]+)['"]/g)]
    .map((m) => resolveImport(file, m[1]))
    .filter((dep): dep is string => dep !== null);
  const namespaces = [
    ...source.matchAll(/(?:useTranslations|getTranslations)\(\s*['"]([A-Za-z][A-Za-z0-9]*)/g),
  ].map((m) => m[1]);
  const info = { deps, namespaces, needs: messageNeedsOf(source) };
  infoCache.set(file, info);
  return info;
}

function namespacesReachableFrom(entry: string): string[] {
  const seen = new Set<string>();
  const found = new Set<string>();
  const stack = [entry];
  while (stack.length > 0) {
    const file = stack.pop() as string;
    if (seen.has(file)) continue;
    seen.add(file);
    const info = infoOf(file);
    info.namespaces.forEach((ns) => found.add(ns));
    stack.push(...info.deps);
  }
  return [...found].sort();
}

function needsReachableFrom(entry: string): KeyNeed[] {
  const seen = new Set<string>();
  const found = new Set<KeyNeed>();
  const stack = [entry];
  while (stack.length > 0) {
    const file = stack.pop() as string;
    if (seen.has(file)) continue;
    seen.add(file);
    const info = infoOf(file);
    info.needs.forEach((need) => found.add(need));
    stack.push(...info.deps);
  }
  return [...found].sort();
}

/** A scope entry serves a need when it is the need's path or an ancestor of it. */
const serves = (entry: string, need: KeyNeed) => need === entry || need.startsWith(`${entry}.`);

function pageFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return pageFiles(path);
    return entry.name === 'page.tsx' ? [path] : [];
  });
}

const sorted = (list: readonly string[]) => [...new Set(list)].sort();

const pages = pageFiles(LOCALE_DIR);

describe('message scopes', () => {
  it('the layout ships exactly the namespaces the shell and every sheet use', () => {
    expect(sorted(SHELL_NAMESPACES)).toEqual(
      namespacesReachableFrom(join(LOCALE_DIR, 'layout.tsx')),
    );
  });

  it.each(pages.map((page) => [relative(SRC, page), page]))(
    '%s ships exactly the namespaces its screen uses',
    (_name, page) => {
      const used = namespacesReachableFrom(page);
      const route = /<RouteMessages\s+route="([A-Za-z]+)"/.exec(readFileSync(page, 'utf8'))?.[1];

      if (!route) {
        // Only a page that renders no JSX at all (the locale root just
        // redirects) may skip the wrapper; its barrel imports may still
        // reach translated components it never renders.
        expect(readFileSync(page, 'utf8')).not.toMatch(/return\s*\(?\s*</);
        return;
      }
      expect(Object.keys(ROUTE_NAMESPACES)).toContain(route);
      expect(sorted(ROUTE_NAMESPACES[route as keyof typeof ROUTE_NAMESPACES].map(scopeNamespace))).toEqual(
        used,
      );
    },
  );

  it('every declared route list is used by a page', () => {
    const used = pages
      .map((page) => /<RouteMessages\s+route="([A-Za-z]+)"/.exec(readFileSync(page, 'utf8'))?.[1])
      .filter(Boolean);
    expect(sorted(Object.keys(ROUTE_NAMESPACES))).toEqual(sorted(used as string[]));
  });
});

// --- T-M12-29: sub-key scopes ------------------------------------------------------------------
// A route list may name a subtree (`menopause.home`) instead of a whole namespace. The namespace
// test above still says *which namespaces* a route ships; the tests below say *which keys of them*.

const camel = (file: string) => file.replace(/-([a-z0-9])/g, (_, c: string) => c.toUpperCase());
function loadBundle(locale: string): Record<string, unknown> {
  const dir = join(SRC, '..', 'messages', locale);
  const bundle: Record<string, unknown> = {};
  for (const file of readdirSync(dir).filter((f) => f.endsWith('.json'))) {
    bundle[camel(file.replace(/\.json$/, ''))] = JSON.parse(readFileSync(join(dir, file), 'utf8'));
  }
  return bundle;
}
const existsIn = (bundle: Record<string, unknown>, path: string) =>
  path.split('.').every((segment, i, all) => {
    const node = all.slice(0, i).reduce<unknown>((n, k) => (n as Record<string, unknown> | undefined)?.[k], bundle);
    return typeof node === 'object' && node !== null && Object.hasOwn(node, segment);
  });

describe('sub-key message scopes', () => {
  const pageOf = (route: string) =>
    pages.find((page) => new RegExp(`<RouteMessages\\s+route="${route}"`).test(readFileSync(page, 'utf8')));

  it.each(Object.keys(ROUTE_NAMESPACES))('%s: every key its screen reads is inside its scope', (route) => {
    const page = pageOf(route);
    expect(page, `no page renders route "${route}"`).toBeDefined();
    const entries: readonly string[] = ROUTE_NAMESPACES[route as keyof typeof ROUTE_NAMESPACES];
    // An entry serves a need when it is that path or an ancestor of it (a bare namespace serves all of it).
    // A need whose path is not in the bundle at all (the finder merges same-named `t` variables of one
    // file, so it can pair a key with a prefix it never had) cannot be shipped and is skipped.
    const fa = loadBundle('fa');
    const missing = needsReachableFrom(page as string).filter(
      (need) => existsIn(fa, need) && !entries.some((entry) => serves(entry, need)),
    );
    expect(missing, 'keys read by the screen but outside its scope').toEqual([]);
  });

  it.each(Object.keys(ROUTE_NAMESPACES))('%s: no sub-key entry is dead or misspelled', (route) => {
    const page = pageOf(route) as string;
    const needs = needsReachableFrom(page);
    const fa = loadBundle('fa');
    for (const entry of ROUTE_NAMESPACES[route as keyof typeof ROUTE_NAMESPACES] as readonly string[]) {
      if (!entry.includes('.')) continue;
      expect(
        needs.some((need) => serves(entry, need)),
        `${route}: "${entry}" is not used by any component of the screen`,
      ).toBe(true);
      expect(existsIn(fa, entry), `${route}: "${entry}" does not exist in messages/fa`).toBe(true);
    }
  });

  it('home leaves the heavy mode-specific namespaces trimmed to the subtrees it renders', () => {
    const home = ROUTE_NAMESPACES.home as readonly string[];
    // Whole namespaces would silently bring the 100+ KB back.
    for (const heavy of ['home', 'menopause', 'companions', 'teen', 'fertility', 'learn', 'checkups', 'loss']) {
      expect(home, `${heavy} must stay sub-key scoped`).not.toContain(heavy);
    }
  });
});

describe('messageNeedsOf (the key finder behind the scope test)', () => {
  const needs = (code: string) => messageNeedsOf(code).sort();

  it('reads literal keys under the namespace prefix, for t, t.rich, t.raw, t.has and t.markup', () => {
    expect(
      needs(`const t = useTranslations('a.b');
        t('x'); t.rich('y', {}); t.raw('z'); t.has('w'); t.markup('v');`),
    ).toEqual(['a.b.v', 'a.b.w', 'a.b.x', 'a.b.y', 'a.b.z']);
  });

  it('takes the root namespace when none is given and awaits getTranslations', () => {
    expect(needs(`const t = await getTranslations(); t('home.title')`)).toEqual(['home.title']);
    expect(needs(`const t = await getTranslations({ locale, namespace: 'foo' }); t('k')`)).toEqual(['foo.k']);
  });

  it('a template literal needs the subtree before its first hole', () => {
    expect(needs('const t = useTranslations("n"); t(`items.${i}.title`)')).toEqual(['n.items']);
    expect(needs('const t = useTranslations("n"); t(`prefix.${i}`)')).toEqual(['n.prefix']);
    expect(needs('const t = useTranslations("n"); t(`plain.key`)')).toEqual(['n.plain.key']);
  });

  it('a ternary of two literals needs both, any other expression needs the whole prefix', () => {
    expect(needs(`const t = useTranslations('n'); t(ok ? 'a' : 'b', { x: 1 })`)).toEqual(['n.a', 'n.b']);
    expect(needs(`const t = useTranslations('n.m'); t(tab.key)`)).toEqual(['n.m']);
    expect(needs(`const t = useTranslations('n.m'); t(ok ? 'a' : other)`)).toEqual(['n.m']);
  });

  it('t handed to something else can read anything below its prefix', () => {
    expect(needs(`const t = useTranslations('n.m'); format(t, 1); t('a')`)).toEqual(['n.m', 'n.m.a']);
    expect(needs(`const t = useTranslations('n.m'); return { t };`)).toEqual(['n.m']);
  });

  it('t handed to a component of the same file is not widening, to an imported one it is', () => {
    const local = `function Row({ t }: { t: T }) { return t('row.title'); }
      function Page() { const t = useTranslations('n'); return <Row t={t} />; }`;
    expect(needs(local)).toEqual(['n.row.title']);
    const imported = `function Page() { const t = useTranslations('n'); return <Outside t={t} />; }`;
    expect(needs(imported)).toEqual(['n']);
  });

  it('ignores calls of other functions that merely end in t', () => {
    expect(needs(`const t = useTranslations('n'); format('x'); obj.t('y'); t('z')`)).toEqual(['n.z']);
  });
});

describe('stripComments', () => {
  it('removes line and block comments', () => {
    expect(stripComments("a(); // gone\n/* gone\ntoo */ b();").replace(/\s+/g, ' ').trim()).toBe('a(); b();');
  });

  it('a `/*` inside a line comment does not swallow the code that follows (the P-04 blind spot)', () => {
    const code = "// the `/fertility/*` reads too\nconst t = useTranslations('a');\nt('kept');\n/** doc */\nt('also');";
    expect(messageNeedsOf(stripComments(code)).sort()).toEqual(['a.also', 'a.kept']);
  });

  it('keeps comment markers that sit inside strings and template literals', () => {
    expect(stripComments("const u = 'http://x.y/*z'; // c")).toBe("const u = 'http://x.y/*z'; ");
    expect(stripComments('const u = `a // b`;')).toBe('const u = `a // b`;');
  });
});
