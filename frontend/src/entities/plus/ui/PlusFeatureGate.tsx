'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, fromApiDate } from '@/shared/lib/date';
import { PlusGate, upgradeHelps, type PlusDenial } from '@/shared/ui/plus-gate';

import { usePlusStatus } from '../api/queries';
import type { PlusEntitlement, PlusStatus } from '../model/types';

/** The `/plus/status` entitlement of `feature` (`plus.lab_ai` …), null while unknown. */
export function entitlementOf(status: PlusStatus | undefined, feature: string): PlusEntitlement | null {
  return status?.entitlements.find((e) => e.key === feature) ?? null;
}

/** Whether `feature` is locked for the user now (false while the status is loading or unknown — the API gate decides). */
export function usePlusLocked(feature: string): boolean {
  const status = usePlusStatus();
  const e = entitlementOf(status.data, feature);
  return e !== null && !e.allowed;
}

interface PlusFeatureGateProps {
  /** Entitlement key, e.g. `plus.pdf_share` (backend-go/internal/plus/entitlements.go). */
  feature: string;
  /** The last API refusal for this feature (`plusDenialOf(error)` from `@/shared/ui/plus-gate`). */
  denial?: PlusDenial | null;
  children: ReactNode;
  className?: string;
}

/**
 * `<PlusGate>` wired to the plus entity (B-N2-08): locked state from the
 * `/plus/status` entitlements (or a 402/429 `denial`), the «پلاس» copy, the
 * paywall (`/plus`) as unlock and the renewal date for a spent Plus quota.
 * Needs the `plus` message namespace on the route.
 */
export function PlusFeatureGate({ feature, denial, children, className }: PlusFeatureGateProps) {
  const t = useTranslations('plus.gate');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const locked = usePlusLocked(feature);
  // resets_at is Tehran midnight; its calendar date is the civil renewal day.
  const resets = denial?.resetsAt ? formatDayMonth(fromApiDate(denial.resetsAt.slice(0, 10)), loc) : null;
  return (
    <PlusGate
      locked={locked}
      denial={denial}
      label={t('label')}
      lockedText={t('lockedText')}
      onUnlock={!denial || upgradeHelps(denial) ? () => router.push('/plus') : undefined}
      exhaustedText={resets ? t('exhausted', { date: resets }) : undefined}
      className={className}
    >
      {children}
    </PlusGate>
  );
}
