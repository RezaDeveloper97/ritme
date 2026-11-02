'use client';

import { useLocale, useTranslations } from 'next-intl';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useState } from 'react';

import { useCan, useCurrentAdmin } from '@/features/auth';
import { formatDateTime, formatNumber, nextSearch, useListParams } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  Icon,
  PageHeader,
  Pagination,
  Panel,
  TextInput,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { downloadAuditCsv, useAuditLog, useVerifyChain, type AuditRow } from '../api/access';

/** Chain status card (super only): a verify button walks the hash chain and names the first break. */
export function ChainStatusCard() {
  const t = useTranslations('access');
  const verify = useVerifyChain();
  const notifyError = useNotifyError();
  const r = verify.data;
  return (
    <Panel
      title={t('chainTitle')}
      actions={
        <Button size="sm" loading={verify.isPending} onClick={() => verify.mutate(undefined, { onError: notifyError })}>
          {t('chainVerify')}
        </Button>
      }
    >
      <p className="m-0 text-ink-3">{t('chainNote')}</p>
      <div role="status" aria-live="polite" className="mt-3">
        {r ? (
          r.ok ? (
            <Badge tone="green">{t('chainOk', { count: formatNumber(r.checked, 'en') })}</Badge>
          ) : (
            <div className="flex flex-col gap-1">
              <Badge tone="red">{t('chainBroken')}</Badge>
              {r.break ? (
                <span dir="ltr" className="text-sm">
                  #{r.break.id} · {r.break.reason}
                </span>
              ) : null}
            </div>
          )
        ) : (
          <span className="text-ink-3">{t('chainNotRun')}</span>
        )}
        {r?.head ? (
          <p dir="ltr" className="m-0 mt-2 break-all text-[13px] text-ink-3">
            head #{r.head.id} · {r.head.hash}
          </p>
        ) : null}
      </div>
    </Panel>
  );
}

const FILTERS = { action: '', admin_id: '', target_type: '', from: '', to: '' };

/** /audit-logs — who did what, when (Admin_Roles artboard, lower half); filters live in the URL. */
export function AuditLogScreen() {
  const t = useTranslations('access');
  const tr = useTranslations('roles');
  const locale = useLocale();
  const canExport = useCan('export');
  const isSuper = useCurrentAdmin()?.role === 'super';
  const notifyError = useNotifyError();
  const list = useListParams(FILTERS);
  const router = useRouter();
  const pathname = usePathname();
  const search = useSearchParams();
  const query = useAuditLog(list.query);
  const [draft, setDraft] = useState(list.params.filters);
  const [exporting, setExporting] = useState(false);

  const columns: Column<AuditRow>[] = [
    {
      key: 'time',
      header: t('time'),
      cell: (r) => formatDateTime(r.created_at, locale) || '—',
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    {
      key: 'actor',
      header: t('actor'),
      cell: (r) =>
        r.admin ? (
          <span className="whitespace-nowrap">
            {r.admin.name ?? `#${r.admin.id}`}
            {r.admin.role ? <span className="text-ink-3"> ({tr(r.admin.role as 'super')})</span> : null}
          </span>
        ) : (
          t('system')
        ),
    },
    {
      key: 'action',
      header: t('action'),
      cell: (r) => <code dir="ltr">{r.action}</code>,
    },
    {
      key: 'target',
      header: t('target'),
      cell: (r) => (
        <span dir="ltr" className="whitespace-nowrap">
          {r.target_type}
          {r.target_id ? ` #${formatNumber(r.target_id, 'en')}` : ''}
        </span>
      ),
    },
    {
      key: 'detail',
      header: t('detail'),
      cell: (r) =>
        r.detail ? (
          <span dir="ltr" className="text-ink-3">
            {Object.entries(r.detail)
              .map(([k, v]) => `${k}: ${String(v)}`)
              .join(' · ')}
          </span>
        ) : (
          '—'
        ),
    },
    {
      key: 'ip',
      header: t('ip'),
      cell: (r) => (r.ip ? <span dir="ltr">{r.ip}</span> : '—'),
      className: 'text-ink-3',
    },
  ];

  // One URL update for every filter (setFilter per key would race: each replace starts from the same URL).
  const apply = () => {
    const patch = Object.fromEntries(Object.entries(draft).map(([k, v]) => [k, v.trim()]));
    router.replace(`${pathname}${nextSearch(new URLSearchParams(search.toString()), patch, FILTERS)}`, {
      scroll: false,
    });
  };

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('auditTitle')}
        actions={
          canExport ? (
            <Button
              size="sm"
              loading={exporting}
              onClick={async () => {
                setExporting(true);
                try {
                  await downloadAuditCsv(list.query);
                } catch (e) {
                  notifyError(e);
                } finally {
                  setExporting(false);
                }
              }}
            >
              <Icon name="download" size={15} />
              {t('export')}
            </Button>
          ) : null
        }
      />
      {isSuper ? <ChainStatusCard /> : null}
      <Panel>
        <form
          className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5"
          onSubmit={(e) => {
            e.preventDefault();
            apply();
          }}
        >
          <TextInput
            label={t('filterAction')}
            placeholder={t('filterActionPlaceholder')}
            dir="ltr"
            value={draft.action}
            onChange={(e) => setDraft({ ...draft, action: e.target.value })}
          />
          <TextInput
            label={t('filterAdmin')}
            type="number"
            value={draft.admin_id}
            onChange={(e) => setDraft({ ...draft, admin_id: e.target.value })}
          />
          <TextInput
            label={t('filterTarget')}
            dir="ltr"
            value={draft.target_type}
            onChange={(e) => setDraft({ ...draft, target_type: e.target.value })}
          />
          <TextInput
            label={t('filterFrom')}
            type="date"
            value={draft.from}
            onChange={(e) => setDraft({ ...draft, from: e.target.value })}
          />
          <TextInput
            label={t('filterTo')}
            type="date"
            value={draft.to}
            onChange={(e) => setDraft({ ...draft, to: e.target.value })}
          />
          <div className="flex gap-2 sm:col-span-2 lg:col-span-5">
            <Button type="submit" variant="primary" size="sm">
              {t('apply')}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setDraft(FILTERS);
                list.reset();
              }}
            >
              {t('clear')}
            </Button>
          </div>
        </form>
      </Panel>
      <Panel bodyClassName="">
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('auditEmpty')}
          caption={t('auditTitle')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}
