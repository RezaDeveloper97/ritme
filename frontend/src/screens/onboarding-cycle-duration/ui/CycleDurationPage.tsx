'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';


// Cycle length bounds mirror the POST /profile validation (cycle_duration 15–60).
const MIN = 15;
const MAX = 60;

export function CycleDurationPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { cycleDuration, intention, setCycleDuration } = useOnboardingStore();
  const step = stepPosition('cycleDuration', intention);
  const items = useMemo(
    () => Array.from({ length: MAX - MIN + 1 }, (_, i) => t('dayCount', { days: i + MIN })),
    [t],
  );

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{formatNumber(step.index, loc)}<span className="onb-dim"> / {formatNumber(step.total, loc)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('cycleDuration.title')}</div>
          <p className="sub onb-intro-sub">{t('cycleDuration.subtitle')}</p>
        </div>
        <div className="onb-center">
          <div className="prof-wheel-center">
            <div className="wheel-band prof-wheel-band" />
            <WheelPicker
              id="wC" items={items} selectedIndex={cycleDuration - MIN} width={150}
              onChange={i => setCycleDuration(i + MIN)}
            />
          </div>
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('cycleDuration', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
