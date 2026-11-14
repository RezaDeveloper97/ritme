'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { formatNumber } from '@/shared/lib';
import { Badge, Button, confirm, DataTable, Icon, Panel, RowActions, toast, useNotifyError, type Column } from '@/shared/ui';

import { modelsApi, useRemoveModel, useSyncModels, type AiModel } from '../api/ai-credits';
import { CapabilityBadges, Usd, useCanManage } from './parts';

/** The provider the «همگام‌سازی» button asks (the ArvanCloud AI gateway lists its models). */
const SYNC_PROVIDER = 'arvan';

/** /ai/models — the AI model catalogue: what each model can do and what it costs us (admin-api.md §29). */
export function ModelsScreen() {
  const t = useTranslations('aiCredits.models');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const canManage = useCanManage();
  const notifyError = useNotifyError();
  const query = modelsApi.useList();
  const sync = useSyncModels();
  const remove = useRemoveModel();
  const configured = query.data?.configured_providers ?? [];
  const canSync = configured.includes(SYNC_PROVIDER);

  const onSync = () =>
    sync.mutate(SYNC_PROVIDER, {
      onSuccess: (res) =>
        toast.success(
          res.added.length > 0
            ? t('syncAdded', { count: res.added.length, existing: formatNumber(res.existing, locale) })
            : t('syncNothing', { existing: formatNumber(res.existing, locale) }),
        ),
      onError: notifyError,
    });

  const onRemove = async (m: AiModel) => {
    if (!(await confirm({ message: tc('confirmDelete'), confirmLabel: tc('delete'), tone: 'danger' }))) return;
    remove.mutate(m.id, { onSuccess: () => toast.success(tc('deleted')), onError: notifyError });
  };

  const columns: Column<AiModel>[] = [
    {
      key: 'title',
      header: t('model'),
      cell: (m) => (
        <span className="flex flex-col items-start gap-1">
          <span className="font-semibold" dir="auto">
            {m.title}
          </span>
          <code dir="ltr" className="text-xs text-muted">
            {m.code}
          </code>
        </span>
      ),
    },
    {
      key: 'provider',
      header: t('provider'),
      cell: (m) => (
        <span className="flex flex-col items-start gap-1">
          <span>{t.has(`providers.${m.provider}` as 'providers.arvan') ? t(`providers.${m.provider}` as 'providers.arvan') : m.provider}</span>
          <Badge tone={m.configured ? 'green' : 'amber'}>{m.configured ? t('configured') : t('notConfigured')}</Badge>
        </span>
      ),
    },
    { key: 'upstream', header: t('upstream'), cell: (m) => <code dir="ltr">{m.upstream_model}</code> },
    { key: 'caps', header: t('capabilities'), cell: (m) => <CapabilityBadges caps={m.capabilities} /> },
    {
      key: 'prices',
      header: t('prices'),
      cell: (m) => (
        <span className="flex flex-col items-end text-xs text-ink-3">
          <span>
            {t('inputShort')} <Usd milli={m.input_price_milli} />
          </span>
          <span>
            {t('outputShort')} <Usd milli={m.output_price_milli} />
          </span>
          {m.audio_price_milli > 0 ? (
            <span>
              {t('audioShort')} <Usd milli={m.audio_price_milli} />
            </span>
          ) : null}
        </span>
      ),
      className: 'cell-num',
    },
    { key: 'slots', header: t('slots'), cell: (m) => formatNumber(m.slots_count, locale), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (m) => <Badge tone={m.is_active ? 'green' : 'neutral'}>{m.is_active ? tc('active') : tc('inactive')}</Badge>,
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (m) =>
        canManage ? (
          <RowActions>
            <Link href={`/ai/models/${m.id}`} className="btn btn-sm" onClick={(e) => e.stopPropagation()}>
              {tc('edit')}
            </Link>
            <Button
              size="sm"
              variant="danger"
              disabled={m.slots_count > 0}
              title={m.slots_count > 0 ? t('inUseHint') : undefined}
              onClick={(e) => {
                e.stopPropagation();
                void onRemove(m);
              }}
              loading={remove.isPending && remove.variables === m.id}
            >
              {tc('delete')}
            </Button>
          </RowActions>
        ) : null,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        canManage ? (
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={onSync} loading={sync.isPending} disabled={!canSync} title={canSync ? undefined : t('syncUnavailable')}>
              <Icon name="download" size={15} />
              {t('sync')}
            </Button>
            <Link href="/ai/models/new" className="btn btn-primary btn-sm">
              <Icon name="plus" size={15} />
              {t('new')}
            </Link>
          </div>
        ) : null
      }
    >
      <p className="panel-note">{t('intro')}</p>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(m) => m.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={canManage ? (m) => router.push(`/ai/models/${m.id}`) : undefined}
        emptyText={t('empty')}
        caption={t('title')}
      />
    </Panel>
  );
}
