'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { EmergencyCardView, usePublicEmergencyCard } from '@/entities/health-record';
import { ApiError } from '@/shared/api';
import { replaceUrlPath, usePageHide } from '@/shared/lib/page-hide';
import { EmptyState, InfoNote, PrimaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import '@/shared/ui/styles/rem.css';

/**
 * `/emergency/[token]` — the public emergency card (CB-REC-05, D-72): no session, no navigation into the app, not
 * indexed, no referrer (route metadata). Minimal data only — the server sends first name, blood type, allergies when
 * flagged, chronic illnesses, ongoing medications, pregnancy week when shared and the contact; never insurance.
 */
export function PublicEmergencyPage({ token, scrubTo }: { token: string | null; scrubTo?: string }) {
  const t = useTranslations('record.publicEmergency');
  const queryClient = useQueryClient();
  // dropped on pagehide (shared device, bfcache) and never silently re-fetched
  const [closed, setClosed] = useState(false);
  const query = usePublicEmergencyCard(token ?? '', !!token && !closed);
  const opened = query.isSuccess;

  // after the first open the token leaves the address bar; a refresh lands on the token-less page («open the link again»)
  useEffect(() => {
    if (opened && scrubTo) replaceUrlPath(scrubTo);
  }, [opened, scrubTo]);

  usePageHide(() => {
    setClosed(true);
    queryClient.removeQueries({ queryKey: ['public-emergency-card'] });
  });

  let body;
  if (!token || closed) {
    body = <EmptyState icon="lock" title={t('closedTitle')} body={t('closedBody')} />;
  } else if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rem-skel">
        <Skeleton shape="card" className="rem-skel-card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
    const status = query.error instanceof ApiError ? query.error.response?.status : undefined;
    body =
      status === 404 ? (
        <EmptyState icon="search" title={t('notFoundTitle')} body={t('notFoundBody')} />
      ) : (
        <EmptyState
          icon="warning"
          title={t('errorTitle')}
          body={t('errorBody')}
          action={
            <PrimaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      );
  } else {
    body = (
      <>
        <EmergencyCardView card={query.data} variant="lock" />
        <InfoNote icon="info">{t('note')}</InfoNote>
      </>
    );
  }

  return (
    <div className="view rem-page is-public">
      <SkyLayer />
      <div className="scroll rem-scroll">
        <header className="rem-public-head">
          <h1 className="rem-public-title">{t('title')}</h1>
          <p className="rem-public-sub">{t('sub')}</p>
        </header>
        {body}
      </div>
    </div>
  );
}
