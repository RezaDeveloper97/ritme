'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import {
  CHECKUP_RECORD_FILTERS,
  type CheckupRecord,
  type CheckupRecordFilter,
  type CheckupResult,
  checkupAttachments,
  fetchCheckupRecords,
  useCheckupAttachmentIds,
  useCheckupRecords,
} from '@/entities/checkup';
import { type Locale, Link, useDirection } from '@/shared/i18n';
import { formatLongDate, formatMonthLabel, formatNumber, fromApiDate, toParts, today } from '@/shared/lib/date';
import { type PdfBlock, loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';
import { openSheet } from '@/shared/sheet';
import { Icon } from '@/shared/ui';

import { MARK_DONE_SHEET, editRecordSheetArg, withLocalAttachment } from '../model/view';

const PANEL_ID = 'ckh-panel';

const RESULT_CHIP: Record<CheckupResult, string> = {
  normal: 'bg-(--success-soft) text-(--success)',
  follow_up: 'bg-(--amber-soft) text-(--amber-deep)',
  pending: 'bg-(--surface-2) text-(--ink-3)',
};

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

function RecordItem({ record, last }: { record: CheckupRecord; last: boolean }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const [missing, setMissing] = useState(false);
  const parts = toParts(fromApiDate(record.doneOn), locale);

  const openAttachment = async () => {
    const file = await checkupAttachments.get(record.id);
    if (!file) {
      setMissing(true);
      return;
    }
    const url = URL.createObjectURL(file.blob);
    window.open(url, '_blank', 'noopener');
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  };

  return (
    <li className="relative flex gap-3 ps-1">
      <span className="flex flex-col items-center" aria-hidden>
        <span className="mt-1.5 size-3 shrink-0 rounded-full bg-(--brand-fill) ring-4 ring-(--pink-bg)" />
        {!last && <span className="w-0.5 flex-1 bg-(--line)" />}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-1.5 pb-4 text-start">
        <span className="text-[11.5px] font-bold text-(--ink-3)">
          {formatMonthLabel(parts.year, parts.month, locale)}
        </span>
        <button
          type="button"
          className="card flex w-full flex-col gap-1.5 p-4 text-start"
          aria-label={t('detail.editRecord')}
          onClick={() => openSheet(MARK_DONE_SHEET, editRecordSheetArg(record))}
        >
          <span className="flex items-center justify-between gap-2">
            <span className="text-[13.5px] font-extrabold text-(--ink)">{record.checkupTitle ?? t('title')}</span>
            <Icon name="pencil" size={14} className="shrink-0 text-(--ink-3)" />
          </span>
          <span className="text-[11.5px] text-(--ink-3)">
            {formatLongDate(fromApiDate(record.doneOn), locale)}
          </span>
          <span className="flex flex-wrap gap-1.5">
            <span className={clsx('rounded-full px-2.5 py-0.5 text-[11px] font-extrabold', RESULT_CHIP[record.result])}>
              {t(`result.${record.result}`)}
            </span>
          </span>
        </button>
        {record.hasAttachment && (
          <button
            type="button"
            className="flex w-fit items-center gap-1 rounded-full bg-(--pink-bg) px-2.5 py-1 text-[11px] font-extrabold text-(--brand)"
            aria-label={t('history.openAttachment')}
            onClick={() => void openAttachment()}
          >
            <Icon name="note" size={12} />
            {t('history.attachment')}
          </button>
        )}
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
  const [filter, setFilter] = useState<CheckupRecordFilter>('all');
  const query = useCheckupRecords({ filter, type });
  const localIds = useCheckupAttachmentIds();
  const pdf = useSummaryPdf(type);

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
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href={backHref} className="rmd-hdr-btn" aria-label={t('back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{t('history.title')}</h1>
            {query.data && (
              <p className="rmd-hdr-sub">{t('history.count', { count: formatNumber(total, locale) })}</p>
            )}
          </div>
          <button
            type="button"
            className="rmd-hdr-btn"
            aria-label={t('history.export')}
            disabled={pdf.state === 'working'}
            onClick={() => void pdf.run()}
          >
            <Icon name="download" size={20} strokeWidth={1.8} />
          </button>
        </header>

        <div className="rmd-body flex flex-col gap-3 pb-8">
          <button
            type="button"
            className="card flex items-center gap-3 p-4 text-start"
            disabled={pdf.state === 'working'}
            onClick={() => void pdf.run()}
          >
            <span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-(--pink-bg) text-(--brand)" aria-hidden>
              <Icon name="note" size={18} />
            </span>
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="text-[13.5px] font-extrabold text-(--ink)">{t('history.summary')}</span>
              <span className="text-[11.5px] text-(--ink-3)">
                {pdf.state === 'working' ? t('history.pdf.working') : t('history.summarySub')}
              </span>
            </span>
            <span className="text-[12px] font-bold text-(--brand)">{t('history.export')}</span>
          </button>
          {pdf.state === 'error' && (
            <p className="text-[12px] text-(--danger)" role="alert">
              {t('history.pdf.error')}
            </p>
          )}

          <div className="rmd-tabs" role="tablist" aria-label={t('history.title')}>
            {CHECKUP_RECORD_FILTERS.map((key) => (
              <button
                key={key}
                type="button"
                role="tab"
                id={`ckh-tab-${key}`}
                aria-selected={filter === key}
                aria-controls={PANEL_ID}
                className={clsx('rmd-tab', filter === key && 'on')}
                onClick={() => setFilter(key)}
              >
                {t(`history.tabs.${key}`)}
              </button>
            ))}
          </div>

          <div id={PANEL_ID} role="tabpanel" aria-labelledby={`ckh-tab-${filter}`}>
            {query.isPending ? (
              <p className="rmd-state" role="status">
                {t('loading')}
              </p>
            ) : query.isError ? (
              <div className="rmd-state" role="alert">
                <p>{t('loadError')}</p>
                <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
                  {t('retry')}
                </button>
              </div>
            ) : records.length === 0 && !query.hasNextPage ? (
              <p className="rmd-state">{t('history.empty')}</p>
            ) : (
              <>
                <ol className="flex flex-col">
                  {records.map((r, i) => (
                    <RecordItem key={r.id} record={r} last={i === records.length - 1} />
                  ))}
                </ol>
                <div ref={sentinel} aria-hidden />
                {query.hasNextPage && (
                  <button
                    type="button"
                    className="btn btn-ghost w-full"
                    disabled={query.isFetchingNextPage}
                    onClick={() => void query.fetchNextPage()}
                  >
                    {query.isFetchingNextPage ? t('loading') : t('history.loadMore')}
                  </button>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
