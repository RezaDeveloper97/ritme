import { z } from 'zod';

import type { MilestoneActivity, MilestoneBandRef, MilestoneItem, MilestonesView } from '../model/types';

/* Parser of GET|PUT /children/{id}/milestones (B-N5-02, `MilestoneBandJSON`). */

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

const bandRefSchema = z
  .object({ months: z.number().int(), label: z.string().catch(''), current: z.boolean().catch(false) })
  .transform((d): MilestoneBandRef => d);

const itemSchema = z
  .object({
    code: z.string(),
    title: z.string().catch(''),
    domain: z.string().catch(''),
    checked: z.boolean().catch(false),
    checked_on: text,
  })
  .transform(
    (d): MilestoneItem => ({ code: d.code, title: d.title || d.code, domain: d.domain, checked: d.checked, checkedOn: d.checked_on }),
  );

const activitySchema = z
  .object({ code: z.string(), title: z.string().catch(''), body: text })
  .transform((d): MilestoneActivity => d);

export const milestonesSchema = z
  .object({
    bands: each(bandRefSchema),
    band: z.object({
      months: z.number().int(),
      label: z.string().catch(''),
      checked: z.number().int().catch(0),
      total: z.number().int().catch(0),
      items: each(itemSchema),
      activities: each(activitySchema),
      doctor_note: text,
    }),
    intro: text,
  })
  .transform(
    (d): MilestonesView => ({
      bands: d.bands,
      band: {
        months: d.band.months,
        label: d.band.label,
        checked: d.band.checked,
        total: d.band.total,
        items: d.band.items,
        activities: d.band.activities,
        doctorNote: d.band.doctor_note,
      },
      intro: d.intro,
    }),
  );
