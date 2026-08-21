import type { AbstractIntlMessages } from 'next-intl';

import { env } from '@/shared/config';

import {
  DEFAULT_LOCALE,
  getBundledMessages,
  isBundledLocale,
} from './bundled';
import { getDefaultLocale, type Locale } from './registry';

/**
 * Assembles the message bundle a locale renders with.
 *
 * Three layers, each one filling the gaps in the one before:
 *
 *   1. the bundled DEFAULT locale — guarantees every key resolves to *something*
 *      even during a backend outage or a build-time pre-render;
 *   2. the bundled locale itself, when it is one of the compiled-in ones;
 *   3. the backend's bundle for the locale, which is authoritative — it carries
 *      admin edits and is the only source for a language added after the last
 *      frontend deploy.
 *
 * A key nobody has translated therefore renders the default language's text
 * rather than a blank, which matches how the backend resolves content.
 */

/** How long a fetched bundle is reused. Matches the locale registry's TTL. */
const TTL_MS = 5 * 60_000;

const cache = new Map<Locale, { value: AbstractIntlMessages; expires: number }>();

type MessageTree = Record<string, unknown>;

/**
 * The backend names namespaces after their files (`phase-details`), while the
 * app addresses them in camelCase (`t('phaseDetails')`).
 */
function toNamespaceKey(fileName: string): string {
  return fileName.replace(/-([a-z0-9])/g, (_, char: string) => char.toUpperCase());
}

/** Recursive merge where `override`'s non-empty leaves win. */
function merge(base: MessageTree, override: MessageTree): MessageTree {
  const result: MessageTree = { ...base };

  for (const [key, value] of Object.entries(override)) {
    const existing = result[key];

    if (
      value !== null &&
      typeof value === 'object' &&
      !Array.isArray(value) &&
      existing !== null &&
      typeof existing === 'object' &&
      !Array.isArray(existing)
    ) {
      result[key] = merge(existing as MessageTree, value as MessageTree);
      continue;
    }

    if (value === undefined || value === null || value === '') continue;

    result[key] = value;
  }

  return result;
}

async function fetchRemoteMessages(locale: Locale): Promise<MessageTree | null> {
  try {
    const response = await fetch(
      `${env.apiBaseUrl}/languages/${encodeURIComponent(locale)}/messages`,
      {
        headers: { Accept: 'application/json' },
        signal: AbortSignal.timeout(5_000),
      },
    );

    if (!response.ok) return null;

    const payload = (await response.json()) as {
      data?: { messages?: Record<string, unknown> };
    };
    const messages = payload.data?.messages;

    if (!messages || typeof messages !== 'object') return null;

    return Object.fromEntries(
      Object.entries(messages).map(([name, value]) => [toNamespaceKey(name), value]),
    );
  } catch {
    // An unreachable or slow backend must never blank the UI — the bundled
    // layers below already cover every key.
    return null;
  }
}

export async function getLocaleMessages(locale: Locale): Promise<AbstractIntlMessages> {
  const now = Date.now();
  const cached = cache.get(locale);

  if (cached && cached.expires > now) return cached.value;

  const fallbackLocale = isBundledLocale(await getDefaultLocale())
    ? ((await getDefaultLocale()) as typeof DEFAULT_LOCALE)
    : DEFAULT_LOCALE;

  let messages = getBundledMessages(fallbackLocale) as MessageTree;

  if (isBundledLocale(locale) && locale !== fallbackLocale) {
    messages = merge(messages, getBundledMessages(locale) as MessageTree);
  }

  const remote = await fetchRemoteMessages(locale);

  if (remote) {
    messages = merge(messages, remote);
  }

  const value = messages as unknown as AbstractIntlMessages;
  cache.set(locale, { value, expires: now + TTL_MS });

  return value;
}
