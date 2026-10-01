import { z } from 'zod';

/**
 * `GET /onboarding` (B-N2-01, OnboardingState in backend-go/api/openapi.yaml).
 * Lists are `null` when the step was skipped and `[]` for «هیچ‌کدام».
 */
const nullableList = z.array(z.string()).nullable().catch(null);

export const onboardingStateSchema = z.object({
  name: z.string().nullable().catch(null),
  gender: z.enum(['female', 'male']).nullable().catch(null),
  goal: z.string().nullable().catch(null),
  /** `mode` is the effective life stage: `pregnancy` only while a pregnancy is active. */
  life_stage: z.object({ mode: z.string().catch('cycle') }).catch({ mode: 'cycle' }),
  cycle: z
    .object({
      last_period_start: z.string().nullable().catch(null),
      period_duration: z.number().nullable().catch(null),
      cycle_duration: z.number().nullable().catch(null),
    })
    .catch({ last_period_start: null, period_duration: null, cycle_duration: null }),
  menopause: z
    .object({
      stage: z.enum(['peri', 'meno', 'post', 'unsure']).nullable().catch(null),
      last_period: z.string().nullable().catch(null),
      surgical: z.boolean().nullable().catch(null),
      hrt: z.boolean().nullable().catch(null),
    })
    .catch({ stage: null, last_period: null, surgical: null, hrt: null }),
  conditions: z
    .object({
      chronic_illnesses: nullableList,
      gyn_conditions: nullableList,
      medications: nullableList,
    })
    .catch({ chronic_illnesses: null, gyn_conditions: null, medications: null }),
  health: z
    .object({
      birthday: z.string().nullable().catch(null),
      height: z.number().nullable().catch(null),
      weight: z.number().nullable().catch(null),
    })
    .catch({ birthday: null, height: null, weight: null }),
  completed: z.boolean().catch(false),
});

export type OnboardingState = z.infer<typeof onboardingStateSchema>;

export type MenopauseStage = 'peri' | 'meno' | 'post' | 'unsure';
export const MENOPAUSE_STAGES: readonly MenopauseStage[] = ['peri', 'meno', 'post', 'unsure'];

export const CHRONIC_ILLNESSES = ['diabetes', 'hypertension', 'thyroid', 'asthma', 'anemia', 'migraine', 'other'] as const;
export const GYN_CONDITIONS = ['pcos', 'endometriosis', 'fibroids', 'recurrent_infections', 'other'] as const;
export const MEDICATIONS = ['contraceptive_pill', 'iud', 'hormonal_medication'] as const;

/** Body of `PUT /onboarding/steps/{step}` per step. */
export interface StepBodies {
  name: { name: string };
  gender: { gender: 'female' | 'male' };
  goal: { goal: string };
  cycle: { last_period_start?: string | null; period_duration?: number | null; cycle_duration?: number | null };
  menopause: { stage: MenopauseStage; last_period?: string | null; surgical?: boolean | null; hrt?: boolean | null };
  conditions: { chronic_illnesses?: string[] | null; gyn_conditions?: string[] | null; medications?: string[] | null };
  health: { birthday?: string | null; height?: number | null; weight?: number | null };
}

export type StepName = keyof StepBodies;

/**
 * A multi-select with an exclusive «هیچ‌کدام» chip: picking «none» clears the
 * rest, picking an item drops «none». `null` = untouched (sent as skipped),
 * `[]` = «هیچ‌کدام».
 */
export function toggleListItem(list: readonly string[] | null, item: string | 'none'): string[] {
  if (item === 'none') return [];
  const current = list ?? [];
  return current.includes(item) ? current.filter((x) => x !== item) : [...current, item];
}
