import { z } from 'zod';

import type { LearnTip, LearnTopic, LearnView } from '../model/types';

/* Parser of GET /children/{id}/learn (B-N5-02, `LearnTipJSON`). */

const text = z.string().nullable().catch(null);

function each<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((list) =>
      list.flatMap((raw) => {
        const p = schema.safeParse(raw);
        return p.success ? [p.data] : [];
      }),
    );
}

const topicSchema = z.object({ code: z.string(), label: z.string().catch('') }).transform((d): LearnTopic => d);

export const learnTipSchema = z
  .object({
    code: z.string(),
    topic: z.string().catch(''),
    topic_label: text,
    title: z.string().min(1),
    body: text,
    minutes: z.number().int().nullable().catch(null),
    article: z
      .object({ slug: z.string().min(1) })
      .nullable()
      .catch(null),
  })
  .transform(
    (d): LearnTip => ({
      code: d.code,
      topic: d.topic,
      topicLabel: d.topic_label,
      title: d.title,
      body: d.body,
      minutes: d.minutes,
      articleSlug: d.article?.slug ?? null,
    }),
  );

export const learnViewSchema = z
  .object({
    age_months: z.number().int().catch(0),
    topics: each(topicSchema),
    topic: text,
    featured: learnTipSchema.nullable().catch(null),
    tips: each(learnTipSchema),
    disclaimer: text,
  })
  .transform(
    (d): LearnView => ({
      ageMonths: d.age_months,
      topics: d.topics,
      topic: d.topic,
      featured: d.featured,
      tips: d.tips,
      disclaimer: d.disclaimer,
    }),
  );
