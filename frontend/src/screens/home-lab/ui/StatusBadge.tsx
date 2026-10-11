import { useTranslations } from 'next-intl';

import type { LabStatus } from '@/entities/partner-lab';
import { StatusPill, type Tone } from '@/shared/ui';

const TONE: Record<LabStatus, Tone> = {
  requested: 'brand',
  accepted: 'brand',
  sampled: 'warm',
  result_ready: 'success',
  rejected: 'danger',
  cancelled: 'neutral',
};

/** The status of a request as a verdict pill. */
export function StatusBadge({ status }: { status: LabStatus }) {
  const t = useTranslations('homeLab.status');
  return <StatusPill tone={TONE[status]}>{t(status)}</StatusPill>;
}
