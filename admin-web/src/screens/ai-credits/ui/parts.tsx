'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useCurrentAdmin } from '@/features/auth';
import { formatNumber } from '@/shared/lib';
import { Badge } from '@/shared/ui';

import { milliToUsd } from '../lib/report';

/** Every write of the AI credits admin is super-admin only (admin-api.md §29); reads follow the capability. */
export function useCanManage(): boolean {
  return useCurrentAdmin()?.role === 'super';
}

/** One badge per capability (chat, vision, audio, json). */
export function CapabilityBadges({ caps, missing = [] }: { caps: readonly string[]; missing?: readonly string[] }) {
  const t = useTranslations('aiCredits.capabilities');
  return (
    <span className="flex flex-wrap gap-1">
      {caps.map((c) => (
        <Badge key={c} tone={missing.includes(c) ? 'red' : 'data'}>
          {t.has(c as 'chat') ? t(c as 'chat') : c}
        </Badge>
      ))}
    </span>
  );
}

/** `$2.5` per 1M tokens from milli-USD. */
export function Usd({ milli }: { milli: number }) {
  const locale = useLocale();
  return (
    <span dir="ltr" className="whitespace-nowrap tabular-nums">
      ${formatNumber(milliToUsd(milli), locale)}
    </span>
  );
}
