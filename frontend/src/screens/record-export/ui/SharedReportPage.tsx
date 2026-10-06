'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useSharedReport } from '@/entities/health-record';
import { ApiError } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatLongDate } from '@/shared/lib/date';
import { EmptyState, InfoNote, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { ReportPaper } from './ReportPaper';
import { usePaperModel, usePdfDownload } from './usePaper';

/**
 * The public, read-only doctor report behind a share link (bloom B-N6-04): no session, no navigation into the app,
 * not indexed (route metadata). 410 → the link expired or was revoked; 404 → unknown.
 */
export function SharedReportPage({ token }: { token: string }) {
  const t = useTranslations('recordExport.shared');
  const tp = useTranslations('recordExport.paper');
  const tr = useTranslations('recordExport');
  const loc = useLocale() as Locale;
  const query = useSharedReport(token);
  const model = usePaperModel(query.data, query.data?.question ?? null);
  const pdf = usePdfDownload();

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rx-skel">
        <Skeleton shape="card" className="rx-skel-paper" />
      </SkeletonGroup>
    );
  } else if (query.isError || !model || !query.data) {
    const status = query.error instanceof ApiError ? query.error.response?.status : undefined;
    const data = query.error instanceof ApiError ? (query.error.response?.data as { error_code?: string } | undefined) : undefined;
    if (status === 410) {
      body = (
        <EmptyState
          icon="lock"
          title={t('goneTitle')}
          body={data?.error_code === 'share_link_revoked' ? t('revokedBody') : t('expiredBody')}
        />
      );
    } else if (status === 404) {
      body = <EmptyState icon="search" title={t('notFoundTitle')} body={t('notFoundBody')} />;
    } else {
      body = (
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
    }
  } else {
    body = (
      <>
        <InfoNote icon="clock">{t('expires', { date: formatLongDate(new Date(query.data.expiresAt), loc) })}</InfoNote>
        <ReportPaper model={model} questionLabel={tp('question')} />
        <div className="rx-actions is-inline">
          <SecondaryButton icon="download" loading={pdf.busy} onClick={() => void pdf.run(model)}>
            {tr('download')}
          </SecondaryButton>
        </div>
        {pdf.failed ? (
          <p className="rx-error" role="alert">
            {tr('downloadError')}
          </p>
        ) : null}
      </>
    );
  }

  return (
    <div className="view rx-page is-public">
      <SkyLayer />
      <div className="scroll rx-scroll">
        <header className="rx-public-head">
          <h1 className="rx-public-title">{t('title')}</h1>
          <p className="rx-public-sub">{t('readOnly')}</p>
        </header>
        {body}
      </div>
    </div>
  );
}
