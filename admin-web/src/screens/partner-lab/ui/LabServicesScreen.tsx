'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib';
import { Badge, DataTable, Icon, Panel, type Column } from '@/shared/ui';

import { labServicesApi, type LabServiceRow } from '../api/partnerLab';

/** /lab-services — the orderable services (home blood draw, annual checkup / Pap). Inactive ones hide the Today cards. */
export function LabServicesScreen() {
  const t = useTranslations('partnerLab.services');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const router = useRouter();
  const localize = useLocalized();
  const query = labServicesApi.useList();

  const columns: Column<LabServiceRow>[] = [
    { key: 'id', header: '#', cell: (s) => formatNumber(s.id, locale), className: 'cell-num w-14 text-muted' },
    { key: 'title', header: t('titleField'), cell: (s) => localize(s.title) || '—' },
    { key: 'code', header: t('code'), cell: (s) => <span dir="ltr">{s.code}</span> },
    { key: 'kind', header: t('kind'), cell: (s) => t(`kinds.${s.kind}` as 'kinds.home_draw') },
    { key: 'partner', header: t('partnerName'), cell: (s) => `${s.partner_name} (${s.partner_code})` },
    { key: 'status', header: tc('status'), cell: (s) => (s.is_active ? <Badge tone="green">{tc('active')}</Badge> : <Badge tone="amber">{tc('inactive')}</Badge>) },
    { key: 'sort', header: tc('sortOrder'), cell: (s) => formatNumber(s.sort_order, locale), className: 'cell-num' },
  ];

  return (
    <Panel
      title={t('title')}
      bodyClassName=""
      actions={
        <>
          <Link href="/lab-requests" className="btn btn-sm">
            {t('queue')}
          </Link>
          <Link href="/lab-services/new" className="btn btn-primary btn-sm">
            <Icon name="plus" size={15} />
            {t('new')}
          </Link>
        </>
      }
    >
      <DataTable
        columns={columns}
        items={query.data?.items}
        rowKey={(s) => s.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={(s) => router.push(`/lab-services/${s.id}`)}
        emptyText={t('empty')}
        caption={t('title')}
      />
    </Panel>
  );
}
