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
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span style={{ opacity: .5 }}> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('cycleDuration.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('cycleDuration.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <div style={{ display: 'flex', justifyContent: 'center', position: 'relative' }}>
            <div className="wheel-band" style={{ width: 150, left: '50%', transform: 'translateX(-50%)' }} />
            <WheelPicker
              id="wC" items={ITEMS} selectedIndex={cycleDuration - MIN} width={150}
              onChange={i => setCycleDuration(i + MIN)}
            />
          </div>
        </div>
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('cycleDuration', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
