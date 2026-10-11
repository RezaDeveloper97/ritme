'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type InsuranceQuestionnaire, questionCode, sectionCode } from '@/entities/insurance';
import { type Locale, useDirection } from '@/shared/i18n';
import { formatLongDate, today } from '@/shared/lib/date';
import { loadPdfGenerator, type PdfBlock, shareOrDownloadFile } from '@/shared/lib/pdf';

/**
 * Draws the questionnaire to a PDF on the device (the same canvas → PDF path as the checkup history export) and hands
 * it to the share sheet / a download. Nothing is uploaded. Health data (§11): on failure report only the failure.
 */
export function useQuestionnairePdf(q: InsuranceQuestionnaire) {
  const t = useTranslations('insurance.questionnaire');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const [state, setState] = useState<'idle' | 'working' | 'error'>('idle');

  const run = async () => {
    setState('working');
    try {
      const { renderPdf } = await loadPdfGenerator();
      const blocks: PdfBlock[] = [
        { kind: 'title', text: t('pdf.title') },
        {
          kind: 'subtitle',
          text: t('pdf.subtitle', { purpose: t(`start.purpose.${q.purpose}`), date: formatLongDate(today(), locale) }),
        },
      ];
      for (const section of q.sections.filter((s) => s !== 'review')) {
        const rows = q.questions.filter((x) => x.section === section);
        if (rows.length === 0) continue;
        blocks.push({ kind: 'heading', text: t(`sections.${sectionCode(section)}`) });
        for (const row of rows) {
          const answer = row.answer ? t(row.answer) : t('notAnswered');
          blocks.push({
            kind: 'row',
            cells: [questionCode(row.code) ? t(`q.${questionCode(row.code)!}`) : row.code, row.fromRecord ? `${answer} · ${t('pdf.fromRecord')}` : answer],
            weights: [3, 1.4],
          });
        }
      }
      if (q.explanation.trim()) {
        blocks.push({ kind: 'rule' });
        blocks.push({ kind: 'text', text: t('pdf.explanation', { text: q.explanation.trim() }) });
      }
      blocks.push({ kind: 'rule' });
      blocks.push({ kind: 'muted', text: t('pdf.disclaimer') });
      const blob = await renderPdf({ blocks, dir, locale, footer: t.raw('pdf.footer') as string });
      await shareOrDownloadFile(blob, t('pdf.filename'));
      setState('idle');
    } catch {
      setState('error');
    }
  };
  return { state, run };
}
