'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

import { isApiError } from '@/shared/api';

import { createUpload, deleteUpload, fetchUpload, patchChunk, type Media } from '../api/media-api';
import { nextChunk, removePending, savePending, type PendingUpload } from './pending';

export type UploadPhase = 'idle' | 'uploading' | 'paused' | 'done' | 'error';

export interface UploadState {
  phase: UploadPhase;
  fileName: string | null;
  sent: number;
  total: number;
  error: unknown;
}

const IDLE: UploadState = { phase: 'idle', fileName: null, sent: 0, total: 0, error: null };
const DEFAULT_CHUNK = 8 * 1024 * 1024;
const RETRIES = 3;

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/**
 * Resumable chunked upload over the tus-style /media API (B-N8-02): POST creates, PATCH appends one chunk at the
 * server offset, GET reads the offset back after a 409, a network drop or a reload. `onReady` runs once the server
 * reports the file ready (e.g. publish the lesson).
 */
export function useResumableUpload(onReady?: (pending: PendingUpload, media: Media) => Promise<void> | void) {
  const [state, setState] = useState<UploadState>(IDLE);
  const abort = useRef<AbortController | null>(null);
  const current = useRef<{ pending: PendingUpload; file: File; chunk: number } | null>(null);
  const readyRef = useRef(onReady);
  readyRef.current = onReady;

  useEffect(() => () => abort.current?.abort(), []);

  const run = useCallback(async (offset: number) => {
    const job = current.current;
    if (!job) return;
    const ctrl = new AbortController();
    abort.current = ctrl;
    const { pending, file, chunk } = job;
    setState({ phase: 'uploading', fileName: file.name, sent: offset, total: file.size, error: null });
    let pos = offset;
    let failures = 0;
    try {
      let media: Media | null = null;
      while (pos < file.size) {
        const [start, end] = nextChunk(pos, file.size, chunk);
        try {
          media = await patchChunk(pending.mediaId, start, file.slice(start, end), ctrl.signal);
          pos = media.offset_bytes;
          failures = 0;
        } catch (err) {
          if (ctrl.signal.aborted) throw err;
          const mismatch = isApiError(err) && err.code === 'media_offset_mismatch';
          const retryable =
            mismatch ||
            (isApiError(err) && (err.code === 'media_upload_busy' || err.status === 0 || err.status >= 500));
          if (!retryable || ++failures > RETRIES) throw err;
          if (!mismatch) await sleep(1000 * failures);
          pos = (await fetchUpload(pending.mediaId, ctrl.signal)).media.offset_bytes;
        }
        setState((s) => ({ ...s, sent: pos }));
      }
      media ??= (await fetchUpload(pending.mediaId, ctrl.signal)).media;
      await readyRef.current?.(pending, media);
      removePending(pending.mediaId);
      setState((s) => ({ ...s, phase: 'done', sent: file.size }));
    } catch (err) {
      if (ctrl.signal.aborted) {
        setState((s) => ({ ...s, phase: 'paused' }));
        return;
      }
      setState((s) => ({ ...s, phase: 'error', error: err }));
    }
  }, []);

  /** A new upload of `file` for a lesson the caller just created. */
  const start = useCallback(
    async (file: File, meta: Omit<PendingUpload, 'mediaId' | 'name' | 'size' | 'lastModified' | 'startedAt'>) => {
      setState({ phase: 'uploading', fileName: file.name, sent: 0, total: file.size, error: null });
      try {
        const { media, upload } = await createUpload({ lesson_id: meta.lessonId, size: file.size });
        const pending: PendingUpload = {
          ...meta,
          mediaId: media.id,
          name: file.name,
          size: file.size,
          lastModified: file.lastModified,
          startedAt: Date.now(),
        };
        savePending(pending);
        current.current = { pending, file, chunk: Math.min(upload.chunk_size || DEFAULT_CHUNK, upload.max_chunk_bytes || DEFAULT_CHUNK) };
        await run(media.offset_bytes);
      } catch (err) {
        setState((s) => ({ ...s, phase: 'error', error: err }));
      }
    },
    [run],
  );

  /** Continue a remembered upload with the same file picked again. */
  const resume = useCallback(
    async (pending: PendingUpload, file: File) => {
      setState({ phase: 'uploading', fileName: file.name, sent: 0, total: file.size, error: null });
      try {
        const { media } = await fetchUpload(pending.mediaId);
        current.current = { pending, file, chunk: DEFAULT_CHUNK };
        if (media.status === 'ready') {
          await readyRef.current?.(pending, media);
          removePending(pending.mediaId);
          setState({ phase: 'done', fileName: file.name, sent: file.size, total: file.size, error: null });
          return;
        }
        await run(media.offset_bytes);
      } catch (err) {
        if (isApiError(err) && (err.status === 404 || err.status === 410)) removePending(pending.mediaId);
        setState((s) => ({ ...s, phase: 'error', error: err }));
      }
    },
    [run],
  );

  const pause = useCallback(() => abort.current?.abort(), []);

  const retry = useCallback(async () => {
    const job = current.current;
    if (!job) return;
    try {
      const { media } = await fetchUpload(job.pending.mediaId);
      await run(media.offset_bytes);
    } catch (err) {
      setState((s) => ({ ...s, phase: 'error', error: err }));
    }
  }, [run]);

  const cancel = useCallback(async () => {
    abort.current?.abort();
    const job = current.current;
    current.current = null;
    setState(IDLE);
    if (job) {
      removePending(job.pending.mediaId);
      await deleteUpload(job.pending.mediaId).catch(() => undefined);
    }
  }, []);

  return { state, start, resume, pause, retry, cancel };
}
