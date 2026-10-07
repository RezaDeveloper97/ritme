import { type DocumentKind, MAX_DOCUMENT_FILES, MAX_UPLOAD_BYTES } from '@/entities/health-record';

/* The upload sheet's draft and its client checks, mirroring CB-CORE-05 (photo / PDF, 10 MB each, 10 per document). */

export interface DraftFile {
  key: string;
  file: File;
  kind: 'image' | 'pdf';
}

export type PickError = 'fileType' | 'fileSize' | 'tooMany';
export type DraftError = 'kind' | 'files';

/** Photo types the server accepts (HEIC/HEIF from iOS cameras are sent as-is and the server decides). */
const IMAGE_TYPES: readonly string[] = ['image/jpeg', 'image/png', 'image/webp', 'image/heic', 'image/heif'];

export function fileKindOf(file: Pick<File, 'type' | 'name'>): DraftFile['kind'] | null {
  if (IMAGE_TYPES.includes(file.type)) return 'image';
  if (file.type === 'application/pdf' || (file.type === '' && file.name.toLowerCase().endsWith('.pdf'))) return 'pdf';
  return null;
}

/** Adds picked files, keeping the valid ones up to the cap; reports the first problem. */
export function addFiles(
  current: readonly DraftFile[],
  picked: readonly File[],
  nextKey: () => string,
): { files: DraftFile[]; error: PickError | null } {
  const files = [...current];
  let error: PickError | null = null;
  for (const f of picked) {
    const kind = fileKindOf(f);
    if (!kind) {
      error ??= 'fileType';
      continue;
    }
    if (f.size > MAX_UPLOAD_BYTES) {
      error ??= 'fileSize';
      continue;
    }
    if (files.length >= MAX_DOCUMENT_FILES) {
      error ??= 'tooMany';
      break;
    }
    files.push({ key: nextKey(), file: f, kind });
  }
  return { files, error };
}

export function validateDraft(d: { kind: DocumentKind | null; files: readonly DraftFile[] }): DraftError | null {
  if (!d.kind) return 'kind';
  if (d.files.length === 0) return 'files';
  return null;
}

/** Overall progress 0–1 of `done` finished files plus the running one's share. */
export function overallProgress(done: number, current: number, total: number): number {
  if (total <= 0) return 0;
  return Math.min(1, (done + Math.max(0, Math.min(1, current))) / total);
}
