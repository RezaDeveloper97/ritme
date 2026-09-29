'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useAppointments } from '@/entities/care-reminder';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { type AppointmentRowState, appointmentRowState, sectionStatus } from '../model/view';
import { SectionHead } from './SectionHead';

type T = ReturnType<typeof useTranslations<'care'>>;

function rowMeta(row: AppointmentRowState, t: T, locale: Locale): string {
  const time = row.time ? formatNumber(row.time, locale) : '';
  const kind = t(`appointments.kindShort.${row.kind}`);
  if (!time) return row.detail ? t('appointments.rowMetaNoTime', { kind, place: row.detail }) : kind;
  return row.detail
    ? t('appointments.rowMeta', { time, kind, place: row.detail })
    : t('appointments.rowMetaNoPlace', { time, kind });
}

function DateTile({ row, locale }: { row: AppointmentRowState; locale: Locale }) {
  const parts = row.date ? toParts(fromApiDate(row.date), locale) : null;
  return (
    <span className={clsx('rmd-date', `is-${row.tone}`)} aria-hidden>
      {parts && (
        <>
          <span className="rmd-date-day">{formatNumber(parts.day, locale)}</span>
          <span className="rmd-date-month">{monthName(parts.month, locale)}</span>
        </>
      )}
    </span>
  );
}

/**
 * «نوبت‌ها و مشاوره‌ها» — upcoming visits and consultations (cancelled ones are
 * excluded server-side). A row opens the appointment detail (T-M3-08).
 */
export function AppointmentSection() {
  const t = useTranslations('care');
  const locale = useLocale() as Locale;
  const query = useAppointments('upcoming');
  const status = sectionStatus(query);

  return (
    <section className="rmd-sec" aria-labelledby="rmd-appts-title">
      <SectionHead
        id="rmd-appts-title"
        title={t('appointments.title')}
        addHref="/reminders/appointment/new?kind=in_person"
        addText={t('add')}
        addLabel={t('appointments.add')}
      />

      <div className="rmd-list">
        {status === 'loading' && (
          <div className="rmd-state" aria-busy="true" aria-label={t('loading')}>
            <span className="skeleton-line rmd-row-skel" />
          </div>
        )}
        {status === 'error' && (
          <div className="rmd-state">
            <p className="rmd-empty">{t('loadError')}</p>
            <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
              {t('retry')}
            </button>
          </div>
        )}
        {status === 'empty' && <p className="rmd-empty rmd-state">{t('appointments.empty')}</p>}
        {status === 'ready' &&
          (query.data ?? []).map(appointmentRowState).map((row) => (
            <Link key={row.id} href={`/reminders/appointment/${row.id}`} className="rmd-row rmd-appt">
              <DateTile row={row} locale={locale} />
              <span className="rmd-row-body">
                <span className="rmd-row-title">
                  {row.titleWith
                    ? t('home.appointmentTitle', { title: row.title, with: row.titleWith })
                    : row.title}
                </span>
                <span className="rmd-row-meta is-clamp">{rowMeta(row, t, locale)}</span>
              </span>
              <Icon name="chevronLeft" size={18} strokeWidth={2} className="rmd-chev" />
            </Link>
          ))}
      </div>
    </section>
  );
}
