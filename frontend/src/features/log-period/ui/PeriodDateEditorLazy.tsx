'use client';

import dynamic from 'next/dynamic';

/**
 * `PeriodDateEditor` behind `next/dynamic`: importing the feature barrel (for its hooks) must not pull the editor
 * and its CSS into the importer's first load (T-M12-27b). Same props as the real component.
 */
export const PeriodDateEditor = dynamic(() => import('./PeriodDateEditor').then((m) => m.PeriodDateEditor), {
  ssr: false,
  loading: () => null,
});
