// Public API of the `log-period` feature. Import only from here (CLAUDE.md §3.3).
export { PeriodButton } from './ui/PeriodButton';
export { PeriodEditor } from './ui/PeriodEditor';
export { PeriodDateEditor } from './ui/PeriodDateEditor';
export type { PeriodSaveInfo } from './ui/PeriodEditor';
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
