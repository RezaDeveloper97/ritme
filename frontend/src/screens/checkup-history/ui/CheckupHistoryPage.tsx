'use client';

import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  CHECKUP_ATTACHMENT_ACCEPT,
  CHECKUP_RECORD_FILTERS,
  type CheckupRecord,
  type CheckupRecordFilter,
  checkupAttachments,
  checkupIcon,
  checkupKeys,
  formatCheckupMonth,
  fetchCheckupRecords,
  pruneCheckupAttachmentsSoon,
  useCheckupAttachmentIds,
  useCheckupRecords,
} from '@/entities/checkup';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate, today } from '@/shared/lib/date';
import { openLocalFile } from '@/shared/lib/local-files';
import { type PdfBlock, loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';
import { openSheet } from '@/shared/sheet';
import { EmptyState, Icon, ScreenHeader, SegmentedTabs, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { MARK_DONE_SHEET, editRecordSheetArg, withLocalAttachment } from '../model/view';

const PANEL_ID = 'ckh-panel';

/** Fetch every record page (the doctor summary covers all of them). */
async function fetchAllRecords(type: number | null): Promise<CheckupRecord[]> {
  const all: CheckupRecord[] = [];
  for (let page = 1; page <= 50; page++) {
    const res = await fetchCheckupRecords({ type }, page);
    all.push(...res.records);
    if (res.page >= res.lastPage) break;
  }
  return all;
}

function useSummaryPdf(type: number | null) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const [state, setState] = useState<'idle' | 'working' | 'error'>('idle');

  const run = async () => {
    setState('working');
    try {
      const [records, { renderPdf }] = await Promise.all([fetchAllRecords(type), loadPdfGenerator()]);
      const date = (ymd: string) => formatLongDate(fromApiDate(ymd), locale);
      const blocks: PdfBlock[] = [
        { kind: 'title', text: t('history.pdf.title') },
        {
          kind: 'subtitle',
          text: t('history.pdf.subtitle', {
            date: formatLongDate(today(), locale),
            count: formatNumber(records.length, locale),
          }),
        },
      ];
      if (records.length === 0) blocks.push({ kind: 'text', text: t('history.empty') });
      for (const r of records) {
        blocks.push({ kind: 'rule' });
        blocks.push({ kind: 'heading', text: r.checkupTitle ?? t('title') });
        blocks.push({
          kind: 'text',
          text: [date(r.doneOn), t(`result.${r.result}`)].join(t('separator')),
        });
        if (r.findings.length > 0) {
          blocks.push({ kind: 'text', text: t('history.pdf.findings', { count: formatNumber(r.findings.length, locale) }) });
        }
        if (r.note) blocks.push({ kind: 'text', text: t('history.pdf.note', { note: r.note }) });
        if (r.nextDueOn) blocks.push({ kind: 'text', text: t('history.pdf.nextDue', { date: date(r.nextDueOn) }) });
        if (r.hasAttachment) blocks.push({ kind: 'muted', text: t('history.pdf.attachment') });
      }
      blocks.push({ kind: 'rule' });
      blocks.push({ kind: 'muted', text: t('history.pdf.disclaimer') });
      const blob = await renderPdf({ blocks, dir, locale, footer: t.raw('history.pdf.footer') as string });
      await shareOrDownloadFile(blob, t('history.pdf.filename'));
      setState('idle');
    } catch {
      // Health data: report nothing but the failure itself (§11).
      setState('error');
    }
  };
  return { state, run };
}

/**
 * One timeline entry (v14_History): the type's tone tile with the connector,
 * month + year, title, «result، note», and the «پیوست» chip inline (known 5a).
 */
function RecordItem({ record, last }: { record: CheckupRecord; last: boolean }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const [missing, setMissing] = useState(false);
  const line = [t(`result.${record.result}`), record.note].filter(Boolean).join(t('separator'));

  const openAttachment = async () => {
    const file = await checkupAttachments.get(record.id);
    if (!file) {
      setMissing(true);
      return;
    }
    // Allow-listed photos / PDFs open in a tab; anything else (a file stored
    // before the allow-list, e.g. an SVG) is only downloaded, never rendered
    // as the app's origin (audit M3-M7 #2).
    openLocalFile(file, CHECKUP_ATTACHMENT_ACCEPT);
  };

  return (
    <li className={clsx(`ck-tone-${record.checkupTone}`, 'flex gap-3')}>
      <span className="flex flex-col items-center" aria-hidden>
        <span className="grid size-11 shrink-0 place-items-center rounded-full bg-(--ck-soft) text-(--ck-ink)">
          <Icon name={checkupIcon(record.checkupIcon)} size={19} />
        </span>
        {!last && <span className="ck-tl-line" />}
      </span>
      <div className={clsx('flex min-w-0 flex-1 flex-col gap-1 text-start', !last && 'pb-4')}>
        <button
          type="button"
          className="flex w-full flex-col gap-0.5 text-start"
          aria-label={`${t('detail.editRecord')}: ${record.checkupTitle ?? t('title')}`}
          onClick={() => openSheet(MARK_DONE_SHEET, editRecordSheetArg(record))}
        >
          <span className="text-[11.5px] font-bold text-(--ink-3)">{formatCheckupMonth(record.doneOn, locale)}</span>
          <span className="text-[14px] font-extrabold text-(--ink)">{record.checkupTitle ?? t('title')}</span>
        </button>
        <span className="flex flex-wrap items-center gap-2">
          <span className="text-[12px] leading-5 font-semibold text-(--text-2)">{line}</span>
          {record.hasAttachment && (
            <button
              type="button"
              className="flex min-h-8 items-center gap-1 rounded-full border-[1.5px] border-(--brand) px-3 text-[11.5px] font-extrabold text-(--brand-strong)"
              aria-label={t('history.openAttachment')}
              onClick={() => void openAttachment()}
            >
              <Icon name="note" size={12} />
              {t('history.attachment')}
            </button>
          )}
        </span>
        {missing && (
          <p className="text-[11px] text-(--ink-3)" role="status">
            {t('history.attachmentMissing')}
          </p>
        )}
      </div>
    </li>
  );
}

/**
 * Checkup history (`v14_History`) — `/checkups/history?type=`. Health data
 * (§11): display only; the PDF summary is built on the device and never sent.
 */
export function CheckupHistoryPage({ type }: { type: number | null }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const [filter, setFilter] = useState<CheckupRecordFilter>('all');
  const query = useCheckupRecords({ filter, type });
  const localIds = useCheckupAttachmentIds();
  const pdf = useSummaryPdf(type);
  const queryClient = useQueryClient();

  // Drop report files whose record is gone — e.g. every record of a deleted
  // custom checkup (audit M3-M7 #1) — then refresh the «با پیوست» ids.
  useEffect(() => {
    void pruneCheckupAttachmentsSoon().then((removed) => {
      if (removed > 0) void queryClient.invalidateQueries({ queryKey: checkupKeys.attachmentsAll() });
    });
  }, [queryClient]);

  const loaded = query.data?.pages.flatMap((p) => p.records) ?? [];
  const records = filter === 'with_attachment' ? withLocalAttachment(loaded, localIds.data) : loaded;
  const total = query.data?.pages[0]?.total ?? 0;
  const sentinel = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  // Infinite list: fetch the next page as the end of the timeline scrolls into view.
  useEffect(() => {
    const el = sentinel.current;
    if (!el || !hasNextPage || typeof IntersectionObserver === 'undefined') return;
    const io = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting) && !isFetchingNextPage) void fetchNextPage();
    });
    io.observe(el);
    return () => io.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const backHref = type !== null ? `/checkups/${type}` : '/checkups';

  return (
    <div className="view rmd-page">
      <div className="scroll rmd-screen">
        <SkyLayer />
        <ScreenHeader
          title={t('history.title')}
          subtitle={query.data ? t('history.count', { count: formatNumber(total, locale) }) : undefined}
          onBack={() => router.push(backHref)}
          backLabel={t('back')}
          action={
            <button
              type="button"
              className="nb-hbtn"
              aria-label={t('history.export')}
              disabled={pdf.state === 'working'}
              onClick={() => void pdf.run()}
            >
              <Icon name="export" size={20} strokeWidth={1.8} />
            </button>
          }
        />

        <div className="rmd-body flex flex-col gap-3 pb-8">
          <SegmentedTabs
            tabs={CHECKUP_RECORD_FILTERS.map((key) => ({ value: key, label: t(`history.tabs.${key}`) }))}
            value={filter}
            onChange={setFilter}
            label={t('history.title')}
            panelId={() => PANEL_ID}
          />

          <div id={PANEL_ID} role="tabpanel" aria-label={t(`history.tabs.${filter}`)}>
            {query.isPending ? (
              <SkeletonGroup label={t('loading')} className="rmd-form-skel">
                <Skeleton shape="card" />
                <Skeleton shape="card" />
              </SkeletonGroup>
            ) : query.isError ? (
              <div className="rmd-state" role="alert">
                <p>{t('loadError')}</p>
                <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
                  {t('retry')}
                </button>
              </div>
            ) : records.length === 0 && !query.hasNextPage ? (
              <EmptyState icon="note" title={t('history.empty')} className="card" />
            ) : (
              <>
                <ol className="card flex flex-col p-4">
                  {records.map((r, i) => (
                    <RecordItem key={r.id} record={r} last={i === records.length - 1} />
                  ))}
                </ol>
                <div ref={sentinel} aria-hidden />
                {query.hasNextPage && (
                  <button
                    type="button"
                    className="nb-btn is-outline is-block"
                    disabled={query.isFetchingNextPage}
                    onClick={() => void query.fetchNextPage()}
                  >
                    {query.isFetchingNextPage ? t('loading') : t('history.loadMore')}
                  </button>
                )}
              </>
            )}
          </div>

          <button
            type="button"
            className="card flex items-center gap-3 p-4 text-start"
            disabled={pdf.state === 'working'}
            onClick={() => void pdf.run()}
          >
            <span className="grid size-11 shrink-0 place-items-center rounded-full bg-(--pink-bg) text-(--brand-strong)" aria-hidden>
              <Icon name="note" size={18} />
            </span>
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="text-[13.5px] font-extrabold text-(--ink)">{t('history.summary')}</span>
              <span className="text-[11.5px] text-(--ink-3)">
                {pdf.state === 'working' ? t('history.pdf.working') : t('history.summarySub')}
              </span>
            </span>
            <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={18} className="shrink-0 text-(--ink-3)" />
          </button>
          {pdf.state === 'error' && (
            <p className="text-[12px] text-(--danger)" role="alert">
              {t('history.pdf.error')}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
