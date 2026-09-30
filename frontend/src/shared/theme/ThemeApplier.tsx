'use client';

import { useEffect } from 'react';

import {
  applyMotion,
  applyTextScale,
  clampTextScaleIndex,
  isMotionPreference,
  MOTION_KEY,
  TEXT_SCALE_KEY,
  useDisplayStore,
} from './display';
import { isThemePreference, THEME_KEY, useThemeStore, watchSystemTheme } from './store';

/**
 * Keeps <html data-theme> in sync with the preference after hydration:
 * - the resolved theme from the store,
 * - the OS appearance, live, while the preference is `system`,
 * - the same choice made in another tab / window of the installed PWA,
 * and the display preferences (text size, reduced motion — `display.ts`) the
 * same way. Mounted once in the app layout.
 *
 * The initial paint is handled by the inline script (see `themeInitScript`), so
 * there is no flash before hydration; this component owns everything after it.
 */
export function ThemeApplier() {
  const theme = useThemeStore((s) => s.theme);
  const setPreference = useThemeStore((s) => s.setPreference);
  const syncSystem = useThemeStore((s) => s.syncSystem);
  const textScale = useDisplayStore((s) => s.textScale);
  const motion = useDisplayStore((s) => s.motion);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  // A `system` user whose phone switches to dark at sunset switches with it.
  // The store ignores the event for an explicit light/dark preference.
  useEffect(() => watchSystemTheme(syncSystem), [syncSystem]);

  useEffect(() => applyTextScale(textScale), [textScale]);
  useEffect(() => applyMotion(motion), [motion]);

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (event.key === TEXT_SCALE_KEY) {
        useDisplayStore.getState().setTextScale(clampTextScaleIndex(event.newValue));
        return;
      }
      if (event.key === MOTION_KEY) {
        if (isMotionPreference(event.newValue)) useDisplayStore.getState().setMotion(event.newValue);
        return;
      }
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
 * It also applies the display preferences (`display.ts`): `ritme_text_scale`
 * (index into TEXT_SCALES — keep the array in step) and `ritme_motion`.
 */
export const themeInitScript = `(function(){try{
var s=null;try{s=localStorage.getItem('ritme_theme');}catch(e){}
var d=s==='dark'||(s!=='light'&&!!window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches);
var r=document.documentElement;
r.dataset.theme=d?'dark':'light';
var ts=null,mo=null;try{ts=localStorage.getItem('ritme_text_scale');mo=localStorage.getItem('ritme_motion');}catch(e){}
var k=[0.9,0.95,1,1.1,1.2][parseInt(ts,10)];if(k)r.style.setProperty('--text-scale',String(k));
if(mo==='reduce')r.dataset.motion='reduce';
var c=getComputedStyle(r).getPropertyValue('--page').trim();
if(c){var m=document.querySelectorAll('meta[name="theme-color"]');for(var i=0;i<m.length;i++)m[i].content=c;}
}catch(e){}})();`;
