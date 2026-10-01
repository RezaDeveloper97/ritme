'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useDeactivatePregnancy } from '@/entities/pregnancy';
import { useLifeStage, useLossCopy, useUpdateLifeStage, writeLifeModeHint } from '@/entities/user';
import { useRouter } from '@/shared/i18n';
import {
  Card,
  EmptyState,
  IconCircle,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/**
 * The calm pregnancy exit (`/profile/mode/loss`, B-N2-03): reached only from the
 * mode screen while pregnancy mode is on («اگر بارداری‌ات ادامه پیدا نکرد…»).
 * One quiet confirm step, then a closing note — no congratulation, no
 * illustration, no toast. The copy is admin-edited (`pregnancy_setup/loss_exit`,
 * `GET /profile/life-stage/loss-copy`); the bundle covers any text not written yet.
 *
 * The exit turns the pregnancy profile off (`POST /pregnancy/deactivate`, which
 * stops every pregnancy screen, alert and message) and stores cycle tracking as
 * the mode — never TTC, so no conception nudges follow. All data is kept.
 * roadmap CB-LOSS-02 replaces this screen with the full loss path (`/loss`).
 */
export function PregnancyLossPage() {
  const t = useTranslations('me.mode.loss');
  const router = useRouter();
  const queryClient = useQueryClient();
  const stage = useLifeStage();
  const copy = useLossCopy();
  const deactivate = useDeactivatePregnancy();
  const update = useUpdateLifeStage();
  const [done, setDone] = useState(false);
  const [failed, setFailed] = useState(false);

  const busy = deactivate.isPending || update.isPending;
  const c = copy.data;

  const exit = async () => {
    if (busy) return;
    setFailed(false);
    try {
      await deactivate.mutateAsync();
      await update.mutateAsync({ mode: 'cycle' });
      writeLifeModeHint('cycle');
      setDone(true);
      void queryClient.invalidateQueries();
    } catch {
      setFailed(true);
    }
  };

  let body;
  if (done) {
    body = (
      <Card as="section" className="loss-card" aria-labelledby="loss-done-title">
        <IconCircle icon="heart" tone="brand" size="lg" />
        <h2 id="loss-done-title" className="loss-title" tabIndex={-1}>
          {c?.doneTitle ?? t('doneTitle')}
        </h2>
        <p className="loss-body">{c?.doneBody ?? t('doneBody')}</p>
        <div className="loss-btns">
          <PrimaryButton onClick={() => router.replace('/home')}>{c?.doneAction ?? t('doneAction')}</PrimaryButton>
        </div>
      </Card>
    );
  } else if (stage.isPending || copy.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="loss-card">
        <Skeleton shape="circle" />
        <Skeleton width="medium" />
        <Skeleton />
        <Skeleton />
      </SkeletonGroup>
    );
  } else if (stage.isError) {
    body = (
      <EmptyState
        icon="modeRing"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={stage.isFetching} onClick={() => void stage.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (stage.data?.mode !== 'pregnancy') {
    body = (
      <EmptyState
        icon="modeRing"
        title={t('notPregnant.title')}
        body={t('notPregnant.body')}
        action={<PrimaryButton onClick={() => router.replace('/profile/mode')}>{t('notPregnant.action')}</PrimaryButton>}
      />
    );
  } else {
    body = (
      <Card as="section" className="loss-card" aria-labelledby="loss-title">
        <IconCircle icon="heart" tone="brand" size="lg" />
        <h2 id="loss-title" className="loss-title">
          {c?.title ?? t('title')}
        </h2>
        <p className="loss-body">{c?.body ?? t('body')}</p>
        {failed ? (
          <p className="mode-error" role="alert">
            {t('error.save')}
          </p>
        ) : null}
        <div className="loss-btns">
          <PrimaryButton loading={busy} onClick={() => void exit()}>
            {busy ? t('closing') : (c?.confirm ?? t('confirm'))}
          </PrimaryButton>
          <SecondaryButton variant="text" block disabled={busy} onClick={() => router.push('/profile/mode')}>
            {c?.cancel ?? t('cancel')}
          </SecondaryButton>
        </div>
      </Card>
    );
  }

  return (
    <div className="view mode-page">
      <SkyLayer />
      <div className="scroll mode-scroll">
        <ScreenHeader
          title={t('screenTitle')}
          onBack={done ? undefined : () => router.push('/profile/mode')}
          backLabel={t('back')}
        />
        {body}
      </div>
    </div>
  );
}
