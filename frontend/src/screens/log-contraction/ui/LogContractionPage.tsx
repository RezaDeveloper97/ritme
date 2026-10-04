'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { ContractionTimer, useContractionOverview } from '@/features/pregnancy-tools';
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
 * `/pregnancy/contractions` — «زمان‌سنج انقباض» (B-N5-08, nbl_/nbd_Log_Contraction). A flow
 * screen opened from the pregnancy log sheet's kicks tile: close returns to the
 * log, no bottom nav. A running session (and contraction) is restored from the server on load.
 */
export function LogContractionPage() {
  const t = useTranslations('pregnancyTools');
  const router = useRouter();
  const query = useContractionOverview();
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
        icon="clock"
        title={t('common.loadError')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('common.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <ContractionTimer overview={query.data} />;
  }

  return (
    <div className="view ptl-page">
      <SkyLayer />
      <div className="scroll ptl-scroll">
        <ScreenHeader
          title={t('contraction.title')}
          onBack={() => router.push('/pregnancy/log')}
          backLabel={t('common.close')}
          backIcon="close"
          action={<HeaderButton icon="info" label={t('common.help')} onClick={() => setHelp((v) => !v)} />}
        />
        {help ? (
          <Card className="ptl-help" role="note">
            <b className="ptl-note-title">{t('contraction.helpTitle')}</b>
            <p className="ptl-help-body">{t('contraction.helpBody')}</p>
            <p className="ptl-help-note">{t('contraction.helpNote')}</p>
          </Card>
        ) : null}
        {body}
      </div>
    </div>
  );
}
