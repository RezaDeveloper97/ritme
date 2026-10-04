'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Link, useDirection, useRouter } from '@/shared/i18n';
import { toApiDate, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { Icon, IconCircle, ListRow } from '@/shared/ui';

import { useLossState } from '../api/queries';
import { hideLossCareRow, isLossCareOpen, isLossCareRowHidden } from '../model/care-return';

interface LossCareReturnProps {
  /**
   * `home` — a quiet card on her cycle / TTC home with «پنهان کردن» (per loss, per device);
   * `row` — a plain settings row (/profile/mode), always there while the window is open.
   */
  variant: 'home' | 'row';
  className?: string;
}

/**
 * The calm way back into `/loss/care` (CB-LOSS-03b) while a recent loss exists — see
 * {@link isLossCareOpen} for the window. Renders nothing otherwise (and nothing until
 * `GET /loss` has answered). The copy never names the loss: the home can be glanced at.
 */
export function LossCareReturn({ variant, className }: LossCareReturnProps) {
  const t = useTranslations('loss.return');
  const router = useRouter();
  const mounted = useMounted();
  const state = useLossState();
  const [hidden, setHidden] = useState<number | null>(null);
  const loss = state.data?.loss ?? null;
  const rtl = useDirection() === 'rtl';

  if (!mounted || !loss || !isLossCareOpen(loss, toApiDate(today()))) return null;

  if (variant === 'row') {
    return (
      <div className={['lcr-group', className].filter(Boolean).join(' ')}>
        <ListRow
          icon="heart"
          iconTone="brand"
          title={t('title')}
          description={t('rowSub')}
          onClick={() => router.push('/loss/care')}
        />
      </div>
    );
  }

  if (hidden === loss.id) {
    return (
      <p role="status" className="lcr-hidden">
        {t('hidden')}
      </p>
    );
  }
  if (isLossCareRowHidden(loss.id)) return null;

  return (
    <div className={['lcr-card', className].filter(Boolean).join(' ')}>
      <Link href="/loss/care" className="lcr-main">
        <IconCircle icon="heart" tone="brand" size="sm" />
        <span className="lcr-text">
          <b>{t('title')}</b>
          <span>{t('sub')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="lcr-chev" />
      </Link>
      <button
        type="button"
        className="lcr-hide"
        aria-label={t('hide')}
        onClick={() => {
          hideLossCareRow(loss.id);
          setHidden(loss.id);
        }}
      >
        <Icon name="x" size={16} />
      </button>
    </div>
  );
}
