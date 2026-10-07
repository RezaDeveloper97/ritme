// Public API of the `menopause` entity (CB-MENO-05 on the CB-MENO-02/12 API). Import only from here (CLAUDE.md §3.3).

export {
  MENOPAUSE_STAGE_ANSWERS,
  type MenopauseCatalogItem,
  type MenopauseCheckupRaw,
  type MenopauseFlash,
  type MenopauseMessage,
  type MenopauseMessageKind,
  type MenopauseProfile,
  type MenopauseProfileUpdate,
  type MenopauseScoreSummary,
  type MenopauseStage,
  type MenopauseStageAnswer,
  type MenopauseToday,
  type MenopauseTreatment,
  type MenopauseTrendPoint,
} from './model/types';
export { menopauseKeys } from './api/keys';
export {
  menopauseFlashSchema,
  menopauseMessagesSchema,
  menopauseProfileSchema,
  menopauseTodaySchema,
} from './api/schema';
export {
  fetchMenopauseMessages,
  fetchMenopauseProfile,
  fetchMenopauseToday,
  toMenopauseProfileBody,
  useHotFlashTimer,
  useMenopauseMessages,
  useMenopauseProfile,
  useMenopauseToday,
  useSaveMenopauseProfile,
} from './api/queries';

// CB-MENO-08 — monthly score + patterns.
export {
  MENOPAUSE_SCORE_DOMAINS,
  type MenopausePattern,
  type MenopausePatterns,
  type MenopauseScoreBand,
  type MenopauseScoreDomain,
  type MenopauseScoreEntry,
  type MenopauseScoreHistory,
  type MenopauseScoreQuestion,
} from './model/score';
export {
  menopausePatternsSchema,
  menopauseScoreHistorySchema,
  menopauseScoreQuestionsSchema,
  useMenopausePatterns,
  useMenopauseScoreQuestions,
  useMenopauseScores,
  useSaveMenopauseScore,
} from './api/score';

// CB-MENO-07 — hot-flash timer screen.
export {
  HOT_FLASH_SEVERITIES,
  HOT_FLASH_TRIGGERS,
  type HotFlashDetails,
  type HotFlashSeverity,
  type HotFlashTrigger,
  type MenopauseFlashDay,
} from './model/types';
export {
  fetchHotFlashDay,
  menopauseFlashDaySchema,
  toHotFlashDetailsBody,
  useHotFlashDay,
  useMenopauseTips,
} from './api/hot-flashes';

// CB-MENO-10 — treatment & care.
export {
  SIDE_EFFECT_CODES,
  TREATMENT_GOAL_UNITS,
  TREATMENT_KINDS,
  TREATMENT_LIMITS,
  TREATMENT_SCHEDULES,
  type SideEffectCode,
  type TreatmentDay,
  type TreatmentGoal,
  type TreatmentGoalUnit,
  type TreatmentItem,
  type TreatmentItemInput,
  type TreatmentKind,
  type TreatmentSchedule,
  type TreatmentScreen,
  type TreatmentTip,
  toTreatmentItemBody,
  treatmentItemInput,
} from './model/treatment';
export {
  fetchTreatment,
  treatmentScreenSchema,
  useDeleteTreatmentItem,
  useSaveSideEffects,
  useSaveTreatmentItem,
  useTreatment,
  useTreatmentIntake,
} from './api/treatment';
