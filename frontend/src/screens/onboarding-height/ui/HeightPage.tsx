'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack, RulerPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore, type HeightUnit } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function HeightPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { height, heightUnit, intention, setHeight, setHeightUnit } = useOnboardingStore();
  const step = stepPosition('height', intention);

  const switchUnit = (u: HeightUnit) => {
    setHeightUnit(u);
    if (u === 'ft' && heightUnit === 'cm') setHeight(Math.round(height / 30.48 * 100) / 100);
    if (u === 'cm' && heightUnit === 'ft') setHeight(Math.round(height * 30.48));
  };

  const [min, max] = heightUnit === 'cm' ? [120, 220] : [4, 7];

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('height.title')}</div>
          <p className="sub onb-intro-sub">{t('height.subtitle')}</p>
          <p className="sub onb-hint">{t('height.helper')}</p>
        </div>
        <div className="onb-center">
          <div className="onb-ruler-wrap">
            <div className="seg onb-ruler">
              <button className={heightUnit === 'ft' ? 'on' : ''} onClick={() => switchUnit('ft')}>ft</button>
              <button className={heightUnit === 'cm' ? 'on' : ''} onClick={() => switchUnit('cm')}>cm</button>
            </div>
          </div>
          <RulerPicker
            min={min} max={max} value={height} unit={heightUnit}
            onChange={setHeight}
            toDisplay={v => faNum(v.toFixed(1))}
          />
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('height', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
