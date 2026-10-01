'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useEffect, useId, useState, type FormEvent } from 'react';

import { formatToman, planFromParam, usePlusPlans, usePlusQuote, type PlusQuote } from '@/entities/plus';
import { goToGateway, useCheckout } from '@/features/purchase-plus';
import { ApiError, getApiErrorCode, getApiErrorStatus, type ApiEnvelope } from '@/shared/api';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
  PrimaryButton,
  ProgressSteps,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { useNavMode } from '@/widgets/bottom-nav';

import { codeState } from '../lib/code-state';

/** The server's localized `errors.discount_code[0]` of a rejected code (422 `discount_*`). */
function discountError(error: unknown): string | null | undefined {
  if (getApiErrorStatus(error) !== 422 || !getApiErrorCode(error)?.startsWith('discount_')) return undefined;
  const body = error instanceof ApiError ? (error.response?.data as ApiEnvelope<unknown> | undefined) : undefined;
  const line = body?.errors?.discount_code?.[0];
  return typeof line === 'string' && line.trim() ? line : null;
}

/**
 * «تأیید و پرداخت» (`/plus/checkout?plan=<id>`, B-N2-07, `nbl_Prem_Checkout`):
 * the server-priced summary (subtotal, discount, VAT, total — `POST /plus/checkout`
 * with `preview: true`), a discount-code field checked the same way, the payment
 * methods (web: bank gateway only — Café Bazaar / Myket are Android-only and
 * hidden) and «پرداخت و فعال‌سازی», which creates the invoice and sends the
 * browser to the gateway. A 100% discount is settled at once → success.
 */
export function CheckoutPage() {
  const t = useTranslations('plus');
  const router = useRouter();
  const params = useSearchParams();
  const loc = useLocale() as Locale;
  const navMode = useNavMode().mode;
  const plans = usePlusPlans();
  const plan = plans.data ? planFromParam(plans.data.plans, params.get('plan')) : null;
  const planId = plan?.id ?? null;

  const [code, setCode] = useState('');
  const [applied, setApplied] = useState<string | null>(null);
  const base = usePlusQuote(planId);
  const withCode = usePlusQuote(applied ? planId : null, applied);
  const checkout = useCheckout();
  const [payError, setPayError] = useState<string | null>(null);
  const codeId = useId();

  useEffect(() => {
    if (navMode === 'teen') router.replace('/profile');
  }, [navMode, router]);

  const codeError = applied && withCode.isError ? discountError(withCode.error) : undefined;
  const codeQuoted = applied && withCode.data ? withCode.data : undefined;
  const codeAccepted = codeState(codeQuoted) === 'accepted';
  const codeOutranked = codeState(codeQuoted) === 'outranked';
  const quote: PlusQuote | undefined = codeQuoted ?? base.data;

  const onApply = (event: FormEvent) => {
    event.preventDefault();
    const next = code.trim().toUpperCase();
    setApplied(next || null);
  };

  const onPay = () => {
    if (!planId) return;
    setPayError(null);
    checkout.mutate(
      { planId, discountCode: codeAccepted ? applied : null },
      {
        onSuccess: (result) => {
          if (!result.payment) {
            router.replace('/plus/success');
          } else if (!goToGateway(result.payment.redirectUrl)) {
            setPayError(t('checkout.payFailed'));
          }
        },
        onError: (error) =>
          setPayError(getApiErrorStatus(error) === 503 ? t('checkout.unavailable') : t('checkout.payFailed')),
      },
    );
  };

  const toman = (rials: number) => t('toman', { amount: formatToman(rials, loc) });

  let body;
  if (plans.isPending || (planId !== null && base.isPending)) {
    body = (
      <SkeletonGroup label={t('loading')} className="plus-checkout">
        <Skeleton shape="card" />
        <Skeleton shape="block" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (plans.isError || base.isError || !plan || !quote) {
    body = (
      <EmptyState
        icon="crown"
        title={t('error.title')}
        body={plans.data && !plan ? t('plans.empty') : t('error.body')}
        action={
          <PrimaryButton
            icon="refresh"
            loading={plans.isFetching || base.isFetching}
            onClick={() => void (plans.isError ? plans.refetch() : base.refetch())}
          >
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
              {t('checkout.summary', { plan: quote.plan?.title ?? plan.title })}
            </h2>
            <button type="button" className="plus-link is-brand" onClick={() => router.push(`/plus/plans?plan=${plan.id}`)}>
              {t('checkout.change')}
            </button>
          </div>
          <dl className="plus-lines">
            <div className="plus-line">
              <dt>{t('checkout.price')}</dt>
              <dd>{toman(quote.subtotal)}</dd>
            </div>
            {quote.discount > 0 ? (
              <div className="plus-line is-discount">
                <dt>
                  {quote.discountCode ? (
                    <>
                      {t('checkout.codeLabel')} <bdi dir="ltr">{quote.discountCode}</bdi>
                    </>
                  ) : (
                    t('checkout.offer')
                  )}
                </dt>
                <dd>{t('checkout.minus', { amount: formatToman(quote.discount, loc) })}</dd>
              </div>
            ) : null}
            {quote.vat > 0 ? (
              <div className="plus-line">
                <dt>{t('checkout.vat')}</dt>
                <dd>{toman(quote.vat)}</dd>
              </div>
            ) : null}
            <div className="plus-line is-total">
              <dt>{t('checkout.total')}</dt>
              <dd>
                <span className="plus-total-num" aria-hidden>
                  {formatToman(quote.total, loc)}
                </span>
                <span className="sr-only">{toman(quote.total)}</span>
              </dd>
            </div>
          </dl>
        </section>

        <form className="plus-code" onSubmit={onApply} noValidate>
          <label htmlFor={codeId} className="sr-only">
            {t('checkout.codeLabel')}
          </label>
          <input
            id={codeId}
            className="plus-code-input"
            dir="ltr"
            value={code}
            maxLength={64}
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            placeholder={t('checkout.codePlaceholder')}
            aria-invalid={codeError !== undefined || undefined}
            aria-describedby={codeError !== undefined ? `${codeId}-err` : undefined}
            data-state={codeAccepted ? 'ok' : codeError !== undefined ? 'error' : undefined}
            onChange={(e) => {
              setCode(e.target.value);
              if (applied) setApplied(null);
            }}
          />
          {codeAccepted ? (
            <button
              type="button"
              className="plus-code-btn is-applied"
              aria-label={t('checkout.removeCode')}
              onClick={() => {
                setApplied(null);
                setCode('');
              }}
            >
              <Icon name="check" size={16} strokeWidth={2.4} />
              {t('checkout.applied')}
            </button>
          ) : (
            <button
              type="submit"
              className="plus-code-btn"
              disabled={!code.trim() || withCode.isFetching}
              aria-busy={withCode.isFetching || undefined}
            >
              {t('checkout.apply')}
            </button>
          )}
        </form>
        {codeError !== undefined ? (
          <p id={`${codeId}-err`} className="plus-field-error" role="alert">
            {codeError ?? t('checkout.codeInvalid')}
          </p>
        ) : applied && withCode.isError ? (
          <p className="plus-field-error" role="alert">
            {t('error.body')}
          </p>
        ) : codeOutranked ? (
          <p className="plus-notice" role="status">
            {t('checkout.codeOutranked')}
          </p>
        ) : null}

        <section className="plus-methods" aria-labelledby="plus-methods-title">
          <h2 id="plus-methods-title" className="plus-section-title">
            {t('checkout.methods')}
          </h2>
          {/* Web shows the bank gateway only; Bazaar / Myket in-app billing is Android-only. */}
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
            {t('checkout.secure')}
          </p>
        </section>
      </div>
    );
  }

  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll">
        <ScreenHeader
          title={t('checkout.title')}
          onBack={() => router.push(plan ? `/plus/plans?plan=${plan.id}` : '/plus/plans')}
          backLabel={t('back')}
          center={<ProgressSteps total={3} current={2} label={t('steps', { current: formatNumber(2, loc), total: formatNumber(3, loc) })} className="plus-steps" />}
        />
        <div className="plus-intro">
          <h1 className="plus-title">{t('checkout.title')}</h1>
        </div>
        {body}
      </div>
      {quote && plan ? (
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
