import { z } from 'zod';

/*
 * Parsers of the /api/v1/assistant responses (B-N7-06; schemas in backend-go/api/openapi.yaml). snake_case — the field
 * names are the wire contract. Message text and summaries are health data: never log them, never put them in a URL.
 */

export const TRIAGE_LEVELS = ['self_care', 'non_urgent', 'soon', 'emergency'] as const;
export type TriageLevel = (typeof TRIAGE_LEVELS)[number];
const triage = z.enum(TRIAGE_LEVELS).nullable().catch(null);

export const PROFILE_KEYS = ['age', 'life_mode', 'basics', 'cycle', 'pregnancies', 'conditions', 'medications', 'allergies'] as const;
export type ProfileKey = (typeof PROFILE_KEYS)[number];

const text = z.string().nullable().catch(null);

export const departmentSchema = z.object({
  code: z.string(),
  icon: text,
  name: z.string().catch(''),
  tagline: text,
});
export type Department = z.infer<typeof departmentSchema>;

export const departmentFullSchema = departmentSchema.extend({
  intro: text,
  does_not_do: z.array(z.string()).catch([]),
  quick_replies: z.array(z.string()).catch([]),
  conversations_count: z.number().catch(0),
});
export type DepartmentFull = z.infer<typeof departmentFullSchema>;

export const profileItemSchema = z.object({
  key: z.string(),
  shared: z.boolean().catch(false),
  empty: z.boolean().catch(false),
  data: z.unknown().optional(),
});
export type ProfileItem = z.infer<typeof profileItemSchema>;

export const profileSchema = z.object({
  consent: z.object({ code: z.string(), granted: z.boolean().catch(false) }),
  shared: z.array(z.string()).catch([]),
  items: z.array(profileItemSchema).catch([]),
});
export type AssistantProfile = z.infer<typeof profileSchema>;

export const quotaSchema = z
  .object({
    unlimited: z.boolean().catch(false),
    allowed: z.boolean().catch(true),
    limit: z.number().nullable().catch(null),
    used: z.number().nullable().catch(null),
    remaining: z.number().nullable().catch(null),
  })
  .nullable()
  .catch(null);
export type Quota = z.infer<typeof quotaSchema>;

export const conversationSchema = z.object({
  id: z.number(),
  department: departmentSchema,
  title: text,
  triage_level: triage,
  emergency: z.boolean().catch(false),
  message_count: z.number().catch(0),
  has_summary: z.boolean().catch(false),
  last_message_at: text,
  created_at: text,
});
export type Conversation = z.infer<typeof conversationSchema>;

export const hubSchema = z.object({
  departments: z.array(departmentSchema),
  quota: quotaSchema,
  recent: z.array(conversationSchema).catch([]),
  profile: profileSchema,
  emergency: z.object({ phone: z.string().catch('115') }).catch({ phone: '115' }),
});
export type Hub = z.infer<typeof hubSchema>;

export const departmentPageSchema = z.object({ department: departmentFullSchema, profile: profileSchema });

export const cardSchema = z
  .object({ type: z.string(), phone: z.string().catch('115'), title: z.string().catch(''), body: z.string().catch(''), call_label: z.string().catch('') })
  .nullable()
  .catch(null);

export const messageSchema = z.object({
  id: z.number(),
  role: z.enum(['user', 'assistant', 'card']).catch('assistant'),
  text: z.string().catch(''),
  photo: z.object({ id: z.number() }).nullable().catch(null),
  card: cardSchema,
  triage_level: triage,
  guardrail: text,
  chips: z.array(z.string()).catch([]),
  status: z.string().catch('complete'),
  created_at: text,
});
export type AssistantMessage = z.infer<typeof messageSchema>;

export const summarySchema = z
  .object({ text: z.string(), triage_level: triage, created_at: text, up_to_date: z.boolean().catch(true) })
  .nullable()
  .catch(null);
export type Summary = z.infer<typeof summarySchema>;

export const handoffSchema = z.object({ id: z.number(), booking_id: z.number(), triage_level: triage, created_at: text });

export const conversationPageSchema = z.object({
  conversation: conversationSchema,
  department: departmentFullSchema.omit({ conversations_count: true }),
  messages: z.array(messageSchema),
  summary: summarySchema,
  handoffs: z.array(handoffSchema).catch([]),
});
export type ConversationPage = z.infer<typeof conversationPageSchema>;

export const conversationListSchema = z.object({ conversations: z.array(conversationSchema) });

export const summaryPageSchema = z.object({
  conversation: conversationSchema,
  summary: summarySchema,
  photos_count: z.number().catch(0),
  triage_levels: z.array(z.string()).catch([]),
});
export type SummaryPage = z.infer<typeof summaryPageSchema>;

export const consentSchema = z
  .object({
    consent: z.object({
      code: z.string(),
      version: z.number().int(),
      title: z.string().catch(''),
      body: z.string().catch(''),
      points: z.array(z.string()).catch([]),
      granted: z.boolean().catch(false),
      needs_consent: z.boolean().catch(true),
    }),
  })
  .transform(({ consent }) => consent);
export type AssistantConsent = z.infer<typeof consentSchema>;
