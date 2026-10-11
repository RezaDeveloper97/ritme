'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, toIntOrNull, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, Pagination, Panel, SearchInput, Select, TextInput, toast, useNotifyError, type Column } from '@/shared/ui';

import { stockApi, variantsApi, type StockRow } from '../api/commerce';

import { Money, ShopTabs } from './ShopTabs';

/** /shop/stock — every variant with its product, lowest stock first; quick stock edit (CB-SHOP-05). */
export function ShopStockScreen() {
  const t = useTranslations('shopAdmin.stock');
  const locale = useLocale();
  const localize = useLocalized();
  const list = useListParams({ low: '0' });
  const low = list.params.filters.low ?? '0';
  const query = stockApi.useList(list.query);

  const columns: Column<StockRow>[] = [
    {
      key: 'product',
      header: t('product'),
      cell: (r) => (
        <Link href={`/shop/${r.product_id}/commerce`} className="text-brand no-underline hover:underline">
          {localize(r.product_title) || `#${formatNumber(r.product_id, locale)}`}
        </Link>
      ),
    },
    { key: 'variant', header: t('variant'), cell: (r) => [localize(r.size), localize(r.color), localize(r.volume)].filter(Boolean).join(' · ') || '—' },
    { key: 'sku', header: t('sku'), cell: (r) => <span dir="ltr">{r.sku ?? '—'}</span> },
    { key: 'price', header: t('price'), cell: (r) => <Money value={r.price} /> },
    {
      key: 'stock',
      header: t('stock'),
      cell: (r) => (
        <span className="flex items-center gap-2">
          {formatNumber(r.stock, locale)}
          {r.low ? <Badge tone="red">{t('low')}</Badge> : null}
        </span>
      ),
    },
    { key: 'edit', header: <span className="sr-only">{t('edit')}</span>, cell: (r) => <StockCell row={r} />, className: 'cell-actions' },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ShopTabs />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('search')} />
          <Select
            aria-label={t('filter')}
            value={low}
            onChange={(e) => list.setFilter('low', e.target.value)}
            options={[
              { value: '0', label: t('all') },
              { value: '1', label: t('lowOnly') },
            ]}
          />
        </div>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

function StockCell({ row }: { row: StockRow }) {
  const t = useTranslations('shopAdmin.stock');
  const notifyError = useNotifyError();
  const setStock = variantsApi.useAction('stock');
  const [value, setValue] = useState(String(row.stock));
  return (
    <div className="flex items-end gap-2">
      <TextInput aria-label={t('stock')} type="number" value={value} onChange={(e) => setValue(e.target.value)} className="w-24" />
      <Button
        size="sm"
        disabled={toIntOrNull(value) === row.stock}
        loading={setStock.isPending}
        onClick={() =>
          setStock.mutate({ id: row.id, body: { stock: toIntOrNull(value) } }, { onSuccess: () => toast.success(t('saved')), onError: notifyError })
        }
      >
        {t('edit')}
      </Button>
    </div>
  );
}
