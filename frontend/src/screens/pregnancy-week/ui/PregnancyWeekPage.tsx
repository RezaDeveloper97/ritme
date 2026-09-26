'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type TouchEvent, useRef } from 'react';

import {
  clampV2Week,
  type PregnancyWeek,
  usePregnancyToday,
  usePregnancyWeek,
  V2_TERM_WEEKS,
} from '@/entities/pregnancy';
import { useUpdateWeekState } from '@/features/track-pregnancy';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';
import { FetusSize } from '@/shared/ui/illustrations';
import { BottomNav } from '@/widgets/bottom-nav';

import { swipeTarget } from '../model/swipe';
import { WeekStrip } from './WeekStrip';
import { WeekTabs } from './WeekTabs';

const CARD = 'rounded-2xl border border-(--line) bg-(--surface)';

interface Props {
  /** Week from the URL; `null` (`/pregnancy/weeks`) opens the current week. */
  week: number | null;
}

/** «هفته‌به‌هفته» — `/pregnancy/weeks/[n]` (Week.dc.html, T-M7-11). */
export function PregnancyWeekPage({ week: requested }: Props) {
  const t = useTranslations('pregnancyV2');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const today = usePregnancyToday();
  const currentWeek = today.data ? clampV2Week(today.data.progress.week) : null;
  const week = requested != null ? clampV2Week(requested) : currentWeek;
  const query = usePregnancyWeek(week);
  const bookmark = useUpdateWeekState();
  const touch = useRef<{ x: number; y: number } | null>(null);
  const data = query.data;

  const onTouchStart = (e: TouchEvent) => {
    const p = e.touches[0];
    touch.current = p ? { x: p.clientX, y: p.clientY } : null;
  };
  const onTouchEnd = (e: TouchEvent) => {
    const start = touch.current;
    const p = e.changedTouches[0];
    touch.current = null;
    if (!start || !p || week == null) return;
    const next = swipeTarget(week, p.clientX - start.x, p.clientY - start.y, dir);
    if (next != null) router.replace(`/pregnancy/weeks/${next}`, { scroll: false });
  };

  const trimester = data?.trimester ?? null;
  const rangeText = data
    ? (data.rangeLabel ??
      (data.range
        ? t('common.dateRange', {
            from: formatLongDate(fromApiDate(data.range.from), locale),
            to: formatLongDate(fromApiDate(data.range.to), locale),
          })
        : null))
    : null;
  const subtitle = [trimester ? t(`common.trimester.${trimester as 1 | 2 | 3}`) : null, rangeText]
    .filter(Boolean)
    .join(' · ');

  return (
    <div className="view preg-page">
      <div className="scroll" onTouchStart={onTouchStart} onTouchEnd={onTouchEnd}>
        <header className="rmd-hdr">
          <Link href="/pregnancy" className="rmd-hdr-btn" aria-label={t('common.back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">
              {week != null ? t('common.weekOf', { week, total: V2_TERM_WEEKS }) : t('common.title')}
            </h1>
            {subtitle && <p className="rmd-hdr-sub">{subtitle}</p>}
          </div>
          {data ? (
            <button
              type="button"
              className="rmd-hdr-btn"
              aria-pressed={data.bookmarked}
              aria-label={data.bookmarked ? t('week.unbookmark') : t('week.bookmark')}
              onClick={() => bookmark.mutate({ week: data.week, state: { bookmarked: !data.bookmarked } })}
            >
              <Icon
                name="bookmark"
                size={20}
                strokeWidth={1.8}
                className={clsx(data.bookmarked && 'fill-(--brand) text-(--brand)')}
              />
            </button>
          ) : (
            <span className="size-11 shrink-0" aria-hidden />
          )}
        </header>

        {week != null && <WeekStrip week={week} currentWeek={currentWeek} />}

        <div className="flex flex-col gap-3.5 px-4 pt-1 pb-28">
          {query.isPending && week != null ? (
            <p className="m-0 py-10 text-center text-sm font-semibold text-(--ink-3)">{t('common.loading')}</p>
          ) : query.isError || today.isError ? (
            <div className="flex flex-col items-center gap-3 py-10 text-center">
              <p role="alert" className="m-0 text-sm font-semibold text-(--ink-3)">
                {t('common.loadError')}
              </p>
              <button type="button" className="btn btn-primary" onClick={() => void query.refetch()}>
                {t('common.retry')}
              </button>
            </div>
          ) : data ? (
            <WeekBody data={data} />
          ) : today.isPending ? (
            <p className="m-0 py-10 text-center text-sm font-semibold text-(--ink-3)">{t('common.loading')}</p>
          ) : (
            <div className="flex flex-col items-center gap-3 py-10 text-center">
              <p className="m-0 text-sm font-semibold text-(--ink-3)">{t('common.notActive')}</p>
              <Link href="/pregnancy" className="btn btn-primary no-underline">
                {t('common.back')}
              </Link>
            </div>
          )}
        </div>
      </div>
      <BottomNav />
    </div>
  );
}

function WeekBody({ data }: { data: PregnancyWeek }) {
  const t = useTranslations('pregnancyV2.week');
  const locale = useLocale() as Locale;
  const d = data.details;
  const stats = [
    { value: d.length, unit: t('stats.cm') },
    { value: d.weight, unit: t('stats.g') },
    { value: d.heartRate, unit: t('stats.bpm') },
  ].filter((s): s is { value: string; unit: string } => !!s.value);

  return (
    <>
      <section
        className={clsx(CARD, 'flex flex-col items-center gap-2.5 rounded-[22px] px-4 pt-[22px] pb-4 shadow-md')}
      >
        <FetusSize illustrationKey={d.illustrationKey} label={t('illustration')} />
        {d.sizeLabel && (
          <span className="inline-flex h-7 items-center rounded-[14px] bg-(--pink-soft) px-3 text-xs font-extrabold text-(--preg-pink-ink)">
            {t('sizePill', { item: d.sizeLabel })}
          </span>
        )}
        {d.headline && (
          <h2 className="m-0 text-center text-[21px] leading-[1.6] font-black text-(--ink)">{d.headline}</h2>
        )}
        {stats.length > 0 && (
          <dl className="m-0 mt-1 grid w-full grid-cols-3 gap-2">
            {stats.map((s) => (
              <div key={s.unit} className="flex flex-col-reverse rounded-[14px] bg-(--surface-2) px-1.5 py-2.5 text-center">
                <dt className="text-[11px] font-bold text-(--ink-3)">{s.unit}</dt>
                <dd className="m-0 text-[17px] font-black text-(--brand-deep)">{s.value}</dd>
              </div>
            ))}
          </dl>
        )}
        {stats.length > 0 && (
          <span className="text-center text-[11.5px] leading-[1.7] font-semibold text-(--ink-3)">
            {t('averagesNote')}
          </span>
        )}
      </section>

      <WeekTabs data={data} />

      {d.warning && (
        <section className="pg2-warn flex items-start gap-3 rounded-2xl p-3.5">
          <Icon name="warning" size={20} className="pg2-warn-icon" />
          <p className="m-0 text-[12.5px] leading-[1.9]">{d.warning}</p>
        </section>
      )}

      <div className="flex items-start gap-2.5 px-1">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-(--line) text-(--ink-3)" aria-hidden>
          <Icon name="doctor" size={16} />
        </span>
        <div className="min-w-0 text-[11.5px] leading-[1.7] font-semibold text-(--ink-3)">
          {d.reviewerName ? (
            <>
              {t('reviewedBy', { name: d.reviewerName })}
              {d.reviewedAt && <> · {formatLongDate(fromApiDate(d.reviewedAt.slice(0, 10)), locale)}</>}
            </>
          ) : (
            t('notReviewed')
          )}
          {d.sources.length > 0 && (
            <details className="mt-0.5">
              <summary className="cursor-pointer font-bold text-(--brand)">{t('sources')}</summary>
              <ul className="m-0 mt-1 list-disc ps-4">
                {d.sources.map((s, i) => (
                  <li key={`${s.title}-${i}`}>
                    {s.url ? (
                      <a href={s.url} target="_blank" rel="noopener noreferrer" className="text-(--brand)">
                        {s.title}
                      </a>
                    ) : (
                      s.title
                    )}
                  </li>
                ))}
              </ul>
            </details>
          )}
        </div>
      </div>
    </>
  );
}
