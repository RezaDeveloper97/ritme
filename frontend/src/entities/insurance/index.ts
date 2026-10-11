// Public API of the `insurance` entity (CB-INS-03). Import only from here (CLAUDE.md §3.3).
export {
  CENTRE_TYPES,
  COVERAGE_CATEGORIES,
  type CentreType,
  type InsuranceCentre,
  type ClaimStatus,
  type CoverageCategory,
  type CoverageStatus,
  type InsuranceClaimRow,
  type InsurancePolicy,
  type InsurancePolicyState,
  type InsurerOption,
  type MemberKind,
  type PolicyCoverage,
  type PolicyMember,
  type PolicyPayload,
} from './model/types';
export {
  type CompactAmount,
  capped,
  categoryKey,
  costTypeKey,
  compactAmount,
  deductibleTiles,
  homeCoverages,
  inMillions,
  maskedTail,
  remainingOf,
  TOMAN_MILLION,
} from './model/display';
export { useAmountFormat } from './lib/use-amount';
export {
  buildTimeline,
  evidenceLabel,
  QUESTIONNAIRE_STEPS,
  questionCode,
  sectionCode,
  type QuestionCode,
  type SectionCode,
  type TimelineRowId,
  type EvidenceText,
  type InsuranceQuestionnaire,
  type QuestionAnswer,
  type QuestionnaireEvent,
  type QuestionnairePurpose,
  type QuestionnaireQuestion,
  type QuestionnaireStatus,
  type QuestionnaireStep,
  type QuestionnaireUpdate,
  type TimelineRow,
} from './model/questionnaire';
export {
  useAdvanceQuestionnaire,
  useConfirmQuestionnaire,
  useQuestionnaire,
  useQuestionnaires,
  useStartQuestionnaire,
  useUpdateQuestionnaire,
} from './api/questionnaire-queries';
export { insuranceKeys } from './api/keys';
export { useCentres, useInsurancePolicy, useInsurers, useRecentClaims, useSaveInsurancePolicy } from './api/queries';
