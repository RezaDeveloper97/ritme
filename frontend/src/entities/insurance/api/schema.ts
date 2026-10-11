import { z } from 'zod';

import type {
  ClaimStatus,
  CoverageStatus,
  InsuranceClaimsSummary,
  InsuranceCentre,
  InsurancePolicyState,
  InsurerOption,
  MemberKind,
} from '../model/types';

/** Boundary parsers (CLAUDE.md §10). Shapes: openapi.yaml `InsurancePolicyEnvelope`, `InsuranceInsurerList`, `InsuranceClaimList`. */

const nstr = z
  .string()
  .nullish()
  .transform((v) => v ?? null);
const nint = z
  .number()
  .nullish()
  .transform((v) => v ?? null);

const memberSchema = z.object({
  id: z.number(),
  kind: z.enum(['self', 'spouse', 'child', 'other']).catch('other'),
  child_id: nint,
  name: z.string(),
});

const coverageSchema = z.object({
  category: z.string(),
  status: z.enum(['covered', 'conditional', 'not_covered']).catch('covered'),
  cap_amount: nint,
  deductible_pct: nint,
  waiting_days: nint,
  note: nstr,
  used: z.number().catch(0),
  remaining: nint,
  waiting_ends_on: nstr,
  waiting_over: z.boolean().catch(false),
});

const policySchema = z.object({
  id: z.number(),
  insurer: z.object({ id: z.number() }).nullish(),
  insurer_name: z.string(),
  plan_name: nstr,
  number_last4: nstr,
  valid_from: nstr,
  valid_to: nstr,
  claim_deadline_days: nint,
  iban_last4: nstr,
  not_covered_note: nstr,
  members: z.array(memberSchema),
  coverages: z.array(coverageSchema),
});

export const policyStateSchema = z
  .object({ policy: policySchema.nullable(), needs_action: z.number().catch(0) })
  .transform(
    (v): InsurancePolicyState => ({
      needsAction: v.needs_action,
      policy: v.policy && {
        id: v.policy.id,
        insurerId: v.policy.insurer?.id ?? null,
        insurerName: v.policy.insurer_name,
        planName: v.policy.plan_name,
        numberLast4: v.policy.number_last4,
        validFrom: v.policy.valid_from,
        validTo: v.policy.valid_to,
        claimDeadlineDays: v.policy.claim_deadline_days,
        ibanLast4: v.policy.iban_last4,
        notCoveredNote: v.policy.not_covered_note,
        members: v.policy.members.map((m) => ({ id: m.id, kind: m.kind as MemberKind, childId: m.child_id, name: m.name })),
        coverages: v.policy.coverages.map((c) => ({
          category: c.category,
          status: c.status as CoverageStatus,
          capAmount: c.cap_amount,
          deductiblePct: c.deductible_pct,
          waitingDays: c.waiting_days,
          note: c.note,
          used: c.used,
          remaining: c.remaining,
          waitingEndsOn: c.waiting_ends_on,
          waitingOver: c.waiting_over,
        })),
      },
    }),
  );

const ncoord = z
  .union([z.number(), z.string()])
  .nullish()
  .transform((v) => {
    if (v === null || v === undefined || v === '') return null;
    const n = typeof v === 'number' ? v : Number.parseFloat(v);
    return Number.isFinite(n) ? n : null;
  });

/** GET /insurance/centres (CB-INS-06): `lat`/`lng` arrive as decimal strings. */
export const centresSchema = z
  .object({
    centres: z.array(
      z.object({
        id: z.number(),
        name: z.string(),
        type: z.string(),
        services: nstr,
        city: nstr,
        address: nstr,
        phone: nstr,
        lat: ncoord,
        lng: ncoord,
        is_direct: z.boolean().catch(false),
        insurer_name: nstr,
      }),
    ),
  })
  .transform(({ centres }) =>
    centres.map(
      (c): InsuranceCentre => ({
        id: c.id,
        name: c.name,
        type: c.type,
        services: c.services,
        city: c.city,
        address: c.address,
        phone: c.phone,
        lat: c.lat,
        lng: c.lng,
        isDirect: c.is_direct,
        insurerName: c.insurer_name,
      }),
    ),
  );

export const insurersSchema = z
  .object({ insurers: z.array(z.object({ id: z.number(), name: z.string() })) })
  .transform((v): InsurerOption[] => v.insurers.map((i) => ({ id: i.id, name: i.name })));

export const claimsSchema = z
  .object({
    claims: z.array(
      z.object({
        id: z.number(),
        title: nstr,
        cost_type: z.string(),
        member: z.object({ name: z.string() }).nullish(),
        service_date: z.string(),
        amount: z.number(),
        paid_amount: nint,
        status: z.enum(['submitted', 'reviewing', 'docs_requested', 'paid', 'rejected', 'appealed']).catch('submitted'),
        needs_action: z.boolean().catch(false),
      }),
    ),
  })
  .transform(
    (v): InsuranceClaimsSummary => ({
      needsAction: v.claims.filter((c) => c.needs_action).length,
      claims: v.claims.map((c) => ({
        id: c.id,
        title: c.title,
        costType: c.cost_type,
        memberName: c.member?.name ?? null,
        serviceDate: c.service_date,
        amount: c.amount,
        paidAmount: c.paid_amount,
        status: c.status as ClaimStatus,
        needsAction: c.needs_action,
      })),
    }),
  );
