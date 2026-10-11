'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, useListParams } from '@/shared/lib';
import { Badge, CrudRowActions, DataTable, Icon, Pagination, Panel, useRowCommands, type Column } from '@/shared/ui';

import { shopProductsApi, type ShopProduct } from '../api/shop';

/** /shop — the link-out products (B-N10-03); categories live under /shop/categories. */
export function ShopProductsScreen() {
  const t = useTranslations('shop');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams();
  const query = shopProductsApi.useList(list.query);
  const commands = useRowCommands({ toggle: shopProductsApi.useAction('toggle'), remove: shopProductsApi.useRemove() });

  const columns: Column<ShopProduct>[] = [
    { key: 'id', header: '#', cell: (p) => formatNumber(p.id, locale), className: 'cell-num w-14 text-muted' },
    {
      key: 'image',
      header: t('image'),
      // eslint-disable-next-line @next/next/no-img-element -- API-hosted upload
      cell: (p) => (p.image_url ? <img src={p.image_url} alt="" className="thumb" loading="lazy" /> : '—'),
    },
    { key: 'title', header: t('titleShort'), cell: (p) => localize(p.title) || '—' },
    { key: 'category', header: t('category'), cell: (p) => p.category_slug, className: 'whitespace-nowrap' },
    { key: 'price', header: t('priceText'), cell: (p) => p.price_text ?? '—' },
    {
      key: 'link',
      header: t('link'),
      cell: (p) =>
        p.external_url ? (
          <span dir="ltr" className="block max-w-40 truncate text-xs text-muted">
            {p.external_url}
          </span>
        ) : (
          <Badge tone="neutral">{t('noLink')}</Badge>
        ),
    },
    { key: 'clicks', header: t('clicks30d'), cell: (p) => formatNumber(p.clicks_30d, locale), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (p) => (p.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (p) => <CrudRowActions id={p.id} editHref={`/shop/${p.id}`} commands={commands} active={p.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <>
          <Link href="/shop/categories" className="btn btn-sm">
            {t('categories')}
          </Link>
          <Link href="/shop/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        </>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(p) => p.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(p) => router.push(`/shop/${p.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
