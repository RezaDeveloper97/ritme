export { LOG_CATEGORIES } from './model/categories';
export type {
  CategoryDef,
  CategoryKey,
  EnumKey,
  FieldControl,
  FieldDef,
  HealthLogEnums,
  HealthLogField,
  HealthLogInput,
} from './model/types';
export {
  healthLogKeys,
  useHealthLog,
  useHealthLogEnums,
  useSaveHealthLog,
} from './api/queries';

// Log taxonomy v2 (B-N3-01/02/03): `/logs/taxonomy`, `/logs/days*`, `/logs/preferences`.
export {
  fetchLogDay,
  fetchLogDays,
  fetchLogPreferences,
  fetchLogTaxonomy,
  logKeys,
  saveLogDay,
  useLogDay,
  useLogDays,
  useLogPreferences,
  useLogTaxonomy,
} from './api/log-v2';
export {
  addLogCustomItem,
  deleteLogCustomItem,
  renameLogCustomItem,
  resetLogPreferences,
  saveLogPreferences,
  type LogCustomHost,
  type LogPreferencesChanges,
} from './api/log-prefs';
export type {
  LogCategory,
  LogCustomItem,
  LogDay,
  LogDayChanges,
  LogDaysRange,
  LogDayValues,
  LogItemValue,
  LogLabeled,
  LogOption,
  LogParam,
  LogParamType,
  LogParamValue,
  LogPrefCategory,
  LogPreferences,
  LogRange,
  LogTaxonomy,
} from './model/log-v2';
