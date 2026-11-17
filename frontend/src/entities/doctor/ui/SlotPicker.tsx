'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { addDays, formatMonthLabel, formatNumber, formatWeekday, formatWeekdayDayMonth, fromApiDate, toApiDate, toParts } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Card, EmptyState, Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { useDoctorSlots } from '../api/queries';
import type { VisitMode } from '../api/schema';
import { formatClock } from '../model/format';

export interface PickedSlot {
  date: string;
  time: string;
}

/**
 * Week strip + free-time grid of one doctor and visit mode (profile + reschedule). 7 days per page, «ماه بعد» pages
 * forward (the API serves 60 days). Selecting a time reports `{date, time}` (Tehran wall clock).
 */
export function SlotPicker({
  doctorId,
  mode,
  value,
  onChange,
}: {
  doctorId: number;
  mode: VisitMode | null;
  value: PickedSlot | null;
  onChange: (slot: PickedSlot | null) => void;
}) {
  const t = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const [week, setWeek] = useState(0);
  const [picked, setPicked] = useState<string | null>(value?.date ?? null);
  const from = useMemo(() => toApiDate(addDays(new Date(), week * 7)), [week]);
  const q = useDoctorSlots(doctorId, mode, from);

  const days = q.data?.days ?? [];
  const activeDate = picked && days.some((d) => d.date === picked && d.count > 0) ? picked : (days.find((d) => d.count > 0)?.date ?? null);
  const active = days.find((d) => d.date === activeDate);
  const first = days[0] ? fromApiDate(days[0].date) : new Date();
  const p = toParts(first, locale);

  let body;
  if (q.isPending && mode !== null) {
    body = (
      <SkeletonGroup label={t('common.loading')}>
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={t('common.loadError')}
        action={
          <button type="button" className="dr-link" onClick={() => void q.refetch()}>
            {t('common.retry')}
          </button>
        }
      />
    );
  } else {
    body = (
      <>
        <div className="dr-days" role="group" aria-label={t('slots.days')} {...pilot('doctor.day', 'choice')}>
          {days.map((d) => {
            const dt = fromApiDate(d.date);
            const w = formatWeekday(dt, locale);
            const on = d.date === activeDate;
            return (
              <button
                key={d.date}
                type="button"
                className={clsx('dr-day', on && 'is-on')}
                disabled={d.count === 0}
                aria-pressed={on}
                {...pilotOption(d.date)}
                onClick={() => {
                  setPicked(d.date);
                  onChange(null);
                }}
              >
                <span className="dr-day-w">{locale === 'fa' ? Array.from(w)[0] : w.slice(0, 3)}</span>
                <span className="dr-day-n">{formatNumber(toParts(dt, locale).day, locale)}</span>
              </button>
            );
          })}
        </div>
        {active ? (
          <>
            <p className="dr-slots-sub">{t('slots.free', { date: formatWeekdayDayMonth(fromApiDate(active.date), locale), count: formatNumber(active.count, locale) })}</p>
            <div className="dr-slots" {...pilot('doctor.slot', 'choice')}>
              {active.slots.map((s) => {
                const on = value?.date === active.date && value.time === s.time;
                return (
                  <button
                    key={s.starts_at}
                    type="button"
                    className={clsx('dr-slot', on && 'is-on')}
                    aria-pressed={on}
                    {...pilotOption(s.time)}
                    onClick={() => onChange({ date: active.date, time: s.time })}
                  >
                    {formatClock(s.time, locale)}
                  </button>
                );
              })}
            </div>
          </>
        ) : (
          <p className="dr-slots-sub">{t('slots.none')}</p>
        )}
      </>
    );
  }

  return (
    <Card as="section" aria-label={t('slots.title')} className="dr-slotcard">
      <div className="dr-slot-head">
        <h2 className="dr-h2">{formatMonthLabel(p.year, p.month, locale)}</h2>
        <div className="dr-slot-nav">
          {week > 0 ? (
            <button type="button" className="dr-link" {...pilot('doctor.slots-prev')} onClick={() => setWeek((w) => w - 1)}>
              {t('slots.prev')}
            </button>
          ) : null}
          <button type="button" className="dr-link" {...pilot('doctor.slots-next')} onClick={() => setWeek((w) => w + 1)}>
            {t('slots.next')}
            <Icon name="chevronLeft" size={14} className="dr-flip" />
          </button>
        </div>
      </div>
      {body}
    </Card>
  );
}
