'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { useCurrentAdmin } from '@/features/auth';
import { useNumber } from '@/shared/lib';
import { Badge, ErrorState, Icon, PageHeader, Panel, Skeleton } from '@/shared/ui';

import { POSTPARTUM_GROUPS, usePostpartumCoverage, type PostpartumGroup } from '../api/postpartum-content';

/**
 * /postpartum-content — the postpartum copy at a glance (week tips, alerts, EPDS safety messages): how many rows
 * each group has and how many slots still use the built-in copy, with a way into the messages editor
 * (admin-api.md §18; writes are super-admin only).
 */
export function PostpartumContentScreen() {
  const t = useTranslations('postpartumContent');
  const isSuper = useCurrentAdmin()?.role === 'super';
  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} />
      <p className="field-hint m-0">{t('hint')}</p>
      {isSuper ? null : (
        <p className="m-0 rounded-lg border border-line bg-[var(--amber-soft)] px-3 py-2 text-sm text-[var(--amber-deep)]" role="note">
          {t('readOnly')}
        </p>
      )}
      <div className="grid gap-4 lg:grid-cols-3">
        {POSTPARTUM_GROUPS.map((g) => (
          <GroupCard key={g} group={g} />
        ))}
      </div>
      <Panel title={t('fixedTitle')}>
        <p className="m-0 text-sm text-ink-3">{t('fixedBody')}</p>
      </Panel>
    </div>
  );
}

function GroupCard({ group }: { group: PostpartumGroup }) {
  const t = useTranslations('postpartumContent');
  const n = useNumber();
  const q = usePostpartumCoverage(group);
  return (
    <Panel
      title={t(`groups.${group}.title`)}
      actions={
        <Link href={`/messages?group=${group}`} className="btn btn-sm btn-primary">
          <Icon name="message" size={14} />
          {t('open')}
        </Link>
      }
    >
      <p className="mt-0 text-sm text-ink-3">{t(`groups.${group}.body`)}</p>
      <p dir="ltr" className="cell-mono m-0 mb-3 text-xs text-muted">
        {group}
      </p>
      {q.error && !q.data ? (
        <ErrorState error={q.error} onRetry={() => q.refetch()} />
      ) : !q.data ? (
        <Skeleton className="h-8 w-full" />
      ) : (
        <div className="flex flex-wrap gap-2">
          <Badge tone="brand">{t('rows', { count: n(q.data.rows) })}</Badge>
          <Badge tone="green">{t('live', { count: n(q.data.live) })}</Badge>
          {q.data.missing ? <Badge tone="amber">{t('missing', { count: n(q.data.missing) })}</Badge> : <Badge tone="green">{t('complete')}</Badge>}
        </div>
      )}
    </Panel>
  );
}
