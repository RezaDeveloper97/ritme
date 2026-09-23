'use client';

import { useTranslations } from 'next-intl';

/** `rtl` / `ltr` → "Right to left" / "Left to right". */
export function useDirectionLabel() {
  const t = useTranslations('languages.directions');
  return (value: string) => (t.has(value as 'rtl') ? t(value as 'rtl') : value);
}
