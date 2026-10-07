'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { HealthReport } from '@/entities/health-record';
import type { Locale } from '@/shared/i18n';
import { useDirection } from '@/shared/i18n';
import { loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';

import { buildPaper, paperPdfBlocks, type PaperModel, type Translate } from '../model/paper';

/** The paper model of a report, translated for the current locale. */
export function usePaperModel(report: HealthReport | undefined, question: string | null): PaperModel | null {
  const t = useTranslations('recordExport.paper');
  const hr = useTranslations('healthRecord');
  const tm = useTranslations('menopause.report.paper');
  const loc = useLocale() as Locale;
  if (!report) return null;
  return buildPaper(report, question, t as unknown as Translate, hr as unknown as Translate, loc, tm as unknown as Translate);
}

/** Pages the PDF will take, roughly (A4, the renderer's line heights) — for «۲ صفحه» on the export card. */
export function estimatePages(m: PaperModel | null): number {
  if (!m) return 1;
  const units = 8 + m.blocks.reduce((sum, b) => sum + (b.kind === 'heading' ? 2 : b.kind === 'text' ? 2 : 1.1), 0) + (m.question ? 3 : 0);
  return Math.max(1, Math.ceil(units / 34));
}

/** Draws the PDF on the device and hands it to the share sheet / a download. Nothing is uploaded. */
export function usePdfDownload() {
  const t = useTranslations('recordExport.paper');
  const loc = useLocale() as Locale;
  const dir = useDirection();
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const run = async (m: PaperModel) => {
    setBusy(true);
    setFailed(false);
    try {
      const { renderPdf } = await loadPdfGenerator();
      const blob = await renderPdf({ blocks: paperPdfBlocks(m, t('question')), dir, locale: loc, footer: t.raw('pdfFooter') as string });
      await shareOrDownloadFile(blob, `${t('filename')}.pdf`);
    } catch {
      setFailed(true); // never log the report
    } finally {
      setBusy(false);
    }
  };
  return { run, busy, failed };
}
