'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack, RulerPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore, type WeightUnit } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function WeightPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { weight, weightUnit, intention, setWeight, setWeightUnit } = useOnboardingStore();
  const step = stepPosition('weight', intention);

  const switchUnit = (u: WeightUnit) => {
    setWeightUnit(u);
    if (u === 'lb' && weightUnit === 'kg') setWeight(Math.round(weight * 2.205 * 10) / 10);
    if (u === 'kg' && weightUnit === 'lb') setWeight(Math.round(weight / 2.205 * 10) / 10);
  };

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('weight.title')}</div>
          <p className="sub onb-intro-sub">{t('weight.subtitle')}</p>
          <p className="sub onb-hint">{t('weight.helper')}</p>
        </div>
        <div className="onb-center">
          <div className="onb-ruler-wrap">
            <div className="seg onb-ruler">
              <button className={weightUnit === 'lb' ? 'on' : ''} onClick={() => switchUnit('lb')}>lb</button>
              <button className={weightUnit === 'kg' ? 'on' : ''} onClick={() => switchUnit('kg')}>kg</button>
            </div>
          </div>
          <RulerPicker
            min={30} max={150} value={weight} unit={weightUnit}
            onChange={setWeight}
            toDisplay={v => faNum(v.toFixed(1))}
          />
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('weight', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
