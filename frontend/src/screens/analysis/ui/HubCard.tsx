'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, useDirection, useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { Icon, type IconName } from '@/shared/ui';
import { PlusBadge } from '@/shared/ui/plus-gate';
import '@/shared/ui/styles/an.css';

/** Accent of a card's icon tile — one of the Night & Bloom `.nb-tone-*` tones. */
export type HubTone = 'brand' | 'data' | 'warm' | 'bloom' | 'period';

interface HubCardProps {
  title: string;
  /** The card's icon, drawn in a soft tile of `tone` so the hub scans by shape, not only by text. */
  icon: IconName;
  tone: HubTone;
  /** Small line under the title («۶ سیکل اخیر»). */
  sub?: ReactNode;
  /** Detail screen; the whole card becomes one link. Omitted for a card without a screen yet. */
  href?: string;
  /** Autopilot key of this card when the href alone is not unique. */
  pilotKey?: string;
  /** A Plus section: shows the «پلاس» badge. */
  plus?: boolean;
  /**
   * Plus and the user isn't entitled (the server sent no data): the card collapses to one line saying what the
   * analysis shows, plus the unlock button — no blurred fake chart.
   */
  locked?: boolean;
  /** The one-line pitch of a locked card («ببین حالت در هر فاز چطور فرق می‌کند»). */
  lockedBody?: string;
  className?: string;
  children?: ReactNode;
}

/**
 * One An_Hub card: flat surface, a tone icon tile + title 15/800 + a chevron chip, then the body. An open card
 * is one link to its detail screen; a locked Plus card is not a link (it holds the paywall button instead, and
 * nesting it in a link would be invalid).
 */
export function HubCard({
  title,
  icon,
  tone,
  sub,
  href,
  plus,
  locked,
  lockedBody,
  className,
  children,
  pilotKey,
}: HubCardProps) {
  const tPlus = useTranslations('plus.gate');
  const t = useTranslations('analysis.hub');
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const head = (
    <span className="an-card-head">
      <span className={clsx('an-card-icon', `nb-tone-${tone}`)} aria-hidden>
        <Icon name={icon} size={18} strokeWidth={2} />
      </span>
      <span className="an-card-titles">
        <span className="an-card-title-row">
          <b className="an-card-title">{title}</b>
          {plus ? <PlusBadge label={tPlus('label')} /> : null}
        </span>
        {sub ? <span className="an-card-sub">{sub}</span> : null}
      </span>
      {href && !locked ? (
        <span className="an-card-go" aria-hidden>
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} strokeWidth={2.2} />
        </span>
      ) : null}
    </span>
  );

  if (locked) {
    return (
      <section className={clsx('nb-card an-card is-locked', className)} aria-label={title}>
        {head}
        {lockedBody ? <p className="an-card-note">{lockedBody}</p> : null}
        <button type="button" className="an-unlock" onClick={() => router.push('/plus')}>
          <Icon name="lock" size={15} strokeWidth={2} />
          {t('unlock')}
          <span className="sr-only">{tPlus('lockedText')}</span>
        </button>
      </section>
    );
  }
  if (href) {
    return (
      <Link
        href={href}
        className={clsx('nb-card an-card is-link', className)}
        {...pilot(`analysis.card:${pilotKey ?? href.split('?')[0].split('/').filter((s) => s && s !== 'analysis').join('.')}`)}
      >
        {head}
        {children}
      </Link>
    );
  }
  return (
    <section className={clsx('nb-card an-card', className)} aria-label={title}>
      {head}
      {children}
    </section>
  );
}

/** The quiet «not enough data yet» line inside a card. */
export function NotReady({ children }: { children: ReactNode }) {
  return <p className="an-card-note is-pending">{children}</p>;
}
