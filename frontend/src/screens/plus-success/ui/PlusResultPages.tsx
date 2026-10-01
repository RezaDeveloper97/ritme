'use client';

import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useEffect, useMemo, useRef, type ReactNode } from 'react';

import { parseGatewayReturn, usePlusStatus } from '@/entities/plus';
import { useVerifyPayment } from '@/features/purchase-plus';
import { getApiErrorCode, getApiErrorStatus } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { EmptyState, Icon, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import { SuccessView } from './SuccessView';

/** Error codes after which re-trying verify cannot help: the payment itself did not happen. */
const FINAL_FAILURES = new Set(['payment_failed', 'invoice_closed', 'invoice_not_found', 'authority_mismatch', 'amount_mismatch']);

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll is-center">{children}</div>
    </div>
  );
}

/**
 * Gateway return page (`/plus/return`, `PLUS_CALLBACK_URL`, B-N2-05): the API's
 * return hop 303s here with `reference`, `authority` and `status`; this page
 * settles the invoice with the authenticated `POST /plus/verify` (idempotent,
 * so a reload is safe) and shows the success view or a calm failure.
 */
export function PlusReturnPage() {
  const t = useTranslations('plus');
  const router = useRouter();
  const params = useSearchParams();
  const ret = useMemo(() => parseGatewayReturn(params), [params]);
  const verify = useVerifyPayment();
  const started = useRef(false);

  useEffect(() => {
    if (!ret || started.current) return;
    started.current = true;
    verify.mutate(ret);
  }, [ret, verify]);

  if (!ret) {
    return (
      <Shell>
        <EmptyState
          icon="warning"
          title={t('return.invalidTitle')}
          body={t('return.invalidBody')}
          action={<PrimaryButton onClick={() => router.replace('/plus')}>{t('return.backToPlans')}</PrimaryButton>}
        />
      </Shell>
    );
  }

  if (verify.isSuccess) {
    return <SuccessView status={verify.data.status} refId={verify.data.invoice.receipt?.refId} />;
  }

  if (verify.isError) {
    const code = getApiErrorCode(verify.error);
    const canRetry = !(code && FINAL_FAILURES.has(code)) && getApiErrorStatus(verify.error) !== 422;
    return (
      <Shell>
        <EmptyState
          icon="warning"
          title={t('return.failedTitle')}
          body={t('return.failedBody')}
          action={
            <div className="plus-actions">
              {canRetry ? (
                <PrimaryButton icon="refresh" onClick={() => verify.mutate(ret)}>
                  {t('retry')}
                </PrimaryButton>
              ) : null}
              <SecondaryButton onClick={() => router.replace('/plus/plans')}>{t('return.backToPlans')}</SecondaryButton>
            </div>
          }
        />
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="plus-verifying" role="status" aria-live="polite">
        <Icon name="loader" size={32} className="plus-spin" />
        <p>{t('return.verifying')}</p>
      </div>
    </Shell>
  );
}

/**
 * `/plus/success` — the success view from the current status: after a 100%
 * discount (no gateway) or a started trial (`?trial=1`). Without Plus it
 * sends the user back to the paywall.
 */
export function PlusSuccessPage() {
  const t = useTranslations('plus');
  const router = useRouter();
  const status = usePlusStatus();
  const isPlus = status.data?.isPlus ?? false;

  useEffect(() => {
    if (status.data && !status.data.isPlus) router.replace('/plus');
  }, [status.data, router]);

  if (status.isError) {
    return (
      <Shell>
        <EmptyState
          icon="crown"
          title={t('error.title')}
          body={t('error.body')}
          action={
            <PrimaryButton icon="refresh" loading={status.isFetching} onClick={() => void status.refetch()}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Shell>
    );
  }
  if (!status.data || !isPlus) {
    return (
      <Shell>
        <SkeletonGroup label={t('loading')} className="plus-hero is-success">
          <Skeleton shape="circle" />
          <Skeleton width="medium" />
          <Skeleton width="short" />
        </SkeletonGroup>
      </Shell>
    );
  }
  return <SuccessView status={status.data} />;
}
