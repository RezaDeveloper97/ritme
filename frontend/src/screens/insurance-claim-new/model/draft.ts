import type { ClaimCostType, ClaimDocLabel, ClaimPayload } from '@/entities/insurance';

/** The three document slots of nbl_Ins_ClaimNew; anything else she attaches is `other`. */
export const DOC_SLOTS = ['prescription', 'invoice', 'report'] as const satisfies readonly ClaimDocLabel[];

export const MAX_AMOUNT = 999_999_999;
export const MAX_CENTRE = 120;

export interface DraftDoc {
  id: number;
  label: ClaimDocLabel;
  title: string | null;
  kind: string;
}

export interface ClaimDraft {
  memberId: number | null;
  costType: ClaimCostType;
  centreName: string;
  /** `Y-m-d`. */
  serviceDate: string;
  amount: number | undefined;
  docs: DraftDoc[];
}

export type DraftProblem = 'amount' | 'date';

export function draftProblems(d: ClaimDraft): DraftProblem[] {
  const out: DraftProblem[] = [];
  if (!d.amount || d.amount < 1) out.push('amount');
  if (!d.serviceDate) out.push('date');
  return out;
}

/** Attach a record document under a label; the same document is never listed twice (it moves to the new label). */
export function withDoc(docs: readonly DraftDoc[], doc: DraftDoc): DraftDoc[] {
  const rest = docs.filter((d) => d.id !== doc.id);
  // One document per named slot: a new pick replaces the previous one of that slot.
  return doc.label === 'other' ? [...rest, doc] : [...rest.filter((d) => d.label !== doc.label), doc];
}

export function draftToPayload(d: ClaimDraft): ClaimPayload {
  const centre = d.centreName.trim();
  return {
    member_id: d.memberId,
    cost_type: d.costType,
    centre_name: centre === '' ? null : centre,
    service_date: d.serviceDate,
    amount: d.amount ?? 0,
    documents: d.docs.map((x) => ({ id: x.id, label: x.label })),
  };
}
