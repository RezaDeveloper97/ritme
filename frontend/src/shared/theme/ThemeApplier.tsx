'use client';

import { useEffect } from 'react';

import { isThemePreference, THEME_KEY, useThemeStore, watchSystemTheme } from './store';

/**
 * Keeps <html data-theme> in sync with the preference after hydration:
 * - the resolved theme from the store,
 * - the OS appearance, live, while the preference is `system`,
 * - the same choice made in another tab / window of the installed PWA.
 * Mounted once in the app layout.
 *
 * The initial paint is handled by the inline script (see `themeInitScript`), so
 * there is no flash before hydration; this component owns everything after it.
 */
export function ThemeApplier() {
  const theme = useThemeStore((s) => s.theme);
  const setPreference = useThemeStore((s) => s.setPreference);
  const syncSystem = useThemeStore((s) => s.syncSystem);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  // A `system` user whose phone switches to dark at sunset switches with it.
  // The store ignores the event for an explicit light/dark preference.
  useEffect(() => watchSystemTheme(syncSystem), [syncSystem]);

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== THEME_KEY) return;
      if (isThemePreference(event.newValue)) setPreference(event.newValue);
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [setPreference]);

  return null;
}

/**
 * Inline bootstrap: resolves the stored preference before first paint, so
 * neither a dark-mode user nor a `system` user on a dark phone sees a light
 * flash. Rendered as a <script> in the root layout (a server component), so it
 * must stay a plain string with no imports — it is the one deliberate duplicate
 * of `resolveTheme` + `applyTheme`.
 *
 * Stored 'light' / 'dark' win; anything else (nothing stored, 'system', junk)
 * follows prefers-color-scheme — the store's DEFAULT_THEME is 'system'.
 *
 * `ritme_theme` here is `THEME_KEY`; keep the two in step (lint:dark checks).
 */
export const themeInitScript = `(function(){try{
var s=null;try{s=localStorage.getItem('ritme_theme');}catch(e){}
var d=s==='dark'||(s!=='light'&&!!window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches);
var r=document.documentElement;
r.dataset.theme=d?'dark':'light';
var c=getComputedStyle(r).getPropertyValue('--page').trim();
if(c){var m=document.querySelectorAll('meta[name="theme-color"]');for(var i=0;i<m.length;i++)m[i].content=c;}
}catch(e){}})();`;
