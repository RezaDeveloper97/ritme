// Public API of the `log-period` feature. Import only from here (CLAUDE.md §3.3).
export { PeriodButton } from './ui/PeriodButton';
// Lazy wrapper: the editor and its CSS load when a screen first renders it, not with the barrel (T-M12-27b).
export { PeriodDateEditor } from './ui/PeriodDateEditorLazy';
export {
  useStartPeriod,
  useEndPeriod,
  usePeriodStatus,
  useLogPeriodRange,
  usePeriodHistory,
  useUpdatePeriod,
  useDeletePeriod,
  useReconcilePeriods,
} from './api/mutations';
export type { PeriodStatus, LoggedPeriod, PeriodSegment } from './api/mutations';
