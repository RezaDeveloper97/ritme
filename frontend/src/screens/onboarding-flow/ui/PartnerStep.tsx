'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { OnbFrame } from '@/features/auth';
import { JoinCodeStep, JoinConfirmStep, useJoinCompanion } from '@/features/join-companion';
import { useRouter } from '@/shared/i18n';
import { Icon, IconCircle } from '@/shared/ui';

import { PARTNER_LINKED_ROUTE } from '../model/flow';
import { StepScreen, type StepContext } from './StepScreen';
import '@/shared/ui/styles/cmh.css';

type Stage = 'intro' | 'code';

function PartnerView({ ctx }: { ctx: StepContext }) {
  // Partner-step copy lives with the companion member side (companionHome.onboardingPartner, companion v2).
  const t = useTranslations('companionHome');
  const router = useRouter();
  const [stage, setStage] = useState<Stage>('intro');
  const join = useJoinCompanion(() => router.push(PARTNER_LINKED_ROUTE));
  const skip = () => ctx.next();

  if (stage === 'intro') {
    return (
      <OnbFrame
        progress={ctx.progress}
        onBack={ctx.back}
        privacyNote={false}
        primary={{ label: t('onboardingPartner.enterCode'), onClick: () => setStage('code') }}
        secondary={{ label: t('onboardingPartner.skip'), onClick: skip }}
      >
        <div className="cmh2-join">
          <span className="cmh2-illus" aria-hidden>
            <Icon name="users" size={34} />
          </span>
          <div className="cmh2-join-intro">
            <h1 className="cmh2-title">{t('onboardingPartner.title')}</h1>
            <p className="cmh2-sub">{t('onboardingPartner.sub')}</p>
          </div>
          <div className="nb-card cmh2-how">
            <div className="cmh2-how-row">
              <IconCircle icon="book" tone="brand" size="sm" />
              <span className="cmh2-how-text">
                <b>{t('onboardingPartner.forAllTitle')}</b>
                {t('onboardingPartner.forAll')}
              </span>
            </div>
          </div>
        </div>
      </OnbFrame>
    );
  }

  if (join.step === 'confirm') {
    return (
      <OnbFrame
        progress={ctx.progress}
        onBack={join.backToCode}
        privacyNote={false}
        primary={{ label: t('join.confirm.start'), onClick: join.confirm, loading: join.accepting }}
      >
        <JoinConfirmStep join={join} />
      </OnbFrame>
    );
  }

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={() => setStage('intro')}
      onSkip={skip}
      primary={{
        label: join.checking ? t('code.connecting') : t('code.connect'),
        onClick: join.submitCode,
        disabled: !join.complete,
        loading: join.checking,
      }}
      secondary={{ label: t('onboardingPartner.skip'), onClick: skip }}
    >
      <JoinCodeStep join={join} autoFocus />
    </OnbFrame>
  );
}

/**
 * Male onboarding, companion v2 `ob-male-code`: an optional invitation — «واردکردن کد همراه» opens the code
 * entry and its confirm step (shared with `/companion/links`, features/join-companion); accepting opens the
 * linked screen. «فعلاً رد می‌کنم» finishes onboarding without a link (Ready → `/companion`, his personal
 * panel, where the «کد همراه داری؟» banner offers the code again).
 */
export function PartnerStep() {
  return <StepScreen step="partner">{(ctx) => <PartnerView ctx={ctx} />}</StepScreen>;
}
