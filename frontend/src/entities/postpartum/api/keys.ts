import type { EpdsKind } from '../model/types';

/** Query keys of the postpartum reads (B-N5-04). */
export const postpartumKeys = {
  all: ['postpartum'] as const,
  overview: () => [...postpartumKeys.all, 'overview'] as const,
  recovery: (date: string | null) => [...postpartumKeys.all, 'recovery', date ?? 'today'] as const,
  questions: (kind: EpdsKind, locale: string) => [...postpartumKeys.all, 'epds-questions', kind, locale] as const,
  history: () => [...postpartumKeys.all, 'epds-history'] as const,
};
