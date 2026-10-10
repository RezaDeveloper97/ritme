/**
 * Unfinished uploads survive a reload: the browser forgets the File, so the panel remembers which upload it was
 * (media id + the file's name / size / mtime) and asks for the same file again; the server's offset is the resume
 * point (GET /media/{id}). Only ids and the file name live here — no student data.
 */
export interface PendingUpload {
  mediaId: number;
  lessonId: number;
  courseId: number;
  lessonTitle: string;
  name: string;
  size: number;
  lastModified: number;
  /** Publish the lesson (and a standalone item) once the file is ready. */
  publish: boolean;
  standalone: boolean;
  startedAt: number;
}

export const PENDING_KEY = 'ritme_instructor_uploads';
/** Server UploadTTL (internal/media): older entries cannot resume. */
export const PENDING_TTL_MS = 24 * 60 * 60 * 1000;

interface KV {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

function storage(): KV | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage;
  } catch {
    return null;
  }
}

export function readPending(now = Date.now(), kv: KV | null = storage()): PendingUpload[] {
  if (!kv) return [];
  try {
    const raw = JSON.parse(kv.getItem(PENDING_KEY) ?? '[]') as unknown;
    if (!Array.isArray(raw)) return [];
    return raw.filter(
      (p): p is PendingUpload =>
        typeof p === 'object' &&
        p !== null &&
        typeof (p as PendingUpload).mediaId === 'number' &&
        typeof (p as PendingUpload).startedAt === 'number' &&
        now - (p as PendingUpload).startedAt < PENDING_TTL_MS,
    );
  } catch {
    return [];
  }
}

function write(list: PendingUpload[], kv: KV | null): void {
  try {
    kv?.setItem(PENDING_KEY, JSON.stringify(list));
  } catch {
    // storage full / blocked: resume just won't be offered
  }
}

export function savePending(p: PendingUpload, kv: KV | null = storage()): void {
  write([...readPending(Date.now(), kv).filter((x) => x.mediaId !== p.mediaId && x.lessonId !== p.lessonId), p], kv);
}

export function removePending(mediaId: number, kv: KV | null = storage()): void {
  write(readPending(Date.now(), kv).filter((x) => x.mediaId !== mediaId), kv);
}

/** Is `file` the one this pending upload was started with? */
export function isSameFile(p: PendingUpload, file: { name: string; size: number; lastModified: number }): boolean {
  return p.name === file.name && p.size === file.size && p.lastModified === file.lastModified;
}

/** The byte range [start, end) of the next chunk. */
export function nextChunk(offset: number, size: number, chunkSize: number): [number, number] {
  return [offset, Math.min(size, offset + Math.max(1, chunkSize))];
}

/** Whole percent, 100 only when complete. */
export function percentOf(sent: number, total: number): number {
  if (total <= 0) return 0;
  if (sent >= total) return 100;
  return Math.min(99, Math.floor((sent / total) * 100));
}
