'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber, optionLabel, useListParams } from '@/shared/lib';
import {
  Badge,
  CrudRowActions,
  DataTable,
  Icon,
  Pagination,
  Panel,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { articlesApi, type Article } from '../api/articles';

/** /articles (Blade articles.index): title, phases, category, published state. */
export function ArticlesScreen() {
  const t = useTranslations('articles');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams();
  const query = articlesApi.useList(list.query);
  const phases = articlesApi.useOptions().data?.phases;
  const commands = useRowCommands({ toggle: articlesApi.useAction('toggle'), remove: articlesApi.useRemove() });

  const columns: Column<Article>[] = [
    { key: 'id', header: '#', cell: (a) => formatNumber(a.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'title', header: t('titleField'), cell: (a) => localize(a.title) || '—', className: 'cell-wrap font-semibold' },
    {
      key: 'phases',
      header: t('phases'),
      cell: (a) =>
        a.cycle_phases.length ? (
          <span className="flex flex-wrap gap-1">
            {a.cycle_phases.map((p) => (
              <Badge key={p}>{optionLabel(phases, p)}</Badge>
            ))}
          </span>
        ) : (
          <span className="text-muted">{t('allPhases')}</span>
        ),
      className: 'cell-wrap',
    },
    { key: 'category', header: t('category'), cell: (a) => a.category || '—' },
    {
      key: 'status',
      header: tc('status'),
      cell: (a) => (a.is_published ? <Badge tone="green">{t('published')}</Badge> : <Badge tone="amber">{t('draft')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (a) => formatNumber(a.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (a) => (
        <CrudRowActions
          id={a.id}
          editHref={`/articles/${a.id}`}
          commands={commands}
          active={a.is_published}
          activeLabels={[t('unpublish'), t('publish')]}
        />
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/articles/new" className="btn btn-primary btn-sm">
          <Icon name="plus" size={15} />
          {t('new')}
        </Link>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(a) => a.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(a) => router.push(`/articles/${a.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
