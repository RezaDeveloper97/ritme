import { z } from 'zod';

import type { VaccineSummary, VaccineVisit } from '@/entities/child';

import type { VaccineDose, VaccineSchedule, VaccineVisitDetail } from '../model/types';

/* Parser of GET /children/{id}/vaccines and its mark / unmark responses (B-N5-02, `internal/children/views.go`). */

const text = z.string().nullable().catch(null);
const date = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

const doseSchema = z
  .object({
    code: z.string(),
    title: z.string().catch(''),
    protects_against: text,
    status: z.string().catch('upcoming'),
    status_label: text,
    given_on: date.nullable().catch(null),
    note: text,
  })
  .transform(
    (d): VaccineDose => ({
      code: d.code,
      title: d.title || d.code,
      protectsAgainst: d.protects_against,
      status: d.status,
      statusLabel: d.status_label,
      givenOn: d.given_on,
      note: d.note,
    }),
  );

const visitFields = {
  code: z.string(),
  age_months: z.number().int().catch(0),
  label: z.string().catch(''),
  due_date: date,
  days_left: z.number().int().catch(0),
  status: z.string().catch('upcoming'),
  status_label: text,
  given: z.number().int().catch(0),
  total: z.number().int().catch(0),
  dose_names: z
    .array(z.string().nullable())
    .catch([])
    .transform((l) => l.filter((s): s is string => typeof s === 'string' && s !== '')),
};

type RawVisit = z.infer<z.ZodObject<typeof visitFields>>;

const toVisit = (d: RawVisit): VaccineVisit => ({
  code: d.code,
  ageMonths: d.age_months,
  label: d.label,
  dueDate: d.due_date,
  daysLeft: d.days_left,
  status: d.status,
  statusLabel: d.status_label,
  given: d.given,
  total: d.total,
  doseNames: d.dose_names,
});

const visitSchema = z.object(visitFields).transform(toVisit);

const visitDetailSchema = z
  .object({
    ...visitFields,
    doses: z
      .array(z.unknown())
      .catch([])
      .transform((l) =>
        l.flatMap((raw) => {
          const p = doseSchema.safeParse(raw);
          return p.success ? [p.data] : [];
        }),
      ),
  })
  .transform((d): VaccineVisitDetail => ({ ...toVisit(d), doses: d.doses }));

const summarySchema = z
  .object({
    given: z.number().int().catch(0),
    total: z.number().int().catch(0),
    completed_visits: z.number().int().catch(0),
    up_to_date: z.boolean().catch(false),
    complete: z.boolean().catch(false),
    next: visitSchema.nullable().catch(null),
  })
  .transform(
    (d): VaccineSummary => ({
      given: d.given,
      total: d.total,
      completedVisits: d.completed_visits,
      upToDate: d.up_to_date,
      complete: d.complete,
      next: d.next,
    }),
  );

export const vaccineScheduleSchema = z
  .object({
    summary: summarySchema,
    visits: z
      .array(z.unknown())
      .catch([])
      .transform((l) =>
        l.flatMap((raw) => {
          const p = visitDetailSchema.safeParse(raw);
          return p.success ? [p.data] : [];
        }),
      ),
    reminders: z.object({ label: text }).nullable().catch(null),
    note: text,
  })
  .transform(
    (d): VaccineSchedule => ({
      summary: d.summary,
      visits: d.visits,
      reminderLabel: d.reminders?.label ?? null,
      note: d.note,
    }),
  );
