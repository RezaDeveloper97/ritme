'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';

import { useChild, useChildren } from '@/entities/child';
import { DiaperCard, FeedFinishBar, FeedPanel, SleepCard, feedingHref, sectionOf, useFeedTimer } from '@/features/baby-log';
import { getApiErrorStatus } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  ChipGroup,
  EmptyState,
  HeaderButton,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { defaultChildId } from '../model/pick';

type T = ReturnType<typeof useTranslations<'babyLog'>>;

function Shell({ children, footer }: { children: React.ReactNode; footer?: React.ReactNode }) {
  return (
    <div className="view bfl-page">
      <SkyLayer />
      <div className="scroll bfl-scroll">{children}</div>
      {footer}
    </div>
  );
}

function Loading({ t }: { t: T }) {
  return (
    <SkeletonGroup label={t('loading')} className="bfl-skel bfl-body">
      <Skeleton shape="block" />
      <Skeleton shape="card" />
      <Skeleton shape="block" />
    </SkeletonGroup>
  );
}

/**
 * `/children/feeding` — the postpartum log sheet's feeding / sleep / diaper
 * tiles land here without a child id: it opens the first own child's screen
 * (`GET /children` lists own children youngest first), or asks to add one.
 */
function ResolveChild({ section }: { section: string | null }) {
  const t = useTranslations('babyLog');
  const router = useRouter();
  const list = useChildren();
  const id = list.data ? defaultChildId(list.data.children) : null;

  useEffect(() => {
    if (id) router.replace(feedingHref(id, sectionOf(section)));
  }, [id, router, section]);

  if (list.isError) {
    return (
      <Shell>
        <ScreenHeader title={t('titleNoName')} backIcon="close" backLabel={t('close')} onBack={() => router.push('/postpartum')} />
        <div className="bfl-body">
          <EmptyState
            icon="warning"
            title={t('errors.loadTitle')}
            body={t('errors.load')}
            action={
              <SecondaryButton icon="refresh" onClick={() => void list.refetch()} loading={list.isFetching}>
                {t('errors.retry')}
              </SecondaryButton>
            }
          />
        </div>
      </Shell>
    );
  }
  if (list.data && !id) {
    return (
      <Shell>
        <ScreenHeader title={t('titleNoName')} backIcon="close" backLabel={t('close')} onBack={() => router.push('/postpartum')} />
        <div className="bfl-body">
          <EmptyState
            icon="sprout"
            title={t('noChild.title')}
            body={t('noChild.body')}
            action={
              list.data.canAdd ? (
                <PrimaryButton block={false} onClick={() => router.push('/children/new')}>
                  {t('noChild.cta')}
                </PrimaryButton>
              ) : undefined
            }
          />
        </div>
      </Shell>
    );
  }
  return (
    <Shell>
      <ScreenHeader title={t('titleNoName')} backIcon="close" backLabel={t('close')} onBack={() => router.push('/postpartum')} />
      <Loading t={t} />
    </Shell>
  );
}

function FeedScreen({ id, section }: { id: number; section: string | null }) {
  const t = useTranslations('babyLog');
  const router = useRouter();
  const child = useChild(id);
  const list = useChildren();
  const timer = useFeedTimer(id);
  const [info, setInfo] = useState(false);
  const name = child.data?.name ?? '';
  const readOnly = child.data ? !child.data.canEdit : true;
  const others = list.data?.children ?? [];
  const target = sectionOf(section);

  // `?section=sleep|diapers` (child home rows, log sheet tiles): bring that card into view once it exists.
  const ready = !!child.data;
  useEffect(() => {
    if (!ready || target === 'feeding') return;
    document.getElementById(target)?.scrollIntoView({ block: 'start' });
  }, [ready, target]);

  const header = (
    <ScreenHeader
      title={name ? t('title', { name }) : t('titleNoName')}
      backIcon="close"
      backLabel={t('close')}
      onBack={() => router.push(`/children/${id}`)}
      action={<HeaderButton icon="info" label={t('info')} onClick={() => setInfo((v) => !v)} />}
    />
  );

  if (id <= 0 || getApiErrorStatus(child.error) === 404) {
    return (
      <Shell>
        {header}
        <div className="bfl-body">
          <EmptyState
            icon="sprout"
            title={t('notFound.title')}
            body={t('notFound.body')}
            action={<PrimaryButton block={false} onClick={() => router.push('/children')}>{t('notFound.cta')}</PrimaryButton>}
          />
        </div>
      </Shell>
    );
  }
  if (child.isError) {
    return (
      <Shell>
        {header}
        <div className="bfl-body">
          <EmptyState
            icon="warning"
            title={t('errors.loadTitle')}
            body={t('errors.load')}
            action={
              <SecondaryButton icon="refresh" onClick={() => void child.refetch()} loading={child.isFetching}>
                {t('errors.retry')}
              </SecondaryButton>
            }
          />
        </div>
      </Shell>
    );
  }
  if (!child.data) {
    return (
      <Shell>
        {header}
        <Loading t={t} />
      </Shell>
    );
  }

  return (
    <Shell footer={readOnly ? null : <FeedFinishBar timer={timer} />}>
      {header}
      <div className="bfl-body">
        {others.length > 1 ? (
          <ChipGroup label={t('picker')} className="bfl-picker">
            {others.map((c) => (
              <PillChip
                key={c.id}
                pressed={c.id === id}
                onPressedChange={() => {
                  if (c.id !== id) router.replace(feedingHref(c.id));
                }}
              >
                {c.name}
              </PillChip>
            ))}
          </ChipGroup>
        ) : null}
        {info ? <InfoNote>{t('infoBody')}</InfoNote> : null}
        {readOnly ? <InfoNote icon="eye">{t('readOnly')}</InfoNote> : null}
        <FeedPanel timer={timer} readOnly={readOnly} />
        <SleepCard childId={id} name={name} readOnly={readOnly} />
        <DiaperCard childId={id} readOnly={readOnly} />
      </div>
    </Shell>
  );
}

/**
 * `/children/[id]/feeding` (nbl_/nbd_Log_Feed, B-N5-07): the feed timer with
 * «پایان و ذخیره», then the baby's sleep and diapers. A child picker shows
 * when there is more than one child; a spouse sees a shared child read-only.
 * Without an `id` (`/children/feeding`) it resolves the child first.
 */
export function LogFeedPage({ id, section = null }: { id?: number; section?: string | null }) {
  const t = useTranslations('babyLog');
  const mounted = useMounted();
  if (!mounted) {
    return (
      <Shell>
        <Loading t={t} />
      </Shell>
    );
  }
  if (id === undefined) return <ResolveChild section={section} />;
  if (!Number.isFinite(id) || id <= 0) return <FeedScreen id={0} section={section} />;
  return <FeedScreen id={id} section={section} />;
}
