'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, useDirection, useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
import { PlusBadge, PlusGate } from '@/shared/ui/plus-gate';

interface TtcCardProps {
  title: string;
  sub?: ReactNode;
  href?: string;
  plus?: boolean;
  locked?: boolean;
  className?: string;
  children: ReactNode;
}

/**
 * One An_Hub_TTC card — the same flat card, title row and Plus lock as the
 * cycle hub (`.an-card` styles, B-N3-08). An open card is one link to its
 * detail; a locked Plus card holds the paywall button instead.
 */
export function TtcCard({ title, sub, href, plus, locked, className, children }: TtcCardProps) {
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
