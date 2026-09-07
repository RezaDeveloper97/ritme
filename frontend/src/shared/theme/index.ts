// Public API of the `theme` shared slice. Import only from here (§3.3).
export {
  useThemeStore,
  applyTheme,
  resolveTheme,
  isThemePreference,
  THEME_KEY,
  THEME_PREFERENCES,
  type ThemePreference,
  type ResolvedTheme,
} from './store';
export { ThemeApplier, themeInitScript } from './ThemeApplier';
