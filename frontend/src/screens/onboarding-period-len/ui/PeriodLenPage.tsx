'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';



export function PeriodLenPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { periodLen, intention, setPeriodLen } = useOnboardingStore();
  const step = stepPosition('periodLen', intention);
  const items = useMemo(
    () => Array.from({ length: 10 }, (_, i) => t('dayCount', { days: i + 1 })),
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
          <div className="titr">{t('periodLen.title')}</div>
          <p className="sub onb-intro-sub">{t('periodLen.subtitle')}</p>
        </div>
        <div className="onb-center">
          <div className="prof-wheel-center">
            <div className="wheel-band prof-wheel-band" />
            <WheelPicker
              id="wP" items={items} selectedIndex={periodLen - 1} width={150}
              onChange={i => setPeriodLen(i + 1)}
            />
          </div>
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('periodLen', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
