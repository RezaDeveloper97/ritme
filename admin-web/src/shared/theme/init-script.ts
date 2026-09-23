import { DEFAULT_THEME, THEME_KEY } from './store-constants';

/**
 * Runs before first paint (inline in <head>) so a dark-mode admin never sees a
 * light flash. Plain JS duplicate of applyTheme(); keep the two in step.
 */
export const themeInitScript = `(function(){try{var t=localStorage.getItem(${JSON.stringify(
  THEME_KEY,
)});document.documentElement.dataset.theme=(t==='dark'||t==='light')?t:${JSON.stringify(
  DEFAULT_THEME,
)};}catch(e){document.documentElement.dataset.theme=${JSON.stringify(DEFAULT_THEME)};}})();`;
