'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack, RulerPicker } from '@/shared/ui';
import { useOnboardingStore, type HeightUnit } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function HeightPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { height, heightUnit, setHeight, setHeightUnit } = useOnboardingStore();

  const switchUnit = (u: HeightUnit) => {
    setHeightUnit(u);
    if (u === 'ft' && heightUnit === 'cm') setHeight(Math.round(height / 30.48 * 100) / 100);
    if (u === 'cm' && heightUnit === 'ft') setHeight(Math.round(height * 30.48));
  };

  const [min, max] = heightUnit === 'cm' ? [120, 220] : [4, 7];

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(5)}<span style={{ opacity: .5 }}> / ۸</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('height.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('height.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 30 }}>
            <div className="seg" style={{ width: 170 }}>
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

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push('/onboarding/period-len')}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
