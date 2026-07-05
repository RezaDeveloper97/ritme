// Public API of the `message` entity. Import only from here (CLAUDE.md §3.3).
export type {
  AppMode,
  UserMode,
  PrimaryMessage,
  DailyMessage,
} from './model/types';
export {
  messageKeys,
  useUserMode,
  useDailyMessage,
  fetchUserMode,
  fetchDailyMessage,
} from './api/queries';
