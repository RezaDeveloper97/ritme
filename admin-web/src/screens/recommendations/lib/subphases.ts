/**
 * The sub-phases a recommendation can target once `phase` is chosen
 * (recommendations/subphase-picker.blade.php): no phase → every sub-phase;
 * otherwise only those inside the phase. A phase with fewer than two
 * sub-phases can't be narrowed, so the picker is hidden (`narrowable`).
 */
export function subphasesFor<O extends { value: string }>(
  subphases: readonly O[],
  phaseOf: Readonly<Record<string, string>>,
  phase: string,
): { visible: O[]; narrowable: boolean } {
  const visible = phase ? subphases.filter((s) => phaseOf[s.value] === phase) : [...subphases];
  return { visible, narrowable: visible.length >= 2 };
}

/** Drop selections that became impossible, so what the admin sees is what is sent. */
export function keepReachable(selected: readonly string[], visible: ReadonlyArray<{ value: string }>, narrowable: boolean): string[] {
  if (!narrowable) return [];
  const allowed = new Set(visible.map((v) => v.value));
  return selected.filter((s) => allowed.has(s));
}
