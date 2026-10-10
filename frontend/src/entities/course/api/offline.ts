'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { OfflineError, deleteOffline, offlineSupported, purgeOffline, saveOffline, useOfflineItem, type OfflineErrorCode } from '@/shared/pwa/offline';

import { stillAccessible } from '../model/format';
import { playbackSchema } from './schema';
import { useMyCourses } from './queries';

/*
 * B-N8-04 offline lessons: the download action and the access purge. The
 * encrypted store itself is shared/pwa/offline/offline-media.ts.
 */

export interface DownloadTarget {
  lessonId: number;
  courseId: number;
  courseTitle: string;
  title: string;
  kind: 'video' | 'audio' | 'pdf';
}

export type DownloadState = 'checking' | 'idle' | 'downloading' | 'saved' | 'unsupported';

/** Download / remove one lesson; the signed URL is minted fresh for the fetch (10-min TTL). */
export function useLessonDownload(target: DownloadTarget) {
  const item = useOfflineItem(target.lessonId);
  const [busy, setBusy] = useState(false);
  const [percent, setPercent] = useState<number | null>(null);
  const [error, setError] = useState<Exclude<OfflineErrorCode, 'aborted'> | null>(null);
  const ctrl = useRef<AbortController | null>(null);
  const targetRef = useRef(target);
  targetRef.current = target;

  useEffect(() => () => ctrl.current?.abort(), []);

  const start = useCallback(async () => {
    const t = targetRef.current;
    setError(null);
    setPercent(null);
    setBusy(true);
    const abort = new AbortController();
    ctrl.current = abort;
    try {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/learning/lessons/${t.lessonId}/playback`);
      const pb = playbackSchema.parse(data.data);
      let res: Response;
      try {
        res = await fetch(pb.url, { credentials: 'omit', cache: 'no-store', signal: abort.signal });
      } catch {
        throw new OfflineError(abort.signal.aborted ? 'aborted' : 'network');
      }
      await saveOffline(t, res, {
        signal: abort.signal,
        onProgress: (got, total) => setPercent(total ? Math.min(99, Math.floor((got / total) * 100)) : null),
      });
    } catch (e) {
      const code = e instanceof OfflineError ? e.code : 'network';
      if (code !== 'aborted') setError(code);
    } finally {
      setBusy(false);
      setPercent(null);
      ctrl.current = null;
    }
  }, []);

  const cancel = useCallback(() => ctrl.current?.abort(), []);
  const remove = useCallback(() => deleteOffline(targetRef.current.lessonId).catch(() => undefined), []);

  const state: DownloadState = !offlineSupported()
    ? 'unsupported'
    : busy
      ? 'downloading'
      : item === undefined
        ? 'checking'
        : item
          ? 'saved'
          : 'idle';
  return { state, percent, error, start, cancel, remove, size: item?.size ?? null };
}

let purgedThisSession = false;

/**
 * Drops downloads whose course access ended or was revoked. Runs once per app
 * session after a successful online read of «دوره‌های من»; offline (query
 * error) nothing is removed — that waits for the next online check.
 */
export function useOfflinePurge() {
  const q = useMyCourses();
  const data = q.isSuccess && !q.isFetching ? q.data : null;
  useEffect(() => {
    if (!data || purgedThisSession || !offlineSupported()) return;
    purgedThisSession = true;
    void purgeOffline((i) => stillAccessible(data, i.courseId)).catch(() => undefined);
  }, [data]);
}
