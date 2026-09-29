import { createLocalFileStore, matchesAccept } from '@/shared/lib/local-files';

/**
 * Checkup report files (MarkDone «پیوست گزارش»: photo or PDF), stored on this
 * device only and keyed by record id. The server never receives the file — the
 * record just carries `has_attachment` (docs/checkups/README.md). Reads live
 * here so the detail, history and PDF-summary screens can show them; writes go
 * through `features/record-checkup` together with the record itself.
 *
 * Every session end wipes these files (`shared/session` → `clearAllLocalFiles`),
 * and `pruneCheckupAttachments` drops files whose record is gone.
 */
/**
 * Photos a phone camera / gallery produces. An explicit list, never `image/*`:
 * that admits `image/svg+xml`, which can carry script (security audit M3-M7 #2).
 */
export const CHECKUP_ATTACHMENT_IMAGE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/heic',
  'image/heif',
] as const;

export const CHECKUP_ATTACHMENT_ACCEPT = [...CHECKUP_ATTACHMENT_IMAGE_TYPES, 'application/pdf'] as const;

/** An allow-listed photo type — safe to show in an `<img>` preview. */
export function isCheckupAttachmentImage(type: string): boolean {
  return matchesAccept(type, CHECKUP_ATTACHMENT_IMAGE_TYPES);
}

/** 15 MB — a phone photo of a report or a lab PDF fits comfortably. */
export const CHECKUP_ATTACHMENT_MAX_BYTES = 15 * 1024 * 1024;

/** 300 MB across all reports, well under typical per-origin quotas. */
export const CHECKUP_ATTACHMENTS_MAX_TOTAL_BYTES = 300 * 1024 * 1024;

export const checkupAttachments = createLocalFileStore({
  namespace: 'checkup-reports',
  maxFileBytes: CHECKUP_ATTACHMENT_MAX_BYTES,
  maxTotalBytes: CHECKUP_ATTACHMENTS_MAX_TOTAL_BYTES,
  accept: CHECKUP_ATTACHMENT_ACCEPT,
});
