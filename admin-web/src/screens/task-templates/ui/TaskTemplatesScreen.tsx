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

import { taskTemplatesApi, type TaskTemplate } from '../api/task-templates';

/** /task-templates (Blade task-templates.index). */
export function TaskTemplatesScreen() {
  const t = useTranslations('taskTemplates');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams();
  const query = taskTemplatesApi.useList(list.query);
  const options = taskTemplatesApi.useOptions().data;
  const commands = useRowCommands({ toggle: taskTemplatesApi.useAction('toggle'), remove: taskTemplatesApi.useRemove() });

  const columns: Column<TaskTemplate>[] = [
    { key: 'id', header: '#', cell: (r) => formatNumber(r.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'key', header: t('key'), cell: (r) => <span dir="ltr">{r.key}</span>, className: 'cell-mono' },
    { key: 'title', header: t('titleField'), cell: (r) => localize(r.title) || '—', className: 'cell-wrap font-semibold' },
    { key: 'category', header: tc('category'), cell: (r) => optionLabel(options?.categories, r.category) },
    { key: 'phase', header: tc('phase'), cell: (r) => optionLabel(options?.phases, r.cycle_phase) || tc('allPhases') },
    {
      key: 'status',
      header: tc('status'),
      cell: (r) => (r.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge>{tc('inactive')}</Badge>),
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (r) => <CrudRowActions id={r.id} editHref={`/task-templates/${r.id}`} commands={commands} active={r.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/task-templates/new" className="btn btn-primary btn-sm">
          <Icon name="plus" size={15} />
          {t('new')}
        </Link>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(r) => r.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(r) => router.push(`/task-templates/${r.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
