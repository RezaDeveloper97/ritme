'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { HomeIndicator,  NavBack, RulerPicker, StatusBar } from '@/shared/ui';
import { useOnboardingStore, type WeightUnit } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function WeightPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { weight, weightUnit, setWeight, setWeightUnit } = useOnboardingStore();

  const switchUnit = (u: WeightUnit) => {
    setWeightUnit(u);
    if (u === 'lb' && weightUnit === 'kg') setWeight(Math.round(weight * 2.205 * 10) / 10);
    if (u === 'kg' && weightUnit === 'lb') setWeight(Math.round(weight / 2.205 * 10) / 10);
  };

  return (
    <div className="view" style={{ background: '#fff' }}>
      <StatusBar />
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(4)}<span style={{ opacity: .5 }}> / ۷</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('weight.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('weight.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 30 }}>
            <div className="seg" style={{ width: 170 }}>
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

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push('/onboarding/height')}>
          {t('continue')}
        </button>
      </div>
      <HomeIndicator />
    </div>
  );
}
