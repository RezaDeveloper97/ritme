import { DEFAULT_THEME, THEME_KEY } from './store-constants';

/**
 * Runs before first paint (inline in <head>) so a dark-mode admin never sees a
 * light flash. Plain JS duplicate of applyTheme(); keep the two in step.
 */
export const themeInitScript = `(function(){try{var t=localStorage.getItem(${JSON.stringify(
  THEME_KEY,
)});if(t!=='dark'&&t!=='light'&&t!=='system')t=${JSON.stringify(
  DEFAULT_THEME,
)};if(t==='system')t=(window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches)?'dark':'light';document.documentElement.dataset.theme=t;}catch(e){document.documentElement.dataset.theme=${JSON.stringify(DEFAULT_THEME)};}})();`;
