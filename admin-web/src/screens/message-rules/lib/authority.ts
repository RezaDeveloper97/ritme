/** Who may do what to a rule — mirrors the API (SPEC §5.8): the UI hides what the API would refuse with 403 / 409. */
export interface Authority {
  /** Nothing is editable (clinical kind without the clinical capability, or a locked rule for a non-clinical admin). */
  readOnly: boolean;
  /** A locked rule: only its copy is editable, and only with the clinical capability. */
  copyOnly: boolean;
  canDelete: boolean;
  canSwitchOff: boolean;
  /** Publish (switch on) — a locked rule may be switched on again, never off. */
  canPublish: boolean;
  /** Whether the admin may author this kind (the kind select disables the rest). */
  canAuthorKind: (kind: string) => boolean;
  canSave: boolean;
}

export function authorityFor(
  rule: { kind: string; locked: boolean } | null,
  canClinical: boolean,
  approvalKinds: readonly string[],
): Authority {
  const authorKind = (kind: string) => canClinical || !approvalKinds.includes(kind);
  const medicalKind = rule ? approvalKinds.includes(rule.kind) : false;
  const locked = rule?.locked ?? false;
  const readOnly = rule ? (medicalKind && !canClinical) || (locked && !canClinical) : false;
  return {
    readOnly,
    copyOnly: locked && canClinical,
    canDelete: !!rule && !locked && !readOnly && (rule.kind !== 'safety' || canClinical),
    canSwitchOff: !!rule && !locked && !readOnly && (rule.kind !== 'safety' || canClinical),
    canPublish: !!rule && !readOnly && (rule.kind !== 'safety' || canClinical),
    canAuthorKind: authorKind,
    canSave: !readOnly,
  };
}
