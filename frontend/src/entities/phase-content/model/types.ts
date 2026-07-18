// Domain shape for the DB-driven "Phase Details" educational content. The
// backend keys content by the fine-grained cycle sub-phase (cycle_view.subphase)
// and returns nine prose sections. Sections with no copy for the active locale
// are omitted by the API, so the map is intentionally partial (§ graceful hide).

/**
 * The nine content sections, in the order they should render on screen. Adding
 * a section later means appending one key here plus one i18n label — no screen
 * rewrite (task requirement: scalable structure).
 */
export const PHASE_SECTION_KEYS = [
  'symptom_prediction',
  'vaginal_discharge',
  'fertility',
  'hormonal_changes',
  'sex_tips',
  'nutrition',
  'exercise',
  'skin_care',
  'sleep',
] as const;

export type PhaseSectionKey = (typeof PHASE_SECTION_KEYS)[number];

export interface PhaseContent {
  /** The sub-phase key this content belongs to (CycleSubphase value). */
  phase: string;
  /** Localized human label for the phase, e.g. «باروری بالا». */
  phaseLabel: string;
  /** Present sections only; missing/empty ones are absent. */
  sections: Partial<Record<PhaseSectionKey, string>>;
}
