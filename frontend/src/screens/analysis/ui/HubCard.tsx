'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, useDirection, useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
import { PlusBadge, PlusGate } from '@/shared/ui/plus-gate';

interface HubCardProps {
  title: string;
  /** Small line under the title («۶ سیکل اخیر»). */
  sub?: ReactNode;
  /** Detail screen; the whole card becomes one link. Omitted for a card without a screen yet. */
  href?: string;
  /** A Plus section: shows the «پلاس» badge. */
  plus?: boolean;
  /** Plus and the user isn't entitled: the body is a blurred teaser behind the lock (the server sent no data). */
  locked?: boolean;
  className?: string;
  children: ReactNode;
}

/**
 * One An_Hub card: flat surface, title 15/800 + chevron, body. An open card
 * is one link to its detail screen; a locked Plus card is not a link (it holds
 * the paywall button instead, and nesting it in a link would be invalid).
 */
export function HubCard({ title, sub, href, plus, locked, className, children }: HubCardProps) {
  const tPlus = useTranslations('plus.gate');
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  const head = (
    <span className="an-card-head">
      <span className="an-card-titles">
        <span className="an-card-title-row">
          <b className="an-card-title">{title}</b>
          {plus ? <PlusBadge label={tPlus('label')} /> : null}
        </span>
        {sub ? <span className="an-card-sub">{sub}</span> : null}
      </span>
      {href && !locked ? (
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} strokeWidth={2} className="an-card-chev" />
      ) : null}
    </span>
  );

  if (locked) {
    return (
      <section className={clsx('nb-card an-card is-locked', className)} aria-label={title}>
        {head}
        <PlusGate locked label={tPlus('label')} lockedText={tPlus('lockedText')} onUnlock={() => router.push('/plus')}>
          {children}
        </PlusGate>
      </section>
    );
  }
  if (href) {
    return (
      <Link href={href} className={clsx('nb-card an-card is-link', className)}>
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
  return <p className="an-card-note">{children}</p>;
}
