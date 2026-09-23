'use client';

import { useTranslations } from 'next-intl';

import { Badge } from '@/shared/ui';

import type { AdminRole } from '../model/admin';

export function RoleBadge({ role }: { role: AdminRole }) {
  const t = useTranslations('roles');
  return <Badge tone={role === 'super' ? 'brand' : 'neutral'}>{t(role)}</Badge>;
}
