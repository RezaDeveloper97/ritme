'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { excerpt, formatNumber, optionLabel, useListParams } from '@/shared/lib';
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

import { affirmationsApi, type Affirmation } from '../api/affirmations';

/** /affirmations (Blade affirmations.index). */
export function AffirmationsScreen() {
  const t = useTranslations('affirmations');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams();
  const query = affirmationsApi.useList(list.query);
  const phases = affirmationsApi.useOptions().data?.phases;
  const commands = useRowCommands({ toggle: affirmationsApi.useAction('toggle'), remove: affirmationsApi.useRemove() });

  const columns: Column<Affirmation>[] = [
    { key: 'id', header: '#', cell: (a) => formatNumber(a.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'text', header: t('text'), cell: (a) => excerpt(localize(a.text), 120) || '—', className: 'cell-wrap' },
    { key: 'phase', header: tc('phase'), cell: (a) => optionLabel(phases, a.cycle_phase) || tc('allPhases') },
    {
      key: 'status',
      header: tc('status'),
      cell: (a) => (a.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge>{tc('inactive')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (a) => formatNumber(a.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (a) => <CrudRowActions id={a.id} editHref={`/affirmations/${a.id}`} commands={commands} active={a.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/affirmations/new" className="btn btn-primary btn-sm">
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
        onRowClick={(a) => router.push(`/affirmations/${a.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
