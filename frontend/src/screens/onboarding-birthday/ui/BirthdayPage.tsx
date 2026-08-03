'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { allMonthNames, birthYearRange, formatNumber } from '@/shared/lib/date';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

export function BirthdayPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { birth, intention, setBirth } = useOnboardingStore();
  const step = stepPosition('birthday', intention);

  // The wheels speak the locale's calendar — Jalali years/months for fa,
  // Gregorian for en (§7). `OnboardingCalendarSync` has already re-expressed
  // `birth` in that calendar by the time this renders.
  const months = useMemo(() => [...allMonthNames(loc)], [loc]);
  const days = useMemo(() => Array.from({ length: 31 }, (_, i) => formatNumber(i + 1, loc)), [loc]);
  const years = useMemo(() => birthYearRange(loc), [loc]);
  const yearItems = useMemo(
    () => Array.from({ length: years.max - years.min + 1 }, (_, i) => formatNumber(years.min + i, loc)),
    [years, loc],
  );
  const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

  // WheelPicker binds its scroll handler once on mount, so these callbacks are
  // the first-render closures. Patching the latest value from a ref (instead of
  // the captured `birth`) keeps the three wheels from clobbering each other —
  // and keeps a scroll settling after a locale switch from writing the old
  // calendar's numbers back over the converted ones.
  const birthRef = useRef(birth);
  useEffect(() => {
    birthRef.current = birth;
  }, [birth]);
  const patch = (part: Partial<typeof birth>) => setBirth({ ...birthRef.current, ...part });

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">
          {formatNumber(step.index, loc)}
          <span className="onb-dim"> / {formatNumber(step.total, loc)}</span>
        </span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('birthday.title')}</div>
          <p className="sub onb-intro-sub">{t('birthday.subtitle')}</p>
        </div>
        <div className="onb-center">
          <div className="onb-wheels">
            <div className="wheel-band" />
            <WheelPicker
              id="wD" items={days} selectedIndex={clamp(birth.d - 1, 0, 30)} width={56}
              onChange={i => patch({ d: i + 1 })}
            />
            {/* `birth.m` is a 1-based month; the wheel index is 0-based. */}
            <WheelPicker
              id="wM" items={months} selectedIndex={clamp(birth.m - 1, 0, 11)} width={112}
              onChange={i => patch({ m: i + 1 })}
            />
            <WheelPicker
              id="wY" items={yearItems}
              selectedIndex={clamp(birth.y - years.min, 0, years.max - years.min)}
              width={80}
              onChange={i => patch({ y: years.min + i })}
            />
          </div>
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('birthday', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
