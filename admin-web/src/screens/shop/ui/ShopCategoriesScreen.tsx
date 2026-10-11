'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib';
import { Badge, CrudRowActions, DataTable, Icon, Panel, useRowCommands, type Column } from '@/shared/ui';

import { shopCategoriesApi, type ShopCategory } from '../api/shop';

/** /shop/categories — «سیسمونی و نوزاد», «آرایشی و بهداشتی», … (B-N10-03). A category with products can't be deleted. */
export function ShopCategoriesScreen() {
  const t = useTranslations('shop');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const query = shopCategoriesApi.useList();
  const commands = useRowCommands({ remove: shopCategoriesApi.useRemove() });

  const columns: Column<ShopCategory>[] = [
    { key: 'id', header: '#', cell: (c) => formatNumber(c.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'title', header: t('titleShort'), cell: (c) => localize(c.title) || '—' },
    { key: 'slug', header: t('slug'), cell: (c) => <span dir="ltr">{c.slug}</span> },
    { key: 'products', header: t('productsCount'), cell: (c) => formatNumber(c.products_count, locale), className: 'cell-num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (c) => (c.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (c) => formatNumber(c.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (c) => <CrudRowActions id={c.id} editHref={`/shop/categories/${c.id}`} commands={commands} canDelete={c.products_count === 0} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('categories')}
      bodyClassName=""
      actions={
        <>
          <Link href="/shop" className="btn btn-sm">
            {t('title')}
          </Link>
          <Link href="/shop/categories/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('newCategory')}
          </Link>
        </>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(c) => c.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(c) => router.push(`/shop/categories/${c.id}`)}
        emptyText={t('emptyCategories')}
        caption={t('categories')}
      />
    </Panel>
  );
}
