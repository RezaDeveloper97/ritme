// Public API of the `pregnancy` entity. Import only from here (CLAUDE.md §3.3).

// ── Model: pure logic & config ─────────────────────────────────
export {
  clampWeek,
  derivePregnancyProgress,
  trimesterOfWeek,
  type PregnancyProgress,
  type Trimester,
} from './model/progress';
export { pickBilingual, resolveFlags } from './model/localize';
export {
  CONTENT_MODULE_ORDER,
  CRITICAL_SYMPTOMS,
  isCriticalSymptom,
  SYMPTOM_GROUPS,
  SYMPTOM_KEYS,
  TOTAL_WEEKS,
  WEEKLY_MENTAL,
} from './model/config';

// ── Model: types ───────────────────────────────────────────────
export type {
  AgeSource,
  AlertCounts,
  AlertLevel,
  AlertSummary,
  Bilingual,
  ContentModuleKey,
  EstimatedDueDate,
  FetalMovementInput,
  GestationalAge,
  MovementStatus,
  OnboardingInput,
  PregnancyActivation,
  PregnancyAlert,
  PregnancyEnums,
  PregnancyFlags,
  PregnancyProfile,
  PregnancyStatus,
  SymptomEnums,
  SymptomKey,
  SymptomLogInput,
  WeeklyContent,
  WeeklyEnums,
  WeeklyFaqItem,
  WeeklyLogInput,
} from './model/types';

// ── API: keys, hooks, fetchers ─────────────────────────────────
export {
  pregnancyKeys,
  // status & profile
  usePregnancyStatus,
  usePregnancyProfile,
  fetchPregnancyStatus,
  fetchPregnancyProfile,
  // enums
  usePregnancyEnums,
  useSymptomEnums,
  useWeeklyEnums,
  // content
  useWeeklyContent,
  fetchWeeklyContent,
  // daily symptoms
  useSymptomLog,
  useSaveSymptomLog,
  // weekly checkup
  useWeeklyLog,
  useSaveWeeklyLog,
  // fetal movement
  useFetalMovement,
  useSaveFetalMovement,
  // alerts
  usePregnancyAlerts,
  useAlertSummary,
  useAlertAction,
  useMarkAllAlertsRead,
  // mode & onboarding
  useActivatePregnancy,
  useDeactivatePregnancy,
  useCompleteOnboarding,
} from './api/queries';
