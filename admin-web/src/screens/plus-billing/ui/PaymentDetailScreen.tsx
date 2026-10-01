'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError, isApiError } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatDate, formatDateTime, formatNumber } from '@/shared/lib';
import { Button, LoadGate, PageHeader, Panel, TextArea, toast, useNotifyError } from '@/shared/ui';

import { paymentsApi, type Payment } from '../api/billing';
import { FormDialog } from './FormDialog';
import { StatusBadge, Toman, useCanManage, UserCell } from './parts';

/** /plus/payments/:id — invoice, receipt (no card data), periods bought, admin actions, refund (B-N2-09). */
export function PaymentDetailScreen({ id }: { id: number }) {
  const t = useTranslations('plus.payments');
  const detail = paymentsApi.useDetail(id);
  return (
    <LoadGate queries={[detail]} header={<PageHeader title={t('detailTitle')} backHref="/plus/payments" backLabel={t('backToList')} />}>
      {() => (detail.data ? <PaymentDetail payment={detail.data.payment} /> : null)}
    </LoadGate>
  );
}

type RefundMode = 'gateway' | 'manual';

function PaymentDetail({ payment }: { payment: Payment }) {
  const t = useTranslations('plus.payments');
  const tp = useTranslations('plus');
  const locale = useLocale();
  const localize = useLocalized();
  const canManage = useCanManage();
  const [mode, setMode] = useState<RefundMode | null>(null);
  const [gatewayUnsupported, setGatewayUnsupported] = useState(false);
  const pct = formatNumber(payment.vat_rate_bps / 100, locale);
  const canRefund = canManage && payment.refund.refundable;

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={<span dir="ltr">{payment.reference}</span>}
        backHref="/plus/payments"
        backLabel={t('backToList')}
        meta={
          <>
            <StatusBadge status={payment.status} />
            <span>{formatDateTime(payment.created_at, locale)}</span>
          </>
        }
        actions={
          canRefund ? (
            <>
              {payment.refund.gateway_available && !gatewayUnsupported ? (
                <Button variant="danger" onClick={() => setMode('gateway')}>
                  {t('refundGateway')}
                </Button>
              ) : null}
              <Button onClick={() => setMode('manual')}>{t('refundManual')}</Button>
            </>
          ) : null
        }
      />
      {canRefund && (!payment.refund.gateway_available || gatewayUnsupported) ? (
        <p role="note" className="m-0 rounded-lg border border-line p-3 text-sm text-ink-3">
          {t('gatewayUnavailableNote')}
        </p>
      ) : null}

      <Panel title={t('amounts')} bodyClassName="">
        <dl className="facts">
          <div>
            <dt>{t('subtotal')}</dt>
            <dd>
              <Toman rials={payment.subtotal_rials} />
            </dd>
          </div>
          <div>
            <dt>{t('discount')}</dt>
            <dd className="flex flex-col items-start">
              <Toman rials={payment.discount_rials} />
              {payment.discount_code ? (
                <span dir="ltr" className="text-xs text-muted">
                  {payment.discount_code}
                </span>
              ) : null}
            </dd>
          </div>
          <div>
            <dt>{t('vatLine', { rate: pct })}</dt>
            <dd>
              <Toman rials={payment.vat_rials} />
            </dd>
          </div>
          <div>
            <dt>{t('total')}</dt>
            <dd>
              <Toman rials={payment.total_rials} />
            </dd>
          </div>
        </dl>
      </Panel>

      <Panel title={t('details')} bodyClassName="">
        <dl className="facts">
          <div>
            <dt>{t('payer')}</dt>
            <dd>
              <UserCell user={payment.user} />
            </dd>
          </div>
          <div>
            <dt>{t('plan')}</dt>
            <dd>
              {payment.plan ? localize(payment.plan.title) || payment.plan.code : tp('plans.months', { count: payment.duration_months })}
            </dd>
          </div>
          <div>
            <dt>{t('gateway')}</dt>
            <dd dir="ltr" className="rtl:text-end">
              {payment.gateway ?? t('noGateway')}
            </dd>
          </div>
          <div>
            <dt>{t('bankRef')}</dt>
            <dd dir="ltr" className="break-all rtl:text-end">
              {payment.receipt?.ref_id ?? '—'}
            </dd>
          </div>
          <div>
            <dt>{t('paidAt')}</dt>
            <dd>{formatDateTime(payment.paid_at, locale) || '—'}</dd>
          </div>
        </dl>
      </Panel>

      <Panel title={t('periods')}>
        {payment.subscriptions.length ? (
          <ul className="m-0 flex list-none flex-col gap-2 p-0">
            {payment.subscriptions.map((s) => (
              <li key={s.id} className="flex flex-wrap items-center gap-2 text-sm">
                <StatusBadge status={s.status} />
                <span className="tabular-nums text-ink-3">
                  {formatDate(s.starts_at, locale)} — {formatDate(s.ends_at, locale)}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="m-0 text-ink-3">{t('noPeriods')}</p>
        )}
      </Panel>

      <Panel title={t('history')}>
        {payment.actions.length ? (
          <ul className="m-0 flex list-none flex-col gap-3 p-0">
            {payment.actions.map((a) => (
              <li key={a.id} className="flex flex-col gap-0.5 text-sm">
                <span className="font-semibold">
                  {tp.has(`actions.${a.action}` as 'actions.invoice.refund') ? tp(`actions.${a.action}` as 'actions.invoice.refund') : a.action}
                  {a.amount_rials ? <> · <Toman rials={a.amount_rials} /></> : null}
                </span>
                <span className="text-xs text-muted">
                  {[a.admin.name, formatDateTime(a.created_at, locale), a.gateway_ref].filter(Boolean).join(' · ')}
                </span>
                {a.note ? <span className="text-ink-3" dir="auto">{a.note}</span> : null}
              </li>
            ))}
          </ul>
        ) : (
          <p className="m-0 text-ink-3">{t('noHistory')}</p>
        )}
      </Panel>

      <RefundDialog
        payment={payment}
        mode={mode}
        onClose={() => setMode(null)}
        onUnsupported={() => {
          setGatewayUnsupported(true);
          setMode('manual');
        }}
      />
    </div>
  );
}

function RefundDialog({
  payment,
  mode,
  onClose,
  onUnsupported,
}: {
  payment: Payment;
  mode: RefundMode | null;
  onClose: () => void;
  onUnsupported: () => void;
}) {
  const t = useTranslations('plus.payments');
  const notifyError = useNotifyError();
  const refund = paymentsApi.useAction('refund');
  const [note, setNote] = useState('');
  const err = (name: string) => fieldError(refund.error, name);

  const close = () => {
    refund.reset();
    setNote('');
    onClose();
  };
  const submit = () => {
    if (!mode) return;
    refund.mutate(
      { id: payment.id, body: { mode, note: note.trim() || null } },
      {
        onSuccess: () => {
          toast.success(mode === 'manual' ? t('markedRefunded') : t('refunded'));
          close();
        },
        onError: (error) => {
          if (isApiError(error) && error.code === 'refund_not_supported') {
            refund.reset();
            toast.error(t('unsupportedToast'));
            onUnsupported();
            return;
          }
          notifyError(error);
        },
      },
    );
  };

  return (
    <FormDialog
      open={mode !== null}
      title={mode === 'manual' ? t('refundManualTitle') : t('refundGatewayTitle')}
      onClose={close}
      onSubmit={submit}
      submitLabel={mode === 'manual' ? t('refundManualSubmit') : t('refundGatewaySubmit')}
      saving={refund.isPending}
      tone="danger"
    >
      <p className="m-0 text-sm text-ink-3">
        {mode === 'manual' ? t('refundManualIntro') : t('refundGatewayIntro')} <Toman rials={payment.refund.amount_rials} />
      </p>
      <TextArea
        label={t('note')}
        hint={mode === 'manual' ? t('noteManualHint') : t('noteHint')}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        rows={3}
        maxLength={500}
        required={mode === 'manual'}
        error={err('note')}
      />
    </FormDialog>
  );
}
