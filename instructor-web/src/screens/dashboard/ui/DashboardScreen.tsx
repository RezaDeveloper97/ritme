'use client';

import { useTranslations } from 'next-intl';

import type { Instructor } from '@/entities/instructor';
import { Icon } from '@/shared/ui';

const dateFormat = new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  timeZone: 'Asia/Tehran',
});

/**
 * / — the dashboard frame. Stats, courses and recent activity (nbl_Ins_Dashboard)
 * are built by B-N8-06 / B-N8-07 on real data; until then an honest empty state.
 */
export function DashboardScreen({ instructor }: { instructor: Instructor }) {
  const t = useTranslations('dashboard');
  const approved = instructor.approved_at ? dateFormat.format(new Date(instructor.approved_at)) : null;
  return (
    <div className="page">
      <section className="welcome">
        <h1 className="welcome__title">{t('hello', { name: instructor.display_name })}</h1>
        <p className="welcome__lead">{t('lead')}</p>
        {approved ? (
          <span className="chip chip--success">
            <Icon name="shield" size={16} />
            {t('approvedSince', { date: approved })}
          </span>
        ) : null}
      </section>
      <section className="empty-card">
        <span className="empty-card__icon">
          <Icon name="dashboard" size={26} />
        </span>
        <h2 className="empty-card__title">{t('emptyTitle')}</h2>
        <p className="empty-card__text">{t('emptyText')}</p>
      </section>
    </div>
  );
}
