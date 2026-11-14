import { z } from 'zod';

import type {
  AiCredits,
  AiUsagePage,
  CreditPack,
  CreditPackCatalog,
  CreditPackQuote,
} from '../model/types';

const featureSchema = z.object({
  key: z.string(),
  title: z.string().default(''),
  credit_cost: z.number().default(0),
  available: z.boolean().default(true),
  affordable: z.boolean().default(true),
});

export const creditsSchema = z
  .object({
    can_use: z.boolean(),
    period: z.enum(['plus', 'trial']).nullable().default(null),
    period_ends_at: z.string().nullable().default(null),
    plan: z.object({ total: z.number(), used: z.number(), remaining: z.number() }),
    extra: z.object({ remaining: z.number(), locked: z.boolean().default(false) }),
    spendable: z.number().default(0),
    features: z.array(featureSchema).nullable().default([]),
  })
  .transform(
    (c): AiCredits => ({
      canUse: c.can_use,
      period: c.period,
      periodEndsAt: c.period_ends_at,
      plan: c.plan,
      extra: c.extra,
      spendable: c.spendable,
      features: (c.features ?? []).map((f) => ({
        key: f.key,
        title: f.title,
        creditCost: f.credit_cost,
        available: f.available,
        affordable: f.affordable,
      })),
    }),
  );

export const usageSchema = z
  .object({
    days: z.number().default(30),
    since: z.string().nullable().default(null),
    total: z.number().default(0),
    by_feature: z
      .array(z.object({ feature: z.string(), title: z.string().default(''), uses: z.number(), credits: z.number() }))
      .nullable()
      .default([]),
    by_day: z.array(z.object({ date: z.string(), credits: z.number() })).nullable().default([]),
    history: z
      .object({
        items: z
          .array(z.object({ id: z.number(), feature: z.string(), title: z.string().default(''), credits: z.number(), at: z.string() }))
          .nullable()
          .default([]),
        next: z.number().nullable().default(null),
      })
      .default({ items: [], next: null }),
  })
  .transform(
    (u): AiUsagePage => ({
      days: u.days,
      since: u.since,
      total: u.total,
      byFeature: u.by_feature ?? [],
      byDay: u.by_day ?? [],
      history: { items: u.history.items ?? [], next: u.history.next },
    }),
  );

const packShape = z.object({
  id: z.number(),
  code: z.string(),
  title: z.string(),
  badge: z.string().nullable().optional(),
  credits: z.number(),
  price: z.number().int().nonnegative(),
  is_highlighted: z.boolean().optional(),
});

export const packSchema = packShape.transform(
  (p): CreditPack => ({
    id: p.id,
    code: p.code,
    title: p.title,
    badge: p.badge ?? null,
    credits: p.credits,
    price: p.price,
    isHighlighted: p.is_highlighted ?? false,
  }),
);

export const packCatalogSchema = z
  .object({
    currency: z.string().default('IRR'),
    vat_rate_bps: z.number().default(0),
    packs: z.array(packSchema).nullable().default([]),
  })
  .transform((c): CreditPackCatalog => ({ currency: c.currency, vatRateBps: c.vat_rate_bps, packs: c.packs ?? [] }));

export const packQuoteSchema = z
  .object({
    quote: z.object({
      pack: packSchema,
      currency: z.string().default('IRR'),
      subtotal: z.number(),
      vat_rate_bps: z.number().default(0),
      vat: z.number(),
      total: z.number(),
    }),
  })
  .transform(
    ({ quote: q }): CreditPackQuote => ({
      pack: q.pack,
      currency: q.currency,
      subtotal: q.subtotal,
      vatRateBps: q.vat_rate_bps,
      vat: q.vat,
      total: q.total,
    }),
  );
