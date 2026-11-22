import type { AbstractIntlMessages } from 'next-intl';

/** A top-level message namespace (`home`, `logPeriod`, …) — one per message file. */
export type MessageNamespace = keyof IntlMessages & string;

/**
 * What a client subtree reads from one namespace: the whole namespace (`'common'`) or only a subtree
 * of it (`'home.nb.predictions'`, `'menopause.home'`). Sub-key scopes keep the big, mode-specific
 * namespaces (`home`, `menopause`, `companions`, …) from shipping every string to every route
 * (T-M12-29). The path is dot-separated, exactly like the keys `t('…')` takes.
 */
export type MessageScope = MessageNamespace | `${MessageNamespace}.${string}`;

/** The top-level namespace a scope entry belongs to (`'home.nb.log'` → `'home'`). */
export function scopeNamespace(scope: string): string {
  const dot = scope.indexOf('.');
  return dot === -1 ? scope : scope.slice(0, dot);
}

function isPlainObject(value: unknown): value is AbstractIntlMessages {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * The subset of a locale's bundle that a client subtree actually reads.
 *
 * Every message a `NextIntlClientProvider` receives is serialised into the page
 * (HTML + RSC payload). Handing it the whole bundle shipped all namespaces
 * (~50 KB, 14.5 KB gzip for `fa`) with every route (perf baseline §1.3, §3 #9),
 * so providers get only what their subtree uses: whole namespaces, or the
 * subtrees named by a dotted {@link MessageScope}. A namespace or path missing
 * from the bundle is simply skipped; next-intl then reports the missing key
 * exactly as it would have before. A whole-namespace entry wins over a sub-key
 * entry of the same namespace.
 */
export function pickNamespaces(
  messages: AbstractIntlMessages,
  scopes: readonly MessageScope[],
): AbstractIntlMessages {
  const picked: AbstractIntlMessages = {};
  // Overlapping entries (`home.nb` + `home.nb.log`, or `home` + `home.nb`): the broader one wins, so
  // no entry is ever written into a subtree that an ancestor already shares by reference.
  const unique = [...new Set(scopes)];
  const roots = unique.filter(
    (scope) => !unique.some((other) => other !== scope && scope.startsWith(`${other}.`)),
  );
  for (const scope of roots) {
    const path = scope.split('.');
    // Walk down the source tree; stop quietly where the path does not exist.
    let source: unknown = messages;
    for (const segment of path) {
      if (!isPlainObject(source) || !Object.hasOwn(source, segment)) {
        source = undefined;
        break;
      }
      source = source[segment];
    }
    if (source === undefined) continue;
    // Rebuild the same path in the result and attach the subtree at its end.
    let target: AbstractIntlMessages = picked;
    for (const segment of path.slice(0, -1)) {
      const next = target[segment];
      if (isPlainObject(next)) target = next;
      else {
        const created: AbstractIntlMessages = {};
        target[segment] = created;
        target = created;
      }
    }
    target[path[path.length - 1] as string] = source as AbstractIntlMessages[string];
  }
  return picked;
}
