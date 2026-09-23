// Public API of the `record-checkup` feature (CLAUDE.md §3.3).
export {
  useCreateCheckupRecord,
  useDeleteCheckupRecord,
  useUpdateCheckupRecord,
  type AttachmentInput,
  type CreateCheckupRecordVars,
  type DeleteCheckupRecordVars,
  type RecordCheckupOutcome,
  type UpdateCheckupRecordVars,
} from './api/mutations';
export {
  selfExamResult,
  toCheckupRecordBody,
  toggleFinding,
  type CheckupRecordInput,
  type CheckupRecordPatch,
} from './model/body';
