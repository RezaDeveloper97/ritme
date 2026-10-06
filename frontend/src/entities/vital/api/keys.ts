import type { GlucoseFilter, ReportRange, VitalType } from '../model/types';

/** Query keys of the vitals reads (B-N6-02). Every write invalidates `all`. */
export const vitalsKeys = {
  all: ['vitals'] as const,
  hub: () => [...vitalsKeys.all, 'hub'] as const,
  plan: () => [...vitalsKeys.all, 'plan'] as const,
  thresholds: () => [...vitalsKeys.all, 'thresholds'] as const,
  readings: (type: VitalType, from: string, to: string) => [...vitalsKeys.all, 'readings', type, from, to] as const,
  reports: () => [...vitalsKeys.all, 'report'] as const,
  report: (type: VitalType, range: ReportRange, filter: GlucoseFilter | null) =>
    [...vitalsKeys.reports(), type, range, filter ?? 'all'] as const,
};
