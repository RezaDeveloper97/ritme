import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { describe, expect, it } from 'vitest';

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

interface FileInfo {
  deps: string[];
  namespaces: string[];
}
const infoCache = new Map<string, FileInfo>();

function infoOf(file: string): FileInfo {
  const cached = infoCache.get(file);
  if (cached) return cached;
  // Commented-out code (e.g. a temporarily hidden widget) must not count.
  const source = readFileSync(file, 'utf8')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '');
  const deps = [...source.matchAll(/(?:from|import)\s*\(?\s*['"]([^'"]+)['"]/g)]
    .map((m) => resolveImport(file, m[1]))
    .filter((dep): dep is string => dep !== null);
  const namespaces = [
    ...source.matchAll(/(?:useTranslations|getTranslations)\(\s*['"]([A-Za-z][A-Za-z0-9]*)/g),
  ].map((m) => m[1]);
  const info = { deps, namespaces };
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

function pageFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return pageFiles(path);
    return entry.name === 'page.tsx' ? [path] : [];
  });
}

const sorted = (list: readonly string[]) => [...new Set(list)].sort();

describe('message scopes', () => {
  it('the layout ships exactly the namespaces the shell and every sheet use', () => {
    expect(sorted(SHELL_NAMESPACES)).toEqual(
      namespacesReachableFrom(join(LOCALE_DIR, 'layout.tsx')),
    );
  });

  const pages = pageFiles(LOCALE_DIR);

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
      expect(sorted(ROUTE_NAMESPACES[route as keyof typeof ROUTE_NAMESPACES])).toEqual(used);
    },
  );

  it('every declared route list is used by a page', () => {
    const used = pages
      .map((page) => /<RouteMessages\s+route="([A-Za-z]+)"/.exec(readFileSync(page, 'utf8'))?.[1])
      .filter(Boolean);
    expect(sorted(Object.keys(ROUTE_NAMESPACES))).toEqual(sorted(used as string[]));
  });
});
