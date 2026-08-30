'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import {
  allMonthNames,
  type DateParts,
  daysInCalendarMonth,
  formatNumber,
  recentYearRange,
  todayParts,
} from '@/shared/lib/date';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, previousOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

/** Is `a` strictly after `b`? Both are parts of the *same* calendar. */
const isAfter = (a: DateParts, b: DateParts) =>
  a.year !== b.year ? a.year > b.year : a.month !== b.month ? a.month > b.month : a.day > b.day;

export function CycleLenPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { lastPeriod, intention, setLastPeriod } = useOnboardingStore();
  const step = stepPosition('cycleLen', intention);

  // The wheels speak the locale's calendar — Jalali for fa, Gregorian for en
  // (§7). `OnboardingCalendarSync` has already re-expressed `lastPeriod` in
  // that calendar by the time this renders.
  const today = todayParts(loc);
  const value = lastPeriod ?? today;

  // The wheels always show *some* date, so the shown date is the answer: park
  // an untouched picker on today rather than letting it read as "answered"
  // while the store still holds null.
  useEffect(() => {
    if (!lastPeriod) setLastPeriod(todayParts(loc));
  }, [lastPeriod, loc, setLastPeriod]);

  const months = useMemo(() => [...allMonthNames(loc)], [loc]);
  const years = useMemo(() => recentYearRange(loc), [loc]);
  const yearItems = useMemo(
    () => Array.from({ length: years.max - years.min + 1 }, (_, i) => formatNumber(years.min + i, loc)),
    [years, loc],
  );
  // Esfand is 29 or 30 days, February 28 or 29 — the wheel follows the month.
  const dayCount = daysInCalendarMonth(value.year, value.month, loc);
  const days = useMemo(
    () => Array.from({ length: dayCount }, (_, i) => formatNumber(i + 1, loc)),
    [dayCount, loc],
  );

  // WheelPicker binds its scroll handler once on mount, so these callbacks are
  // the first-render closures. Patching from a ref keeps the three wheels from
  // clobbering each other, and keeps a scroll settling after a locale switch
  // from writing the old calendar's numbers over the converted ones.
  const valueRef = useRef(value);
  valueRef.current = value;
  const patch = (part: Partial<DateParts>) => {
    const next = { ...valueRef.current, ...part };
    next.day = clamp(next.day, 1, daysInCalendarMonth(next.year, next.month, loc));
    // A period that hasn't happened yet is not a *last* period; the profile
    // mapper drops a future date anyway, so clamp instead of losing the answer.
    setLastPeriod(isAfter(next, todayParts(loc)) ? todayParts(loc) : next);
  };

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.replace(previousOnboardingRoute('cycleLen', intention) ?? '/signup')} />
        <span className="stepcount">
          {formatNumber(step.index, loc)}
          <span className="onb-dim"> / {formatNumber(step.total, loc)}</span>
        </span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('cycleLen.title')}</div>
          <p className="sub onb-intro-sub">{t('cycleLen.subtitle')}</p>
        </div>
        <div className="onb-center">
          <div className="onb-wheels">
            <div className="wheel-band" />
            <WheelPicker
              id="lpD" items={days} selectedIndex={clamp(value.day - 1, 0, dayCount - 1)} width={56}
              onChange={i => patch({ day: i + 1 })}
            />
            {/* `value.month` is 1-based; the wheel index is 0-based. */}
            <WheelPicker
              id="lpM" items={months} selectedIndex={clamp(value.month - 1, 0, 11)} width={112}
              onChange={i => patch({ month: i + 1 })}
            />
            <WheelPicker
              id="lpY" items={yearItems}
              selectedIndex={clamp(value.year - years.min, 0, years.max - years.min)}
              width={80}
              onChange={i => patch({ year: years.min + i })}
            />
          </div>
          <p className="sub onb-center-text">{t('cycleLen.hint')}</p>
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('cycleLen', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
