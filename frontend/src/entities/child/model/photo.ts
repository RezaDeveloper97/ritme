'use client';

import { useEffect, useState } from 'react';

import { createLocalFileStore, LocalFilesError, type LocalFilesErrorCode } from '@/shared/lib/local-files';

/**
 * The child's photo (v15_AddChild «عکس · اختیاری · فقط روی گوشی تو»): kept in
 * this device's IndexedDB only, keyed by child id — never uploaded, never sent
 * to the API (the server has no photo column). Every session end wipes it with
 * the other on-device files (`shared/session` → `clearAllLocalFiles`).
 */

/** Explicit photo types — never `image/*`, which admits script-carrying SVG. */
export const CHILD_PHOTO_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/heic', 'image/heif'] as const;
export const CHILD_PHOTO_MAX_BYTES = 10 * 1024 * 1024;

export const childPhotos = createLocalFileStore({
  namespace: 'child-photos',
  maxFileBytes: CHILD_PHOTO_MAX_BYTES,
  maxTotalBytes: 120 * 1024 * 1024,
  accept: CHILD_PHOTO_TYPES,
});

const listeners = new Set<() => void>();
const notify = () => listeners.forEach((l) => l());

/** Pre-check a picked file (type / size) before showing it. */
export function checkChildPhoto(file: Blob): LocalFilesErrorCode | null {
  return childPhotos.check(file);
}

/** Stores the photo for `childId`; rejects with a {@link LocalFilesError} code. */
export async function saveChildPhoto(childId: number, file: Blob): Promise<void> {
  await childPhotos.put(childId, file, 'photo');
  notify();
}

export async function deleteChildPhoto(childId: number): Promise<void> {
  await childPhotos.delete(childId);
  notify();
}

export function photoErrorCode(error: unknown): LocalFilesErrorCode {
  return error instanceof LocalFilesError ? error.code : 'failed';
}

/** An object URL of the stored photo (null while loading / none); revoked on change and unmount. */
export function useChildPhoto(childId: number | null | undefined): string | null {
  const [url, setUrl] = useState<string | null>(null);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    const bump = () => setVersion((v) => v + 1);
    listeners.add(bump);
    return () => {
      listeners.delete(bump);
    };
  }, []);

  useEffect(() => {
    if (childId == null) return undefined;
    let alive = true;
    let created: string | null = null;
    void childPhotos.get(childId).then((file) => {
      if (!alive) return;
      created = file ? URL.createObjectURL(file.blob) : null;
      setUrl(created);
    });
    return () => {
      alive = false;
      if (created) URL.revokeObjectURL(created);
    };
  }, [childId, version]);

  return childId == null ? null : url;
}
