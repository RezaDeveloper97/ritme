'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  type CreatedShareCode,
  MAX_SHARE_DOCUMENTS,
  REPORT_SECTIONS,
  type ReportSection,
  type ShareCodeInput,
  type TimelineItem,
  useCreateShareCode,
  useRecordTimeline,
} from '@/entities/health-record';
import { ApiError } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatTimeOfDay, fromApiDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { Checkbox, ChipGroup, PillChip, PrimaryButton, QrCode, SecondaryButton, Skeleton } from '@/shared/ui';

import { codeEntryUrl, qrTarget, toggleDocument, toggleSection } from '../model/share-code';

const RANGES: ShareCodeInput['range'][] = ['3m', '6m', '1y'];

/**
 * «ساخت کد» (nbl_Rec_Share): window + sections + up to 10 of her own documents → POST /health-record/share-codes,
 * then the code and its QR, shown once. Everything (code, token, URL) lives in the mutation's memory only and is
 * dropped when the sheet closes.
 */
export function ShareCodeSheet({ open, onClose, maxActive }: { open: boolean; onClose: () => void; maxActive: number }) {
  const t = useTranslations('record.shareCode');
  const ts = useTranslations('record.share');
  const loc = useLocale() as Locale;
  const create = useCreateShareCode();
  const [range, setRange] = useState<ShareCodeInput['range']>('6m');
  const [sections, setSections] = useState<ReportSection[]>([...REPORT_SECTIONS]);
  const [documentIds, setDocumentIds] = useState<number[]>([]);

  useEffect(() => {
    if (open) return;
    // closing forgets the code: reset the mutation (its data holds the code + token) and the form
    create.reset();
    setRange('6m');
    setSections([...REPORT_SECTIONS]);
    setDocumentIds([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const status = create.error instanceof ApiError ? create.error.response?.status : undefined;
  const created = create.data;

  const footer = created ? (
    <PrimaryButton onClick={onClose}>{t('done')}</PrimaryButton>
  ) : (
    <div className="rsh-sheet-btns">
      <PrimaryButton
        loading={create.isPending}
        disabled={sections.length === 0}
        onClick={() => create.mutate({ range, sections, documentIds })}
      >
        {t('submit')}
      </PrimaryButton>
      <SecondaryButton variant="text" onClick={onClose}>
        {t('cancel')}
      </SecondaryButton>
    </div>
  );

  return (
    <AppSheet open={open} onClose={onClose} size="full" title={created ? t('resultTitle') : t('title')} footer={footer}>
      {created ? (
        <CodeResult created={created} />
      ) : (
        <div className="rsh-form">
          <fieldset className="rsh-field">
            <legend className="rsh-label">{t('range')}</legend>
            <ChipGroup label={t('range')} layout="fill">
              {RANGES.map((r) => (
                <PillChip key={r} pressed={range === r} onPressedChange={() => setRange(r)}>
                  {t(`ranges.${r}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </fieldset>
          <fieldset className="rsh-field">
            <legend className="rsh-label">{t('sections')}</legend>
            <ChipGroup label={t('sections')}>
              {REPORT_SECTIONS.map((s) => (
                <PillChip
                  key={s}
                  mode="multi"
                  pressed={sections.includes(s)}
                  onPressedChange={() => setSections((cur) => toggleSection(cur, s))}
                >
                  {t(`section.${s}`)}
                </PillChip>
              ))}
            </ChipGroup>
            {sections.length === 0 ? <p className="rsh-hint">{t('needSection')}</p> : null}
          </fieldset>
          <DocumentPicker selected={documentIds} onToggle={(id) => setDocumentIds((cur) => toggleDocument(cur, id))} />
          {create.isError ? (
            <p className="rsh-error" role="alert">
              {status === 409 ? ts('limit', { max: formatNumber(maxActive, loc) }) : status === 503 ? ts('unavailable') : t('error')}
            </p>
          ) : null}
        </div>
      )}
    </AppSheet>
  );
}

function DocumentPicker({ selected, onToggle }: { selected: number[]; onToggle: (id: number) => void }) {
  const t = useTranslations('record.shareCode');
  const tk = useTranslations('record.kinds');
  const loc = useLocale() as Locale;
  const timeline = useRecordTimeline('all');
  const docs: TimelineItem[] = (timeline.data?.pages ?? [])
    .flatMap((p) => p.months)
    .flatMap((m) => m.items)
    .filter((i) => i.type === 'document');
  const full = selected.length >= MAX_SHARE_DOCUMENTS;
  const max = formatNumber(MAX_SHARE_DOCUMENTS, loc);

  return (
    <fieldset className="rsh-field">
      <legend className="rsh-label">{t('documents')}</legend>
      <p className="rsh-hint">{t('documentsHint', { max })}</p>
      {timeline.isPending ? (
        <Skeleton shape="card" className="rsh-skel-card" />
      ) : docs.length === 0 ? (
        <p className="rsh-hint">{t('documentsEmpty')}</p>
      ) : (
        <div className="rsh-docs">
          {docs.map((d) => {
            const kind = tk(d.kind === 'lab' ? 'lab' : d.kind);
            const on = selected.includes(d.id);
            return (
              <Checkbox
                key={d.id}
                checked={on}
                disabled={!on && full}
                onCheckedChange={() => onToggle(d.id)}
                label={d.title || t('untitled', { kind })}
                description={`${kind} · ${formatDayMonth(fromApiDate(d.date), loc)}`}
              />
            );
          })}
        </div>
      )}
      {full ? <p className="rsh-hint">{t('documentsMax', { max })}</p> : null}
    </fieldset>
  );
}

function CodeResult({ created }: { created: CreatedShareCode }) {
  const t = useTranslations('record.shareCode');
  const loc = useLocale() as Locale;
  const [copied, setCopied] = useState(false);
  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  const target = qrTarget(created, origin);
  const entry = codeEntryUrl(origin, loc);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(created.code);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  };

  return (
    <div className="rsh-result">
      <p className="rsh-result-body">
        {t('resultBody', { time: formatTimeOfDay(new Date(created.meta.expiresAt), loc) })}
      </p>
      <div className="rsh-code-box">
        <span className="rsh-code-label">{t('codeLabel')}</span>
        <output className="rsh-code-value" dir="ltr">
          {created.code}
        </output>
        <span className="rsh-code-meta">
          {t('documentsCount', { count: created.documentsCount, n: formatNumber(created.documentsCount, loc) })}
        </span>
      </div>
      {target ? <QrCode value={target} label={t('qrLabel')} className="rsh-qr" /> : null}
      {/* LTR isolate (U+2066 … U+2069) keeps the URL in order inside RTL copy. */}
      <p className="rsh-enter-at">{t('enterAt', { url: `\u2066${entry}\u2069` })}</p>
      <SecondaryButton icon={copied ? 'check' : 'copy'} onClick={() => void copy()}>
        {copied ? t('copied') : t('copy')}
      </SecondaryButton>
    </div>
  );
}
