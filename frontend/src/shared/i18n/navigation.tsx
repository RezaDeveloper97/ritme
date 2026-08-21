'use client';

import NextLink from 'next/link';
import {
  usePathname as useNextPathname,
  useRouter as useNextRouter,
} from 'next/navigation';
import { useLocale } from 'next-intl';
import type { ComponentProps } from 'react';

import type { Locale } from './registry';

/**
 * Locale-aware navigation helpers. Always import `Link`, `useRouter` and
 * `usePathname` from here (via `@/shared/i18n`) instead of `next/link` /
 * `next/navigation`, so the active locale prefix is preserved automatically.
 *
 * These are hand-rolled rather than next-intl's `createNavigation` because the
 * set of locales is not known at build time — an admin can add one at any
 * moment (CLAUDE.md §6). Everything here works off the *active* locale, which
 * the server already resolved, so no locale list is needed on the client.
 */

/** The cookie the middleware reads to decide where an unprefixed URL lands. */
export const LOCALE_COOKIE = 'NEXT_LOCALE';

/** `/home` + `fa` -> `/fa/home`. External and hash hrefs pass through. */
export function localizeHref(href: string, locale: Locale): string {
  if (!href.startsWith('/')) return href;

  return href === '/' ? `/${locale}` : `/${locale}${href}`;
}

/** `/fa/home` -> `/home`. Safe to call on an already-stripped path. */
export function stripLocale(pathname: string, locale: Locale): string {
  if (pathname === `/${locale}`) return '/';

  return pathname.startsWith(`/${locale}/`)
    ? pathname.slice(locale.length + 1)
    : pathname;
}

/** The current path WITHOUT its locale prefix — what you'd pass to `Link`. */
export function usePathname(): string {
  return stripLocale(useNextPathname(), useLocale());
}

interface NavigateOptions {
  /** Navigate under a different locale (used by the language switcher). */
  locale?: Locale;
  scroll?: boolean;
}

export function useRouter() {
  const router = useNextRouter();
  const active = useLocale();

  const resolve = (href: string, options?: NavigateOptions) =>
    localizeHref(href, options?.locale ?? active);

  return {
    push: (href: string, options?: NavigateOptions) =>
      router.push(resolve(href, options), { scroll: options?.scroll }),
    replace: (href: string, options?: NavigateOptions) =>
      router.replace(resolve(href, options), { scroll: options?.scroll }),
    prefetch: (href: string, options?: NavigateOptions) =>
      router.prefetch(resolve(href, options)),
    back: () => router.back(),
    forward: () => router.forward(),
    refresh: () => router.refresh(),
  };
}

type LinkProps = Omit<ComponentProps<typeof NextLink>, 'href'> & {
  href: string;
  locale?: Locale;
};

export function Link({ href, locale, ...props }: LinkProps) {
  const active = useLocale();

  return <NextLink href={localizeHref(href, locale ?? active)} {...props} />;
}
