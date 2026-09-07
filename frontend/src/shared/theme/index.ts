// Public API of the `theme` shared slice. Import only from here (§3.3).
export {
  useThemeStore,
  applyTheme,
  isThemePreference,
  THEME_KEY,
  THEME_PREFERENCES,
  DEFAULT_THEME,
  type ThemePreference,
} from './store';
export { ThemeApplier, themeInitScript } from './ThemeApplier';
