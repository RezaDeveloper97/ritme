'use client';

import { useTranslations } from 'next-intl';

import { Icon, InfoNote } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { StepScreen, type StepContext } from './StepScreen';

const CODE_LENGTH = 6;

function PartnerView({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow.partner');
  const sees = ['phase', 'fertile', 'tips'] as const;
  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      title={t('title')}
      subtitle={t('subtitle')}
      primary={{ label: t('connect'), onClick: () => undefined, disabled: true }}
      secondary={{ label: t('later'), onClick: () => ctx.next() }}
    >
      {/* Stub until B-N4-05: the code boxes are shown but linking is not live yet. */}
      <div dir="ltr" className="onb2-otp is-code" aria-hidden>
        {Array.from({ length: CODE_LENGTH }, (_, i) => (
          <span key={i} className="onb2-otp-box is-static" />
        ))}
      </div>
      <InfoNote icon="clock">{t('soon')}</InfoNote>
      <section className="nb-card onb2-panel">
        <h2 className="onb2-panel-title">{t('seesTitle')}</h2>
        <ul className="onb2-sees">
          {sees.map((k) => (
            <li key={k} className="onb2-sees-item">
              <Icon name="check" size={18} strokeWidth={2.4} className="onb2-sees-yes" />
              {t(`sees.${k}`)}
            </li>
          ))}
          <li className="onb2-sees-item is-no">
            <Icon name="x" size={18} strokeWidth={2.4} className="onb2-sees-no" />
            {t('sees.private')}
          </li>
        </ul>
        <p className="onb2-panel-note">{t('revoke')}</p>
      </section>
    </OnbFrame>
  );
}

/** nbl_Onb_Partner — stub until the companion link ships (B-N4-05); «بعداً وصل می‌شوم» finishes onboarding. */
export function PartnerStep() {
  return <StepScreen step="partner">{(ctx) => <PartnerView ctx={ctx} />}</StepScreen>;
}
