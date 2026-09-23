'use client';

import { useTranslations } from 'next-intl';

/** "Help & support" / "Privacy" / … for a group key; unknown keys as-is. */
export function useGroupLabel() {
  const t = useTranslations('infoSections.groups');
  return (group: string) => (t.has(group as 'help') ? t(group as 'help') : group);
}
