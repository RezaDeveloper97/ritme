import type { LabInterpretation, LabMarker, MarkerReference, OutOfRangeItem } from './types';

/*
 * Per-value confirmation and the JSON summary (T-M10-20, D-171). Pure helpers so
 * the verify and result screens stay presentational.
 */

/** Ids of the values read with low confidence that still wait for the user's tap, in sheet order. */
export function markersToConfirm(markers: readonly LabMarker[]): number[] {
  return markers.filter((m) => m.needsConfirmation).map((m) => m.id);
}

/** The out-of-range markers the summary explains (the AI's plain sentence is present), in the rules' order. */
export function explainedOutOfRange(interp: LabInterpretation | null | undefined): OutOfRangeItem[] {
  if (!interp) return [];
  return interp.outOfRange.filter((o) => o.explanation !== null);
}

/** Whether a range was printed on the sheet or is the general one (the sheet printed none); null without a range. */
export function referenceKind(ref: MarkerReference): 'sheet' | 'general' | null {
  if (ref.source === 'sheet') return 'sheet';
  if (ref.source === 'typical') return 'general';
  return null;
}

/** «نزدیک مرز» hint: just outside the range but not shown as a borderline state (lab.borderline off). */
export function showsNearEdge(m: Pick<LabMarker, 'nearEdge' | 'state'>): boolean {
  return m.nearEdge && (m.state === 'low' || m.state === 'high');
}
