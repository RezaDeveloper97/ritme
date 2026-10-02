'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

/** The server refuses longer recordings (backend-go/internal/voicelog MaxDurationMs); we stop at 60 s. */
export const MAX_RECORDING_MS = 60_000;
/** Shorter than this is a mis-tap, not a sentence. */
export const MIN_RECORDING_MS = 800;

/** Containers in preference order: Chrome/Edge/Firefox Opus in WebM/Ogg, Safari AAC in MP4. */
const MIME_CANDIDATES = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4', 'audio/ogg;codecs=opus', 'audio/ogg'];

/** The first container the browser can record, or `undefined` to let it choose. */
export function pickMimeType(isTypeSupported: ((type: string) => boolean) | undefined): string | undefined {
  if (!isTypeSupported) return undefined;
  return MIME_CANDIDATES.find((type) => {
    try {
      return isTypeSupported(type);
    } catch {
      return false;
    }
  });
}

/** `m:ss` of an elapsed time (digits localised by the caller). */
export function formatElapsed(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`;
}

export type RecorderStatus = 'idle' | 'requesting' | 'recording' | 'paused' | 'denied' | 'unsupported' | 'nomic' | 'tooShort' | 'error';

/** Maps a getUserMedia / MediaRecorder failure to what the user can do about it. */
export function recorderFailure(error: unknown): RecorderStatus {
  const name = error && typeof error === 'object' && 'name' in error ? String((error as { name: unknown }).name) : '';
  if (name === 'NotAllowedError' || name === 'PermissionDeniedError' || name === 'SecurityError') return 'denied';
  if (name === 'NotFoundError' || name === 'DevicesNotFoundError' || name === 'OverconstrainedError') return 'nomic';
  if (name === 'NotSupportedError' || name === 'TypeError') return 'unsupported';
  return 'error';
}

/** Whether this browser can record at all (secure context + getUserMedia + MediaRecorder). */
export function canRecord(): boolean {
  return (
    typeof window !== 'undefined' &&
    window.isSecureContext !== false &&
    typeof navigator !== 'undefined' &&
    typeof navigator.mediaDevices?.getUserMedia === 'function' &&
    typeof window.MediaRecorder === 'function'
  );
}

export interface VoiceRecorder {
  status: RecorderStatus;
  elapsedMs: number;
  start: () => void;
  /** Stops and hands the recording to `onComplete` (unless it was too short). */
  stop: () => void;
  /** Pauses / resumes the clip (nbl_Voice_Record «II»); the timer holds while paused. */
  pause: () => void;
  resume: () => void;
  /** «لغو»: stops the microphone and drops the clip — nothing is uploaded. */
  cancel: () => void;
  /** Back to idle (after an error). */
  reset: () => void;
}

/**
 * Browser microphone recording with MediaRecorder: asks for permission, records one mono clip, auto-stops
 * at {@link MAX_RECORDING_MS} and calls `onComplete` with the blob. The audio stays in memory only (no
 * storage, no object URL); unmounting stops the microphone and drops the clip.
 */
export function useVoiceRecorder(onComplete: (audio: Blob, durationMs: number) => void, maxMs = MAX_RECORDING_MS): VoiceRecorder {
  const [status, setStatus] = useState<RecorderStatus>('idle');
  const [elapsedMs, setElapsedMs] = useState(0);
  const recorder = useRef<MediaRecorder | null>(null);
  const stream = useRef<MediaStream | null>(null);
  const chunks = useRef<Blob[]>([]);
  const startedAt = useRef(0);
  /** Paused time so far, and when the current pause began (0 = not paused). */
  const pausedMs = useRef(0);
  const pausedAt = useRef(0);
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);
  const discard = useRef(false);
  const complete = useRef(onComplete);
  complete.current = onComplete;

  const release = useCallback(() => {
    if (timer.current) clearInterval(timer.current);
    timer.current = null;
    stream.current?.getTracks().forEach((track) => track.stop());
    stream.current = null;
    recorder.current = null;
  }, []);

  /** Recorded time without the pauses. */
  const activeMs = useCallback(() => {
    const now = performance.now();
    const pausing = pausedAt.current ? now - pausedAt.current : 0;
    return now - startedAt.current - pausedMs.current - pausing;
  }, []);

  const stop = useCallback(() => {
    const r = recorder.current;
    if (r && r.state !== 'inactive') r.stop();
  }, []);

  const pause = useCallback(() => {
    const r = recorder.current;
    if (!r || r.state !== 'recording' || typeof r.pause !== 'function') return;
    r.pause();
    pausedAt.current = performance.now();
    setStatus('paused');
  }, []);

  const resume = useCallback(() => {
    const r = recorder.current;
    if (!r || r.state !== 'paused') return;
    r.resume();
    pausedMs.current += performance.now() - pausedAt.current;
    pausedAt.current = 0;
    setStatus('recording');
  }, []);

  const cancel = useCallback(() => {
    discard.current = true;
    const r = recorder.current;
    if (r && r.state !== 'inactive') r.stop();
    else release();
    setStatus('idle');
    setElapsedMs(0);
  }, [release]);

  const start = useCallback(() => {
    if (recorder.current) return;
    if (!canRecord()) {
      setStatus('unsupported');
      return;
    }
    setStatus('requesting');
    setElapsedMs(0);
    discard.current = false;
    navigator.mediaDevices
      .getUserMedia({ audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true } })
      .then((media) => {
        if (discard.current) {
          media.getTracks().forEach((track) => track.stop());
          return;
        }
        stream.current = media;
        const mimeType = pickMimeType(window.MediaRecorder.isTypeSupported?.bind(window.MediaRecorder));
        const r = new MediaRecorder(media, mimeType ? { mimeType, audioBitsPerSecond: 32_000 } : undefined);
        recorder.current = r;
        chunks.current = [];
        r.ondataavailable = (event) => {
          if (event.data.size > 0) chunks.current.push(event.data);
        };
        r.onstop = () => {
          const duration = activeMs();
          const blob = new Blob(chunks.current, { type: r.mimeType || mimeType || 'audio/webm' });
          chunks.current = [];
          release();
          if (discard.current) return;
          if (duration < MIN_RECORDING_MS || blob.size === 0) {
            setStatus('tooShort');
            return;
          }
          setStatus('idle');
          complete.current(blob, Math.min(duration, maxMs));
        };
        r.onerror = () => {
          discard.current = true;
          release();
          setStatus('error');
        };
        startedAt.current = performance.now();
        pausedMs.current = 0;
        pausedAt.current = 0;
        r.start(1000);
        setStatus('recording');
        timer.current = setInterval(() => {
          const elapsed = activeMs();
          setElapsedMs(elapsed);
          if (elapsed >= maxMs) stop();
        }, 250);
      })
      .catch((error: unknown) => {
        release();
        setStatus(recorderFailure(error));
      });
  }, [activeMs, maxMs, release, stop]);

  const reset = useCallback(() => {
    setStatus('idle');
    setElapsedMs(0);
  }, []);

  // Leaving the tab / closing the sheet: stop the microphone and drop the clip.
  useEffect(
    () => () => {
      discard.current = true;
      const r = recorder.current;
      if (r && r.state !== 'inactive') r.stop();
      release();
    },
    [release],
  );

  return { status, elapsedMs, start, stop, pause, resume, cancel, reset };
}
