'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useRouter } from '@/shared/i18n';
import { Icon, InfoNote, ListGroup, ListRow, ScreenHeader, SkyLayer, StatusPill, type IconName } from '@/shared/ui';
import { pilot } from '@/shared/pilot';

import { DEVICE_KINDS, readInterest, writeInterest, type DeviceKind } from '../model/interest';

const ICON: Record<DeviceKind, IconName> = {
  watch: 'watch',
  band: 'watch',
  scale: 'box',
  thermometer: 'thermo',
  pressure: 'heartLine',
  glucose: 'drop',
};

/**
 * Watches & gadgets (B-N10-07, `nbl_Me_Hub` «ساعت و گجت‌ها») at `/profile/devices`. Honest «soon» state: the web
 * cannot pair devices, so there is no fake connected-device list — only the kinds we plan for and a local
 * «I'm interested» toggle (see `model/interest.ts` for the integration notes).
 */
export function DevicesPage() {
  const t = useTranslations('me');
  const router = useRouter();
  const [interest, setInterest] = useState<DeviceKind[]>([]);
  useEffect(() => setInterest(readInterest()), []);

  const toggle = (kind: DeviceKind) => {
    const next = interest.includes(kind) ? interest.filter((k) => k !== kind) : [...interest, kind];
    setInterest(next);
    writeInterest(next);
  };

  return (
    <div className="view dvc-page">
      <SkyLayer />
      <div className="scroll dvc-scroll">
        <ScreenHeader title={t('devices.title')} onBack={() => router.push('/profile')} backLabel={t('devices.back')} />

        <section className="nb-card dvc-hero" aria-labelledby="dvc-soon">
          <span className="dvc-hero-ic" aria-hidden>
            <Icon name="watch" size={26} />
          </span>
          <div className="dvc-hero-body">
            <h2 id="dvc-soon" className="dvc-hero-title">
              {t('devices.soonTitle')}
              <StatusPill tone="neutral">{t('soon')}</StatusPill>
            </h2>
            <p className="dvc-hero-text">{t('devices.soonBody')}</p>
          </div>
        </section>

        <section className="dvc-sec" aria-labelledby="dvc-kinds">
          <h2 id="dvc-kinds" className="dvc-label">
            {t('devices.kindsTitle')}
          </h2>
          <ListGroup>
            {DEVICE_KINDS.map((kind) => {
              const on = interest.includes(kind);
              return (
                <ListRow
                  key={kind}
                  icon={ICON[kind]}
                  iconTone="data"
                  title={t(`devices.kinds.${kind}`)}
                  description={t(`devices.kinds.${kind}Sub`)}
                  trailing={
                    <button
                      type="button"
                      className={clsx('dvc-notify', on && 'is-on')}
                      aria-pressed={on}
                      onClick={() => toggle(kind)}
                      {...pilot(`devices.notify:${kind}`)}
                    >
                      {on ? <Icon name="check" size={14} /> : null}
                      {on ? t('devices.interested') : t('devices.notify')}
                    </button>
                  }
                />
              );
            })}
          </ListGroup>
          <p className="dvc-note">{t('devices.localNote')}</p>
        </section>

        <InfoNote icon="heartLine">{t('devices.manual')}</InfoNote>
        <button type="button" className="dvc-link" onClick={() => router.push('/vitals')} {...pilot('devices.vitals')}>
          {t('devices.toVitals')}
        </button>
      </div>
    </div>
  );
}
