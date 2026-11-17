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
  useUpdateCompanionChildren,
  useUpdateCompanionGrants,
  useUpdateCompanionPreferences,
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
  COMPANION_ACTIONS,
  COMPANION_GOALS,
  COMPANION_SECTIONS,
  COMPANION_SIGNALS,
  COMPANION_TYPES,
  EXCLUSIVE_COMPANION_GOALS,
  type AccessLevel,
  type CompanionAction,
  type CompanionAuditAction,
  type CompanionAuditEntry,
  type CompanionGoal,
  type CompanionGrants,
  type CompanionInvite,
  type CompanionPreferences,
  type CompanionSection,
  type CompanionSignal,
  type CompanionSignalScope,
  type CompanionStatus,
  type CompanionType,
  type CreateCompanionInput,
  type CreatedCompanion,
  type OwnerCompanion,
  type UpdateCompanionPreferencesInput,
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
  fetchInvitePreview,
  type InvitePreview,
  isPreviewUnsupported,
  useAcceptCompanion,
  useInvitePreview,
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
  type CompanionLinkSettings,
  type CompanionPartner,
  type CompanionPhase,
  type CompanionSignals,
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
