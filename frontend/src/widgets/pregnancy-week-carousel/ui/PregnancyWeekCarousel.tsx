'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import { useRef, useState } from 'react';

import { type Confidence, V2_TERM_WEEKS, type WeekSummary } from '@/entities/pregnancy';
import { Link, useDirection } from '@/shared/i18n';
import { Icon } from '@/shared/ui';
import { FetusSize } from '@/shared/ui/illustrations';

interface Props {
  slides: readonly WeekSummary[];
  confidence: Confidence;
  uncertaintyDays: number | null;
}

const SWIPE_PX = 40;

/**
 * Today hero — swipeable prev / current / next week slides (Main artboard).
 * Opens on the current week; «بازگشت به امروز» appears once the user moves.
 */
export function PregnancyWeekCarousel({ slides, confidence, uncertaintyDays }: Props) {
  const t = useTranslations('pregnancyV2');
  const isRtl = useDirection() === 'rtl';
  const currentIndex = Math.max(0, slides.findIndex((s) => s.relation === 'current'));
  const [index, setIndex] = useState(currentIndex);
  const startX = useRef<number | null>(null);

  const i = Math.min(index, slides.length - 1);
  const slide = slides[i];
  if (!slide) return null;

  const go = (next: number) => setIndex(Math.min(slides.length - 1, Math.max(0, next)));
  const prev = () => go(i - 1);
  const next = () => go(i + 1);

  const onTouchEnd = (x: number) => {
    if (startX.current == null) return;
    const dx = x - startX.current;
    startX.current = null;
    if (Math.abs(dx) < SWIPE_PX) return;
    // In RTL the earlier week sits on the right, so a left→right swipe goes forward.
    const forward = isRtl ? dx > 0 : dx < 0;
    if (forward) next();
    else prev();
  };

  const level = confidence.level;
  const confidenceText = level
    ? uncertaintyDays != null
      ? t('common.confidence.withRange', { level: t(`common.confidence.levels.${level}`), days: uncertaintyDays })
      : t('common.confidence.label', { level: t(`common.confidence.levels.${level}`) })
    : null;
  const trimester = slide.trimester === 1 || slide.trimester === 2 || slide.trimester === 3 ? slide.trimester : null;

  return (
    <section
      aria-roledescription="carousel"
      className="mx-4 shrink-0 overflow-hidden rounded-3xl bg-linear-to-b from-(--preg-hero-start) to-(--preg-hero-end) p-4 text-(--on-accent)"
      onTouchStart={(e) => (startX.current = e.touches[0]?.clientX ?? null)}
      onTouchEnd={(e) => onTouchEnd(e.changedTouches[0]?.clientX ?? 0)}
    >
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          className="iconbtn text-(--on-accent) disabled:opacity-40"
          aria-label={t('today.prevWeek')}
          disabled={i === 0}
          onClick={prev}
        >
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <div className="min-w-0 text-center" aria-live="polite">
          <div className="text-[12px] font-bold opacity-90">
            {trimester
              ? t('common.trimesterWeek', {
                  trimester: t(`common.trimester.${trimester}`),
                  week: slide.week,
                  total: V2_TERM_WEEKS,
                })
              : t('common.weekOf', { week: slide.week, total: V2_TERM_WEEKS })}
          </div>
          <div className="mt-1 text-[22px] font-black">{slide.title ?? t('common.weekOf', { week: slide.week, total: V2_TERM_WEEKS })}</div>
        </div>
        <button
          type="button"
          className="iconbtn text-(--on-accent) disabled:opacity-40"
          aria-label={t('today.nextWeek')}
          disabled={i === slides.length - 1}
          onClick={next}
        >
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      <div className="mt-2 flex min-h-6 justify-center">
        {slide.relation === 'current' ? (
          confidenceText && (
            <span className="rounded-full bg-(--on-accent)/20 px-3 py-1 text-[11px] font-bold">{confidenceText}</span>
          )
        ) : (
          <button
            type="button"
            className="rounded-full bg-(--on-accent) px-3 py-1 text-[11px] font-black text-(--brand-deep)"
            onClick={() => go(currentIndex)}
          >
            {t('today.backToToday')}
          </button>
        )}
      </div>

      <div className="mt-2 flex justify-center">
        <FetusSize illustrationKey={slide.illustrationKey} size={150} onHero label={t('week.illustration')} />
      </div>

      {slide.sizeLine && <p className="mt-2 text-center text-[13.5px] font-bold">{slide.sizeLine}</p>}

      <div className="mt-3 flex items-center justify-between">
        <div className="flex gap-1.5" aria-hidden>
          {slides.map((s, k) => (
            <span
              key={s.week}
              className={clsx('h-1.5 rounded-full bg-(--on-accent)', k === i ? 'w-5' : 'w-1.5 opacity-50')}
            />
          ))}
        </div>
        <Link
          href={`/pregnancy/weeks/${slide.week}`}
          className="flex items-center gap-1 text-[12.5px] font-black text-(--on-accent) no-underline"
        >
          {t('today.browseWeeks')}
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={16} />
        </Link>
      </div>
    </section>
  );
}
