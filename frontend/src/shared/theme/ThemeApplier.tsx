'use client';

import { useEffect } from 'react';

import { THEME_KEY, useThemeStore } from './store';

/**
 * Keeps <html data-theme> in sync with the stored preference, including OS
 * theme changes while 'system' is selected, and the same choice made in another
 * tab. Mounted once in the app layout.
 *
 * The initial paint is handled by the inline script (see `themeInitScript`), so
 * there is no light-flash before hydration; this component owns everything that
 * happens *after* it.
 */
export function ThemeApplier() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const syncResolved = useThemeStore((s) => s.syncResolved);

  useEffect(() => {
    syncResolved();
  }, [theme, syncResolved]);

  // Follow the OS only while the user has not overridden it.
  useEffect(() => {
    if (theme !== 'system') return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => syncResolved();
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, [theme, syncResolved]);

  // Another tab (or another window of the installed PWA) changing the
  // preference must not leave this one on the old theme.
  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key !== THEME_KEY) return;
      const next = event.newValue;
      if (next === 'light' || next === 'dark' || next === 'system') setTheme(next);
    };
    window.addEventListener('storage', onStorage);
    return () => window.removeEventListener('storage', onStorage);
  }, [setTheme]);

  return null;
}

/**
 * Inline bootstrap: resolves and applies the stored theme before first paint,
 * so a dark-mode user never sees a white flash. Rendered as a <script> in the
 * root layout (a server component), so it must stay a plain string with no
 * imports — it is the one deliberate duplicate of `applyTheme`.
 *
 * `ritme_theme` here is `THEME_KEY`; keep the two in step.
 */
export const themeInitScript = `(function(){try{
var p=localStorage.getItem('ritme_theme');
if(p!=='light'&&p!=='dark')p='system';
var d=p==='dark'||(p==='system'&&matchMedia('(prefers-color-scheme: dark)').matches);
var r=document.documentElement;
r.dataset.theme=d?'dark':'light';
var c=getComputedStyle(r).getPropertyValue('--page').trim();
if(c){var m=document.querySelectorAll('meta[name="theme-color"]');for(var i=0;i<m.length;i++)m[i].content=c;}
}catch(e){}})();`;
