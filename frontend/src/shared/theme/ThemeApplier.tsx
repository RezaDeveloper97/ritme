'use client';

import { useEffect } from 'react';

import { isThemePreference, THEME_KEY, useThemeStore } from './store';

/**
 * Keeps <html data-theme> in sync with the stored preference, including the
 * same choice made in another tab. Mounted once in the app layout.
 *
 * The initial paint is handled by the inline script (see `themeInitScript`), so
 * there is no light-flash before hydration; this component owns everything that
 * happens *after* it.
 *
 * The OS setting is deliberately not consulted anywhere: light is the app's
 * default until the user turns dark mode on in Profile (see `store.ts`).
 */
export function ThemeApplier() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  // Another tab (or another window of the installed PWA) changing the
  // preference must not leave this one on the old theme.
  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== THEME_KEY) return;
      if (isThemePreference(event.newValue)) setTheme(event.newValue);
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [setTheme]);

  return null;
}

/**
 * Inline bootstrap: applies the stored theme before first paint, so a dark-mode
 * user never sees a white flash. Rendered as a <script> in the root layout (a
 * server component), so it must stay a plain string with no imports — it is the
 * one deliberate duplicate of `applyTheme`.
 *
 * Anything other than a stored 'dark' resolves to light, including a phone in
 * dark mode: the OS does not decide this app's theme.
 *
 * `ritme_theme` here is `THEME_KEY`; keep the two in step.
 */
export const themeInitScript = `(function(){try{
var d=localStorage.getItem('ritme_theme')==='dark';
var r=document.documentElement;
r.dataset.theme=d?'dark':'light';
var c=getComputedStyle(r).getPropertyValue('--page').trim();
if(c){var m=document.querySelectorAll('meta[name="theme-color"]');for(var i=0;i<m.length;i++)m[i].content=c;}
}catch(e){}})();`;
