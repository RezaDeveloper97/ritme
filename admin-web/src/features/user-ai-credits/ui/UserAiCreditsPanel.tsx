'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { fieldError } from '@/shared/api';
import { formatDate, formatDateTime, formatNumber, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  Button,
  confirm,
  DataTable,
  ErrorState,
  FormDialog,
  Panel,
  Select,
  Skeleton,
  TextArea,
  TextInput,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { useAdjustCredits, useUserCredits, type CreditGrant, type CreditSpend, type UserCredits } from '../api/credits';

/**
 * A user's AI credits on /users/:id: balance (the running period's credits + the never-expiring extra ones), the
 * credit buckets, the last 30 uses and the manual adjustments; super admins can gift or take back extra credits.
 * The screen passes `canView` (the billing capability — the API answers 403 without it; the panel then renders
 * nothing) and `canManage` (super admin: the adjust button).
 */
export function UserAiCreditsPanel({ userId, canView, canManage }: { userId: number; canView: boolean; canManage: boolean }) {
  const t = useTranslations('userAiCredits');
  const query = useUserCredits(userId, canView);
  const [adjusting, setAdjusting] = useState(false);
  if (!canView) return null;

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        canManage && query.data ? (
          <Button size="sm" onClick={() => setAdjusting(true)}>
            {t('adjust')}
          </Button>
        ) : null
      }
    >
      {query.error && !query.data ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : !query.data ? (
        <div className="p-4 sm:p-5">
          <Skeleton className="h-24 w-full" />
        </div>
      ) : (
        <Body data={query.data} />
      )}
      {adjusting && query.data ? (
        <AdjustDialog userId={userId} data={query.data} onClose={() => setAdjusting(false)} />
      ) : null}
    </Panel>
  );
}

function Body({ data }: { data: UserCredits }) {
  const t = useTranslations('userAiCredits');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  const b = data.balance;

  const tiles = [
    {
      key: 'period',
      value: b.active ? `${n(b.plan_remaining)} / ${n(b.plan_total)}` : '—',
      sub: b.active
        ? t('periodUntil', { kind: t(`periodKinds.${b.period === 'trial' ? 'trial' : 'plus'}`), date: formatDate(b.period_ends_at, locale) })
        : t('noSubscription'),
    },
    { key: 'extra', value: n(b.extra), sub: b.extra_locked ? t('extraLocked') : t('extraHint') },
    { key: 'spendable', value: n(b.spendable), sub: null },
  ];

  const grantColumns: Column<CreditGrant>[] = [
    {
      key: 'kind',
      header: t('kind'),
      cell: (g) => (
        <span className="flex flex-wrap items-center gap-1">
          {t.has(`kinds.${g.kind}` as 'kinds.admin') ? t(`kinds.${g.kind}` as 'kinds.admin') : g.kind}
          {!g.current ? <Badge tone="neutral">{t('ended')}</Badge> : null}
        </span>
      ),
    },
    { key: 'amount', header: t('amount'), cell: (g) => n(g.amount), className: 'cell-num' },
    { key: 'remaining', header: t('remaining'), cell: (g) => n(g.remaining), className: 'cell-num' },
    { key: 'until', header: t('validUntil'), cell: (g) => (g.valid_until ? formatDate(g.valid_until, locale) : t('noExpiry')) },
    { key: 'note', header: t('note'), cell: (g) => <span dir="auto">{g.note ?? '—'}</span> },
    { key: 'at', header: t('createdAt'), cell: (g) => formatDateTime(g.created_at, locale) },
  ];

  const spendColumns: Column<CreditSpend>[] = [
    { key: 'feature', header: t('feature'), cell: (s) => <code dir="ltr">{s.feature}</code> },
    { key: 'credits', header: t('credits'), cell: (s) => n(s.credits), className: 'cell-num' },
    {
      key: 'from',
      header: t('from'),
      cell: (s) => t('split', { plan: n(s.plan_credits), extra: n(s.extra_credits) }),
      className: 'text-xs text-ink-3',
    },
    {
      key: 'status',
      header: t('status'),
      cell: (s) => <Badge tone={s.status === 'refunded' ? 'amber' : 'green'}>{t.has(`statuses.${s.status}` as 'statuses.spent') ? t(`statuses.${s.status}` as 'statuses.spent') : s.status}</Badge>,
    },
    { key: 'at', header: t('at'), cell: (s) => formatDateTime(s.created_at, locale) },
  ];

  return (
    <>
      <dl className="ledger m-0">
        {tiles.map((tile) => (
          <div key={tile.key} className="ledger-cell">
            <dt className="text-[13px] font-semibold text-ink-3">{t(`tiles.${tile.key}` as 'tiles.extra')}</dt>
            <dd className="m-0 mt-1 flex flex-col">
              <span className="ledger-value">{tile.value}</span>
              {tile.sub ? <span className="text-xs text-ink-3">{tile.sub}</span> : null}
            </dd>
          </div>
        ))}
      </dl>
      <h3 className="m-0 px-4 pt-4 text-sm font-semibold sm:px-5">{t('grants')}</h3>
      <DataTable columns={grantColumns} items={data.grants} rowKey={(g) => g.id} emptyText={t('noGrants')} caption={t('grants')} />
      <h3 className="m-0 px-4 pt-4 text-sm font-semibold sm:px-5">{t('spends')}</h3>
      <DataTable columns={spendColumns} items={data.spends} rowKey={(s) => s.id} emptyText={t('noSpends')} caption={t('spends')} />
      {data.adjustments.length > 0 ? (
        <>
          <h3 className="m-0 px-4 pt-4 text-sm font-semibold sm:px-5">{t('adjustments')}</h3>
          <ul className="m-0 flex list-none flex-col gap-2 p-4 text-sm sm:p-5">
            {data.adjustments.map((a) => {
              const delta = a.details?.delta ?? 0;
              return (
                <li key={a.id} className="flex flex-wrap items-center gap-2">
                  <Badge tone={delta > 0 ? 'green' : 'red'}>
                    {delta > 0
                      ? t('granted', { n: n(a.details?.granted ?? delta) })
                      : t('taken', { n: n(a.details?.taken ?? 0), asked: n(-delta) })}
                  </Badge>
                  <span dir="auto">{a.note ?? ''}</span>
                  <span className="text-xs text-muted">
                    {a.admin.name ?? '—'} · {formatDateTime(a.created_at, locale)}
                  </span>
                </li>
              );
            })}
          </ul>
        </>
      ) : null}
    </>
  );
}

function AdjustDialog({ userId, data, onClose }: { userId: number; data: UserCredits; onClose: () => void }) {
  const t = useTranslations('userAiCredits');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const adjust = useAdjustCredits(userId);
  const [mode, setMode] = useState<'grant' | 'deduct'>('grant');
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const n = (v: number) => formatNumber(v, locale);

  const submit = async () => {
    const value = toIntOrNull(amount);
    if (!value || value <= 0 || !note.trim()) {
      toast.error(t('adjustInvalid'));
      return;
    }
    const message = mode === 'grant' ? t('confirmGrant', { n: n(value) }) : t('confirmDeduct', { n: n(value), extra: n(data.balance.extra) });
    if (!(await confirm({ message, tone: mode === 'deduct' ? 'danger' : 'default' }))) return;
    adjust.mutate(
      { delta: mode === 'grant' ? value : -value, note: note.trim() },
      {
        onSuccess: (res) => {
          toast.success(res.delta > 0 ? t('grantedToast', { n: n(res.granted) }) : t('takenToast', { n: n(res.taken) }));
          onClose();
        },
        onError: notifyError,
      },
    );
  };

  return (
    <FormDialog
      open
      title={t('adjustTitle')}
      onClose={onClose}
      onSubmit={() => void submit()}
      submitLabel={t('adjustSubmit')}
      saving={adjust.isPending}
      tone={mode === 'deduct' ? 'danger' : 'primary'}
    >
      <Select
        label={t('mode')}
        value={mode}
        onChange={(e) => setMode(e.target.value as 'grant' | 'deduct')}
        options={[
          { value: 'grant', label: t('modes.grant') },
          { value: 'deduct', label: t('modes.deduct') },
        ]}
      />
      <TextInput
        label={t('amount')}
        hint={mode === 'deduct' ? t('deductHint', { extra: n(data.balance.extra) }) : t('grantHint')}
        type="number"
        min={1}
        max={data.limits.max_delta}
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        required
        error={fieldError(adjust.error, 'delta')}
      />
      <TextArea
        label={t('note')}
        hint={t('noteHint')}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        maxLength={data.limits.max_note}
        required
        rows={3}
        error={fieldError(adjust.error, 'note')}
      />
    </FormDialog>
  );
}
