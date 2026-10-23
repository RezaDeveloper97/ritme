/**
 * «شیر می‌دهی؟» switch on the postpartum home (engine v6 I-04, T-M9-28). The profile answer is
 * true / false / null (not asked yet); the switch shows on only for true, and its caption says
 * which of the three it is. While a save runs, the value being saved wins (optimistic).
 */
export type BreastfeedingCaption = 'notAsked' | 'yes' | 'no';

export interface BreastfeedingView {
  checked: boolean;
  caption: BreastfeedingCaption;
}

export function breastfeedingView(stored: boolean | null, saving?: boolean | null): BreastfeedingView {
  const value = saving === undefined ? stored : saving;
  if (value === null) return { checked: false, caption: 'notAsked' };
  return { checked: value, caption: value ? 'yes' : 'no' };
}

/** The answer a switch flip saves: on = yes, off = no (an answered «no», never back to «not asked»). */
export function breastfeedingAnswer(next: boolean): boolean {
  return next;
}
