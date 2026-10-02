'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import type { LogCategory, LogDayValues, LogParamValue } from '@/entities/health-log';
import { getApiErrorMessage } from '@/shared/api';
import { EmptyState, type IconName, SecondaryButton, type Tone } from '@/shared/ui';
import { plusDenialOf } from '@/shared/ui/plus-gate';

import { useVoiceCommit, useVoiceLog, type VoiceResult, type VoiceSuggestion } from '../api/voice';
import { mergeSuggestions, paramKey } from '../model/merge';
import { useVoiceRecorder, type RecorderStatus } from '../model/recorder';
import {
  chooseItem,
  commitItems,
  isDiaryTarget,
  logSuggestions,
  revalueItem,
  reviewItems,
  toggleItem,
  type ReviewItem,
  type SavedRow,
} from '../model/review';
import { VoiceEntry } from './VoiceEntry';
import { VoiceRecording } from './VoiceRecording';
import { VoiceReview } from './VoiceReview';
import { VoiceSaved } from './VoiceSaved';

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
  iconOf: (category: string) => IconName;
  entries: readonly { key: string; label: string; summary: string }[];
  save: (onDone?: () => void) => void;
  saving: boolean;
  saveError: boolean;
  onManual: () => void;
  onDone: () => void;
  onImmersive?: (on: boolean) => void;
}

type Blocking = Extract<RecorderStatus, 'denied' | 'unsupported' | 'nomic' | 'error'>;

function isBlocking(status: RecorderStatus): status is Blocking {
  return status === 'denied' || status === 'unsupported' || status === 'nomic' || status === 'error';
}

/**
 * «با صدا» (B-N3-05, canvas-v1 nbl_Voice_Entry / Record / Review / Saved — CB-VOICE-02): entry with examples
 * and today's status → recording → the server transcribes and maps the words → review (drop, re-pick an
 * ambiguous word, re-value) → «تأیید و ثبت»: log items merge into the sheet's draft and go out in its one
 * PUT (`voice_params`), then diary items (hot flash, pain diary, pill, bladder) go to
 * `POST /logs/voice/commit` — in that order, a pain score needs its location saved first → saved list +
 * nightly reminder. Nothing is saved without that tap; the recording lives only in memory for the upload.
 */
export function VoiceLogPanel({
  date,
  mode,
  categories,
  values,
  setParam,
  markVoice,
  openSection,
  toneOf,
  iconOf,
  entries,
  save,
  saving,
  saveError,
  onManual,
  onDone,
  onImmersive,
}: VoiceLogPanelProps) {
  const t = useTranslations('voiceLog');
  const upload = useVoiceLog();
  const commit = useVoiceCommit();
  const [result, setResult] = useState<VoiceResult | null>(null);
  const [items, setItems] = useState<ReviewItem[]>([]);
  const [saved, setSaved] = useState<SavedRow[] | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  const onResult = (r: VoiceResult) => {
    setResult(r);
    setItems(reviewItems(r, categories));
  };
  const recorder = useVoiceRecorder((audio, durationMs) => {
    setResult(null);
    upload.mutate({ audio, durationMs }, { onSuccess: onResult });
  });

  // Another day picked in the strip: the review belonged to the previous one.
  const { reset: resetUpload } = upload;
  useEffect(() => {
    setResult(null);
    setItems([]);
    setSaved(null);
    resetUpload();
  }, [date, resetUpload]);

  // The day's PUT failed (the sheet's save reports through `saving` / `saveError`).
  const sawSaving = useRef(false);
  useEffect(() => {
    if (!confirming) {
      sawSaving.current = false;
      return;
    }
    if (saving) sawSaving.current = true;
    else if (sawSaving.current && saveError) {
      sawSaving.current = false;
      setConfirming(false);
      setError(t('review.error'));
    }
  }, [confirming, saving, saveError, t]);

  const recording = recorder.status === 'recording' || recorder.status === 'paused';
  const requesting = recorder.status === 'requesting';
  const phase = saved
    ? 'saved'
    : recording || requesting || upload.isPending
      ? 'record'
      : result && items.length > 0
        ? 'review'
        : 'entry';

  const lastPhase = useRef(phase);
  useEffect(() => {
    onImmersive?.(phase !== 'entry');
    if (lastPhase.current === phase) return;
    lastPhase.current = phase;
    rootRef.current?.scrollIntoView({ block: 'start' });
  }, [phase, onImmersive]);
  useEffect(() => () => onImmersive?.(false), [onImmersive]);

  const mergeLogs = (): VoiceSuggestion[] => {
    const logs = logSuggestions(items);
    const merged = mergeSuggestions(values, logs, categories);
    for (const m of merged) setParam(m.category, m.param, m.value);
    markVoice(merged.map(paramKey));
    return logs;
  };

  const startRecording = () => {
    setResult(null);
    setItems([]);
    setError(null);
    upload.reset();
    recorder.start();
  };

  const retryUpload = () => {
    if (upload.variables) upload.mutate(upload.variables, { onSuccess: onResult });
  };

  const onConfirm = () => {
    setError(null);
    const logs = mergeLogs();
    const diary = commitItems(items);
    const logRows: SavedRow[] = logs.map((s, i) => ({
      key: `log:${i}`,
      category: s.category,
      label: s.label,
      group: categories.find((c) => c.code === s.category)?.label,
    }));
    setConfirming(true);
    const finish = (rows: SavedRow[]) => {
      setConfirming(false);
      setSaved(rows);
    };
    const sendDiary = () => {
      if (!diary.length) {
        finish(logRows);
        return;
      }
      commit.mutate(
        { date, items: diary },
        {
          onSuccess: (r) =>
            finish([
              ...logRows,
              ...r.saved.map((s, i) => ({
                key: `diary:${i}`,
                category: s.target,
                label: s.label,
                group: isDiaryTarget(s.target) ? t(`targets.${s.target}`) : undefined,
              })),
            ]),
          onError: (e) => {
            setConfirming(false);
            setError(getApiErrorMessage(e) ?? t('review.error'));
          },
        },
      );
    };
    if (logs.length) save(sendDiary);
    else sendDiary();
  };

  if (phase === 'saved' && saved) {
    return (
      <div ref={rootRef}>
        <VoiceSaved rows={saved} iconOf={iconOf} toneOf={toneOf} onDone={onDone} />
      </div>
    );
  }

  if (phase === 'record') {
    return (
      <div ref={rootRef}>
        <VoiceRecording
          elapsedMs={recorder.elapsedMs}
          paused={recorder.status === 'paused'}
          processing={upload.isPending}
          requesting={requesting}
          canPause={typeof window !== 'undefined' && typeof window.MediaRecorder?.prototype?.pause === 'function'}
          onStop={recorder.stop}
          onPause={recorder.pause}
          onResume={recorder.resume}
          onCancel={recorder.cancel}
        />
      </div>
    );
  }

  if (phase === 'review' && result) {
    return (
      <div ref={rootRef}>
        <VoiceReview
          items={items}
          transcript={result.transcript}
          categories={categories}
          iconOf={iconOf}
          toneOf={toneOf}
          onToggle={(id) => setItems((all) => toggleItem(all, id))}
          onChoose={(id, index) => setItems((all) => chooseItem(all, id, index))}
          onRevalue={(id, value, label) => setItems((all) => revalueItem(all, id, value, label))}
          onEdit={(category) => {
            mergeLogs();
            openSection(category);
          }}
          onAdd={() => {
            mergeLogs();
            onManual();
          }}
          onConfirm={onConfirm}
          onAgain={startRecording}
          saving={confirming}
          error={error}
        />
      </div>
    );
  }

  let notice = null;
  if (isBlocking(recorder.status)) {
    const copy = {
      denied: [t('error.deniedTitle'), t('error.deniedBody')],
      unsupported: [t('error.unsupportedTitle'), t('error.unsupportedBody')],
      nomic: [t('error.noMicTitle'), t('error.noMicBody')],
      error: [t('error.uploadTitle'), t('error.uploadBody')],
    }[recorder.status];
    notice = (
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
  } else if (upload.isError) {
    const denial = plusDenialOf(upload.error);
    notice = (
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
  } else if (result) {
    const silent = result.transcript.trim() === '';
    notice = (
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
  }

  return (
    <div ref={rootRef}>
      <VoiceEntry
        mode={mode}
        categories={categories}
        entries={entries}
        iconOf={iconOf}
        toneOf={toneOf}
        status={recorder.status === 'tooShort' ? t('error.tooShort') : null}
        onStart={startRecording}
        openSection={openSection}
        notice={notice}
      />
    </div>
  );
}
