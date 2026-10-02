// Public API of the bottom-nav widget (Night & Bloom, B-N1-04). Import only from here (§3.3).
export { BottomNav, type BottomNavProps } from './ui/BottomNav';
export { LogSheet, LogSheetTitle } from './ui/LogSheet';
export { useNavMode, type NavModeState } from './model/use-nav-mode';
export {
  activeTabKey,
  IVF_TREATMENT_FALLBACK,
  modeTab,
  NAV_READY,
  navConfig,
  resolveNavMode,
  todayHref,
  type NavConfig,
  type NavKey,
  type NavMode,
  type NavReady,
  type NavTab,
} from './model/nav-items';
