// Public API of the `skip-period` feature. Import only from here (CLAUDE.md §3.3).
// Lazy wrapper: the sheet loads when first rendered, not with the barrel (T-M12-27b).
export { SkipPeriodSheet } from './ui/SkipPeriodSheetLazy';
export { SkipUndo } from './ui/SkipUndo';
export { useSkipPeriod, useUndoSkip, skipErrorKind } from './api/mutations';
export type { SkipErrorKind } from './api/mutations';
export { canOfferSkip } from './model/visibility';
export type { SkipOfferInput } from './model/visibility';
