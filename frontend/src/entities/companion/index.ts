// Public API of the `companion` entity (همدم & family, owner side — B-N4-04 on B-N4-02's API).
// Import only from here (CLAUDE.md §3.3).
export {
  companionKeys,
  createCompanion,
  fetchCompanion,
  fetchCompanionAudit,
  fetchCompanions,
  useCompanion,
  useCompanionAudit,
  useCompanions,
  useCreateCompanion,
  useRenewCompanionInvite,
  useRevokeCompanion,
  useUpdateCompanionGrants,
} from './api/queries';
export {
  companionName,
  emptyGrants,
  familySpouse,
  grantsByLevel,
  hoursUntil,
  sameGrants,
  sharedSections,
} from './lib/grants';
export {
  ACCESS_LEVELS,
  COMPANION_SECTIONS,
  COMPANION_TYPES,
  type AccessLevel,
  type CompanionAuditAction,
  type CompanionAuditEntry,
  type CompanionGrants,
  type CompanionInvite,
  type CompanionSection,
  type CompanionStatus,
  type CompanionType,
  type CreateCompanionInput,
  type CreatedCompanion,
  type OwnerCompanion,
} from './model/types';
export { CompanionCard } from './ui/CompanionCard';
export { FamilyStrip, PersonBubble, type PersonTone } from './ui/FamilyStrip';
// B-N4-05: the companion's side — accept a code, links, leave, the companion panel home.
export {
  acceptErrorKind,
  type AcceptErrorKind,
  companionSideKeys,
  fetchCompanionHome,
  fetchCompanionLinks,
  useAcceptCompanion,
  useCompanionHome,
  useCompanionLinks,
  useLeaveCompanionLink,
} from './api/companion-side';
export { isCompleteCompanionCode, normalizeCompanionCode } from './lib/code';
export {
  COMPANION_CODE_LENGTH,
  COMPANION_PHASES,
  type CompanionArticle,
  type CompanionHome,
  type CompanionPartner,
  type CompanionPhase,
  type CompanionTip,
  type SharedAppointment,
  type SharedCycleView,
  type SharedMedication,
  type SharedPregnancyView,
  type ViewerLink,
} from './model/companion-side';
export { CompanionCodeField } from './ui/CompanionCodeField';
export { appInviteUrl, shareAppInvite, type InviteShareOutcome } from './lib/share-invite';
// B-N4-06: «ثبت برای چه کسی؟» — who a companion with edit may record meds / appointments for.
export {
  canRecordFor,
  initialRecordTarget,
  isCompanionForbidden,
  parseForUserId,
  recordTargets,
  showRecordForPicker,
  type RecordSection,
  type RecordTarget,
} from './lib/record-for';
export { RecordedFor, RecordForRow, RecordForSheet } from './ui/RecordForSheet';
export { type RecordForState, useRecordFor } from './api/record-for';
