'use client';

import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  acceptErrorKind,
  appInviteUrl,
  CompanionCodeField,
  isCompleteCompanionCode,
  shareAppInvite,
  useAcceptCompanion,
} from '@/entities/companion';
import { useDirection, useRouter } from '@/shared/i18n';
import { Icon, IconCircle } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { PARTNER_LINKED_ROUTE } from '../model/flow';
import { StepScreen, type StepContext } from './StepScreen';

function PartnerView({ ctx }: { ctx: StepContext }) {
  const t = useTranslations('onboarding.flow.partner');
  const router = useRouter();
  const hintId = useId();
  const rtl = useDirection() === 'rtl';
  const accept = useAcceptCompanion();
  const [code, setCode] = useState('');
  const [copied, setCopied] = useState(false);
  const sees = ['phase', 'fertile', 'tips'] as const;
  const complete = isCompleteCompanionCode(code);

  const connect = () => {
    if (!complete || accept.isPending) return;
    // The code lives only in this component's state and the request body.
    accept.mutate(code, { onSuccess: () => router.push(PARTNER_LINKED_ROUTE) });
  };

  const invite = async () => {
    const outcome = await shareAppInvite(t('inviteShareTitle'), t('inviteShareText', { url: appInviteUrl() }));
    setCopied(outcome === 'copied');
  };

  return (
    <OnbFrame
      progress={ctx.progress}
      onBack={ctx.back}
      onSkip={() => ctx.next()}
      title={t('title')}
      subtitle={t('subtitle')}
      primary={{
        label: accept.isPending ? t('connecting') : t('connect'),
        onClick: connect,
        disabled: !complete || accept.isPending,
        loading: accept.isPending,
      }}
      secondary={{ label: t('later'), onClick: () => ctx.next() }}
    >
      <CompanionCodeField
        value={code}
        onChange={(next) => {
          setCode(next);
          if (accept.isError) accept.reset();
        }}
        onSubmit={connect}
        label={t('codeLabel')}
        describedBy={hintId}
        invalid={accept.isError}
        disabled={accept.isPending}
      />
      {accept.isError ? (
        <p className="onb2-error is-center" role="alert">
          {t(`errors.${acceptErrorKind(accept.error)}`)}
        </p>
      ) : (
        <p id={hintId} className="cmh-code-hint">
          {t('hint')}
        </p>
      )}
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
      <button type="button" className="nb-card cmh-invite" onClick={() => void invite()}>
        <IconCircle icon="share" tone="data" size="md" />
        <span className="cmh-invite-text">
          <b className="cmh-invite-title">{t('inviteTitle')}</b>
          <span className="cmh-invite-sub">{t('inviteSub')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="cmh-invite-chev" />
      </button>
      {copied ? (
        <p className="cmh-code-hint" role="status">
          {t('inviteCopied')}
        </p>
      ) : null}
    </OnbFrame>
  );
}

/**
 * nbl_Onb_Partner (B-N4-05): a man enters the 6-character companion code his
 * partner sent; «اتصال» accepts it (`POST /companions/accept`) and opens the
 * linked screen. «بعداً وصل می‌شوم» / «رد کردن» finish onboarding without a
 * link (Ready → `/companion`, where the empty state offers the code again).
 */
export function PartnerStep() {
  return <StepScreen step="partner">{(ctx) => <PartnerView ctx={ctx} />}</StepScreen>;
}
