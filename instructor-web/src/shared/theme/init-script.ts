import { DEFAULT_THEME, THEME_KEY } from './store-constants';

/**
 * Runs before first paint (inline in <head>) so a dark-mode instructor never
 * sees a light flash. Plain JS duplicate of resolveTheme(); keep the two in step.
 */
export const themeInitScript = `(function(){var d=document.documentElement,t=${JSON.stringify(
  DEFAULT_THEME,
)};try{var s=localStorage.getItem(${JSON.stringify(
  THEME_KEY,
)});if(s==='light'||s==='dark'||s==='system')t=s;}catch(e){}if(t==='system'){t=window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light';}d.dataset.theme=t;})();`;
