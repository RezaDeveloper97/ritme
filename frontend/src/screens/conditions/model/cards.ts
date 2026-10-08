import type { ConditionCatalogItem, ConditionProgram, ConditionsOverview } from '@/entities/condition';

/** A hub card: her enrolment plus the admin copy (null = use the bundled `conditions.programs.<code>` string). */
export interface ProgramCard {
  code: ConditionProgram;
  enrolled: boolean;
  enrolledOn: string | null;
  title: string | null;
  sub: string | null;
  logs: string | null;
}

/** The API's program order, each joined with its `condition_programs` catalog row (missing → bundled copy). */
export function programCards(overview: ConditionsOverview, catalog: readonly ConditionCatalogItem[] | undefined): ProgramCard[] {
  const byCode = new Map((catalog ?? []).map((item) => [item.code, item]));
  return overview.programs.map((p) => {
    const item = byCode.get(p.code);
    return {
      code: p.code,
      enrolled: p.enrolled,
      enrolledOn: p.enrolledOn,
      title: item?.title ?? null,
      sub: item?.body ?? null,
      logs: item?.logs ?? null,
    };
  });
}

/** The «not a diagnosis» note from `condition_alerts/not_a_diagnosis`, or null for the bundled one. */
export function notADiagnosis(
  alerts: readonly ConditionCatalogItem[] | undefined,
): { title: string; body: string | null } | null {
  const item = alerts?.find((a) => a.code === 'not_a_diagnosis');
  return item?.title ? { title: item.title, body: item.body } : null;
}
