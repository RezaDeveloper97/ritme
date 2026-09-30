'use client';

import { useTranslations } from 'next-intl';

import { PrimaryButton, SecondaryButton } from '@/shared/ui';
import { CycleDotRing } from '@/widgets/intro-carousel';

interface Props {
  onStart: () => void;
  onLogin: () => void;
}

/** Night & Bloom `Onb_Welcome`: the brand ring, one line of promise, start / sign in. */
export function WelcomeCard({ onStart, onLogin }: Props) {
  const t = useTranslations('welcome.card');
  return (
    <div className="view ib-page">
      <span className="ib-glow" aria-hidden />
      <main className="ib-welcome">
        <CycleDotRing today={11}>
          <h1 className="ib-brand">{t('brand')}</h1>
          <span className="ib-brand-tag">{t('tagline')}</span>
        </CycleDotRing>
        <p className="ib-body ib-welcome-body">{t('body')}</p>
      </main>
      <footer className="ib-foot is-welcome">
        <PrimaryButton onClick={onStart}>{t('start')}</PrimaryButton>
        <SecondaryButton variant="text" className="ib-login" onClick={onLogin}>
          {t('login')}
        </SecondaryButton>
      </footer>
    </div>
  );
}
