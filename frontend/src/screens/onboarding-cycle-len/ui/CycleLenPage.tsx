'use client';

import { useLocale, useTranslations } from 'next-intl';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { CalendarPicker, NavBack } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';


export function CycleLenPage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { lastPeriod, intention, setLastPeriod } = useOnboardingStore();
  const step = stepPosition('cycleLen', intention);

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{formatNumber(step.index, loc)}<span className="onb-dim"> / {formatNumber(step.total, loc)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('cycleLen.title')}</div>
          <p className="sub onb-intro-sub">{t('cycleLen.subtitle')}</p>
        </div>
        <div className="onb-center">
          <CalendarPicker value={lastPeriod} onSelect={setLastPeriod} />
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
