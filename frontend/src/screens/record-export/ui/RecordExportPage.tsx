'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import { MAX_REPORT_QUESTION, REPORT_RANGES, type ReportRange, useHealthReport } from '@/entities/health-record';
import { useLanguages } from '@/entities/language';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate, partsToDate, toApiDate, today, toParts, type DateParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  CalendarPicker,
  Card,
  ChipGroup,
  Icon,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import { isValidFrom, REPORT_GROUPS, selectionOf, useReportDraft } from '../model/selection';
import { ShareLinkSheet } from './ShareLinkSheet';
import { estimatePages, usePaperModel, usePdfDownload } from './usePaper';

/**
 * «گزارش برای پزشک» (bloom B-N6-04, nbl_Record_Export): range, section toggles and the patient's question, then a
 * PDF drawn on the device or a 7-day share link (Plus). The question never leaves the device unless a link is made.
 */
export function RecordExportPage() {
  const t = useTranslations('recordExport');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const draft = useReportDraft();
  const selection = selectionOf(draft);
  const [picking, setPicking] = useState(false);
  const [sharing, setSharing] = useState(false);
  const questionId = useId();
  const noneId = useId();

  const ready = selection.sections.length > 0 && (draft.range !== 'custom' || !!draft.from);
  const report = useHealthReport(selection, ready);
  const model = usePaperModel(report.data, draft.question);
  const pdf = usePdfDownload();
  const pages = estimatePages(model);
  const language = useLanguages().data?.find((l) => l.code === loc)?.name ?? '';

  const pickRange = (r: ReportRange) => {
    if (r === 'custom') setPicking(true);
    else draft.setRange(r);
  };

  return (
    <div className="view rx-page">
      <SkyLayer />
      <div className="scroll rx-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/record')} backLabel={t('back')} />

        <Card className="rx-preview-card">
          <button
            type="button"
            className="rx-preview-btn"
            onClick={() => router.push('/record/export/preview')}
            disabled={!ready}
            aria-label={t('previewCard.open')}
          >
            <span className="rx-thumb" aria-hidden>
              <span className="rx-thumb-bar" />
              <span className="rx-thumb-line" />
              <span className="rx-thumb-line is-short" />
              <span className="rx-thumb-block" />
              <span className="rx-thumb-line" />
              <span className="rx-thumb-line is-short" />
            </span>
            <span className="rx-preview-text">
              <span className="rx-preview-title">{t('previewCard.title')}</span>
              <span className="rx-preview-meta">
                {t('previewCard.meta', { pages, language })}
              </span>
            </span>
            <Icon name="chevronLeft" size={20} className="rx-preview-chev" />
          </button>
        </Card>

        <Card className="rx-card">
          <h2 className="rx-card-title">{t('range.title')}</h2>
          <ChipGroup label={t('range.title')}>
            {REPORT_RANGES.map((r) => (
              <PillChip key={r} pressed={draft.range === r} onPressedChange={() => pickRange(r)}>
                {t(`range.${r}`)}
              </PillChip>
            ))}
          </ChipGroup>
          {draft.range === 'custom' && draft.from ? (
            <button type="button" className="rx-custom" onClick={() => setPicking(true)}>
              {t('range.customFrom', { date: formatLongDate(fromApiDate(draft.from), loc) })}
            </button>
          ) : null}
        </Card>

        <Card className="rx-card" padding="none">
          <h2 className="rx-card-title rx-pad">{t('sections.title')}</h2>
          <ul className="rx-toggles">
            {REPORT_GROUPS.map((g) => (
              <li key={g} className="rx-toggle">
                <span className="rx-toggle-text">
                  <span className="rx-toggle-title" id={`rx-${g}`}>
                    {t(`sections.${g}`)}
                  </span>
                  <span className="rx-toggle-sub">{t(`sections.${g}Sub`)}</span>
                </span>
                <Switch checked={draft.groups[g]} onCheckedChange={(on) => draft.toggle(g, on)} labelledBy={`rx-${g}`} />
              </li>
            ))}
          </ul>
          {selection.sections.length === 0 ? (
            <p id={noneId} className="rx-error rx-pad" role="alert">
              {t('sections.none')}
            </p>
          ) : null}
        </Card>

        <div className="rx-question">
          <label htmlFor={questionId} className="rx-question-label">
            {t('question.label')}
          </label>
          <div className="rx-question-field">
            <Icon name="fileDoc" size={20} className="rx-question-icon" />
            <textarea
              id={questionId}
              className="rx-textarea"
              rows={1}
              maxLength={MAX_REPORT_QUESTION}
              value={draft.question}
              placeholder={t('question.placeholder')}
              onChange={(e) => draft.setQuestion(e.target.value)}
            />
          </div>
          {draft.question.length > MAX_REPORT_QUESTION - 60 ? (
            <span className="rx-count">
              {t('question.count', { count: formatNumber(draft.question.length, loc), max: formatNumber(MAX_REPORT_QUESTION, loc) })}
            </span>
          ) : null}
        </div>

        <Card className="rx-note">
          <Icon name="lock" size={18} className="rx-note-icon" />
          <p>{t('note')}</p>
        </Card>

        {pdf.failed || report.isError ? (
          <p className="rx-error" role="alert">
            {report.isError ? t('preview.error') : t('downloadError')}
          </p>
        ) : null}

        <div className="rx-actions">
          <PrimaryButton
            loading={pdf.busy || (ready && report.isPending)}
            disabled={!ready || !model}
            onClick={() => model && void pdf.run(model)}
          >
            {pdf.busy ? t('downloading') : t('download')}
          </PrimaryButton>
          <SecondaryButton icon="share" disabled={!ready} onClick={() => setSharing(true)}>
            {t('share')}
          </SecondaryButton>
        </div>
      </div>

      <CustomRangeSheet
        open={picking}
        initial={draft.from}
        onClose={() => setPicking(false)}
        onPick={(from) => {
          draft.setRange('custom', from);
          setPicking(false);
        }}
      />
      <ShareLinkSheet open={sharing} onClose={() => setSharing(false)} selection={selection} />
    </div>
  );
}

function CustomRangeSheet({
  open,
  initial,
  onClose,
  onPick,
}: {
  open: boolean;
  initial: string | null;
  onClose: () => void;
  onPick: (from: string) => void;
}) {
  const t = useTranslations('recordExport.range');
  const loc = useLocale() as Locale;
  const [value, setValue] = useState<DateParts | null>(initial ? toParts(fromApiDate(initial), loc) : null);
  const iso = value ? toApiDate(partsToDate(value, loc)) : null;
  const valid = isValidFrom(iso, toApiDate(today()));
  return (
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('pickTitle')}
      footer={
        <PrimaryButton disabled={!valid} onClick={() => iso && onPick(iso)}>
          {t('pickDone')}
        </PrimaryButton>
      }
    >
      <p className="rx-pick-hint">{t('pickHint')}</p>
      <CalendarPicker value={value} onSelect={setValue} />
      {value && !valid ? (
        <p className="rx-error" role="alert">
          {t('pickInvalid')}
        </p>
      ) : null}
    </AppSheet>
  );
}
