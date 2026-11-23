'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';

import { getApiErrorCode, getApiErrorMessage } from '@/shared/api';
import { Avatar, Card, Icon, EmptyState, PrimaryButton, Skeleton, SkeletonGroup, type IconName } from '@/shared/ui';

import type { VisitMode } from '../api/schema';
import '@/shared/ui/styles/dr.css';

export const MODE_ICON: Record<VisitMode, IconName> = { video: 'video', phone: 'phone', in_person: 'mapPin' };
const MODE_TONE: Record<VisitMode, string> = { video: 'nb-tone-data', phone: 'nb-tone-brand', in_person: 'nb-tone-warm' };

/** «دکتر سارا احمدی» / «خانم لیلا مهدوی» — the honorific is client copy chosen by `kind` (QUESTIONS #120). */
export function useDoctorTitle() {
  const t = useTranslations('doctors');
  return (d: { kind: 'doctor' | 'midwife'; name: string }) => t(`title.${d.kind}`, { name: d.name });
}

export function useModeLabel() {
  const t = useTranslations('doctors');
  return (m: VisitMode) => t(`mode.${m}`);
}

/** Round avatar: the doctor's photo, else the first letter on the avatar gradient. */
export function DoctorAvatar({ name, photoUrl, size = 'lg' }: { name: string; photoUrl: string | null; size?: 'sm' | 'md' | 'lg' }) {
  // eslint-disable-next-line @next/next/no-img-element -- the admin-uploaded WebP is already optimised
  return photoUrl ? <img src={photoUrl} alt="" className={clsx('dr-photo', `is-${size}`)} /> : <Avatar name={name} size={size} />;
}

/** Outlined mode tag («ویدیویی» · «تلفنی» · «حضوری»). */
export function ModeTag({ mode }: { mode: VisitMode }) {
  const label = useModeLabel();
  return (
    <span className={clsx('dr-mode', MODE_TONE[mode])}>
      {label(mode)}
      <Icon name={MODE_ICON[mode]} size={14} strokeWidth={1.8} />
    </span>
  );
}

const KNOWN = new Set([
  'slot_unavailable', 'payment_unavailable', 'payment_failed', 'amount_mismatch', 'authority_mismatch', 'booking_closed',
  'booking_refunded', 'cancel_window_closed', 'consent_not_allowed', 'too_many_bookings', 'too_many_holds', 'reschedule_limit',
  'chat_closed', 'chat_unavailable', 'message_limit', 'prescription_already_added', 'medication_limit', 'video_unavailable',
]);

/** Localized text for a failed telemed call: known `error_code` → our copy, else the server's message, else generic. */
export function useTelemedError() {
  const t = useTranslations('doctors');
  return (error: unknown): string => {
    const code = getApiErrorCode(error);
    if (code && KNOWN.has(code)) return t(`errors.${code}` as "errors.generic");
    return getApiErrorMessage(error) ?? t('errors.generic');
  };
}

/** Error card with a retry button (every doctors screen's load failure). */
export function LoadError({ onRetry, busy, title }: { onRetry: () => void; busy?: boolean; title?: string }) {
  const t = useTranslations('doctors');
  return (
    <Card>
      <EmptyState
        icon="warning"
        title={title ?? t('common.loadError')}
        action={
          <PrimaryButton icon="refresh" loading={busy} onClick={onRetry}>
            {t('common.retry')}
          </PrimaryButton>
        }
      />
    </Card>
  );
}

/** Skeleton of a screen body while its first request is in flight. */
export function BodySkeleton({ rows = 3 }: { rows?: number }) {
  const t = useTranslations('doctors');
  return (
    <SkeletonGroup label={t('common.loading')} className="dr-body">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} shape="block" />
      ))}
    </SkeletonGroup>
  );
}
