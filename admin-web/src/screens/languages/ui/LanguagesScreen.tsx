'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { RequireSuper } from '@/features/auth';
import { formatNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  Icon,
  Panel,
  RowActions,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { languagesApi, type Language } from '../api/languages';
import { useDirectionLabel } from './direction';

/** /languages — the content-language registry (super admins; Blade languages.index). */
export function LanguagesScreen() {
  return (
    <RequireSuper>
      <LanguagesList />
    </RequireSuper>
  );
}

function LanguagesList() {
  const t = useTranslations('languages');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const direction = useDirectionLabel();
  const query = languagesApi.useList();
  const commands = useRowCommands({
    toggle: languagesApi.useAction('toggle'),
    remove: languagesApi.useRemove(),
    confirmDelete: t('confirmDelete'),
  });

  const columns: Column<Language>[] = [
    { key: 'id', header: '#', cell: (l) => formatNumber(l.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'code', header: t('code'), cell: (l) => <span dir="ltr">{l.code}</span>, className: 'cell-mono' },
    {
      key: 'name',
      header: t('name'),
      cell: (l) => (
        <span className="flex flex-wrap items-center gap-2">
          <span dir={l.direction === 'rtl' ? 'rtl' : 'ltr'} className="font-semibold">
            {l.name}
          </span>
          {l.is_default ? <Badge tone="brand">{t('default')}</Badge> : null}
        </span>
      ),
    },
    { key: 'english', header: t('englishName'), cell: (l) => <span dir="ltr">{l.english_name}</span> },
    { key: 'direction', header: t('direction'), cell: (l) => direction(l.direction) },
    {
      key: 'status',
      header: tc('status'),
      cell: (l) => (l.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge>{tc('inactive')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (l) => formatNumber(l.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (l) => (
        <RowActions>
          <Link href={`/languages/${l.id}/translations`} className="btn btn-sm">
            {t('translations')}
          </Link>
          <Link href={`/languages/${l.id}`} className="btn btn-sm">
            {tc('edit')}
          </Link>
          <Button size="sm" onClick={() => commands.toggle(l.id)} loading={commands.isToggling(l.id)}>
            {l.is_active ? tc('deactivate') : tc('activate')}
          </Button>
          {l.is_default ? null : (
            <Button size="sm" variant="danger" onClick={() => commands.remove(l.id)} loading={commands.isRemoving(l.id)}>
              {tc('delete')}
            </Button>
          )}
        </RowActions>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/languages/new" className="btn btn-primary btn-sm">
          <Icon name="plus" size={15} />
          {t('new')}
        </Link>
      }
    >
      <p className="panel-note">{t('intro')}</p>
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(l) => l.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(l) => router.push(`/languages/${l.id}`)}
        emptyText={t('empty')}
        skeletonRows={3}
        caption={t('title')}
      />
    </Panel>
  );
}
