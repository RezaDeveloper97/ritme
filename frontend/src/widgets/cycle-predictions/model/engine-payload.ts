import type { CycleView } from '@/entities/cycle';

/** A v6 payload carries `today_ui` (today) or `limits` (any date). */
export function isEnginePayload(view: Pick<CycleView, 'todayUi' | 'limits'> | null | undefined): boolean {
  return view != null && (view.todayUi != null || view.limits != null);
}
