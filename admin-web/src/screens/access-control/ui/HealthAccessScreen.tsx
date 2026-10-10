'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCan, useCurrentAdmin } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { formatDate, formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  ErrorState,
  PageHeader,
  Pagination,
  Panel,
  Select,
  Skeleton,
  TextArea,
  TextInput,
  toast,
  useNotifyError,
  type BadgeTone,
  type Column,
} from '@/shared/ui';

import {
  ACCESS_STATUSES,
  HEALTH_ACCESS_MINUTES,
  useAccessRequests,
  useDecideAccess,
  useHealthData,
  useRequestAccess,
  type AccessRequest,
} from '../api/access';

const TONE: Record<AccessRequest['status'], BadgeTone> = {
  pending: 'amber',
  approved: 'green',
  rejected: 'red',
  revoked: 'neutral',
  expired: 'neutral',
};

/**
 * /health-access — temporary access to one user's health data: clinical support asks (reason + ticket), another
 * super admin approves, the grant lasts 30 minutes, every view is audited (admin-api.md §21).
 */
export function HealthAccessScreen() {
  const t = useTranslations('access');
  const locale = useLocale();
  const canRequest = useCan('health', 'manage');
  const [viewing, setViewing] = useState<number | null>(null);

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('healthTitle')} meta={t('healthSubtitle', { minutes: formatNumber(HEALTH_ACCESS_MINUTES, locale) })} />
      {canRequest ? <RequestForm /> : null}
      {viewing !== null ? <HealthPanel userId={viewing} onClose={() => setViewing(null)} /> : null}
      <Queue onView={setViewing} />
    </div>
  );
}

function RequestForm() {
  const t = useTranslations('access');
  const save = useRequestAccess();
  const notifyError = useNotifyError();
  const [userId, setUserId] = useState('');
  const [reason, setReason] = useState('');
  const [ticket, setTicket] = useState('');
  const err = (f: string) => fieldError(save.error, f);

  return (
    <Panel title={t('requestTitle')}>
      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate(
            { user_id: userId, reason, ticket },
            {
              onSuccess: () => {
                toast.success(t('requested'));
                setUserId('');
                setReason('');
                setTicket('');
              },
              onError: (e2) => {
                if (!fieldError(e2, 'user_id') && !fieldError(e2, 'reason') && !fieldError(e2, 'ticket')) notifyError(e2);
              },
            },
          );
        }}
      >
        <TextInput
          label={t('userId')}
          type="number"
          required
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
          error={err('user_id')}
        />
        <TextInput
          label={t('ticket')}
          required
          maxLength={64}
          dir="ltr"
          value={ticket}
          onChange={(e) => setTicket(e.target.value)}
          error={err('ticket')}
        />
        <TextArea
          className="sm:col-span-2"
          label={t('reason')}
          hint={t('reasonHint')}
          required
          minLength={10}
          maxLength={500}
          rows={3}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          error={err('reason')}
        />
        <div className="sm:col-span-2">
          <Button type="submit" variant="primary" loading={save.isPending}>
            {t('submit')}
          </Button>
        </div>
      </form>
    </Panel>
  );
}

function Queue({ onView }: { onView: (userId: number) => void }) {
  const t = useTranslations('access');
  const locale = useLocale();
  const me = useCurrentAdmin();
  const isSuper = me?.role === 'super';
  const list = useListParams({ status: '' });
  const query = useAccessRequests(list.query);
  const decide = useDecideAccess();
  const notifyError = useNotifyError();

  const act = (id: number, action: 'approve' | 'reject' | 'revoke') =>
    decide.mutate(
      { id, action },
      {
        onSuccess: () => toast.success(t(action === 'approve' ? 'approved' : action === 'reject' ? 'rejected' : 'revoked')),
        onError: notifyError,
      },
    );

  const columns: Column<AccessRequest>[] = [
    {
      key: 'time',
      header: t('time'),
      cell: (r) => formatDateTime(r.created_at, locale) || '—',
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    { key: 'requester', header: t('requester'), cell: (r) => r.admin.name ?? `#${r.admin.id}` },
    { key: 'user', header: t('user'), cell: (r) => <span dir="ltr">#{formatNumber(r.user_id, 'en')}</span> },
    {
      key: 'reason',
      header: t('reason'),
      cell: (r) => (
        <span className="flex flex-col">
          <span>{r.reason}</span>
          <span className="text-ink-3" dir="ltr">
            {r.ticket}
          </span>
        </span>
      ),
    },
    {
      key: 'status',
      header: t('status'),
      cell: (r) => (
        <span className="flex flex-col gap-1">
          <Badge tone={TONE[r.status]}>{t(`statuses.${r.status}`)}</Badge>
          {r.status === 'approved' && r.expires_at ? (
            <span className="text-ink-3 text-xs">{t('healthExpires', { time: formatDateTime(r.expires_at, locale) })}</span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{t('action')}</span>,
      cell: (r) => (
        <span className="flex flex-wrap gap-2">
          {r.status === 'approved' && r.admin.id === me?.id ? (
            <Button size="sm" variant="primary" onClick={() => onView(r.user_id)}>
              {t('view')}
            </Button>
          ) : null}
          {isSuper && r.status === 'pending' && r.admin.id !== me?.id ? (
            <>
              <Button size="sm" variant="primary" onClick={() => act(r.id, 'approve')} disabled={decide.isPending}>
                {t('approve')}
              </Button>
              <Button size="sm" variant="ghost" onClick={() => act(r.id, 'reject')} disabled={decide.isPending}>
                {t('reject')}
              </Button>
            </>
          ) : null}
          {(r.status === 'pending' || r.status === 'approved') && (isSuper || r.admin.id === me?.id) ? (
            <Button size="sm" variant="ghost" onClick={() => act(r.id, 'revoke')} disabled={decide.isPending}>
              {t('revoke')}
            </Button>
          ) : null}
        </span>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('queueTitle')}
      bodyClassName=""
      actions={
        <Select
          aria-label={t('status')}
          value={list.params.filters.status}
          onChange={(e) => list.setFilter('status', e.target.value)}
          options={[
            { value: '', label: t('allStatuses') },
            ...ACCESS_STATUSES.filter((s) => s !== 'expired').map((s) => ({ value: s, label: t(`statuses.${s}`) })),
          ]}
        />
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(r) => r.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        emptyText={t('queueEmpty')}
        caption={t('queueTitle')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}

function HealthPanel({ userId, onClose }: { userId: number; onClose: () => void }) {
  const t = useTranslations('access');
  const locale = useLocale();
  const query = useHealthData(userId);
  const d = query.data;
  const rows: [string, string][] = d?.profile
    ? [
        ['birthday', formatDate(d.profile.birthday, locale) || '—'],
        ['last_period_start', formatDate(d.profile.last_period_start, locale) || '—'],
        ['cycle_duration', d.profile.cycle_duration === null ? '—' : formatNumber(d.profile.cycle_duration, locale)],
        ['period_duration', d.profile.period_duration === null ? '—' : formatNumber(d.profile.period_duration, locale)],
        ['pregnancy_intention', d.profile.pregnancy_intention ?? '—'],
        ['user_goal', d.profile.user_goal ?? '—'],
        ['health_logs', formatNumber(d.stats.health_logs, locale)],
        ['reminders', formatNumber(d.stats.reminders, locale)],
      ]
    : [];

  return (
    <Panel
      title={t('healthData', { id: userId })}
      actions={
        <Button size="sm" variant="ghost" onClick={onClose}>
          {t('close')}
        </Button>
      }
    >
      {query.isPending ? (
        <Skeleton className="h-24 w-full" />
      ) : query.error ? (
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      ) : (
        <div className="flex flex-col gap-3">
          {d?.expires_at ? (
            <p className="m-0 text-sm text-ink-3">{t('healthExpires', { time: formatDateTime(d.expires_at, locale) })}</p>
          ) : null}
          {d?.profile ? (
            <dl className="grid gap-x-6 gap-y-2 sm:grid-cols-2">
              {rows.map(([k, v]) => (
                <div key={k} className="flex justify-between gap-3 border-b border-line py-1">
                  <dt className="text-ink-3">{t(`f.${k}` as 'f.birthday')}</dt>
                  <dd className="m-0 font-medium">{v}</dd>
                </div>
              ))}
            </dl>
          ) : (
            <p className="m-0 text-ink-3">{t('noProfile')}</p>
          )}
        </div>
      )}
    </Panel>
  );
}
