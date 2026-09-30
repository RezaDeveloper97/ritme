'use client';

import { useTranslations } from 'next-intl';

import { Badge } from '@/shared/ui';

/** Open = amber (waiting for the team), resolved = green; unknown values as-is. */
export function StatusBadge({ status }: { status: string }) {
  const t = useTranslations('supportReports.status');
  if (status === 'open') return <Badge tone="amber">{t('open')}</Badge>;
  if (status === 'resolved') return <Badge tone="green">{t('resolved')}</Badge>;
  return <Badge>{status}</Badge>;
}

/** The reporter as one line: name, else mobile, else #id. */
export function reporterName(user: { id: number; name: string | null; mobile: string | null }): string {
  return user.name || user.mobile || `#${user.id}`;
}
