import { z } from 'zod';

import { safeInAppPath } from '../model/programs';
import {
  CONDITION_NUDGE_KEYS,
  CONDITION_PROGRAMS,
  type ConditionCatalogItem,
  type ConditionEnrolment,
  type ConditionNudge,
  type ConditionsOverview,
} from '../model/types';

/*
 * Zod parsers of `GET /conditions` (+ enrol / leave), `GET /catalog/condition_*` and `GET /messages/nudges`
 * (backend-go/contract/golden/{conditions,messages}). Lenient: an unknown program or a malformed row is dropped,
 * never the screen.
 */

const date = z.string().regex(/^\d{4}-\d{2}-\d{2}$/).nullable().catch(null);
const text = z.string().trim().min(1).nullable().catch(null);

const enrolmentSchema = z.object({
  code: z.enum(CONDITION_PROGRAMS),
  enrolled: z.boolean().catch(false),
  enrolled_on: date,
});

function rows<S extends z.ZodTypeAny>(list: unknown[], schema: S): z.output<S>[] {
  return list.flatMap((raw) => {
    const parsed = schema.safeParse(raw);
    return parsed.success ? [parsed.data] : [];
  });
}

export const conditionsOverviewSchema = z
  .object({ programs: z.array(z.unknown()).catch([]) })
  .transform(
    (d): ConditionsOverview => ({
      programs: rows(d.programs, enrolmentSchema).map(
        (p): ConditionEnrolment => ({ code: p.code, enrolled: p.enrolled, enrolledOn: p.enrolled ? p.enrolled_on : null }),
      ),
    }),
  );

const catalogItemSchema = z
  .object({
    code: z.string(),
    title: text,
    body: text,
    meta: z.record(z.string(), z.unknown()).nullable().catch(null),
  })
  .transform(
    (i): ConditionCatalogItem => ({
      code: i.code,
      title: i.title,
      body: i.body,
      logs: typeof i.meta?.logs === 'string' && i.meta.logs.trim() ? i.meta.logs.trim() : null,
    }),
  );

export const conditionCatalogSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): ConditionCatalogItem[] => rows(g.items, catalogItemSchema));

const nudgeSchema = z
  .object({
    key: z.enum(CONDITION_NUDGE_KEYS),
    program: z.enum(CONDITION_PROGRAMS),
    title: z.string().trim().min(1),
    body: z.string().catch(''),
    action: z.string().catch(''),
    link: z.unknown(),
    doctor: z
      .object({ action: z.string().trim().min(1), link: z.unknown() })
      .nullable()
      .catch(null),
    days: z.number().int().catch(0),
  })
  .transform((n): ConditionNudge => {
    const doctorLink = n.doctor ? safeInAppPath(n.doctor.link) : null;
    return {
      key: n.key,
      program: n.program,
      title: n.title,
      body: n.body,
      action: n.action,
      link: safeInAppPath(n.link),
      doctor: n.doctor && doctorLink ? { action: n.doctor.action, link: doctorLink } : null,
      days: n.days,
    };
  });

export const conditionNudgesSchema = z
  .object({ nudges: z.array(z.unknown()).catch([]) })
  .catch({ nudges: [] })
  .transform((d): ConditionNudge[] => rows(d.nudges, nudgeSchema));
