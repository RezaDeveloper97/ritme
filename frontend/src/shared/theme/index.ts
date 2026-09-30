// Public API of the `theme` shared slice. Import only from here (§3.3).
export {
  useThemeStore,
  applyTheme,
  isThemePreference,
  resolveTheme,
  systemTheme,
  watchSystemTheme,
  THEME_KEY,
  THEME_PREFERENCES,
  DEFAULT_THEME,
  type ThemePreference,
  type ResolvedTheme,
} from './store';
export { ThemeApplier, themeInitScript } from './ThemeApplier';
