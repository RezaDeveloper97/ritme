'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { HomeIndicator, Icon, NavBack, StatusBar } from '@/shared/ui';
import { useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function NamePage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { name, setName } = useOnboardingStore();
  const [value, setValue] = useState(name);

  useEffect(() => { setValue(name); }, [name]);

  const canContinue = value.trim().length > 0;

  const handleNext = () => {
    if (!canContinue) return;
    setName(value.trim());
    router.push('/onboarding/gender');
  };

  return (
    <div className="view" style={{ background: '#fff' }}>
      <StatusBar />
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(1)}<span style={{ opacity: .5 }}> / ۷</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('name.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('name.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <div style={{ marginTop: 8 }}>
            <label className="lbl">{t('name.label')}</label>
            <div className="field">
              <input
                placeholder={t('name.placeholder')}
                value={value}
                onChange={e => setValue(e.target.value)}
              />
              <span style={{ color: '#A9B2BC' }}>
                <Icon name="pencil" size={18} stroke="#A9B2BC" />
              </span>
            </div>
          </div>
        </div>
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" disabled={!canContinue} onClick={handleNext}>
          {t('continue')}
        </button>
      </div>
      <HomeIndicator />
    </div>
  );
}
