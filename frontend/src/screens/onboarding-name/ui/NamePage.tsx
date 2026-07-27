'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { Icon, NavBack } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function NamePage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { name, intention, setName } = useOnboardingStore();
  const [value, setValue] = useState(name);

  useEffect(() => { setValue(name); }, [name]);

  const step = stepPosition('name', intention);
  const canContinue = value.trim().length > 0;

  const handleNext = () => {
    if (!canContinue) return;
    setName(value.trim());
    router.push(nextOnboardingRoute('name', intention));
  };

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span className="onb-dim"> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll onb-body">
        <div className="onb-intro">
          <div className="titr">{t('name.title')}</div>
          <p className="sub onb-intro-sub">{t('name.subtitle')}</p>
        </div>
        <div className="onb-center">
          <div className="onb-mt8">
            <label className="lbl">{t('name.label')}</label>
            <div className="field">
              <input
                placeholder={t('name.placeholder')}
                value={value}
                onChange={e => setValue(e.target.value)}
              />
              <span className="placeholder-soft">
                <Icon name="pencil" size={18} stroke="var(--muted-soft)" />
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="onb-actions">
        <button className="btn btn-primary" disabled={!canContinue} onClick={handleNext}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}
