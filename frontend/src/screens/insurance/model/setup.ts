import {
  type CoverageCategory,
  type InsurancePolicy,
  type MemberKind,
  type PolicyPayload,
  TOMAN_MILLION,
} from '@/entities/insurance';

/**
 * Setup / edit form for the self-tracked policy (CB-INS-03). Pure draft logic: the form keeps what she typed, this
 * turns it into the `PUT /insurance/policy` body. Financial data (§11): never log a draft or a payload.
 */

/** Categories the form asks a cap for (`other` is left to the server). */
export const FORM_CATEGORIES: readonly CoverageCategory[] = [
  'inpatient',
  'delivery',
  'paraclinic',
  'visit_drug',
  'infertility',
  'dental',
  'optical',
];

export const MEMBER_KINDS: readonly MemberKind[] = ['self', 'spouse', 'child', 'other'];
export const MAX_MEMBERS = 10;
const DAYS_PER_MONTH = 30;

export interface DraftMember {
  id?: number;
  kind: MemberKind;
  name: string;
}

export interface DraftCoverage {
  /** Million toman. */
  capMillions: number | undefined;
  deductiblePct: number | undefined;
  waitingMonths: number | undefined;
}

export interface PolicyDraft {
  insurerId: number | null;
  /** True when she types an insurer instead of picking one. */
  custom: boolean;
  insurerName: string;
  planName: string;
  policyNumber: string;
  validFrom: string | null;
  validTo: string | null;
  claimDeadlineDays: number | undefined;
  members: DraftMember[];
  coverages: Record<string, DraftCoverage>;
}

const emptyCoverage = (): DraftCoverage => ({ capMillions: undefined, deductiblePct: undefined, waitingMonths: undefined });

export function emptyDraft(selfName: string): PolicyDraft {
  return {
    insurerId: null,
    custom: false,
    insurerName: '',
    planName: '',
    policyNumber: '',
    validFrom: null,
    validTo: null,
    claimDeadlineDays: undefined,
    members: [{ kind: 'self', name: selfName }],
    coverages: Object.fromEntries(FORM_CATEGORIES.map((c) => [c, emptyCoverage()])),
  };
}

/** The draft of an existing policy. The number stays empty: only its last four are stored, and empty keeps it. */
export function draftFromPolicy(policy: InsurancePolicy, selfName: string): PolicyDraft {
  const draft = emptyDraft(selfName);
  draft.insurerId = policy.insurerId;
  draft.custom = policy.insurerId === null;
  draft.insurerName = policy.insurerId === null ? policy.insurerName : '';
  draft.planName = policy.planName ?? '';
  draft.validFrom = policy.validFrom;
  draft.validTo = policy.validTo;
  draft.claimDeadlineDays = policy.claimDeadlineDays ?? undefined;
  if (policy.members.length > 0) {
    draft.members = policy.members.map((m) => ({ id: m.id, kind: m.kind, name: m.name }));
  }
  for (const cov of policy.coverages) {
    if (!(cov.category in draft.coverages)) continue;
    draft.coverages[cov.category] = {
      capMillions: cov.capAmount !== null ? Math.round((cov.capAmount / TOMAN_MILLION) * 10) / 10 : undefined,
      deductiblePct: cov.deductiblePct ?? undefined,
      waitingMonths: cov.waitingDays ? Math.round(cov.waitingDays / DAYS_PER_MONTH) : undefined,
    };
  }
  return draft;
}

export type DraftProblem = 'insurer' | 'dates' | 'member';

export function draftProblems(draft: PolicyDraft): Partial<Record<DraftProblem, true>> {
  const problems: Partial<Record<DraftProblem, true>> = {};
  if (draft.insurerId === null && draft.insurerName.trim() === '') problems.insurer = true;
  if (draft.validFrom && draft.validTo && draft.validTo < draft.validFrom) problems.dates = true;
  if (draft.members.some((m) => m.name.trim() === '')) problems.member = true;
  return problems;
}

/** `PUT /insurance/policy` body from a valid draft. */
export function draftToPayload(draft: PolicyDraft): PolicyPayload {
  const coverages: PolicyPayload['coverages'] = [];
  for (const category of FORM_CATEGORIES) {
    const c = draft.coverages[category];
    if (!c || (c.capMillions === undefined && c.deductiblePct === undefined && c.waitingMonths === undefined)) continue;
    coverages.push({
      category,
      status: 'covered',
      cap_amount: c.capMillions === undefined ? null : Math.round(c.capMillions * TOMAN_MILLION),
      deductible_pct: c.deductiblePct ?? null,
      waiting_days: c.waitingMonths === undefined ? null : c.waitingMonths * DAYS_PER_MONTH,
    });
  }
  const number = draft.policyNumber.trim();
  return {
    insurer_id: draft.custom ? null : draft.insurerId,
    insurer_name: draft.custom ? draft.insurerName.trim() : null,
    plan_name: draft.planName.trim() || null,
    // Omitted when empty: the server keeps the stored last four.
    ...(number ? { policy_number: number } : {}),
    valid_from: draft.validFrom,
    valid_to: draft.validTo,
    claim_deadline_days: draft.claimDeadlineDays ?? null,
    members: draft.members.map((m) => ({ ...(m.id ? { id: m.id } : {}), kind: m.kind, name: m.name.trim() })),
    coverages,
  };
}
