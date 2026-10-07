// Public API of shared/theme.
export {
  useThemeStore,
  applyTheme,
  resolveTheme,
  isThemePreference,
  THEME_KEY,
  DEFAULT_THEME,
} from './store';
export type { ThemePreference, ResolvedTheme } from './store';
export { themeInitScript } from './init-script';
