'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useDoctorTitle, whenLabel } from '@/entities/doctor';
import { type Locale, useDirection } from '@/shared/i18n';
import { formatLongDate, formatNumber, today } from '@/shared/lib/date';
import { type PdfBlock, loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';

import type { FeedItem } from '../model/feed';

/**
 * «خروجی PDF برای پزشک»: the visible list as a PDF, built on the device (nothing uploaded). Titles, dates and
 * status only — never message bodies or summaries (assistant chats are health data, §11).
 */
export function useHistoryPdf(feed: readonly FeedItem[]) {
  const t = useTranslations('assistant');
  const dt = useTranslations('doctors');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const doctorTitle = useDoctorTitle();
  const [state, setState] = useState<'idle' | 'working' | 'error'>('idle');

  const run = async () => {
    setState('working');
    try {
      const { renderPdf } = await loadPdfGenerator();
      const day = (iso: string | null) => (iso && !Number.isNaN(Date.parse(iso)) ? formatLongDate(new Date(iso), locale) : '');
      const blocks: PdfBlock[] = [
        { kind: 'title', text: t('history.pdf.title') },
        { kind: 'subtitle', text: t('history.pdf.subtitle', { count: formatNumber(feed.length, locale), date: formatLongDate(today(), locale) }) },
        { kind: 'muted', text: t('history.pdf.notice') },
        { kind: 'rule' },
      ];
      for (const it of feed) {
        if (it.kind === 'assistant') {
          const c = it.conversation;
          blocks.push({
            kind: 'row',
            weights: [3, 3, 2, 2],
            cells: [c.title || t('history.untitled'), `${t('history.pdf.kind.assistant')} · ${c.department.name}`, day(c.last_message_at ?? c.created_at), c.triage_level ? t(`triage.${c.triage_level}`) : ''],
          });
        } else {
          const b = it.booking;
          blocks.push({
            kind: 'row',
            weights: [3, 3, 2, 2],
            cells: [doctorTitle(b.doctor), `${t('history.pdf.kind.visit')} · ${dt(`mode.${b.mode}`)}`, whenLabel(b.date, b.time, locale), dt(`status.${b.status}`)],
          });
        }
      }
      const blob = await renderPdf({ blocks, dir, locale, footer: t.raw('history.pdf.footer') as string });
      await shareOrDownloadFile(blob, t('history.pdf.filename'));
      setState('idle');
    } catch {
      setState('error');
    }
  };
  return { state, run };
}
