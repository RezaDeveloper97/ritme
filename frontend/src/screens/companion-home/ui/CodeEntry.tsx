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
  type ViewerLink,
} from '@/entities/companion';
import { useDirection } from '@/shared/i18n';
import { Icon, IconCircle, PrimaryButton } from '@/shared/ui';

interface CodeEntryProps {
  onLinked?: (link: ViewerLink) => void;
  autoFocus?: boolean;
}

/**
 * Companion code entry outside onboarding — the companion home's empty state
 * and `/companion/links` (Me «کد همدم»): the 6 boxes, «اتصال», the error line
 * and the «شریکم هنوز ریتمی ندارد» invite link.
 */
export function CodeEntry({ onLinked, autoFocus }: CodeEntryProps) {
  const t = useTranslations('companionHome');
  const hintId = useId();
  const rtl = useDirection() === 'rtl';
  const accept = useAcceptCompanion();
  const [code, setCode] = useState('');
  const [copied, setCopied] = useState(false);
  const [linkedName, setLinkedName] = useState<string | null | undefined>(undefined);
  const complete = isCompleteCompanionCode(code);

  const connect = () => {
    if (!complete || accept.isPending) return;
    accept.mutate(code, {
      onSuccess: (link) => {
        setCode('');
        setLinkedName(link.owner.name);
        onLinked?.(link);
      },
    });
  };

  const invite = async () => {
    const outcome = await shareAppInvite(t('invite.shareTitle'), t('invite.shareText', { url: appInviteUrl() }));
    setCopied(outcome === 'copied');
  };

  return (
    <div className="cmh-entry">
      <CompanionCodeField
        value={code}
        onChange={(next) => {
          setCode(next);
          setLinkedName(undefined);
          if (accept.isError) accept.reset();
        }}
        onSubmit={connect}
        label={t('code.label')}
        describedBy={hintId}
        invalid={accept.isError}
        disabled={accept.isPending}
        autoFocus={autoFocus}
      />
      {accept.isError ? (
        <p className="onb2-error is-center" role="alert">
          {t(`code.errors.${acceptErrorKind(accept.error)}`)}
        </p>
      ) : linkedName !== undefined ? (
        <p className="cmh-code-hint is-success" role="status">
          {linkedName ? t('code.success', { name: linkedName }) : t('code.successNoName')}
        </p>
      ) : (
        <p id={hintId} className="cmh-code-hint">
          {t('code.hint')}
        </p>
      )}
      <PrimaryButton block onClick={connect} disabled={!complete || accept.isPending} loading={accept.isPending}>
        {accept.isPending ? t('code.connecting') : t('code.connect')}
      </PrimaryButton>
      <button type="button" className="nb-card cmh-invite" onClick={() => void invite()}>
        <IconCircle icon="share" tone="data" size="md" />
        <span className="cmh-invite-text">
          <b className="cmh-invite-title">{t('invite.title')}</b>
          <span className="cmh-invite-sub">{t('invite.sub')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="cmh-invite-chev" />
      </button>
      {copied ? (
        <p className="cmh-code-hint" role="status">
          {t('invite.copied')}
        </p>
      ) : null}
    </div>
  );
}
