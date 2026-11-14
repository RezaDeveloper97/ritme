'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { formatCredits, useAiCredits, useCreditPackQuote } from '@/entities/ai-credit';
import { formatToman } from '@/entities/plus';
import { goToGateway, useCreditPackCheckout } from '@/features/purchase-plus';
import { getApiErrorStatus } from '@/shared/api';
import { useBillingChannel } from '@/shared/billing';
import { useRouter, type Locale } from '@/shared/i18n';
import { EmptyState, Icon, InfoNote, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';
import { useNavMode } from '@/widgets/bottom-nav';

/**
 * «تأیید و پرداخت» for a credit pack (`/plus/checkout?pack=<id>`): the server-priced summary (preview), the bank
 * gateway, then the same gateway return as a plan (`/plus/return` → POST /plus/verify). No discount code for
 * packs. Not offered to teen / companion accounts or store builds.
 */
export function PackCheckout({ packId }: { packId: number }) {
  const t = useTranslations('plus');
  const tc = useTranslations('credits');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const navMode = useNavMode().mode;
  const { channel, ready } = useBillingChannel();
  const quote = useCreditPackQuote(packId);
  const credits = useAiCredits();
  const checkout = useCreditPackCheckout();
  const [payError, setPayError] = useState<string | null>(null);
  const blocked = navMode === 'teen' || navMode === 'companion';

  useEffect(() => {
    if (blocked) router.replace('/profile');
  }, [blocked, router]);

  const onPay = () => {
    setPayError(null);
    checkout.mutate(packId, {
      onSuccess: (result) => {
        if (!result.payment) router.replace('/profile/credits');
        else if (!goToGateway(result.payment.redirectUrl)) setPayError(t('checkout.payFailed'));
      },
      onError: (error) => setPayError(getApiErrorStatus(error) === 503 ? t('checkout.unavailable') : t('checkout.payFailed')),
    });
  };

  const toman = (rials: number) => t('toman', { amount: formatToman(rials, loc) });
  const q = quote.data;

  let body;
  if (!ready || quote.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="plus-checkout">
        <Skeleton shape="card" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (channel !== 'web') {
    body = <EmptyState icon="sparkle" title={tc('buy.storeTitle')} body={tc('buy.storeBody')} />;
  } else if (quote.isError || !q) {
    body = (
      <EmptyState
        icon="sparkle"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={quote.isFetching} onClick={() => void quote.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = (
      <div className="plus-checkout">
        <section className="nb-card plus-summary" aria-labelledby="plus-summary-title">
          <div className="plus-summary-head">
            <h2 id="plus-summary-title" className="plus-summary-title">
              {tc('checkout.summary', { pack: q.pack.title })}
            </h2>
            <button type="button" className="plus-link is-brand" onClick={() => router.push('/plus/credits')}>
              {t('checkout.change')}
            </button>
          </div>
          <dl className="plus-lines">
            <div className="plus-line">
              <dt>{tc('checkout.credits')}</dt>
              <dd>{tc('checkout.creditsValue', { count: formatCredits(q.pack.credits, loc) })}</dd>
            </div>
            <div className="plus-line">
              <dt>{t('checkout.price')}</dt>
              <dd>{toman(q.subtotal)}</dd>
            </div>
            {q.vat > 0 ? (
              <div className="plus-line">
                <dt>{t('checkout.vat')}</dt>
                <dd>{toman(q.vat)}</dd>
              </div>
            ) : null}
            <div className="plus-line is-total">
              <dt>{t('checkout.total')}</dt>
              <dd>
                <span className="plus-total-num" aria-hidden>
                  {formatToman(q.total, loc)}
                </span>
                <span className="sr-only">{toman(q.total)}</span>
              </dd>
            </div>
          </dl>
        </section>
        <InfoNote>{credits.data && !credits.data.canUse ? tc('buy.noSubscription') : tc('checkout.note')}</InfoNote>
        <section className="plus-methods" aria-labelledby="plus-methods-title">
          <h2 id="plus-methods-title" className="plus-section-title">
            {t('checkout.methods')}
          </h2>
          <div role="radiogroup" aria-labelledby="plus-methods-title">
            <div role="radio" aria-checked tabIndex={0} className="plus-method">
              <span className="plus-plan-dot" aria-hidden />
              <span className="plus-method-text">
                <b>{t('checkout.gateway')}</b>
                <span>{t('checkout.gatewaySub')}</span>
              </span>
              <Icon name="card" size={22} strokeWidth={1.8} className="plus-method-icon" />
            </div>
          </div>
          <p className="plus-secure">
            <Icon name="lock" size={14} />
            {tc('checkout.secure')}
          </p>
        </section>
      </div>
    );
  }

  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll">
        <ScreenHeader title={t('checkout.title')} onBack={() => router.push('/plus/credits')} backLabel={t('back')} />
        <div className="plus-intro">
          <h1 className="plus-title">{t('checkout.title')}</h1>
        </div>
        {body}
      </div>
      {q && channel === 'web' ? (
        <div className="plus-footer">
          {payError ? (
            <p className="plus-field-error is-center" role="alert">
              {payError}
            </p>
          ) : null}
          <PrimaryButton loading={checkout.isPending || checkout.isSuccess} onClick={onPay}>
            {t('checkout.pay')}
          </PrimaryButton>
        </div>
      ) : null}
    </div>
  );
}
