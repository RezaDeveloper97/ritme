'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { appInviteUrl, shareAppInvite } from '@/entities/companion';
import { useDirection } from '@/shared/i18n';
import { Icon, IconCircle } from '@/shared/ui';
import '@/shared/ui/styles/cmh.css';

/** «شریکم هنوز ریتمی ندارد» — share the app link (moved here from the old companion code entry). */
export function InviteAppLink() {
  const t = useTranslations('companionHome.invite');
  const rtl = useDirection() === 'rtl';
  const [copied, setCopied] = useState(false);
  const invite = async () => {
    const outcome = await shareAppInvite(t('shareTitle'), t('shareText', { url: appInviteUrl() }));
    setCopied(outcome === 'copied');
  };
  return (
    <>
      <button type="button" className="nb-card cmh-invite" onClick={() => void invite()}>
        <IconCircle icon="share" tone="data" size="md" />
        <span className="cmh-invite-text">
          <b className="cmh-invite-title">{t('title')}</b>
          <span className="cmh-invite-sub">{t('sub')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="cmh-invite-chev" />
      </button>
      {copied ? (
        <p className="cmh-code-hint" role="status">
          {t('copied')}
        </p>
      ) : null}
    </>
  );
}
