'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { V2_MAX_WEEK, V2_MIN_WEEK, weekRelation, type WeekRelation } from '@/entities/pregnancy';
import { type Locale, Link } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

const WEEKS = Array.from({ length: V2_MAX_WEEK - V2_MIN_WEEK + 1 }, (_, i) => V2_MIN_WEEK + i);

/** past = solid white, current = brand fill, future = dashed (Week artboard). */
const CHIP: Record<WeekRelation, string> = {
  past: 'bg-(--surface) border border-(--line) text-(--ink)',
  current: 'bg-(--brand-fill) border border-(--brand-fill) text-(--on-accent) font-black',
  future: 'bg-(--surface) border border-dashed border-(--brand-line) text-(--ink-3)',
};

const SWATCH: Record<WeekRelation, string> = {
  past: 'bg-(--surface) border border-(--line)',
  current: 'bg-(--brand-fill)',
  future: 'bg-(--surface) border border-dashed border-(--brand-line)',
};

interface Props {
  /** The week on screen (gets `aria-current` and a ring). */
  week: number;
  /** Today's gestational week — drives past / current / future. */
  currentWeek: number | null;
}

/** Chip strip 1–42, scrolled so the viewed week sits in the middle, + legend. */
export function WeekStrip({ week, currentWeek }: Props) {
  const t = useTranslations('pregnancyV2.week');
  const locale = useLocale() as Locale;
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const strip = ref.current;
    const chip = strip?.querySelector<HTMLElement>(`[data-week="${week}"]`);
    if (!strip || !chip) return;
    // scrollIntoView would also scroll the page; move only the strip.
    const delta =
      chip.getBoundingClientRect().left +
      chip.offsetWidth / 2 -
      (strip.getBoundingClientRect().left + strip.clientWidth / 2);
    strip.scrollBy({ left: delta, behavior: 'instant' as ScrollBehavior });
  }, [week]);

  return (
    <>
      <nav aria-label={t('stripLabel')}>
        <div ref={ref} className="scroll-x flex gap-1.5 px-4 py-1.5">
          {WEEKS.map((n) => {
            const rel = weekRelation(n, currentWeek);
            const on = n === week;
            return (
              <Link
                key={n}
                data-week={n}
                href={`/pregnancy/weeks/${n}`}
                replace
                scroll={false}
                aria-current={on ? 'page' : undefined}
                className={clsx(
                  'flex h-[54px] w-11 shrink-0 flex-col items-center justify-center gap-0.5 rounded-xl text-[12.5px] font-bold no-underline focus-visible:shadow-(--ring) focus-visible:outline-none',
                  CHIP[rel],
                  on && rel !== 'current' && 'ring-2 ring-(--brand)',
                )}
              >
                <span className="text-[9.5px]">{t('chipWeek')}</span>
                {formatNumber(n, locale)}
              </Link>
            );
          })}
        </div>
      </nav>
      <div className="flex justify-center gap-3.5 px-4 pb-2 text-[10.5px] font-bold text-(--ink-3)">
        {(['past', 'current', 'future'] as const).map((rel) => (
          <span key={rel} className="inline-flex items-center gap-1.5">
            <span aria-hidden className={clsx('size-2.5 rounded-[3px]', SWATCH[rel])} />
            {t(`legend.${rel}`)}
          </span>
        ))}
      </div>
    </>
  );
}
