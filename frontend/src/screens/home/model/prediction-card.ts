import type { CycleRegularity, CycleTodayUi, OvulationSource } from '@/entities/cycle';

/** What the «پیش‌بینی‌ها» card shows when the engine sent `today_ui.prediction_card` (E-17). */
export interface EnginePredictionRows {
  period: { start: string; end: string } | null;
  pms: { start: string; end: string } | null;
  /** Null = no display window (suppression, or a long period swallowed it): the row is hidden. */
  fertile: { start: string; end: string } | null;
  ovulation: { date: string; source: OvulationSource } | null;
  cycleLength: number | null;
  regularity: CycleRegularity | null;
}

const range = (r: { start: string | null; end: string | null } | null | undefined): { start: string; end: string } | null =>
  r?.start ? { start: r.start, end: r.end ?? r.start } : null;

/** Null without a v6 card, so the Laravel path keeps its own derivation untouched. */
export function enginePredictionRows(ui: CycleTodayUi | null | undefined): EnginePredictionRows | null {
  const card = ui?.predictionCard;
  if (!card) return null;
  return {
    period: range(card.period),
    pms: range(card.pms),
    fertile: range(card.fertile),
    ovulation: card.ovulation?.date ? { date: card.ovulation.date, source: card.ovulation.source } : null,
    cycleLength: card.cycleLength,
    regularity: card.cycleRegularity,
  };
}
