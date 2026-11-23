'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useEffect, useState } from 'react';

import { type SharedReport, useSharedReport } from '@/entities/health-record';
import { ApiError } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatLongDate } from '@/shared/lib/date';
import { replaceUrlPath, usePageHide } from '@/shared/lib/page-hide';
import { EmptyState, InfoNote, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { ReportPaper } from './ReportPaper';
import { SharedDocuments } from './SharedDocuments';
import { usePaperModel, usePdfDownload } from './usePaper';
import './rx.css';

/**
 * The public, read-only doctor report behind a share link (bloom B-N6-04): no session, no navigation into the app,
 * not indexed (route metadata). 410 → the link expired or was revoked; 404 → unknown (CB-REC-05: also an expired /
 * revoked 24h summary). A summary (`kind: summary`) also lists the documents the owner chose.
 */
export function SharedReportPage({
  token,
  scrubTo,
  summaryScrubTo,
}: {
  token: string;
  /** Always scrub the token + drop the report on pagehide (`/d/[token]`). */
  scrubTo?: string;
  /** CB-REC-06 (M1): the same, but only once the opened report is a 24h summary (`/shared/report/[token]`). */
  summaryScrubTo?: string;
}) {
  const t = useTranslations('recordExport.shared');
  const td = useTranslations('record.doctor');
  const queryClient = useQueryClient();
  // CB-REC-05 `/d/[token]`: once closed (pagehide) the report is dropped and never silently re-fetched.
  const [closed, setClosed] = useState(false);
  const query = useSharedReport(token, !closed);
  const opened = query.isSuccess;
  // CB-REC-06 (M1): a summary opened through bloom's `/shared/report/[token]` (the server's QR path) gets the `/d`
  // treatment too; bloom's 7-day report links keep their behaviour (no scrub, no pagehide reset).
  const scrub = scrubTo ?? (query.data?.kind === 'summary' ? summaryScrubTo : undefined);

  // CB-REC-05 `/d/[token]`: after the first successful open, the token leaves the address bar (a refresh then lands
  // on the code page).
  useEffect(() => {
    if (opened && scrub) replaceUrlPath(scrub);
  }, [opened, scrub]);

  usePageHide(() => {
    if (!scrub) return;
    setClosed(true);
    queryClient.removeQueries({ queryKey: ['shared-report', token] });
  });

  let body;
  if (closed) {
    body = <EmptyState icon="lock" title={td('closedTitle')} body={td('closedBody')} />;
  } else if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="rx-skel">
        <Skeleton shape="card" className="rx-skel-paper" />
      </SkeletonGroup>
    );
  } else if (query.isError || !query.data) {
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
    body = <SharedReportBody report={query.data} />;
  }

  return <SharedReportShell>{body}</SharedReportShell>;
}

/** The public page frame: sky, title, «read only» — no header buttons, no nav (CB-REC-05 reuses it for `/d`). */
export function SharedReportShell({ children, title }: { children: ReactNode; title?: string }) {
  const t = useTranslations('recordExport.shared');
  return (
    <div className="view rx-page is-public">
      <SkyLayer />
      <div className="scroll rx-scroll">
        <header className="rx-public-head">
          <h1 className="rx-public-title">{title ?? t('title')}</h1>
          <p className="rx-public-sub">{t('readOnly')}</p>
        </header>
        {children}
      </div>
    </div>
  );
}

/** One opened report: expiry note, the paper, a summary's documents, the on-device PDF. */
export function SharedReportBody({ report }: { report: SharedReport }) {
  const t = useTranslations('recordExport.shared');
  const tp = useTranslations('recordExport.paper');
  const tr = useTranslations('recordExport');
  const loc = useLocale() as Locale;
  const model = usePaperModel(report, report.question ?? null);
  const pdf = usePdfDownload();
  if (!model) return null;
  return (
    <>
      <InfoNote icon="clock">{t('expires', { date: formatLongDate(new Date(report.expiresAt), loc) })}</InfoNote>
      <ReportPaper model={model} questionLabel={tp('question')} />
      {report.kind === 'summary' && report.documents.length ? <SharedDocuments documents={report.documents} /> : null}
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
