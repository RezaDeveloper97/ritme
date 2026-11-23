'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import type { AiGateHooks } from '@/entities/ai-credit';
import type { LogCategory, LogDayValues } from '@/entities/health-log';
import { getApiErrorMessage } from '@/shared/api';
import { runAutopilot } from '@/shared/autopilot';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import type { AutopilotManifest } from '@/shared/pilot';
import { EmptyState, InfoNote, SecondaryButton } from '@/shared/ui';

import { useVoicePilot, type VoicePilotResult } from '../api/voice';
import { voiceScript } from '../model/pilot';
import { formatElapsed, useVoiceRecorder, type RecorderStatus } from '../model/recorder';
import { VoiceOrb, type OrbState } from './VoiceOrb';
import './vlog.css';

/**
 * What the log sheet hands the recorder — a subset of the `VoiceLogSlotProps` of `features/log-day`
 * (declared here too: a feature can't import another feature; the screen wires the two).
 */
export interface VoiceLogPanelProps {
  /** `useAiGate('voice_log')`, wired by `widgets/voice-log-slot` (a feature can't import the ai-gate feature). */
  ai: AiGateHooks;
  /** The log sheet's controls (`logSheetManifest`), or null while the taxonomy loads. */
  manifest: AutopilotManifest | null;
  date: string;
  categories: readonly LogCategory[];
  /** The day's draft: what is already picked stays picked (model/pilot.ts). */
  values: LogDayValues;
  /** Switches the sheet to «ثبت دستی», where the autopilot fills the form. */
  onManual: () => void;
}

type Blocking = Extract<RecorderStatus, 'denied' | 'unsupported' | 'nomic' | 'error'>;

function isBlocking(status: RecorderStatus): status is Blocking {
  return status === 'denied' || status === 'unsupported' || status === 'nomic' || status === 'error';
}

const EXAMPLES = ['bleeding', 'pain', 'mood'] as const;

/**
 * «ثبت با صدا»: one orb. Tap → the AI gate is asked, then the microphone records; tap again → the clip and
 * the log sheet's manifest go to `POST /logs/voice/pilot`; the answer is a UI autopilot script. What survives
 * `voiceScript` is played on «ثبت دستی» in front of the user — nothing is saved until she presses «ذخیره»
 * herself. The recording lives only in memory for the upload.
 */
export function VoiceLogPanel({ ai, manifest, date, categories, values, onManual }: VoiceLogPanelProps) {
  const t = useTranslations('voiceLog');
  const locale = useLocale() as Locale;
  const upload = useVoicePilot();
  // An answer with nothing to fill in (silence, or words that match no field).
  const [unmatched, setUnmatched] = useState<VoicePilotResult | null>(null);

  const onAnswer = (result: VoicePilotResult) => {
    ai.refresh();
    const script = manifest ? voiceScript(result.script, manifest, values, categories) : null;
    if (!script) {
      setUnmatched(result);
      return;
    }
    onManual();
    // After the manual tab has painted, so the engine finds the form instead of looking for its surface.
    requestAnimationFrame(() =>
      requestAnimationFrame(() => {
        void runAutopilot(script, { preset: 'brisk', onStepError: 'skip' }).done;
      }),
    );
  };

  const recorder = useVoiceRecorder((audio, durationMs) => {
    if (!manifest) return;
    upload.mutate({ audio, durationMs, manifest }, { onSuccess: onAnswer, onError: ai.handleError });
  });

  // Another day picked in the strip: the last answer belonged to the previous one.
  const { reset: resetUpload } = upload;
  useEffect(() => {
    setUnmatched(null);
    resetUpload();
  }, [date, resetUpload]);

  const listening = recorder.status === 'recording' || recorder.status === 'paused';
  const requesting = recorder.status === 'requesting';
  const state: OrbState = upload.isPending ? 'thinking' : listening ? 'listening' : 'idle';

  const press = () => {
    if (listening) {
      recorder.stop();
      return;
    }
    setUnmatched(null);
    upload.reset();
    if (ai.guard()) recorder.start();
  };

  const retryUpload = () => {
    if (upload.variables) upload.mutate(upload.variables, { onSuccess: onAnswer, onError: ai.handleError });
  };

  const hint = upload.isPending
    ? t('hint.thinking')
    : listening
      ? t('hint.listening')
      : requesting
        ? t('requesting')
        : recorder.status === 'tooShort'
          ? t('error.tooShort')
          : manifest
            ? t('hint.idle')
            : t('hint.loading');

  const manualButton = (
    <SecondaryButton block={false} variant="text" onClick={onManual}>
      {t('manual')}
    </SecondaryButton>
  );

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
              {manualButton}
            </div>
          }
        />
      </section>
    );
  } else if (upload.isError) {
    notice = (
      <section className="nb-card vlog-card">
        <EmptyState
          icon="warning"
          title={t('error.uploadTitle')}
          body={getApiErrorMessage(upload.error) ?? t('error.uploadBody')}
          action={
            <div className="vlog-actions">
              <SecondaryButton block={false} onClick={retryUpload}>
                {t('error.retry')}
              </SecondaryButton>
              {manualButton}
            </div>
          }
        />
      </section>
    );
  } else if (unmatched) {
    const silent = unmatched.transcript.trim() === '';
    notice = (
      <section className="nb-card vlog-card">
        {silent ? null : <p className="vlog-heard">{t('heard', { text: unmatched.transcript })}</p>}
        <EmptyState
          icon={silent ? 'mic' : 'search'}
          title={silent ? t('empty.silentTitle') : t('empty.title')}
          body={silent ? t('empty.silentBody') : t('empty.body')}
          action={manualButton}
        />
      </section>
    );
  }

  return (
    <div className="vlog">
      <section className="nb-card vlog-hero" aria-labelledby="vlog-hero-title">
        <h3 id="vlog-hero-title" className="vlog-hero-title">
          {t('title')}
        </h3>
        <p className="vlog-lead">{t('lead')}</p>
        <VoiceOrb state={state} label={listening ? t('stop') : t('start')} disabled={!manifest || requesting} onPress={press} />
        {listening ? (
          <p className="vlog-timer" aria-live="off">
            {formatNumber(formatElapsed(recorder.elapsedMs), locale)}
          </p>
        ) : null}
        <p className="vlog-hint" aria-live="polite">
          {hint}
        </p>
        {listening ? (
          <button type="button" className="vlog-cancel" onClick={recorder.cancel}>
            {t('cancel')}
          </button>
        ) : null}
      </section>

      {notice}

      <section className="vlog-sec" aria-labelledby="vlog-ex-title">
        <h3 id="vlog-ex-title" className="vlog-sec-title">
          {t('examples.title')}
        </h3>
        <ul className="vlog-exs">
          {EXAMPLES.map((key) => (
            <li key={key} className="vlog-ex">
              {t(`examples.${key}`)}
            </li>
          ))}
        </ul>
      </section>

      <InfoNote icon="shield" className="vlog-privacy">
        {t('privacy')}
      </InfoNote>
    </div>
  );
}
