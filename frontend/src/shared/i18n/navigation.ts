import { createNavigation } from 'next-intl/navigation';

import { routing } from './routing';

/**
 * Locale-aware navigation helpers. Always import `Link`, `useRouter`, etc.
 * from here (via `@/shared/i18n`) instead of `next/link` / `next/navigation`,
 * so the active locale prefix is preserved automatically.
 */
export const { Link, redirect, usePathname, useRouter, getPathname } =
  createNavigation(routing);
