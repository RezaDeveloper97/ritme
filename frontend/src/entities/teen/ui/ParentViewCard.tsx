'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { isEmptyParentView } from '../lib/parent-view';
import type { TeenParentView } from '../model/types';

interface ParentViewCardProps {
  /** The teen's name as the parent sees it. */
  name: string;
  view: TeenParentView;
  /** Next to the name, e.g. the «فقط دیدنی» pill on the parent's side. */
  badge?: ReactNode;
  /** `preview` = the dashed «مادرت این را می‌بیند» card on the teen's screen. */
  variant?: 'preview' | 'card';
  footer?: ReactNode;
  className?: string;
}

/**
 * The read-only teen card (nbl_Teen_Parent): the name, a week bucket for the
 * next period (never a date), whether the school kit is ready, and the note
 * she chose to share. One component for both sides, so the teen's live
 * preview is exactly what her parent's card shows.
 */
export function ParentViewCard({ name, view, badge, variant = 'card', footer, className }: ParentViewCardProps) {
  const t = useTranslations('teen.parent.view');
  const facts: string[] = [];
  if (view.nextPeriodWeek) facts.push(t(`week.${view.nextPeriodWeek}`));
  if (view.kitReady !== null) facts.push(t(view.kitReady ? 'kitReady' : 'kitNotReady'));

  return (
    <article className={clsx('nb-card tnp-view', variant === 'preview' && 'is-preview', className)}>
      <div className="tnp-view-head">
        <b className="tnp-view-name">{name}</b>
        {badge}
      </div>
      {facts.length ? <p className="tnp-view-facts">{facts.join(' ')}</p> : null}
      {view.note ? (
        <p className="tnp-view-note">
          <span className="tnp-view-note-label">{t('noteLabel')}</span>
          <bdi>{view.note}</bdi>
        </p>
      ) : null}
      {isEmptyParentView(view) ? <p className="tnp-view-empty">{t('nothing')}</p> : null}
      {footer}
    </article>
  );
}
