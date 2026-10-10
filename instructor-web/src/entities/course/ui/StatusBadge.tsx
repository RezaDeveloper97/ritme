'use client';

import { useTranslations } from 'next-intl';

import { cn } from '@/shared/lib';

/** Outlined pill: منتشرشده (data), پیش‌نویس (warm), مستقل (brand). */
export function StatusBadge({ status }: { status: 'draft' | 'published' | 'standalone' }) {
  const t = useTranslations('content.status');
  return <span className={cn('status-badge', `status-badge--${status}`)}>{t(status)}</span>;
}
