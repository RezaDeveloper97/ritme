'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatDateTime, formatNumber, optionLabel, useListParams } from '@/shared/lib';
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

import { bannersApi, type Banner } from '../api/banners';
import { scheduleState } from '../lib/schedule';

/** /banners — image, slot, link, display window (Blade banners.index). */
export function BannersScreen() {
  const t = useTranslations('banners');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const list = useListParams();
  const query = bannersApi.useList(list.query);
  const options = bannersApi.useOptions().data;
  const commands = useRowCommands({ toggle: bannersApi.useAction('toggle'), remove: bannersApi.useRemove() });

  const columns: Column<Banner>[] = [
    { key: 'id', header: '#', cell: (b) => formatNumber(b.id, locale), className: 'cell-num w-14 text-muted' },
    {
      key: 'image',
      header: t('image'),
      // eslint-disable-next-line @next/next/no-img-element -- API-hosted upload
      cell: (b) => (b.image_url ? <img src={b.image_url} alt="" className="thumb" loading="lazy" /> : '—'),
    },
    { key: 'title', header: t('titleShort'), cell: (b) => localize(b.title) || '—' },
    { key: 'position', header: t('position'), cell: (b) => optionLabel(options?.positions, b.position), className: 'whitespace-nowrap' },
    {
      key: 'link',
      header: t('link'),
      cell: (b) =>
        b.link_url ? (
          <span className="flex flex-col items-start gap-1">
            <Badge tone={b.link_type === 'external' ? 'amber' : 'brand'}>{optionLabel(options?.link_types, b.link_type)}</Badge>
            <span dir="ltr" className="block max-w-36 truncate text-xs text-muted">
              {b.link_url}
            </span>
          </span>
        ) : (
          '—'
        ),
    },
    {
      key: 'window',
      header: t('window'),
      cell: (b) => {
        const state = scheduleState(b.starts_at, b.ends_at);
        return (
          <span className="flex flex-col gap-0.5 text-xs text-ink-3">
            <Badge tone={state === 'live' ? 'data' : state === 'scheduled' ? 'brand' : 'neutral'}>{t(`schedule.${state}`)}</Badge>
            <span className="tabular-nums">{b.starts_at ? formatDateTime(b.starts_at, locale) : t('fromNow')}</span>
            <span className="tabular-nums">{b.ends_at ? t('until', { date: formatDateTime(b.ends_at, locale) }) : t('noEnd')}</span>
          </span>
        );
      },
    },
    {
      key: 'status',
      header: tc('status'),
      cell: (b) => (b.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>),
    },
    { key: 'sort', header: tc('sortOrder'), cell: (b) => formatNumber(b.sort_order, locale), className: 'cell-num' },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (b) => <CrudRowActions id={b.id} editHref={`/banners/${b.id}`} commands={commands} active={b.is_active} />,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <Link href="/banners/new" className="btn btn-primary btn-sm">
          <Icon name="plus" size={15} />
          {t('new')}
        </Link>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(b) => b.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(b) => router.push(`/banners/${b.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
      <Pagination meta={query.data?.meta} onPage={list.setPage} />
    </Panel>
  );
}
