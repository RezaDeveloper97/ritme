'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, rialsToToman } from '@/shared/lib';
import { Badge, Button, confirm, DataTable, Icon, Panel, RowActions, toast, useNotifyError, type Column } from '@/shared/ui';

import { packsApi, useRemovePack, type CreditPack } from '../api/ai-credits';
import { useCanManage } from './parts';

/** /ai/credit-packs — the extra «اعتبار» packs users can buy (admin-api.md §29). */
export function PacksScreen() {
  const t = useTranslations('aiCredits.packs');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const canManage = useCanManage();
  const notifyError = useNotifyError();
  const query = packsApi.useList();
  const remove = useRemovePack();
  const n = (v: number) => formatNumber(v, locale);
  const toman = (rials: number) => t('toman', { amount: n(rialsToToman(rials)) });

  const onRemove = async (p: CreditPack) => {
    const used = p.invoices_count > 0;
    const ok = await confirm({
      message: used ? t('confirmDeactivate') : tc('confirmDelete'),
      confirmLabel: used ? tc('deactivate') : tc('delete'),
      tone: 'danger',
    });
    if (!ok) return;
    remove.mutate(p.id, {
      onSuccess: (res) => toast.success(res.deleted ? tc('deleted') : t('deactivated')),
      onError: notifyError,
    });
  };

  const columns: Column<CreditPack>[] = [
    { key: 'sort', header: tc('sortOrder'), cell: (p) => n(p.sort_order), className: 'cell-num w-14 text-muted' },
    {
      key: 'title',
      header: t('pack'),
      cell: (p) => (
        <span className="flex flex-col items-start gap-1">
          <span className="font-semibold">{localize(p.title) || p.code}</span>
          <span className="flex flex-wrap gap-1">
            {p.is_highlighted ? <Badge tone="brand">{t('highlighted')}</Badge> : null}
            {localize(p.badge) ? <Badge tone="data">{localize(p.badge)}</Badge> : null}
          </span>
        </span>
      ),
    },
    { key: 'code', header: t('code'), cell: (p) => <code dir="ltr">{p.code}</code> },
    { key: 'credits', header: t('credits'), cell: (p) => n(p.credits), className: 'cell-num' },
    { key: 'price', header: t('price'), cell: (p) => <span className="whitespace-nowrap">{toman(p.price_rials)}</span>, className: 'cell-num' },
    {
      key: 'per',
      header: t('perCredit'),
      cell: (p) => <span className="whitespace-nowrap text-xs text-ink-3">{toman(p.price_per_credit_rials)}</span>,
      className: 'cell-num',
    },
    { key: 'paid', header: t('sold'), cell: (p) => n(p.paid_count), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (p) => <Badge tone={p.is_active ? 'green' : 'neutral'}>{p.is_active ? tc('active') : tc('inactive')}</Badge>,
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (p) =>
        canManage ? (
          <RowActions>
            <Link href={`/ai/credit-packs/${p.id}`} className="btn btn-sm" onClick={(e) => e.stopPropagation()}>
              {tc('edit')}
            </Link>
            <Button
              size="sm"
              variant="danger"
              onClick={(e) => {
                e.stopPropagation();
                void onRemove(p);
              }}
              loading={remove.isPending && remove.variables === p.id}
            >
              {p.invoices_count > 0 ? tc('deactivate') : tc('delete')}
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
          <Link href="/ai/credit-packs/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        ) : null
      }
    >
      <p className="panel-note">{t('intro')}</p>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(p) => p.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={canManage ? (p) => router.push(`/ai/credit-packs/${p.id}`) : undefined}
        emptyText={t('empty')}
        caption={t('title')}
      />
    </Panel>
  );
}
