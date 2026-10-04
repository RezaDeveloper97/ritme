'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { KickCounter, useKickOverview } from '@/features/pregnancy-tools';
import { useRouter } from '@/shared/i18n';
import {
  Card,
  EmptyState,
  HeaderButton,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/**
 * `/pregnancy/kicks` — «شمارش حرکات جنین» (B-N5-08, nbl_/nbd_Log_Kick). A flow
 * screen opened from the pregnancy log sheet's kicks tile: close returns to the
 * log, no bottom nav. A running count is restored from the server on load.
 */
export function LogKickPage() {
  const t = useTranslations('pregnancyTools');
  const router = useRouter();
  const query = useKickOverview();
  const [help, setHelp] = useState(false);

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
        {body}
      </div>
    </div>
  );
}
