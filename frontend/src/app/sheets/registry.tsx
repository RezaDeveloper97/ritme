'use client';

import dynamic from 'next/dynamic';
import { useTranslations } from 'next-intl';
import type { ComponentType } from 'react';

import { isInfoGroup } from '@/entities/info';
import { ArticleSheet } from '@/screens/article';
import type { SheetContentProps, SheetSize } from '@/shared/sheet';

export interface SheetDefinition {
  size: SheetSize;
  /**
   * Renders the sheet's heading. A component rather than a message key so each
   * title stays inside its own screen's namespace and keeps next-intl's
   * compile-time key checking (a plain string would have to be `string`, which
   * discards it). Omit for content that titles itself, such as an article.
   */
  Title?: ComponentType<SheetContentProps>;
  Component: ComponentType<SheetContentProps>;
}

// ── Headings ──────────────────────────────────────────────────
// The topic reaches these from the URL, so it is validated, not trusted.
function InfoTitle({ arg }: SheetContentProps) {
  const t = useTranslations('profileInfo');
  if (!arg || !isInfoGroup(arg)) return null;
  return <>{t(`${arg}.title`)}</>;
}

function NotificationsTitle() {
  return <>{useTranslations('notifications')('title')}</>;
}

function PhaseTitle() {
  return <>{useTranslations('phaseDetails')('title')}</>;
}

function ArticlesTitle() {
  return <>{useTranslations('articles')('title')}</>;
}

function PersonalTitle() {
  return <>{useTranslations('profileEdit')('personal.title')}</>;
}

function HealthTitle() {
  return <>{useTranslations('profileEdit')('health.title')}</>;
}

function RemindersTitle() {
  return <>{useTranslations('reminders')('title')}</>;
}

function LanguageTitle() {
  return <>{useTranslations('profile')('rows.language')}</>;
}

/**
 * Every screen in this app that is neither a bottom-nav tab nor part of signing
 * up. They have no routes of their own: each is a panel that rises over
 * whatever the user was already looking at, addressed by `?sheet=<id>` and
 * dismissed by the back button. See `shared/sheet`.
 *
 * This map lives in the `app` layer because it is the one place allowed to
 * reach into `screens` — `shared/sheet` only ever deals in ids (FSD §3.1).
 *
 * Sheets load on demand (`ssr: false`): none is part of the first paint, and
 * the host that renders them is client-only regardless.
 */
export const SHEET_REGISTRY: Record<string, SheetDefinition> = {
  /** Privacy / terms / about / help — `arg` is the topic. */
  info: {
    size: 'full',
    Title: InfoTitle,
    Component: dynamic(() => import('@/screens/profile-info').then((m) => m.InfoSheet), {
      ssr: false,
    }),
  },

  notifications: {
    size: 'full',
    Title: NotificationsTitle,
    Component: dynamic(
      () => import('@/screens/profile-notifications').then((m) => m.NotificationsSheet),
      { ssr: false },
    ),
  },

  /** Educational detail for the user's current cycle sub-phase. */
  phase: {
    size: 'full',
    Title: PhaseTitle,
    Component: dynamic(
      () => import('@/screens/phase-details').then((m) => m.PhaseDetailsSheet),
      { ssr: false },
    ),
  },

  articles: {
    size: 'full',
    Title: ArticlesTitle,
    Component: dynamic(() => import('@/screens/articles').then((m) => m.ArticlesSheet), {
      ssr: false,
    }),
  },

  /**
   * One article — `arg` is its slug. The article's own headline is the heading,
   * so it declares no `Title`.
   *
   * Statically imported rather than lazy: it is the sheet most often opened
   * from *inside* another sheet, where a loading gap reads as a stall rather
   * than as a transition.
   */
  article: {
    size: 'full',
    Component: ArticleSheet,
  },

  // ── Profile editors ──
  // Not linked from the profile screen today (the same fields are edited inline
  // via QuickEditSheet), but registered so the rows can be switched back on
  // without re-plumbing anything.
  personal: {
    size: 'full',
    Title: PersonalTitle,
    Component: dynamic(
      () => import('@/screens/profile-personal').then((m) => m.ProfilePersonalSheet),
      { ssr: false },
    ),
  },

  health: {
    size: 'full',
    Title: HealthTitle,
    Component: dynamic(
      () => import('@/screens/profile-health').then((m) => m.ProfileHealthSheet),
      { ssr: false },
    ),
  },

  reminders: {
    size: 'full',
    Title: RemindersTitle,
    Component: dynamic(
      () => import('@/screens/profile-reminders').then((m) => m.RemindersSheet),
      { ssr: false },
    ),
  },

  /** App language. The list is fetched, so this can't be a two-way toggle. */
  language: {
    size: 'half',
    Title: LanguageTitle,
    Component: dynamic(
      () => import('@/screens/profile-language').then((m) => m.LanguageSheet),
      { ssr: false },
    ),
  },
};

export function sheetDefinition(id: string): SheetDefinition | null {
  return Object.prototype.hasOwnProperty.call(SHEET_REGISTRY, id)
    ? SHEET_REGISTRY[id]
    : null;
}
