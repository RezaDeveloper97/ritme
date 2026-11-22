'use client';

import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useEffect, useRef, type ReactNode } from 'react';

import { useTelemedError } from '@/entities/doctor';
import { useVerifyBooking } from '@/features/book-visit';
import { getApiErrorCode, getApiErrorStatus } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { EmptyState, PrimaryButton, SecondaryButton, Skeleton, SkyLayer } from '@/shared/ui';
import '@/shared/ui/styles/dr.css';

/** After these the payment itself did not happen — retrying verify cannot help. */
const FINAL = new Set(['payment_failed', 'authority_mismatch', 'amount_mismatch', 'booking_closed', 'booking_refunded']);

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="view dr-view">
      <SkyLayer />
      <div className="scroll dr-center">{children}</div>
    </div>
  );
}

/**
 * `/services/bookings/return?reference=&authority=&status=` — the gateway hop (PAYMENT_CALLBACK) lands here; the
 * booking is settled by the authenticated, idempotent `POST /telemed/bookings/verify`, then the booked screen opens.
 */
export function DoctorReturnPage() {
  const t = useTranslations('doctors');
  const router = useRouter();
  const params = useSearchParams();
  const errorText = useTelemedError();
  const reference = params.get('reference') ?? '';
  const authority = params.get('authority') ?? '';
  const verify = useVerifyBooking();
  const started = useRef(false);
  const valid = reference !== '' && authority !== '';

  useEffect(() => {
    if (!valid || started.current) return;
    started.current = true;
    verify.mutate(
      { reference, authority },
      { onSuccess: (r) => router.replace(r.booking.status === 'confirmed' || r.booking.status === 'completed' ? `/services/bookings/${r.booking.id}?new=1` : '/services/bookings') },
    );
  }, [valid, reference, authority, verify, router]);

  if (!valid) {
    return (
      <Shell>
        <EmptyState icon="warning" title={t('return.invalidTitle')} body={t('return.invalidBody')} action={<PrimaryButton onClick={() => router.replace('/services/doctors')}>{t('return.toDoctors')}</PrimaryButton>} />
      </Shell>
    );
  }
  if (verify.isError) {
    const code = getApiErrorCode(verify.error);
    const canRetry = !(code && FINAL.has(code)) && getApiErrorStatus(verify.error) !== 422;
    return (
      <Shell>
        <EmptyState
          icon="warning"
          title={t('return.failedTitle')}
          body={errorText(verify.error)}
          action={
            <div className="dr-actions">
              {canRetry ? (
                <PrimaryButton icon="refresh" loading={verify.isPending} onClick={() => verify.mutate({ reference, authority })}>
                  {t('common.retry')}
                </PrimaryButton>
              ) : null}
              <SecondaryButton onClick={() => router.replace('/services/doctors')}>{t('return.toDoctors')}</SecondaryButton>
            </div>
          }
        />
      </Shell>
    );
  }
  return (
    <Shell>
      <div className="dr-verifying" role="status" aria-live="polite">
        <Skeleton shape="block" />
        <p>{t('return.verifying')}</p>
      </div>
    </Shell>
  );
}
