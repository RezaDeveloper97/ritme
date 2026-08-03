'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, NavBack } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';


export function NamePage() {
  const t = useTranslations('onboarding');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const { name, intention, setName } = useOnboardingStore();
  const [value, setValue] = useState(name);
  // Asked here rather than on the phone screen: this is the first step only a
  // brand-new account reaches, so returning users never re-accept.
  const [terms, setTerms] = useState(false);

  useEffect(() => { setValue(name); }, [name]);

  const step = stepPosition('name', intention);
  const canContinue = value.trim().length > 0 && terms;

  const handleNext = () => {
    if (!canContinue) return;
    setName(value.trim());
    router.push(nextOnboardingRoute('name', intention));
  };

  return (
    <div className="view onb-page">
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{formatNumber(step.index, loc)}<span className="onb-dim"> / {formatNumber(step.total, loc)}</span></span>
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

          <div
            role="checkbox"
            aria-checked={terms}
            tabIndex={0}
            className="signup-terms"
            onClick={() => setTerms(v => !v)}
            onKeyDown={e => (e.key === 'Enter' || e.key === ' ') && setTerms(v => !v)}
          >
            <span className="sub signup-terms-t">
              {t.rich('name.terms', {
                link: chunks => <b className="signup-terms-link">{chunks}</b>,
              })}
            </span>
            <span className={`cbx pink${terms ? ' on' : ''}`}>
              <Icon name="check" size={13} stroke="var(--on-accent)" />
            </span>
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
