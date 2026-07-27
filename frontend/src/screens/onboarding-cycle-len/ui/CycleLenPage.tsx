'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { JalaliCalendar, NavBack } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function CycleLenPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { lastPeriod, intention, setLastPeriod } = useOnboardingStore();
  const step = stepPosition('cycleLen', intention);

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('cycleLen.title')}</div>
          <p className="sub onb-intro-sub">{t('cycleLen.subtitle')}</p>
        </div>
        <div className="onb-center">
          <JalaliCalendar value={lastPeriod} onSelect={setLastPeriod} />
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
