/** Insurance self-tracking (CB-INS-03). Shapes: backend-go/api/openapi.yaml `InsurancePolicyEnvelope`, `InsuranceClaimList`. */

export const COVERAGE_CATEGORIES = [
  'inpatient',
  'delivery',
  'paraclinic',
  'visit_drug',
  'infertility',
  'dental',
  'optical',
  'other',
] as const;
export type CoverageCategory = (typeof COVERAGE_CATEGORIES)[number];

export type CoverageStatus = 'covered' | 'conditional' | 'not_covered';
export type MemberKind = 'self' | 'spouse' | 'child' | 'other';

export interface PolicyMember {
  id: number;
  kind: MemberKind;
  childId: number | null;
  name: string;
}

export interface PolicyCoverage {
  /** Server category; unknown codes fall back to `other` in the copy. */
  category: string;
  status: CoverageStatus;
  /** Toman. */
  capAmount: number | null;
  deductiblePct: number | null;
  waitingDays: number | null;
  note: string | null;
  used: number;
  remaining: number | null;
  waitingEndsOn: string | null;
  waitingOver: boolean;
}

export interface InsurancePolicy {
  id: number;
  /** Catalog insurer, or null when she typed the name. */
  insurerId: number | null;
  insurerName: string;
  planName: string | null;
  numberLast4: string | null;
  validFrom: string | null;
  validTo: string | null;
  claimDeadlineDays: number | null;
  ibanLast4: string | null;
  notCoveredNote: string | null;
  members: PolicyMember[];
  coverages: PolicyCoverage[];
}

export interface InsurancePolicyState {
  policy: InsurancePolicy | null;
  /** Claims waiting on her (missing documents). */
  needsAction: number;
}

export interface InsurerOption {
  id: number;
  name: string;
}

export type ClaimStatus = 'submitted' | 'reviewing' | 'docs_requested' | 'paid' | 'rejected' | 'appealed';

export interface InsuranceClaimRow {
  id: number;
  title: string | null;
  costType: string;
  memberName: string | null;
  serviceDate: string;
  amount: number;
  paidAmount: number | null;
  status: ClaimStatus;
  needsAction: boolean;
}

export interface InsuranceClaimsSummary {
  claims: InsuranceClaimRow[];
  needsAction: number;
}

/** PUT /insurance/policy body. The number is sent only when typed (the server keeps the last four). */
export interface PolicyPayload {
  insurer_id?: number | null;
  insurer_name?: string | null;
  plan_name?: string | null;
  policy_number?: string | null;
  valid_from?: string | null;
  valid_to?: string | null;
  claim_deadline_days?: number | null;
  members: { id?: number; kind: MemberKind; name: string }[];
  coverages: {
    category: CoverageCategory;
    status: CoverageStatus;
    cap_amount: number | null;
    deductible_pct: number | null;
    waiting_days: number | null;
  }[];
}
