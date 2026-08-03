'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale } from 'next-intl';
import { useTransition } from 'react';

import { type Locale, routing, usePathname, useRouter } from '@/shared/i18n';

interface SwitchLocale {
  /** The currently active locale. */
  locale: Locale;
  /** All supported locales, in display order. */
  locales: readonly Locale[];
  /** `true` while a locale change is navigating. */
  isPending: boolean;
  /** Re-render the current route under `next` (no-op if already active). */
  switchLocale: (next: Locale) => void;
}

/**
 * Switch the app's display language while staying on the current screen.
 *
 * next-intl keeps the locale in the URL path (CLAUDE.md §6), so switching is a
 * locale-scoped navigation to the same pathname — not client state. The
 * pathname from `usePathname` is already locale-stripped, so we re-issue it
 * under the target locale.
 */
export function useSwitchLocale(): SwitchLocale {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const [isPending, startTransition] = useTransition();

  const switchLocale = (next: Locale) => {
    if (next === locale) return;
    // Every API payload is server-localized: the client sends Accept-Language
    // from the document's `lang` (shared/api), so a cached response belongs to
    // the locale it was fetched under. Switching is a client-side navigation,
    // so the cache survives it — without this the user would keep reading
    // Persian cycle tips and daily messages under /en until each key goes
    // stale. A deliberate language change is rare enough that refetching
    // everything is the right trade.
    queryClient.clear();
    startTransition(() => {
      router.replace(pathname, { locale: next });
    });
  };

  return { locale, locales: routing.locales, isPending, switchLocale };
}
