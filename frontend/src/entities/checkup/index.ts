// Public API of the `checkup` entity (M4). Import only from here (CLAUDE.md §3.3).

// ── Model: types & enums ───────────────────────────────────────
export {
  CHECKUP_CATEGORIES,
  CHECKUP_LIST_FILTERS,
  CHECKUP_PERFORMERS,
  CHECKUP_RECORD_FILTERS,
  CHECKUP_RESULTS,
  CHECKUP_SECTIONS,
  CHECKUP_STATUSES,
  CHECKUP_TONES,
  CUSTOM_INTERVAL_MONTHS,
  type CheckupCategory,
  type CheckupDetail,
  type CheckupFindingOption,
  type CheckupGuideStep,
  type CheckupHome,
  type CheckupItem,
  type CheckupList,
  type CheckupListFilter,
  type CheckupNextPreview,
  type CheckupPerformer,
  type CheckupRecord,
  type CheckupRecordFilter,
  type CheckupRecordPage,
  type CheckupRecordResult,
  type CheckupResult,
  type CheckupSection,
  type CheckupSettings,
  type CheckupStatus,
  type CheckupSummary,
  type CheckupTone,
} from './model/types';
export { checkupIcon } from './model/icon';
export { checkupResultIcon, checkupStatusIcon, formatCheckupMonth } from './model/display';
export {
  CHECKUP_ATTACHMENT_ACCEPT,
  CHECKUP_ATTACHMENT_MAX_BYTES,
  CHECKUP_ATTACHMENTS_MAX_TOTAL_BYTES,
  checkupAttachments,
} from './model/attachments';

// ── API: keys, parsers, reads ──────────────────────────────────
export { checkupKeys, type CheckupRecordFilters } from './api/keys';
export {
  checkupDetailSchema,
  checkupHomeSchema,
  checkupItemSchema,
  checkupListSchema,
  checkupNextPreviewSchema,
  checkupRecordPageSchema,
  checkupRecordResultSchema,
  checkupRecordSchema,
} from './api/schema';
export {
  fetchCheckup,
  fetchCheckupHome,
  fetchCheckupNextPreview,
  fetchCheckupRecords,
  fetchCheckups,
  useCheckup,
  useCheckupAttachment,
  useCheckupAttachmentIds,
  useCheckupHome,
  useCheckupNextPreview,
  useCheckupRecords,
  useCheckups,
} from './api/queries';
