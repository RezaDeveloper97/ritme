'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { excerpt, formatNumber, useListParams } from '@/shared/lib';
import {
  Badge,
  CrudRowActions,
  DataTable,
  Icon,
  LinkTabs,
  Pagination,
  Panel,
  useRowCommands,
  type Column,
} from '@/shared/ui';

import { INFO_GROUPS, infoSectionsApi, type InfoSection } from '../api/info-sections';
import { useGroupLabel } from './group-label';

/** /info-sections?group= — the boxes of one app text screen (Blade info-sections.index). */
export function InfoSectionsScreen() {
  const t = useTranslations('infoSections');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const groupLabel = useGroupLabel();
  const list = useListParams({ group: 'help' });
  const group = list.params.filters.group ?? 'help';
  const query = infoSectionsApi.useList(list.query);
  const groups = infoSectionsApi.useOptions().data?.groups ?? [...INFO_GROUPS];
  const commands = useRowCommands({ toggle: infoSectionsApi.useAction('toggle'), remove: infoSectionsApi.useRemove() });

  const columns: Column<InfoSection>[] = [
    { key: 'id', header: '#', cell: (r) => formatNumber(r.id, locale), className: 'cell-num w-14 text-muted' },
    {
      key: 'heading',
      header: t('heading'),
      cell: (r) => (
        <span className="flex flex-col items-start gap-0.5">
          <span>{localize(r.heading) || '—'}</span>
          {r.key ? (
            <span dir="ltr" className="font-mono text-xs font-normal text-muted">
              {r.key}
            </span>
          ) : null}
        </span>
      ),
      className: 'cell-wrap font-semibold',
    },
    { key: 'body', header: t('body'), cell: (r) => excerpt(localize(r.body), 90) || '—', className: 'cell-wrap text-ink-3' },
    {
      key: 'link',
      header: t('button'),
      cell: (r) =>
        r.link_url ? (
          <span className="flex flex-col items-start gap-1">
            <Badge tone="data">{localize(r.link_label) || t('linkFallback')}</Badge>
            <span dir="ltr" className="block max-w-44 truncate text-xs text-muted">
              {r.link_url}
            </span>
          </span>
        ) : (
          '—'
        ),
    },
    {
      key: 'status',
      header: tc('status'),
      cell: (r) => (r.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge>{tc('inactive')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (r) => formatNumber(r.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (r) => <CrudRowActions id={r.id} editHref={`/info-sections/${r.id}`} commands={commands} active={r.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <LinkTabs
        label={t('pages')}
        items={groups.map((g) => ({
          key: g,
          href: g === 'help' ? '/info-sections' : `/info-sections?group=${g}`,
          label: groupLabel(g),
          active: g === group,
        }))}
      />
      <Panel
        title={groupLabel(group)}
        bodyClassName=""
        actions={
          <Link href={`/info-sections/new?group=${group}`} className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        }
      >
        <p className="panel-note">
          {t('intro', { page: groupLabel(group) })} {group === 'help' ? t('introHelp') : null}
        </p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(r) => router.push(`/info-sections/${r.id}`)}
          emptyText={t('empty')}
          caption={groupLabel(group)}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}
