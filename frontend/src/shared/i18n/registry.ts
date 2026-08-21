import { env } from '@/shared/config';

import {
  BUNDLED_DIRECTIONS,
  BUNDLED_LOCALES,
  BUNDLED_NAMES,
  DEFAULT_LOCALE,
} from './bundled';

/**
 * A locale is any string the backend says the app ships — not a fixed union.
 * Admins add languages in the admin panel (CLAUDE.md §6); the frontend must
 * never enumerate them.
 */
export type Locale = string;

export interface LocaleInfo {
  code: Locale;
  name: string;
  englishName: string;
  direction: 'rtl' | 'ltr';
  isDefault: boolean;
}

export interface LocaleRegistry {
  locales: LocaleInfo[];
  codes: Locale[];
  defaultLocale: Locale;
}

/** What we answer with before the API has ever been reached. */
const FALLBACK: LocaleRegistry = {
  locales: BUNDLED_LOCALES.map((code) => ({
    code,
    name: BUNDLED_NAMES[code],
    englishName: BUNDLED_NAMES[code],
    direction: BUNDLED_DIRECTIONS[code],
    isDefault: code === DEFAULT_LOCALE,
  })),
  codes: [...BUNDLED_LOCALES],
  defaultLocale: DEFAULT_LOCALE,
};

/** How long a fetched registry is reused before we ask the API again. */
const TTL_MS = 5 * 60_000;

// Module-scope so one server/edge worker makes at most one request per TTL,
// no matter how many pages and middleware invocations read the list.
let cache: { value: LocaleRegistry; expires: number } | null = null;
let inFlight: Promise<LocaleRegistry> | null = null;

interface LanguagesResponse {
  data?: {
    default?: string;
    languages?: {
      code?: string;
      name?: string;
      english_name?: string;
      direction?: string;
      is_default?: boolean;
    }[];
  };
}

function parse(payload: LanguagesResponse): LocaleRegistry | null {
  const rows = payload.data?.languages;

  if (!Array.isArray(rows) || rows.length === 0) return null;

  const locales = rows
    .filter((row): row is { code: string } & typeof row => Boolean(row.code))
    .map<LocaleInfo>((row) => ({
      code: row.code,
      name: row.name ?? row.code,
      englishName: row.english_name ?? row.code,
      direction: row.direction === 'rtl' ? 'rtl' : 'ltr',
      isDefault: Boolean(row.is_default),
    }));

  if (locales.length === 0) return null;

  const defaultLocale =
    payload.data?.default ??
    locales.find((locale) => locale.isDefault)?.code ??
    locales[0]!.code;

  return { locales, codes: locales.map((locale) => locale.code), defaultLocale };
}

/**
 * The languages the app currently ships.
 *
 * Read at runtime rather than baked in at build time: that is what lets a
 * language added in the admin panel appear in the app — URL prefix, text
 * direction, picker entry — without redeploying the frontend. Any failure
 * degrades to the bundled locales instead of breaking the page.
 */
export async function getLocaleRegistry(): Promise<LocaleRegistry> {
  const now = Date.now();

  if (cache && cache.expires > now) return cache.value;

  // Collapse concurrent misses (a burst of requests after the TTL expires)
  // onto a single fetch.
  inFlight ??= (async () => {
    try {
      const response = await fetch(`${env.apiBaseUrl}/languages`, {
        headers: { Accept: 'application/json' },
        signal: AbortSignal.timeout(3_000),
      });

      if (!response.ok) throw new Error(`HTTP ${response.status}`);

      const parsed = parse((await response.json()) as LanguagesResponse);

      if (!parsed) throw new Error('empty language list');

      cache = { value: parsed, expires: Date.now() + TTL_MS };
      return parsed;
    } catch {
      // Serve the last good answer if we have one; otherwise the bundled
      // floor. Cache the fallback briefly so an outage doesn't mean a failed
      // fetch on every single request.
      const value = cache?.value ?? FALLBACK;
      cache = { value, expires: Date.now() + 30_000 };
      return value;
    } finally {
      inFlight = null;
    }
  })();

  return inFlight;
}

export async function getSupportedLocales(): Promise<Locale[]> {
  return (await getLocaleRegistry()).codes;
}

export async function getDefaultLocale(): Promise<Locale> {
  return (await getLocaleRegistry()).defaultLocale;
}

export async function isSupportedLocale(value: unknown): Promise<boolean> {
  if (typeof value !== 'string' || value === '') return false;

  return (await getSupportedLocales()).includes(value);
}

/** Text direction for a locale; unknown locales read LTR. */
export async function getDirection(locale: Locale): Promise<'rtl' | 'ltr'> {
  const { locales } = await getLocaleRegistry();

  return locales.find((entry) => entry.code === locale)?.direction ?? 'ltr';
}

/** Resolve an arbitrary value to a locale we can actually render. */
export async function resolveLocale(value: unknown): Promise<Locale> {
  const { codes, defaultLocale } = await getLocaleRegistry();

  return typeof value === 'string' && codes.includes(value) ? value : defaultLocale;
}
