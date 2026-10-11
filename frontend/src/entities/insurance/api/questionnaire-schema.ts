import { z } from 'zod';

import type { InsuranceQuestionnaire } from '../model/questionnaire';

/** Boundary parsers for `/insurance/questionnaires` (CLAUDE.md §10). Shape: openapi.yaml `InsuranceQuestionnaire`. */

const nstr = z
  .string()
  .nullish()
  .transform((v) => v ?? null);
const answer = z
  .enum(['yes', 'no'])
  .nullish()
  .transform((v) => v ?? null);

/** List rows carry no sections / questions / explanation, hence the defaults. */
export const questionnaireSchema = z
  .object({
    id: z.number(),
    purpose: z.enum(['issue', 'renewal']).catch('issue'),
    label: nstr,
    status: z
      .enum(['draft', 'confirmed', 'submitted', 'reviewing', 'doctor_review', 'approved', 'conditional', 'rejected'])
      .catch('draft'),
    prefilled_count: z.number().catch(0),
    answered_count: z.number().catch(0),
    question_count: z.number().catch(0),
    complete: z.boolean().catch(false),
    confirmed_at: nstr,
    timeline: z.array(z.object({ status: z.string(), on: z.string() })).catch([]),
    sections: z.array(z.string()).catch([]),
    explanation: z.string().catch(''),
    questions: z
      .array(
        z.object({
          code: z.string(),
          section: z.string(),
          answer,
          from_record: z.boolean().catch(false),
          evidence: z.array(z.string()).catch([]),
        }),
      )
      .catch([]),
  })
  .transform(
    (v): InsuranceQuestionnaire => ({
      id: v.id,
      purpose: v.purpose,
      label: v.label,
      status: v.status,
      prefilledCount: v.prefilled_count,
      answeredCount: v.answered_count,
      questionCount: v.question_count,
      complete: v.complete,
      confirmedAt: v.confirmed_at,
      timeline: v.timeline,
      sections: v.sections,
      explanation: v.explanation,
      questions: v.questions.map((q) => ({
        code: q.code,
        section: q.section,
        answer: q.answer,
        fromRecord: q.from_record,
        evidence: q.evidence,
      })),
    }),
  );

export const questionnaireListSchema = z
  .object({ questionnaires: z.array(questionnaireSchema) })
  .transform((v): InsuranceQuestionnaire[] => v.questionnaires);
