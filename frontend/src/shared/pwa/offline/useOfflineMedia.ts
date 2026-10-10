'use client';

import { useCallback, useEffect, useState } from 'react';

import {
  getOffline,
  listOffline,
  offlineSupported,
  offlineUsage,
  readOfflineBlob,
  subscribeOffline,
  type OfflineItem,
} from './offline-media';

export type OfflineListStatus = 'loading' | 'ready' | 'error' | 'unsupported';

/** The downloads list + storage meter, re-read whenever the store changes. */
export function useOfflineList() {
  const [status, setStatus] = useState<OfflineListStatus>('loading');
  const [items, setItems] = useState<OfflineItem[]>([]);
  const [usage, setUsage] = useState<{ used: number; quota: number | null }>({ used: 0, quota: null });

  const reload = useCallback(async () => {
    if (!offlineSupported()) {
      setStatus('unsupported');
      return;
    }
    try {
      const [list, use] = await Promise.all([listOffline(), offlineUsage()]);
      setItems(list);
      setUsage(use);
      setStatus('ready');
    } catch {
      setStatus('error');
    }
  }, []);

  useEffect(() => {
    void reload();
    return subscribeOffline(() => void reload());
  }, [reload]);

  return { status, items, usage, reload };
}

/** Whether a lesson is downloaded (live). `null` until checked. */
export function useOfflineItem(lessonId: number | null) {
  const [item, setItem] = useState<OfflineItem | null | undefined>(undefined);
  useEffect(() => {
    if (lessonId == null || !offlineSupported()) {
      setItem(null);
      return;
    }
    let live = true;
    const read = () =>
      getOffline(lessonId).then(
        (v) => live && setItem(v),
        () => live && setItem(null),
      );
    void read();
    const off = subscribeOffline(() => void read());
    return () => {
      live = false;
      off();
    };
  }, [lessonId]);
  return item;
}

/**
 * A `blob:` URL of the decrypted download, for in-app playback only; revoked
 * on unmount. `checked` turns true once we know whether a local copy exists.
 */
export function useOfflineBlobUrl(lessonId: number | null) {
  const item = useOfflineItem(lessonId);
  const [url, setUrl] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const has = !!item;

  useEffect(() => {
    if (!has || lessonId == null) {
      setUrl(null);
      return;
    }
    let live = true;
    let made: string | null = null;
    readOfflineBlob(lessonId).then(
      (blob) => {
        if (!live || !blob) return;
        made = URL.createObjectURL(blob);
        setUrl(made);
      },
      () => live && setFailed(true),
    );
    return () => {
      live = false;
      if (made) URL.revokeObjectURL(made);
    };
  }, [has, lessonId]);

  return { checked: item !== undefined && (!has || url !== null || failed), url, failed };
}
