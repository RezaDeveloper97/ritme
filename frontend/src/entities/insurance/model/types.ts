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

/** Place of a policy in her history: exactly one is current; a past one and its claims are read-only. */
export type PolicyStatus = 'current' | 'upcoming' | 'past';

export interface InsurancePolicy {
  id: number;
  status: PolicyStatus;
  /** Past policy: it and its claims cannot change. */
  readOnly: boolean;
  /** The newest policy — the only one that can be renewed. */
  canRenew: boolean;
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

/** One row of GET /insurance/policies (the policy switcher). */
export interface InsurancePolicySummary {
  id: number;
  insurerName: string;
  planName: string | null;
  validFrom: string | null;
  validTo: string | null;
  status: PolicyStatus;
  readOnly: boolean;
  canRenew: boolean;
  claimsCount: number;
}

export interface InsurancePolicyHistory {
  policies: InsurancePolicySummary[];
  currentId: number | null;
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
  /** The policy the list belongs to (default: the current one). */
  policy: { id: number; readOnly: boolean } | null;
  claims: InsuranceClaimRow[];
  needsAction: number;
  /** Policy-period totals (toman) from the server. */
  totals: { requested: number; paid: number; count: number };
}

export const CLAIM_COST_TYPES = ['paraclinic', 'visit', 'drug', 'inpatient', 'delivery', 'dental', 'optical', 'infertility', 'other'] as const;
export type ClaimCostType = (typeof CLAIM_COST_TYPES)[number];
export const CLAIM_DOC_LABELS = ['prescription', 'invoice', 'report', 'other'] as const;
export type ClaimDocLabel = (typeof CLAIM_DOC_LABELS)[number];

export interface ClaimPayable {
  /** false = the policy has no deductible for this cost type, so no estimate. */
  known: boolean;
  invoice: number;
  overTariff: number;
  deductiblePct: number;
  deductible: number;
  payable: number;
  userShare: number;
  capped: boolean;
}

export interface ClaimEvent {
  id: number;
  status: ClaimStatus;
  occurredOn: string;
  deadlineOn: string | null;
  amount: number | null;
  note: string;
}

export interface ClaimDocument {
  documentId: number;
  label: ClaimDocLabel | null;
  kind: string;
  title: string | null;
  documentDate: string | null;
}

export interface InsuranceClaimDetail extends InsuranceClaimRow {
  policyId: number | null;
  /** The claim belongs to a past policy: no steps, documents or delete. */
  readOnly: boolean;
  centreName: string | null;
  overTariff: number;
  deadlineOn: string | null;
  note: string;
  payable: ClaimPayable;
  events: ClaimEvent[];
  documents: ClaimDocument[];
}

/** POST / PUT /insurance/claims body. */
export interface ClaimPayload {
  member_id?: number | null;
  cost_type: ClaimCostType;
  title?: string | null;
  centre_name?: string | null;
  service_date: string;
  amount: number;
  over_tariff?: number | null;
  note?: string | null;
  documents?: { id: number; label?: ClaimDocLabel | null }[];
}

/** POST /insurance/claims/{id}/events body. */
export interface ClaimEventPayload {
  status: ClaimStatus;
  occurred_on: string;
  deadline_on?: string | null;
  amount?: number | null;
  note?: string | null;
}

/** Contracted centre types (backend `CentreTypes`, CB-INS-06). */
export const CENTRE_TYPES = ['gyn', 'lab', 'imaging', 'hospital', 'pharmacy', 'pediatric', 'other'] as const;
export type CentreType = (typeof CENTRE_TYPES)[number];

/** One row of GET /insurance/centres. `isDirect`: direct contract — she pays only the deductible, no claim. */
export interface InsuranceCentre {
  id: number;
  name: string;
  /** Server type; unknown codes fall back to `other` in the copy. */
  type: string;
  services: string | null;
  city: string | null;
  address: string | null;
  phone: string | null;
  lat: number | null;
  lng: number | null;
  isDirect: boolean;
  insurerName: string | null;
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
