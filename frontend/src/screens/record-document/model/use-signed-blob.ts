'use client';

import { useEffect, useRef, useState } from 'react';

import { fetchSignedFile } from '@/entities/health-record';

/**
 * A signed file link read into an object URL (blob:), revoked on unmount or when the file changes. The signed URL
 * itself is never put in the DOM or opened in a tab (security audit CB-REC-04 L2). Keyed on the file id: a refetch
 * that only re-signs the same file does not reload it. `failed` → the caller shows the icon instead.
 */
export function useSignedBlob(fileId: number, url: string | null, enabled: boolean): { src: string | null; failed: boolean } {
  const [state, setState] = useState<{ id: number; src: string | null; failed: boolean }>({ id: fileId, src: null, failed: false });
  const latestUrl = useRef(url);
  // declared before the loader so it runs first in the same commit
  useEffect(() => {
    latestUrl.current = url;
  }, [url]);
  const hasUrl = url !== null;

  useEffect(() => {
    const signed = latestUrl.current;
    if (!enabled || !signed) {
      setState({ id: fileId, src: null, failed: false });
      return;
    }
    let made: string | null = null;
    let cancelled = false;
    setState({ id: fileId, src: null, failed: false });
    fetchSignedFile(signed)
      .then((blob) => {
        if (cancelled) return;
        made = URL.createObjectURL(blob);
        setState({ id: fileId, src: made, failed: false });
      })
      .catch(() => {
        if (!cancelled) setState({ id: fileId, src: null, failed: true });
      });
    return () => {
      cancelled = true;
      if (made) URL.revokeObjectURL(made);
    };
  }, [fileId, hasUrl, enabled]);

  return state.id === fileId ? { src: state.src, failed: state.failed } : { src: null, failed: false };
}
