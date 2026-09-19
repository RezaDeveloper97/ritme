import type { AbstractIntlMessages } from 'next-intl';

/** A top-level message namespace (`home`, `logPeriod`, …) — one per message file. */
export type MessageNamespace = keyof IntlMessages & string;

/**
 * The subset of a locale's bundle that a client subtree actually reads.
 *
 * Every message a `NextIntlClientProvider` receives is serialised into the page
 * (HTML + RSC payload). Handing it the whole bundle shipped all namespaces
 * (~50 KB, 14.5 KB gzip for `fa`) with every route (perf baseline §1.3, §3 #9),
 * so providers get only the namespaces their subtree uses. A namespace missing
 * from the bundle is simply skipped; next-intl then reports the missing key
 * exactly as it would have before.
 */
export function pickNamespaces(
  messages: AbstractIntlMessages,
  namespaces: readonly MessageNamespace[],
): AbstractIntlMessages {
  const picked: AbstractIntlMessages = {};
  for (const namespace of namespaces) {
    if (Object.hasOwn(messages, namespace)) picked[namespace] = messages[namespace];
  }
  return picked;
}
