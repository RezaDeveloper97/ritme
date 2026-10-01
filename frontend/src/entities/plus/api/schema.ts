import { z } from 'zod';

import type {
  PlusCatalog,
  PlusCheckout,
  PlusDiscountSource,
  PlusEntitlement,
  PlusInvoice,
  PlusOfferPlan,
  PlusPlan,
  PlusPlanRef,
  PlusQuote,
  PlusRestore,
  PlusStatus,
  PlusTier,
  PlusTrialOffer,
  PlusTrialSheet,
  PlusTrialUsage,
  PlusVerification,
} from '../model/types';

/*
 * `/plus/*` payloads (backend-go/internal/plus/view.go), validated at the API
 * boundary (CLAUDE.md §10) and mapped to camelCase. Fields the backend adds
 * later (B-N2-06 trial offer, banner, usage) are optional so an older or newer
 * server never breaks the screens.
 */

const money = z.number().int().nonnegative();
const iso = z.string();

const planShape = z.object({
  id: z.number(),
  code: z.string(),
  title: z.string(),
  badge: z.string().nullable().optional(),
  duration_months: z.number(),
  price: money,
  monthly_price: money.optional(),
  savings_percent: z.number().optional(),
  is_highlighted: z.boolean().optional(),
});

function toPlan(p: z.infer<typeof planShape>): PlusPlan {
  return {
    id: p.id,
    code: p.code,
    title: p.title,
    badge: p.badge ?? null,
    durationMonths: p.duration_months,
    price: p.price,
    monthlyPrice: p.monthly_price ?? Math.round(p.price / Math.max(1, p.duration_months)),
    savingsPercent: p.savings_percent ?? 0,
    isHighlighted: p.is_highlighted ?? false,
  };
}

export const planSchema = planShape.transform(toPlan);

/** A plan card with its trial-offer prices (`OfferPlanJSON`, B-N2-06). */
export const offerPlanSchema = planShape
  .extend({ offer_price: money.nullable().default(null), offer_monthly_price: money.nullable().default(null) })
  .transform(
    (p): PlusOfferPlan => ({ ...toPlan(p), offerPrice: p.offer_price, offerMonthlyPrice: p.offer_monthly_price }),
  );

/** `TrialOfferJSON` — the home banner, `/plus/status.trial_offer` and the trial sheet's offer (null = none). */
export const trialOfferSchema = z
  .object({
    percent: z.number(),
    ends_at: iso,
    seconds_left: z.number().nonnegative(),
    days_left: z.number().default(0),
    currency: z.string().default('IRR'),
    plan: offerPlanSchema,
  })
  .transform(
    (o): PlusTrialOffer => ({
      percent: o.percent,
      endsAt: o.ends_at,
      secondsLeft: o.seconds_left,
      daysLeft: o.days_left,
      currency: o.currency,
      plan: o.plan,
    }),
  )
  .nullable()
  .default(null);

export const catalogSchema = z
  .object({
    currency: z.string().default('IRR'),
    vat_rate_bps: z.number().default(0),
    trial_days: z.number().default(0),
    plans: z.array(planSchema),
  })
  .transform(
    (c): PlusCatalog => ({ currency: c.currency, vatRateBps: c.vat_rate_bps, trialDays: c.trial_days, plans: c.plans }),
  );

const planRefSchema = z
  .object({ id: z.number(), code: z.string(), title: z.string(), duration_months: z.number() })
  .transform((p): PlusPlanRef => ({ id: p.id, code: p.code, title: p.title, durationMonths: p.duration_months }))
  .nullable()
  .default(null);

const entitlementSchema = z
  .object({
    key: z.string(),
    allowed: z.boolean(),
    unlimited: z.boolean().default(false),
    limit: z.number().nullable().default(null),
    used: z.number().default(0),
    remaining: z.number().nullable().default(null),
  })
  .transform((e): PlusEntitlement => e);

const TIERS: readonly PlusTier[] = ['free', 'trial', 'plus'];

function tierOf(raw: string, isPlus: boolean): PlusTier {
  return (TIERS as readonly string[]).includes(raw) ? (raw as PlusTier) : isPlus ? 'plus' : 'free';
}

export const statusSchema = z
  .object({
    tier: z.string(),
    is_plus: z.boolean(),
    subscription: z
      .object({
        id: z.number(),
        plan: planRefSchema,
        status: z.string(),
        source: z.string().default(''),
        starts_at: iso,
        ends_at: iso,
        days_left: z.number(),
        auto_renew: z.boolean(),
        canceled_at: iso.nullable().default(null),
      })
      .nullable()
      .default(null),
    trial: z
      .object({ started_at: iso, ends_at: iso, is_active: z.boolean(), days_left: z.number() })
      .nullable()
      .default(null),
    trial_available: z.boolean().default(false),
    period_start: z.string().default(''),
    resets_at: z.string().default(''),
    entitlements: z.array(entitlementSchema).default([]),
    trial_offer: trialOfferSchema,
  })
  .transform(
    (s): PlusStatus => ({
      tier: tierOf(s.tier, s.is_plus),
      isPlus: s.is_plus,
      subscription: s.subscription
        ? {
            id: s.subscription.id,
            plan: s.subscription.plan,
            status: s.subscription.status,
            source: s.subscription.source,
            startsAt: s.subscription.starts_at,
            endsAt: s.subscription.ends_at,
            daysLeft: s.subscription.days_left,
            autoRenew: s.subscription.auto_renew,
            canceledAt: s.subscription.canceled_at,
          }
        : null,
      trial: s.trial
        ? {
            startedAt: s.trial.started_at,
            endsAt: s.trial.ends_at,
            isActive: s.trial.is_active,
            daysLeft: s.trial.days_left,
          }
        : null,
      trialAvailable: s.trial_available,
      periodStart: s.period_start,
      resetsAt: s.resets_at,
      entitlements: s.entitlements,
      trialOffer: s.trial_offer,
    }),
  );

function discountSource(raw: string | null | undefined, code: string | null, discount: number): PlusDiscountSource {
  if (raw === 'code' || raw === 'trial_offer') return raw;
  if (code) return 'code';
  return discount > 0 ? 'trial_offer' : null;
}

export const quoteSchema = z
  .object({
    plan: planRefSchema,
    currency: z.string().default('IRR'),
    subtotal: money,
    discount: money.default(0),
    discount_code: z.string().nullable().default(null),
    discount_source: z.string().nullable().optional(),
    vat_rate_bps: z.number().default(0),
    vat: money.default(0),
    total: money,
  })
  .transform(
    (q): PlusQuote => ({
      plan: q.plan,
      currency: q.currency,
      subtotal: q.subtotal,
      discount: q.discount,
      discountCode: q.discount_code,
      discountSource: discountSource(q.discount_source, q.discount_code, q.discount),
      vatRateBps: q.vat_rate_bps,
      vat: q.vat,
      total: q.total,
    }),
  );

export const invoiceSchema = z
  .object({
    reference: z.string(),
    status: z.string(),
    plan: planRefSchema,
    duration_months: z.number(),
    currency: z.string().default('IRR'),
    subtotal: money,
    discount: money.default(0),
    discount_code: z.string().nullable().default(null),
    vat_rate_bps: z.number().default(0),
    vat: money.default(0),
    total: money,
    gateway: z.string().nullable().default(null),
    created_at: iso.nullable().default(null),
    expires_at: iso.nullable().default(null),
    paid_at: iso.nullable().default(null),
    receipt: z
      .object({ ref_id: z.string(), card_pan: z.string().nullable().default(null), amount: money, paid_at: iso })
      .nullable()
      .default(null),
  })
  .transform(
    (i): PlusInvoice => ({
      reference: i.reference,
      status: i.status,
      plan: i.plan,
      durationMonths: i.duration_months,
      currency: i.currency,
      subtotal: i.subtotal,
      discount: i.discount,
      discountCode: i.discount_code,
      vatRateBps: i.vat_rate_bps,
      vat: i.vat,
      total: i.total,
      gateway: i.gateway,
      createdAt: i.created_at,
      expiresAt: i.expires_at,
      paidAt: i.paid_at,
      receipt: i.receipt
        ? { refId: i.receipt.ref_id, cardPan: i.receipt.card_pan, amount: i.receipt.amount, paidAt: i.receipt.paid_at }
        : null,
    }),
  );

export const historySchema = z.array(invoiceSchema);

export const checkoutSchema = z
  .object({
    invoice: invoiceSchema,
    payment: z
      .object({ gateway: z.string(), authority: z.string(), redirect_url: z.string().url() })
      .nullable()
      .default(null),
  })
  .transform(
    (c): PlusCheckout => ({
      invoice: c.invoice,
      payment: c.payment
        ? { gateway: c.payment.gateway, authority: c.payment.authority, redirectUrl: c.payment.redirect_url }
        : null,
    }),
  );

export const quoteEnvelopeSchema = z.object({ quote: quoteSchema }).transform((q): PlusQuote => q.quote);

export const verificationSchema = z
  .object({ invoice: invoiceSchema, already_verified: z.boolean().default(false), status: statusSchema })
  .transform((v): PlusVerification => ({ invoice: v.invoice, alreadyVerified: v.already_verified, status: v.status }));

export const restoreSchema = z
  .object({ restored: z.number().default(0), status: statusSchema })
  .transform((r): PlusRestore => r);

const trialUsageSchema = z
  .object({
    key: z.string(),
    used: z.number().default(0),
    unlimited: z.boolean().default(false),
    plus_limit: z.number().nullable().default(null),
  })
  .transform((u): PlusTrialUsage => ({ key: u.key, used: u.used, unlimited: u.unlimited, plusLimit: u.plus_limit }));

/** GET /plus/trial (`TrialSheetJSON`, B-N2-06). */
export const trialSheetSchema = z
  .object({
    tier: z.string(),
    trial: z
      .object({
        started_at: iso,
        ends_at: iso,
        is_active: z.boolean(),
        days_left: z.number(),
        seconds_left: z.number().nonnegative().default(0),
      })
      .nullable()
      .default(null),
    trial_available: z.boolean().default(false),
    offer: trialOfferSchema,
    currency: z.string().default('IRR'),
    plans: z.array(offerPlanSchema).default([]),
    usage: z
      .object({ since: z.string().default(''), features: z.array(trialUsageSchema).default([]) })
      .default({ since: '', features: [] }),
  })
  .transform(
    (t): PlusTrialSheet => ({
      tier: tierOf(t.tier, false),
      trial: t.trial
        ? {
            startedAt: t.trial.started_at,
            endsAt: t.trial.ends_at,
            isActive: t.trial.is_active,
            daysLeft: t.trial.days_left,
            secondsLeft: t.trial.seconds_left,
          }
        : null,
      trialAvailable: t.trial_available,
      offer: t.offer,
      currency: t.currency,
      plans: t.plans,
      usage: t.usage,
    }),
  );
