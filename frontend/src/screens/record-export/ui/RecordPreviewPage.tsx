'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useHealthReport } from '@/entities/health-record';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { EmptyState, HeaderButton, PrimaryButton, ScreenHeader, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { selectionOf, useReportDraft } from '../model/selection';
import { ReportPaper } from './ReportPaper';
import { ShareLinkSheet } from './ShareLinkSheet';
import { estimatePages, usePaperModel, usePdfDownload } from './usePaper';

/** «پیش‌نمایش» (bloom B-N6-04, nbl_Record_Preview): the report as it will print, with download and share. */
export function RecordPreviewPage() {
  const t = useTranslations('recordExport');
  const tp = useTranslations('recordExport.paper');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const draft = useReportDraft();
  const selection = selectionOf(draft);
  const ready = selection.sections.length > 0 && (draft.range !== 'custom' || !!draft.from);
  const report = useHealthReport(selection, ready);
  const model = usePaperModel(report.data, draft.question);
  const pdf = usePdfDownload();
  const [sharing, setSharing] = useState(false);

  let body;
  if (!ready) {
    body = (
      <EmptyState
        icon="fileDoc"
        title={t('sections.none')}
        action={
          <PrimaryButton block={false} onClick={() => router.push('/record/export')}>
            {t('title')}
          </PrimaryButton>
        }
      />
    );
  } else if (report.isPending) {
    body = (
      <SkeletonGroup label={t('preview.loading')} className="rx-skel">
        <Skeleton shape="card" className="rx-skel-paper" />
      </SkeletonGroup>
    );
  } else if (report.isError || !model) {
    body = (
      <EmptyState
        icon="warning"
        title={t('preview.error')}
        body={t('preview.errorBody')}
        action={
          <PrimaryButton icon="refresh" block={false} onClick={() => void report.refetch()}>
            {t('preview.retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    const pages = estimatePages(model);
    body = (
      <ReportPaper
        model={model}
        questionLabel={tp('question')}
        pageLabel={tp('pageOf', { page: formatNumber(1, loc), pages: formatNumber(pages, loc) })}
      />
    );
  }

  return (
    <div className="view rx-page">
      <SkyLayer />
      <div className="scroll rx-scroll">
        <ScreenHeader
          title={t('preview.title')}
          onBack={() => router.push('/record/export')}
          backLabel={t('preview.close')}
          backIcon="close"
          action={<HeaderButton icon="share" label={t('preview.shareAction')} onClick={() => setSharing(true)} />}
        />
        {body}
        {pdf.failed ? (
          <p className="rx-error" role="alert">
            {t('downloadError')}
          </p>
        ) : null}
        <div className="rx-actions">
          <PrimaryButton loading={pdf.busy} disabled={!model} onClick={() => model && void pdf.run(model)}>
            {pdf.busy ? t('downloading') : t('download')}
          </PrimaryButton>
          <SecondaryButton icon="share" disabled={!ready} onClick={() => setSharing(true)}>
            {t('send')}
          </SecondaryButton>
        </div>
      </div>
      <ShareLinkSheet open={sharing} onClose={() => setSharing(false)} selection={selection} />
    </div>
  );
}
