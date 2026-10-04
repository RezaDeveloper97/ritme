'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, useDirection, useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
import { PlusBadge, PlusGate } from '@/shared/ui/plus-gate';

interface PostCardProps {
  title: string;
  sub?: ReactNode;
  /** Detail screen; the whole card becomes one link. */
  href?: string;
  plus?: boolean;
  /** Plus and not entitled: the body is a blurred teaser behind the paywall button. */
  locked?: boolean;
  className?: string;
  children: ReactNode;
}

/**
 * One An_Hub_Post card — the An_Hub card look (`an-card`: flat surface,
 * title 15/800 + chevron). A locked Plus card is not a link: it holds the
 * paywall button, and nesting a button in a link is invalid.
 */
export function PostCard({ title, sub, href, plus, locked, className, children }: PostCardProps) {
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
