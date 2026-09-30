// Public API of the bottom-nav widget (Night & Bloom, B-N1-04). Import only from here (§3.3).
export { BottomNav, type BottomNavProps } from './ui/BottomNav';
export { LogSheet, LogSheetTitle } from './ui/LogSheet';
export {
  activeTabKey,
  modeTab,
  navConfig,
  resolveNavMode,
  todayHref,
  type NavConfig,
  type NavKey,
  type NavMode,
  type NavTab,
} from './model/nav-items';
