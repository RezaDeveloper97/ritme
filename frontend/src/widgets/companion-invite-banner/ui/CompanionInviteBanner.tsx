'use client';

import { useTranslations } from 'next-intl';

import { useModuleEnabled } from '@/entities/app-module';
import { useCompanions } from '@/entities/companion';
import { useLifeStage } from '@/entities/user';
import { Link, useDirection } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { useSessionFlag } from '@/shared/lib/session-flag';
import { Icon } from '@/shared/ui';

const DISMISS_KEY = 'companion_invite_banner';

/**
 * «همراهت را به ریتمی دعوت کن» (companion-v2 `home`): a soft, dismissible entry
 * to `/companions/new` on the cycle home. Only while the «همراه» module is on,
 * for a non-companion account with no invited or active companion; «بستن»
 * hides it for the rest of the session.
 */
export function CompanionInviteBanner() {
  const enabled = useModuleEnabled('companion');
  const life = useLifeStage();
  const companionAccount = Boolean(life.data?.companion);
  const [dismissed, dismiss] = useSessionFlag(DISMISS_KEY);
  const companions = useCompanions(enabled && !companionAccount && !dismissed);

  // Wait for the list: an error or a pending read shows nothing rather than inviting someone who already has a companion.
  if (!enabled || companionAccount || dismissed || !companions.data || companions.data.length > 0) return null;
  return <Banner onDismiss={dismiss} />;
}

function Banner({ onDismiss }: { onDismiss: () => void }) {
  const t = useTranslations('companions.banner');
  const rtl = useDirection() === 'rtl';
  return (
    <div className="cmp2-banner">
      <Link href="/companions/new" className="cmp2-banner-link" {...pilot("companion.open:companions-new")}>
        <span className="cmp2-banner-icon" aria-hidden>
          <Icon name="users" size={22} />
        </span>
        <span className="cmp2-banner-text">
          <b>{t('title')}</b>
          <span>{t('body')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="cmp2-banner-chev" />
      </Link>
      <button type="button" className="cmp2-banner-close" aria-label={t('dismiss')} onClick={onDismiss}>
        <Icon name="x" size={16} />
      </button>
    </div>
  );
}
