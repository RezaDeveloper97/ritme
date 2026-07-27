'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const ITEMS = Array.from({ length: 10 }, (_, i) => `${faNum(i + 1)} روز`);

export function PeriodLenPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { periodLen, intention, setPeriodLen } = useOnboardingStore();
  const step = stepPosition('periodLen', intention);

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
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
              id="wP" items={ITEMS} selectedIndex={periodLen - 1} width={150}
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
