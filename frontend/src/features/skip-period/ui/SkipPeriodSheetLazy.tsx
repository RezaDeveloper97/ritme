'use client';

import dynamic from 'next/dynamic';

/** `SkipPeriodSheet` behind `next/dynamic` so the barrel's hooks stay light (T-M12-27b). Same props. */
export const SkipPeriodSheet = dynamic(() => import('./SkipPeriodSheet').then((m) => m.SkipPeriodSheet), {
  ssr: false,
  loading: () => null,
});
