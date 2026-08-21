'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale } from 'next-intl';
import { useTransition } from 'react';

import { useLanguages, type Language } from '@/entities/language';
import { LOCALE_COOKIE, type Locale, usePathname, useRouter } from '@/shared/i18n';

interface SwitchLocale {
  /** The currently active locale. */
  locale: Locale;
  /** Every language the app offers, in display order. */
  languages: Language[];
  /** `true` while a locale change is navigating. */
  isPending: boolean;
  /** Re-render the current route under `next` (no-op if already active). */
  switchLocale: (next: Locale) => void;
}

/**
 * Switch the app's display language while staying on the current screen.
 *
 * The locale lives in the URL path (CLAUDE.md §6), so switching is a
 * locale-scoped navigation to the same pathname — not client state. The
 * pathname from `usePathname` is already locale-stripped, so we re-issue it
 * under the target locale.
 *
 * The list of languages is fetched, not hardcoded: an admin can add one at any
 * time and it has to appear here without a frontend release.
 */
export function useSwitchLocale(): SwitchLocale {
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();
  const queryClient = useQueryClient();
  const { data: languages } = useLanguages();
  const [isPending, startTransition] = useTransition();

  const switchLocale = (next: Locale) => {
    if (next === locale) return;

    // Remember the choice for URLs that carry no locale prefix (a shared link,
    // the PWA start_url); the middleware reads this cookie when it has to pick
    // one. One year: a language preference is not session state.
    document.cookie = `${LOCALE_COOKIE}=${next};path=/;max-age=31536000;samesite=lax`;

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

  return { locale, languages: languages ?? [], isPending, switchLocale };
}
