'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { formatDateTime, formatNumber } from '@/shared/lib';
import { Button, ErrorState, PageHeader, Panel, Skeleton, Spinner, toast, useNotifyError } from '@/shared/ui';

import { supportReportsApi, useScreenshotUrl, type SupportReport } from '../api/support-reports';
import { reporterName, StatusBadge } from './status';

/** /support-reports/:id — the full report, its screenshot and resolve / reopen (B-N1-12b). */
export function SupportReportDetailScreen({ id }: { id: number }) {
  const t = useTranslations('supportReports');
  const query = supportReportsApi.useDetail(id);

  if (query.error && !query.data) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader title={t('detailTitle')} backHref="/support-reports" backLabel={t('backToList')} />
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      </div>
    );
  }
  if (!query.data) {
    return (
      <div className="flex flex-col gap-4" aria-busy="true">
        <Skeleton className="h-8 w-60" />
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }
  return <ReportDetail report={query.data.support_report} />;
}

function ReportDetail({ report }: { report: SupportReport }) {
  const t = useTranslations('supportReports');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const resolve = supportReportsApi.useAction('resolve');
  const reopen = supportReportsApi.useAction('reopen');
  const isOpen = report.status === 'open';

  const toggle = () => {
    const action = isOpen ? resolve : reopen;
    action.mutate(
      { id: report.id },
      { onSuccess: () => toast.success(isOpen ? t('resolvedToast') : t('reopenedToast')), onError: notifyError },
    );
  };

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={t('detailHeading', { id: formatNumber(report.id, locale) })}
        backHref={isOpen ? '/support-reports' : '/support-reports?status=resolved'}
        backLabel={t('backToList')}
        meta={
          <>
            <StatusBadge status={report.status} />
            <span>{t('sentOn', { date: formatDateTime(report.created_at, locale) })}</span>
          </>
        }
        actions={
          <Button variant={isOpen ? 'primary' : 'default'} onClick={toggle} loading={resolve.isPending || reopen.isPending}>
            {isOpen ? t('resolve') : t('reopen')}
          </Button>
        }
      />

      <Panel title={t('message')}>
        <p className="m-0 whitespace-pre-wrap break-words leading-7" dir="auto">
          {report.message}
        </p>
      </Panel>

      <Panel title={t('screenshot')}>
        {report.has_screenshot ? <Screenshot id={report.id} /> : <p className="m-0 text-ink-3">{t('noScreenshot')}</p>}
      </Panel>

      <Panel title={t('details')} bodyClassName="">
        <dl className="facts">
          <div>
            <dt>{t('reporter')}</dt>
            <dd className="flex flex-col items-start gap-0.5">
              <Link
                href={`/users/${report.user.id}`}
                dir="auto"
                className="font-semibold text-brand no-underline hover:underline"
              >
                {reporterName(report.user)}
              </Link>
              {report.user.name && report.user.mobile ? (
                <span dir="ltr" className="text-xs text-muted">
                  {report.user.mobile}
                </span>
              ) : null}
            </dd>
          </div>
          <div>
            <dt>{t('appVersion')}</dt>
            <dd dir="ltr" className="rtl:text-end">
              {report.app_version || '—'}
            </dd>
          </div>
          <div>
            <dt>{t('updatedAt')}</dt>
            <dd>{formatDateTime(report.updated_at, locale) || '—'}</dd>
          </div>
          <div className="col-span-full">
            <dt>{t('userAgent')}</dt>
            <dd dir="ltr" className="break-all text-xs rtl:text-end">
              {report.user_agent || '—'}
            </dd>
          </div>
        </dl>
      </Panel>
    </div>
  );
}

function Screenshot({ id }: { id: number }) {
  const t = useTranslations('supportReports');
  const shot = useScreenshotUrl(id, true);
  if (shot.error) return <ErrorState error={shot.error} onRetry={() => void shot.refetch()} />;
  if (!shot.url) {
    return (
      <div className="flex items-center gap-2 text-ink-3">
        <Spinner />
        {t('loadingScreenshot')}
      </div>
    );
  }
  return (
    <a href={shot.url} target="_blank" rel="noopener noreferrer" className="inline-block max-w-full">
      {/* eslint-disable-next-line @next/next/no-img-element -- a private blob: URL, not an optimisable asset */}
      <img
        src={shot.url}
        alt={t('screenshotAlt')}
        className="block max-h-[36rem] max-w-full rounded-lg border border-line object-contain"
      />
    </a>
  );
}
