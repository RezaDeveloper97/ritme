'use client';

import { useTranslations } from 'next-intl';

import { Icon, PrimaryButton } from '@/shared/ui';

import { useDeviceNotifications } from './useDeviceNotifications';
import '@/shared/ui/styles/ntf.css';

/**
 * «اعلان‌ها روی این دستگاه» — on / off / blocked / not supported for THIS
 * device, with the enable button (or the hint that fixes it). Inside the
 * Android shell the copy says «این دستگاه» and never mentions a browser.
 */
export function DeviceNotificationsRow() {
  const t = useTranslations('pwa.push.device');
  const { state, channel, busy, failed, enable } = useDeviceNotifications();
  if (state === null) return null;

  const hint =
    state === 'blocked'
      ? t(channel === 'native' ? 'blockedHintShell' : 'blockedHint')
      : state === 'install'
        ? t('installHint')
        : state === 'unsupported'
          ? t('unsupportedHint')
          : state === 'on'
            ? t(channel === 'native' ? 'onHintShell' : 'onHint')
            : t(channel === 'native' ? 'offHintShell' : 'offHint');

  return (
    <aside className="nb-card ntf-neutral" aria-labelledby="ntf-device-title" data-device-state={state}>
      <Icon name="bell" size={18} className="ntf-neutral-icon" />
      <div className="ntf-neutral-text">
        <span id="ntf-device-title" className="ntf-neutral-title">
          {t('title')} — {t(`state.${state}`)}
        </span>
        <p className="ntf-neutral-body" role={failed ? 'alert' : undefined}>
          {failed ? t('failed') : hint}
        </p>
        {state === 'off' ? (
          <PrimaryButton block={false} loading={busy} onClick={() => void enable()}>
            {t('enable')}
          </PrimaryButton>
        ) : null}
      </div>
    </aside>
  );
}
