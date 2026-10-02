'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { formatElapsed } from '../model/recorder';

const WAVE_BARS = 29;
const LISTENS = ['pain', 'bleeding', 'sleep', 'meds', 'mood', 'other'] as const;

interface VoiceRecordingProps {
  elapsedMs: number;
  paused: boolean;
  /** The clip is uploading / being understood. */
  processing: boolean;
  /** Waiting for the microphone permission. */
  requesting: boolean;
  canPause: boolean;
  onStop: () => void;
  onPause: () => void;
  onResume: () => void;
  onCancel: () => void;
}

/**
 * nbl_Voice_Record: «در حال گوش دادن», red dot + timer, live waveform, «متن زنده» and «تا الان فهمیدیم», then
 * «II» · stop · «لغو». `POST /logs/voice` takes the whole clip (no streaming), so the live-text card says the
 * words come after stop and the chips are what we listen for — nothing is shown as understood before the
 * server answers.
 */
export function VoiceRecording({ elapsedMs, paused, processing, requesting, canPause, onStop, onPause, onResume, onCancel }: VoiceRecordingProps) {
  const t = useTranslations('voiceLog.record');
  const locale = useLocale() as Locale;
  const live = !paused && !processing && !requesting;
  return (
    <div className="vlog vlog-recording">
      <h3 className="vlog-step-title">{processing ? t('processing') : t('title')}</h3>
      <div className="vlog-rec-state">
        <span className={clsx('vlog-dot', live && 'is-live')} aria-hidden />
        <span>{paused ? t('paused') : t('recording')}</span>
      </div>
      <p className="vlog-timer" aria-live="off">
        {formatNumber(formatElapsed(elapsedMs), locale)}
      </p>
      <div className={clsx('vlog-wv', live && 'is-live')} aria-hidden>
        {Array.from({ length: WAVE_BARS }, (_, i) => (
          <span key={i} className="vlog-wv-bar" />
        ))}
      </div>

      <section className="nb-card vlog-live" aria-labelledby="vlog-live-title">
        <h4 id="vlog-live-title" className="vlog-live-title">
          {t('liveTitle')}
        </h4>
        {processing ? (
          <SkeletonGroup label={t('processing')} className="vlog-skel">
            <Skeleton width="full" />
            <Skeleton width="medium" />
          </SkeletonGroup>
        ) : (
          <p className="vlog-live-text">
            {t('livePending')}
            <span className="vlog-caret" aria-hidden />
          </p>
        )}
      </section>

      <section className="vlog-sec" aria-labelledby="vlog-listens-title">
        <h4 id="vlog-listens-title" className="vlog-sec-title">
          {t('listensTitle')}
        </h4>
        <ul className="vlog-listens">
          {LISTENS.map((key) => (
            <li key={key} className="vlog-listen">
              {t(`listens.${key}`)}
            </li>
          ))}
        </ul>
      </section>

      <div className="vlog-controls">
        {/* board order: cancel at the start edge, stop in the middle, pause at the end edge */}
        <button type="button" className="vlog-ctl vlog-ctl-text" disabled={processing} onClick={onCancel}>
          {t('cancel')}
        </button>
        <button type="button" className="vlog-stop" aria-label={t('stop')} disabled={processing || requesting} onClick={onStop}>
          {processing || requesting ? <Icon name="loader" size={30} className="vlog-spin" /> : <span className="vlog-stop-glyph" aria-hidden />}
        </button>
        <button
          type="button"
          className="vlog-ctl"
          aria-label={paused ? t('resume') : t('pause')}
          disabled={!canPause || processing || requesting}
          onClick={paused ? onResume : onPause}
        >
          {paused ? <Icon name="mic" size={22} /> : <span className="vlog-pause-glyph" aria-hidden />}
        </button>
      </div>
    </div>
  );
}
