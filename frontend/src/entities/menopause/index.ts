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
