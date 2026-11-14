'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { usePregnancyAlertsV2, type PregnancyAlertV2 } from '@/entities/pregnancy';
import { KickCounter, useKickOverview } from '@/features/pregnancy-tools';
import { useRouter } from '@/shared/i18n';
import { openSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { openKickAlerts, useAckKickAlert } from '../model/kick-alerts';

/**
 * `/pregnancy/kicks` — «شمارش حرکات جنین» (B-N5-08, nbl_/nbd_Log_Kick). A flow
 * screen opened from the pregnancy log sheet's kicks tile: close returns to the
 * log, no bottom nav. A running count is restored from the server on load.
 *
 * A count that ends, or runs past its window, short of the target raises the
 * server's urgent `fetal_movement` alert (T-M9-05). Every change of the kick
 * overview (stop, a kick, the lazy expiry on load) re-reads the alert list, so
 * the alert card shows right away above the counter.
 */
export function LogKickPage() {
  const t = useTranslations('pregnancyTools');
  const router = useRouter();
  const query = useKickOverview();
  const alerts = usePregnancyAlertsV2();
  const [help, setHelp] = useState(false);
  const { refetch: refetchAlerts } = alerts;
  const overviewAt = query.dataUpdatedAt;
  useEffect(() => {
    if (overviewAt) void refetchAlerts();
  }, [overviewAt, refetchAlerts]);
  const urgent = openKickAlerts(alerts.data?.alerts);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="ptl-skel">
        <Skeleton shape="line" width="medium" />
        <Skeleton shape="circle" className="ptl-skel-ring" />
        <div className="ptl-stats">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </div>
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="heart"
        title={t('common.loadError')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('common.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <KickCounter overview={query.data} />;
  }

  return (
    <div className="view ptl-page">
      <SkyLayer />
      <div className="scroll ptl-scroll">
        <ScreenHeader
          title={t('kick.title')}
          onBack={() => router.push('/pregnancy/log')}
          backLabel={t('common.close')}
          backIcon="close"
          action={<HeaderButton icon="info" label={t('common.help')} onClick={() => setHelp((v) => !v)} />}
        />
        {help ? (
          <Card className="ptl-help" role="note">
            <b className="ptl-note-title">{t('kick.helpTitle')}</b>
            <p className="ptl-help-body">{t('kick.helpBody')}</p>
            <p className="ptl-help-note">{t('kick.helpNote')}</p>
          </Card>
        ) : null}
        {urgent.map((a) => (
          <KickAlertCard key={a.id} alert={a} />
        ))}
        {body}
      </div>
    </div>
  );
}

/** The urgent alert as the server raised it: its copy, a call CTA and «دیدم، ممنون». */
function KickAlertCard({ alert }: { alert: PregnancyAlertV2 }) {
  const t = useTranslations('pregnancyTools.kick');
  const ack = useAckKickAlert();
  const phone = alert.contact?.phone?.replace(/[^\d+]/g, '') ?? '';
  const call = alert.actions.find((a) => a.key === 'call');
  const ackAction = alert.actions.find((a) => a.key === 'ack');
  return (
    <Card as="article" className="ptl-advice is-urgent" role="alert" aria-labelledby={`ptl-kick-alert-${alert.id}`}>
      <div className="ptl-advice-head">
        <Icon name="warning" size={18} className="ptl-advice-icon" />
        <b id={`ptl-kick-alert-${alert.id}`} className="ptl-note-title">
          {alert.title}
        </b>
      </div>
      {alert.whatWeSaw ? <p className="ptl-advice-body">{alert.whatWeSaw}</p> : null}
      {alert.advice ? <p className="ptl-advice-body">{alert.advice}</p> : null}
      {alert.howSure ? <p className="ptl-advice-sub">{alert.howSure}</p> : null}
      {alert.contact?.text ? <p className="ptl-advice-sub is-strong">{alert.contact.text}</p> : null}
      {/* With no number on the rule, `tel:` opens the dialer. */}
      <a className="nb-btn is-primary is-block ptl-call" href={`tel:${phone}`}>
        <Icon name="phone" size={18} />
        {call?.label ?? t('call')}
      </a>
      <button
        type="button"
        className="nb-btn is-text is-block"
        disabled={ack.isPending}
        onClick={() => ack.mutate(alert.id)}
      >
        {ackAction?.label ?? t('alertAck')}
      </button>
      <button type="button" className="ptl-link" onClick={() => openSheet('notifications')}>
        {t('seeAlerts')}
      </button>
    </Card>
  );
}
