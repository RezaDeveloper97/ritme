import type { VitalType } from './types';

/** URL segment of a type (routes.md: `/vitals/bp`, `/vitals/glucose`, `/vitals/heart-rate`). */
export function vitalSlug(type: VitalType): string {
  return type === 'hr' ? 'heart-rate' : type;
}

/** `/vitals/<slug>` — the report of a type. */
export function vitalReportPath(type: VitalType): string {
  return `/vitals/${vitalSlug(type)}`;
}

/** `/vitals/<slug>/new` — the add screen of a type. */
export function vitalAddPath(type: VitalType): string {
  return `/vitals/${vitalSlug(type)}/new`;
}
