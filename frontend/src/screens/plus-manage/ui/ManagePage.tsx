'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  formatToman,
  membershipOf,
  remainingShare,
  upgradePlan,
  usePlusHistory,
  usePlusPlans,
  usePlusStatus,
  type PlusInvoice,
  type PlusStatus,
} from '@/entities/plus';
import { useCancelPlus } from '@/features/purchase-plus';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatLongDate, formatNumber } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  IconCircle,
  ListGroup,
  ListRow,
  PrimaryButton,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  Switch,
  type Tone,
} from '@/shared/ui';

const INVOICE_TONE: Record<string, Tone> = { paid: 'success', pending: 'warm', failed: 'danger', expired: 'neutral', refunded: 'neutral' };
const INVOICE_STATUSES = ['paid', 'pending', 'failed', 'expired', 'refunded'] as const;
type InvoiceStatusKey = (typeof INVOICE_STATUSES)[number] | 'other';

/**
 * «اشتراک من» (`/plus/manage`, B-N2-07, `nbl_Prem_Manage` / `nbd_Prem_Manage`):
 * days-left ring with plan + renewal date, upgrade to the longest plan,
 * auto-renew switch (off = `POST /plus/cancel`; the API has no way back on),
 * payment history (expands in place, `GET /plus/history`) and «لغو اشتراک».
 * A running trial shows its own ring and a «خرید اشتراک» row instead.
 */
export function ManagePage() {
  const t = useTranslations('plus');
  const router = useRouter();
  const status = usePlusStatus();

  let body;
  if (status.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="plus-manage">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (status.isError || !status.data) {
    body = (
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
    );
  } else {
    body = <ManageBody status={status.data} />;
  }

  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll">
        <ScreenHeader title={t('manage.title')} onBack={() => router.push('/profile')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}

function ManageBody({ status }: { status: PlusStatus }) {
  const t = useTranslations('plus');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const plans = usePlusPlans();
  const cancel = useCancelPlus();
  const [confirm, setConfirm] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [notice, setNotice] = useState<{ text: string; error?: boolean } | null>(null);
  const history = usePlusHistory();
  const historyId = useId();
  const membership = membershipOf(status);

  if (membership.kind === 'none') {
    return (
      <EmptyState
        icon="crown"
        title={t('manage.noneTitle')}
        body={t('manage.noneBody')}
        action={<PrimaryButton onClick={() => router.push('/plus')}>{t('manage.noneCta')}</PrimaryButton>}
      />
    );
  }

  const sub = membership.kind === 'subscription' ? status.subscription : null;
  const trial = membership.kind === 'trial' ? status.trial : null;
  const period = sub ?? trial;
  if (!period) return null;
  const startsAt = sub ? sub.startsAt : (trial?.startedAt ?? '');
  const endDate = formatLongDate(new Date(period.endsAt), loc);
  const days = formatNumber(period.daysLeft, loc);
  const upgrade = sub ? upgradePlan(plans.data?.plans ?? [], sub.plan?.durationMonths ?? null) : null;
  const invoices = history.data ?? [];
  const paidCount = invoices.filter((i) => i.status === 'paid').length;

  const onConfirmCancel = () => {
    cancel.mutate(undefined, {
      onSuccess: () => {
        setConfirm(false);
        setNotice({ text: t('manage.canceled') });
      },
      onError: () => {
        setConfirm(false);
        setNotice({ text: t('manage.cancelFailed'), error: true });
      },
    });
  };

  return (
    <div className="plus-manage">
      <section className="nb-card plus-status-card" aria-label={t('manage.title')}>
        <div className="plus-status-text">
          <p className="plus-status-title">
            <Icon name="crown" size={18} strokeWidth={1.8} className="plus-status-crown" />
            {sub ? t('manage.plan', { plan: sub.plan?.title ?? '' }) : t('manage.trial')}
          </p>
          <p className="plus-status-sub">
            {sub?.autoRenew ? t('manage.renewsOn', { date: endDate }) : t('manage.endsOn', { date: endDate })}
          </p>
        </div>
        <ProgressRing
          value={remainingShare(period.daysLeft, startsAt, period.endsAt)}
          label={t('manage.ringLabel')}
          valueText={t('manage.ringValue', { days })}
          size={92}
          thickness={8}
          tone="warm"
          className="plus-ring"
        >
          <span className="plus-ring-num">{days}</span>
          <span className="plus-ring-cap">{t('manage.daysLeft')}</span>
        </ProgressRing>
      </section>

      {notice ? (
        <p className={notice.error ? 'plus-field-error is-center' : 'plus-notice'} role={notice.error ? 'alert' : 'status'}>
          {notice.text}
        </p>
      ) : null}

      <ListGroup className="plus-manage-list">
        {upgrade ? (
          <ListRow
            icon="crown"
            iconTone="warm"
            iconOutlined
            title={t('manage.upgrade', { plan: upgrade.title })}
            description={
              upgrade.savingsPercent > 0
                ? t('manage.upgradeSub', { percent: formatNumber(upgrade.savingsPercent, loc) })
                : undefined
            }
            onClick={() => router.push(`/plus/checkout?plan=${upgrade.id}`)}
          />
        ) : null}
        {trial ? (
          <ListRow
            icon="crown"
            iconTone="warm"
            iconOutlined
            title={t('manage.buy')}
            description={t('manage.buySub')}
            onClick={() => router.push('/plus/plans')}
          />
        ) : null}
        {sub ? (
          <ListRow
            id="plus-renew"
            icon="clock"
            iconTone="brand"
            iconOutlined
            title={t('manage.autoRenew')}
            description={sub.autoRenew ? t('manage.autoRenewOn') : t('manage.autoRenewOff')}
            trailing={
              <Switch
                checked={sub.autoRenew}
                labelledBy="plus-renew-title"
                // Turning it off is the cancel flow; the API has no «resume», so off stays off.
                disabled={!sub.autoRenew || cancel.isPending}
                onCheckedChange={(next) => (next ? undefined : setConfirm(true))}
              />
            }
          />
        ) : null}
        <button
          type="button"
          className="nb-row is-action plus-history-toggle"
          aria-expanded={historyOpen}
          aria-controls={historyId}
          onClick={() => setHistoryOpen((v) => !v)}
        >
          <IconCircle icon="note" tone="data" outlined />
          <span className="nb-row-text">
            <span className="nb-row-title">{t('manage.history')}</span>
            <span className="nb-row-desc">
              {history.isPending
                ? t('loading')
                : paidCount
                  ? t('manage.historyCount', { n: paidCount, count: formatNumber(paidCount, loc) })
                  : t('manage.historyEmpty')}
            </span>
          </span>
          <Icon name="chevronDown" size={18} className="nb-row-chev plus-history-chev" />
        </button>
        {historyOpen ? (
          <div id={historyId} className="plus-history" aria-label={t('manage.historyLabel')}>
            {history.isError ? (
              <p className="plus-field-error">{t('manage.historyError')}</p>
            ) : invoices.length ? (
              <ul className="plus-invoices">
                {invoices.map((inv) => (
                  <InvoiceRow key={inv.reference} invoice={inv} />
                ))}
              </ul>
            ) : (
              <p className="plus-history-empty">{t('manage.historyEmpty')}</p>
            )}
          </div>
        ) : null}
      </ListGroup>

      {sub?.autoRenew ? (
        <button type="button" className="plus-cancel" onClick={() => setConfirm(true)}>
          {t('manage.cancel')}
        </button>
      ) : null}

      <AppSheet
        open={confirm}
        onClose={() => (cancel.isPending ? undefined : setConfirm(false))}
        size="half"
        title={t('manage.cancelTitle')}
        footer={
          <div className="plus-actions">
            <SecondaryButton onClick={() => setConfirm(false)} disabled={cancel.isPending}>
              {t('manage.cancelKeep')}
            </SecondaryButton>
            <SecondaryButton variant="text" className="plus-danger-text" loading={cancel.isPending} onClick={onConfirmCancel}>
              {t('manage.cancelConfirm')}
            </SecondaryButton>
          </div>
        }
      >
        <p className="plus-sheet-body">{t('manage.cancelBody', { date: endDate })}</p>
      </AppSheet>
    </div>
  );
}

function InvoiceRow({ invoice }: { invoice: PlusInvoice }) {
  const t = useTranslations('plus');
  const loc = useLocale() as Locale;
  const when = invoice.paidAt ?? invoice.createdAt;
  const key: InvoiceStatusKey = (INVOICE_STATUSES as readonly string[]).includes(invoice.status)
    ? (invoice.status as InvoiceStatusKey)
    : 'other';
  return (
    <li className="plus-invoice">
      <span className="plus-invoice-text">
        <span className="plus-invoice-title">
          {t('manage.invoice', {
            plan: invoice.plan?.title ?? '',
            date: when ? formatLongDate(new Date(when), loc) : '',
          })}
        </span>
        <span className="plus-invoice-amount">{t('toman', { amount: formatToman(invoice.total, loc) })}</span>
      </span>
      <StatusPill tone={INVOICE_TONE[invoice.status] ?? 'neutral'}>{t(`manage.invoiceStatus.${key}`)}</StatusPill>
    </li>
  );
}
