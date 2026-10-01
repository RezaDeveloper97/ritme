'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { useCurrentAdmin } from '@/features/auth';
import { formatNumber } from '@/shared/lib';
import { Badge, type BadgeTone } from '@/shared/ui';

import { rialsToToman } from '../lib/money';

/** Money writes (prices, codes, settings, refunds, extensions) are super-admin only (admin-api.md §15). */
export function useCanManage(): boolean {
  return useCurrentAdmin()?.role === 'super';
}

/** `۹۹٬۰۰۰ تومان` from rials. */
export function Toman({ rials }: { rials: number | null | undefined }) {
  const t = useTranslations('plus');
  const locale = useLocale();
  if (rials === null || rials === undefined) return <>—</>;
  return <span className="whitespace-nowrap tabular-nums">{t('toman', { amount: formatNumber(rialsToToman(rials), locale) })}</span>;
}

const STATUS_TONE: Record<string, BadgeTone> = {
  active: 'green',
  paid: 'green',
  canceled: 'amber',
  pending: 'brand',
  expired: 'neutral',
  failed: 'red',
  refunded: 'red',
  scheduled: 'brand',
  exhausted: 'amber',
  inactive: 'neutral',
};

/** Subscription / invoice / discount status badge. */
export function StatusBadge({ status }: { status: string }) {
  const t = useTranslations('plus.status');
  const key = status as 'active';
  return <Badge tone={STATUS_TONE[status] ?? 'neutral'}>{t.has(key) ? t(key) : status}</Badge>;
}

/** Subscriber: name (or #id) linking to the user, and the masked mobile. */
export function UserCell({ user }: { user: { id: number; name: string | null; mobile: string | null } }) {
  const t = useTranslations('plus');
  const locale = useLocale();
  return (
    <span className="flex flex-col items-start">
      <Link
        href={`/users/${user.id}`}
        dir="auto"
        className="font-semibold text-brand no-underline hover:underline"
        onClick={(e) => e.stopPropagation()}
      >
        {user.name || t('userNumber', { id: formatNumber(user.id, locale) })}
      </Link>
      {user.mobile ? (
        <span dir="ltr" className="text-xs text-muted">
          {user.mobile}
        </span>
      ) : null}
    </span>
  );
}
