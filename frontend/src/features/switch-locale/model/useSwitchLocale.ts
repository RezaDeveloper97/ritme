'use client';

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
  const [isPending, startTransition] = useTransition();

  const switchLocale = (next: Locale) => {
    if (next === locale) return;
    startTransition(() => {
      router.replace(pathname, { locale: next });
    });
  };

  return { locale, locales: routing.locales, isPending, switchLocale };
}
