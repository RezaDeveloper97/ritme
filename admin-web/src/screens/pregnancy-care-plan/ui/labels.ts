'use client';

import { useTranslations } from 'next-intl';

/** Care item kind → its label (unknown kinds as-is). */
export function useKindLabel() {
  const t = useTranslations('pregnancyCarePlan.kinds');
  return (kind: string) => (t.has(kind as 'visit') ? t(kind as 'visit') : kind);
}
