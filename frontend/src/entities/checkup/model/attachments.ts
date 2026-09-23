import { createLocalFileStore } from '@/shared/lib/local-files';

/**
 * Checkup report files (MarkDone «پیوست گزارش»: photo or PDF), stored on this
 * device only and keyed by record id. The server never receives the file — the
 * record just carries `has_attachment` (docs/checkups/README.md). Reads live
 * here so the detail, history and PDF-summary screens can show them; writes go
 * through `features/record-checkup` together with the record itself.
 */
export const CHECKUP_ATTACHMENT_ACCEPT = ['image/*', 'application/pdf'] as const;

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
