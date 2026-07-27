'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

// Cycle length bounds mirror the POST /profile validation (cycle_duration 15–60).
const MIN = 15;
const MAX = 60;
const ITEMS = Array.from({ length: MAX - MIN + 1 }, (_, i) => `${faNum(i + MIN)} روز`);

export function CycleDurationPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { cycleDuration, intention, setCycleDuration } = useOnboardingStore();
  const step = stepPosition('cycleDuration', intention);

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
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
              id="wC" items={ITEMS} selectedIndex={cycleDuration - MIN} width={150}
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
