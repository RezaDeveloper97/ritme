import type { CycleTodayUi } from '@/entities/cycle';

export interface SkipOfferInput {
  isToday: boolean;
  /** `cycle_view.today_ui`; null on a Laravel payload, which has no skip endpoint. */
  todayUi: CycleTodayUi | null | undefined;
  /** `anchors.predicted_next_period_start` (Y-m-d). */
  predictedNext: string | null | undefined;
  /** The user's local today (Y-m-d). */
  todayIso: string;
  inPeriod: boolean;
  /** A trying-to-conceive ring shows the fertility log instead of a period prompt. */
  ttcRing?: boolean;
}

/**
 * «پریود نداشتم» is offered only on a v6 payload, today, once the expected day has come and no period is open,
 * either as the overdue banner or as the ring's start prompt. Under `ask_last_period` the prediction is stale,
 * so the date question comes first. The server still has the last word (422).
 */
export function canOfferSkip(i: SkipOfferInput): boolean {
  const ui = i.todayUi;
  if (!ui || !i.isToday || i.inPeriod || i.ttcRing || !i.predictedNext) return false;
  if (i.todayIso < i.predictedNext) return false;
  if (ui.banner === 'ask_last_period') return false;
  return ui.banner === 'overdue' || ui.ringAction === 'start';
}
