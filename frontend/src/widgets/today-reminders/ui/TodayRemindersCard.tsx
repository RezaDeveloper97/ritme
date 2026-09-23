'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { type NextAppointment, useCareToday } from '@/entities/care-reminder';
import { useLogIntake } from '@/features/log-intake';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

import { cardStatus, doseRowState, type DoseRowState } from '../model/dose-row';


type T = ReturnType<typeof useTranslations<'care'>>;

/** The AddChooser sheet (`app/sheets/registry.tsx`). */
const ADD_SHEET = 'reminders-add';

function Header({ t }: { t: T }) {
  return (
    <div className="trm-head">
      <span className="trm-head-l">
        <span className="trm-head-tile" aria-hidden>
          <Icon name="bellRing" size={16} strokeWidth={1.8} />
        </span>
        <h2 className="trm-title">{t('home.title')}</h2>
      </span>
      <Link href="/reminders" className="trm-all">
        {t('home.all')}
      </Link>
    </div>
  );
}

function AddButton({ t }: { t: T }) {
  return (
    <button type="button" className="trm-add" onClick={() => openSheet(ADD_SHEET)}>
      <Icon name="plus" size={16} strokeWidth={2.4} />
      {t('home.add')}
    </button>
  );
}

function DoseRow({
  row,
  t,
  locale,
  pending,
  onToggle,
}: {
  row: DoseRowState;
  t: T;
  locale: Locale;
  pending: boolean;
  onToggle: () => void;
}) {
  return (
    <div className={clsx('trm-dose', row.taken && 'is-taken')}>
      <span className={clsx('trm-dose-tile', `is-${row.tone}`)} aria-hidden>
        <Icon name={row.icon} size={18} strokeWidth={1.8} />
      </span>
      <div className="trm-dose-body">
        <div className="trm-dose-title">{row.title}</div>
        <div className="trm-dose-time">
          {t('slotWithPeriod', {
            time: formatNumber(row.clock, locale),
            period: t(`slotPeriod.${row.period}`),
          })}
        </div>
      </div>
      <button
        type="button"
        className="trm-check"
        aria-pressed={row.taken}
        aria-label={t(row.taken ? 'today.markNotTaken' : 'today.markTaken', { title: row.title })}
        // Optimistic already; blocking only this row while its write is in
        // flight keeps a double-tap from queueing an undo.
        disabled={pending}
        onClick={onToggle}
      >
        <span className="trm-check-dot" aria-hidden>
          {row.taken && <Icon name="check" size={14} strokeWidth={3} />}
        </span>
      </button>
    </div>
  );
}

function AppointmentRow({ appt, t, locale }: { appt: NextAppointment; t: T; locale: Locale }) {
  const [date, clock = ''] = appt.scheduledAt.split(' ');
  const parts = toParts(fromApiDate(date), locale);
  const when = t('appointmentDetail.daysUntil', { days: appt.daysUntil });
  const time = formatNumber(clock.slice(0, 5), locale);

  return (
    <Link href={`/reminders/appointment/${appt.id}`} className="trm-appt">
      <span className="trm-appt-date" aria-hidden>
        <span className="trm-appt-day">{formatNumber(parts.day, locale)}</span>
        <span className="trm-appt-month">{monthName(parts.month, locale)}</span>
      </span>
      <span className="trm-appt-body">
        <span className="trm-appt-title">
          {appt.withWhom
            ? t('home.appointmentTitle', { title: appt.title, with: appt.withWhom })
            : appt.title}
        </span>
        <span className="trm-appt-meta">
          {appt.location
            ? t('home.nextAppointmentMeta', { when, time, place: appt.location })
            : t('home.nextAppointmentMetaNoPlace', { when, time })}
        </span>
      </span>
      <span className="trm-appt-chip">
        <Icon name="bellPlain" size={13} strokeWidth={2.4} />
        {t('home.remindBefore', { before: t(`appointmentForm.remindOptions.${appt.remindBefore}`) })}
      </span>
    </Link>
  );
}

/**
 * «یادآورهای امروز» — today's medication doses (tick to log) and the next
 * doctor's appointment, on both homes. Artboard: docs/design/reminders-v13/
 * `nbl_v13_Preg_Home` (light) / `nbd_` (dark).
 *
 * Privacy (§11): which doses were taken is health data — rendered, never logged.
 */
export function TodayRemindersCard() {
  const t = useTranslations('care');
  const locale = useLocale() as Locale;
  const query = useCareToday();
  const logIntake = useLogIntake();
  const status = cardStatus(query);

  if (status === 'hidden') return null;

  if (status === 'loading') {
    return (
      <section className="trm-sec" aria-busy="true" aria-label={t('loading')}>
        <div className="trm-card">
          <Header t={t} />
          <span className="skeleton-line trm-skel" />
          <span className="skeleton-line trm-skel is-short" />
        </div>
      </section>
    );
  }

  const data = query.data;
  if (!data) return null;
  const rows = data.doses.map(doseRowState);
  const pendingKey =
    logIntake.isPending && logIntake.variables
      ? `${logIntake.variables.reminderId}-${logIntake.variables.slot}`
      : null;

  return (
    <section className="trm-sec">
      <div className="trm-card">
        <Header t={t} />

        {status === 'empty' && <p className="trm-empty">{t('home.empty')}</p>}

        {rows.map((row) => (
          <DoseRow
            key={row.key}
            row={row}
            t={t}
            locale={locale}
            pending={pendingKey === row.key}
            onToggle={() =>
              logIntake.mutate({
                reminderId: row.reminderId,
                date: data.date,
                slot: row.slot,
                taken: row.nextTaken,
              })
            }
          />
        ))}

        {data.nextAppointment && (
          <>
            {rows.length > 0 && <div className="trm-divider" aria-hidden />}
            <AppointmentRow appt={data.nextAppointment} t={t} locale={locale} />
          </>
        )}

        <AddButton t={t} />
      </div>
    </section>
  );
}
