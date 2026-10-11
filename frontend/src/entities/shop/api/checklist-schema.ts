import { z } from 'zod';

/*
 * Parsers of /api/v1/shop/checklist* and /shop/supply-reminder (CB-SHOP-04 → CB-SHOP-09, backend-go/api/openapi.yaml).
 * Snake_case = wire contract. The checklist is her own list — never logged, never sent to a seller.
 */

export const CHECKLIST_STATES = ['todo', 'have', 'in_cart', 'on_the_way', 'not_needed', 'later'] as const;
export type ChecklistState = (typeof CHECKLIST_STATES)[number];

/** The five groups a custom row may join (the server also sends unknown template groups after these). */
export const CHECKLIST_GROUPS = ['clothing', 'sleep', 'feeding', 'bath', 'outing'] as const;
export type ChecklistGroupKey = (typeof CHECKLIST_GROUPS)[number];

export const shopChecklistItemSchema = z.object({
  id: z.number().nullable().catch(null),
  layette_item_id: z.number().nullable().catch(null),
  is_custom: z.boolean().catch(false),
  title: z.string(),
  note: z.string().nullable().catch(null),
  suggested_qty: z.number().nullable().catch(null),
  is_essential: z.boolean().catch(false),
  is_first_days: z.boolean().catch(false),
  category_slug: z.string().nullable().catch(null),
  state: z.enum(CHECKLIST_STATES).catch('todo'),
  quantity: z.number().nullable().catch(null),
});
export type ShopChecklistItem = z.infer<typeof shopChecklistItemSchema>;

export const shopChecklistSchema = z.object({
  due_date: z.string().nullable().catch(null),
  weeks_left: z.number().nullable().catch(null),
  read_only: z.boolean().catch(false),
  summary: z.object({
    ready: z.number().catch(0),
    total: z.number().catch(0),
    not_needed: z.number().catch(0),
    percent: z.number().catch(0),
  }),
  groups: z
    .array(
      z.object({
        key: z.string(),
        title: z.string(),
        ready: z.number().catch(0),
        total: z.number().catch(0),
        items: z.array(shopChecklistItemSchema).catch([]),
      }),
    )
    .catch([]),
});
export type ShopChecklist = z.infer<typeof shopChecklistSchema>;
export type ShopChecklistGroup = ShopChecklist['groups'][number];

export const shopChecklistItemEnvelopeSchema = z.object({ item: shopChecklistItemSchema });

export const supplyReminderSchema = z.object({
  enabled: z.boolean().catch(false),
  lead_days: z.number().catch(2),
  product_id: z.number().nullable().catch(null),
  remind_on: z.string().nullable().catch(null),
});
export type SupplyReminder = z.infer<typeof supplyReminderSchema>;
