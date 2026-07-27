// Public API of the `message` entity. Import only from here (CLAUDE.md §3.3).
export type {
  AppMode,
  UserMode,
  PrimaryMessage,
  DailyMessage,
  MessageCorrelation,
  MessagePattern,
  SupplementTips,
} from './model/types';
export type { SmartTip, SmartTipSource } from './lib/smart-tip';
export { selectSmartTip } from './lib/smart-tip';
export {
  messageKeys,
  useUserMode,
  useDailyMessage,
  fetchUserMode,
  fetchDailyMessage,
} from './api/queries';
