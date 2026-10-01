'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import { getApiErrorMessage } from '@/shared/api';
import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import {
  ChipGroup,
  EmptyState,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  type Tone,
} from '@/shared/ui';
import { PlusBadge, plusDenialOf } from '@/shared/ui/plus-gate';

import { useVoiceLog, type VoiceResult, type VoiceSuggestion } from '../api/voice';
import { mergeSuggestions, paramKey } from '../model/merge';
import { formatElapsed, useVoiceRecorder, type RecorderStatus } from '../model/recorder';

/**
 * What the log sheet hands the recorder — structurally the `VoiceLogSlotProps` of `features/log-day`
 * (declared here too: a feature can't import another feature; the screen wires the two).
 */
export interface VoiceLogPanelProps {
  date: string;
  mode: string | null;
  categories: readonly LogCategory[];
  values: LogDayValues;
  setParam: (category: string, param: string, value: LogParamValue | null) => void;
  markVoice: (keys: readonly string[]) => void;
  openSection: (category: string) => void;
  toneOf: (category: string) => Tone;
  save: () => void;
  saving: boolean;
  saveError: boolean;
  onManual: () => void;
}

const WAVE_BARS = 17;

type Blocking = Extract<RecorderStatus, 'denied' | 'unsupported' | 'nomic' | 'error'>;

function isBlocking(status: RecorderStatus): status is Blocking {
  return status === 'denied' || status === 'unsupported' || status === 'nomic' || status === 'error';
}

/**
 * «ثبت با صدا» (nbl_/nbd_Log_Day, B-N3-05): record up to 60 s → upload → the server transcribes and maps
 * the words to log items → «این‌ها را فهمیدیم» review chips. Tapping a chip merges the suggestions into the
 * sheet's draft and opens that section in «ثبت دستی» to adjust; «ذخیره» merges and runs the sheet's normal
 * save. Nothing is saved without that tap. The recording lives only in memory for the upload.
 */
export function VoiceLogPanel({
  date,
  categories,
  values,
  setParam,
  markVoice,
  openSection,
  toneOf,
  save,
  saving,
  saveError,
  onManual,
}: VoiceLogPanelProps) {
  const t = useTranslations('voiceLog');
  const locale = useLocale() as Locale;
  const upload = useVoiceLog();
  const [result, setResult] = useState<VoiceResult | null>(null);
  const recorder = useVoiceRecorder((audio, durationMs) => {
    setResult(null);
    upload.mutate({ audio, durationMs }, { onSuccess: setResult });
  });

  // Another day picked in the strip: the review belonged to the previous one.
  const { reset: resetUpload } = upload;
  useEffect(() => {
    setResult(null);
    resetUpload();
  }, [date, resetUpload]);

  // After «ذخیره» went through, the review is done.
  const savingRef = useRef(false);
  const [saveRequested, setSaveRequested] = useState(false);
  useEffect(() => {
    if (saving) savingRef.current = true;
    else if (savingRef.current && saveRequested) {
      savingRef.current = false;
      setSaveRequested(false);
      if (!saveError) setResult(null);
    }
  }, [saving, saveError, saveRequested]);

  const shown = (result?.suggestions ?? []).filter((s) =>
    categories.some((c) => c.code === s.category && c.params.some((p) => p.code === s.param)),
  );

  const mergeAll = () => {
    const merged = mergeSuggestions(values, shown, categories);
    for (const m of merged) setParam(m.category, m.param, m.value);
    markVoice(merged.map(paramKey));
  };

  const onChip = (s: VoiceSuggestion) => {
    mergeAll();
    openSection(s.category);
  };

  const onSave = () => {
    mergeAll();
    setSaveRequested(true);
    save();
  };

  const startRecording = () => {
    setResult(null);
    upload.reset();
    recorder.start();
  };

  const retryUpload = () => {
    if (upload.variables) upload.mutate(upload.variables, { onSuccess: setResult });
  };

  const recording = recorder.status === 'recording';
  const busy = recorder.status === 'requesting' || upload.isPending;
  const label = (s: VoiceSuggestion) => formatDecimal(s.label, locale);

  let status: string;
  if (recording) status = t('listening', { time: formatNumber(formatElapsed(recorder.elapsedMs), locale) });
  else if (recorder.status === 'requesting') status = t('requesting');
  else if (upload.isPending) status = t('uploading');
  else if (recorder.status === 'tooShort') status = t('error.tooShort');
  else status = result ? '' : t('idle');

  let review = null;
  if (isBlocking(recorder.status)) {
    const copy = {
      denied: [t('error.deniedTitle'), t('error.deniedBody')],
      unsupported: [t('error.unsupportedTitle'), t('error.unsupportedBody')],
      nomic: [t('error.noMicTitle'), t('error.noMicBody')],
      error: [t('error.uploadTitle'), t('error.uploadBody')],
    }[recorder.status];
    review = (
      <section className="nb-card vlog-card">
        <EmptyState
          icon={recorder.status === 'denied' ? 'lock' : 'warning'}
          title={copy[0]}
          body={copy[1]}
          action={
            <div className="vlog-actions">
              {recorder.status === 'nomic' || recorder.status === 'error' ? (
                <SecondaryButton block={false} onClick={recorder.reset}>
                  {t('error.retry')}
                </SecondaryButton>
              ) : null}
              <SecondaryButton block={false} variant="text" onClick={onManual}>
                {t('manual')}
              </SecondaryButton>
            </div>
          }
        />
      </section>
    );
  } else if (upload.isPending) {
    review = (
      <section className="nb-card vlog-card" aria-busy="true">
        <SkeletonGroup label={t('uploading')} className="vlog-skel">
          <Skeleton width="medium" />
          <div className="vlog-skel-chips">
            <Skeleton shape="block" />
            <Skeleton shape="block" />
            <Skeleton shape="block" />
          </div>
          <Skeleton width="short" />
        </SkeletonGroup>
      </section>
    );
  } else if (upload.isError) {
    const denial = plusDenialOf(upload.error);
    review = (
      <section className="nb-card vlog-card">
        <EmptyState
          icon="warning"
          title={t('error.uploadTitle')}
          body={denial?.message ?? getApiErrorMessage(upload.error) ?? t('error.uploadBody')}
          action={
            <div className="vlog-actions">
              {denial ? null : (
                <SecondaryButton block={false} onClick={retryUpload}>
                  {t('error.retry')}
                </SecondaryButton>
              )}
              <SecondaryButton block={false} variant="text" onClick={onManual}>
                {t('manual')}
              </SecondaryButton>
            </div>
          }
        />
      </section>
    );
  } else if (result && shown.length === 0) {
    const silent = result.transcript.trim() === '';
    review = (
      <section className="nb-card vlog-card">
        {silent ? null : <p className="vlog-heard">{t('heard', { text: result.transcript })}</p>}
        <EmptyState
          icon={silent ? 'mic' : 'search'}
          title={silent ? t('empty.silentTitle') : t('empty.title')}
          body={silent ? t('empty.silentBody') : t('empty.body')}
          action={
            <SecondaryButton block={false} variant="text" onClick={onManual}>
              {t('manual')}
            </SecondaryButton>
          }
        />
      </section>
    );
  } else if (result) {
    review = (
      <section className="nb-card vlog-card" aria-labelledby="vlog-review-title">
        <div className="vlog-review-head">
          <h3 id="vlog-review-title" className="vlog-review-title">
            {t('review.title')}
          </h3>
          <span className="vlog-review-hint">{t('review.hint')}</span>
        </div>
        <p className="vlog-heard">{t('heard', { text: result.transcript })}</p>
        <ChipGroup label={t('review.title')}>
          {shown.map((s) => (
            <PillChip
              key={`${s.category}.${s.param}.${s.item ?? ''}`}
              pressed
              mode="multi"
              tone={toneOf(s.category)}
              className="vlog-chip"
              aria-label={t('review.open', { label: label(s) })}
              onClick={() => onChip(s)}
            >
              {label(s)}
            </PillChip>
          ))}
        </ChipGroup>
        <p className="vlog-help">{t('review.help')}</p>
      </section>
    );
  }

  return (
    <div className="vlog">
      <section className="nb-card vlog-rec">
        <PlusBadge label={t('plus')} />
        <h3 className="vlog-title">{t('title')}</h3>
        <p className="vlog-example">{t('example')}</p>
        <div className={clsx('vlog-wave', recording && 'is-live')} aria-hidden>
          {Array.from({ length: WAVE_BARS }, (_, i) => (
            <span key={i} className="vlog-bar" />
          ))}
        </div>
        <button
          type="button"
          className={clsx('vlog-mic', recording && 'is-recording')}
          aria-label={recording ? t('stop') : t('start')}
          aria-pressed={recording}
          disabled={busy}
          onClick={recording ? recorder.stop : startRecording}
        >
          {busy ? <Icon name="loader" size={30} className="vlog-spin" /> : <Icon name="mic" size={34} strokeWidth={2} />}
        </button>
        <p className="vlog-status" aria-live="polite">
          {status}
        </p>
        {!recording && !busy && !result ? <p className="vlog-max">{t('maxHint')}</p> : null}
        {result && !recording && !busy ? (
          <SecondaryButton block={false} variant="text" icon="refresh" onClick={startRecording}>
            {t('again')}
          </SecondaryButton>
        ) : null}
      </section>

      {review}

      <InfoNote className="vlog-privacy">{t('privacy')}</InfoNote>

      {result && shown.length > 0 && !upload.isPending ? (
        <div className="lday-foot">
          <div className="lday-foot-text" aria-live="polite">
            <span className="lday-foot-count">{saveError ? t('footer.error') : t('footer.count', { count: shown.length })}</span>
            {saveError ? null : <span className="lday-foot-list">{shown.map(label).join(t('separator'))}</span>}
          </div>
          <PrimaryButton block={false} className="lday-foot-save" onClick={onSave} loading={saving}>
            {saving ? t('footer.saving') : t('footer.save')}
          </PrimaryButton>
        </div>
      ) : null}
    </div>
  );
}
