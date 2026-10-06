'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { VitalAlert } from '@/entities/vital';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { Icon, SecondaryButton, UrgentCard } from '@/shared/ui';

/** Emergency number of the bundled fallback (Iran EMS), used when the alert payload carries no call action. */
const FALLBACK_PHONE = '115';

/**
 * The urgent modal of a saved reading over a safety threshold (B-N6-01:
 * BP > 180/120, glucose < 54 mg/dL). The reading is already stored — saving is
 * never blocked; this only tells her what to do, with a one-tap call. Copy is
 * the server's (admin-editable `vitals_alert`), with a bundled fallback.
 */
export function UrgentAlertSheet({ alert, onDone }: { alert: VitalAlert; onDone: () => void }) {
  const t = useTranslations('vitals.alert');
  const locale = useLocale() as Locale;
  const calls = alert.actions.filter((a) => a.phone);
  const ack = alert.actions.find((a) => a.key === 'ack');
  const callList = calls.length ? calls : [{ key: 'call', label: t('call', { number: formatNumber(FALLBACK_PHONE, locale) }), phone: FALLBACK_PHONE }];
  return (
    <AppSheet open size="half" onClose={onDone} className="vt-alert-sheet">
      <UrgentCard
        title={alert.title ?? t('title')}
        action={
          <div className="vt-alert-calls">
            {callList.map((a) => (
              <a key={a.key + a.phone} className="nb-btn is-primary is-block vt-alert-call" href={`tel:${a.phone}`}>
                <Icon name="phone" size={18} />
                {a.label}
              </a>
            ))}
          </div>
        }
        actions={<SecondaryButton onClick={onDone}>{ack?.label ?? t('ack')}</SecondaryButton>}
      >
        {alert.whatWeSaw || alert.advice || alert.contact ? (
          <>
            {alert.whatWeSaw ? <p className="vt-alert-p is-strong">{alert.whatWeSaw}</p> : null}
            {alert.advice ? <p className="vt-alert-p">{alert.advice}</p> : null}
            {alert.contact ? <p className="vt-alert-p">{alert.contact}</p> : null}
          </>
        ) : (
          <p className="vt-alert-p">{t('body')}</p>
        )}
      </UrgentCard>
    </AppSheet>
  );
}
