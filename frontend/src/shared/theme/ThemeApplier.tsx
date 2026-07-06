'use client';

import { useEffect } from 'react';

import { applyTheme, useThemeStore } from './store';

/**
 * Keeps <html data-theme> in sync with the stored preference, including OS
 * theme changes while 'system' is selected. Mounted once in the app layout.
 * The initial paint is handled by the inline script (see themeInitScript) so
 * there is no light-flash before hydration.
 */
export function ThemeApplier() {
  const theme = useThemeStore((s) => s.theme);

  useEffect(() => {
    applyTheme(theme);
    if (theme !== 'system') return;
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => applyTheme('system');
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, [theme]);

  return null;
}

/**
 * Inline bootstrap: applies the stored theme before first paint. Rendered as a
 * <script> in the root layout (server component), so it must stay a plain
 * string with no imports.
 */
export const themeInitScript = `(function(){try{var t=localStorage.getItem('ritme_theme');var d=t==='dark'||(t!=='light'&&matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.dataset.theme=d?'dark':'light';}catch(e){}})();`;
