'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useLocalized } from '@/shared/i18n';
import { excerpt, useNumber } from '@/shared/lib';
import { Badge, ErrorState, Icon, PageHeader, Panel, Select, Skeleton } from '@/shared/ui';

import { useChildItems } from '../api/child-content';
import { KINDS, TOPICS } from '../lib/meta';
import { useAgeLabel, useCodeLabels } from './labels';
import { ChildTabs, ItemActions, ReadOnlyNotice, ReviewBadges, itemHref, useCanWrite } from './parts';

/** /children-content/learn — age-based learn tips («آموزش کودک»), filterable by topic. */
export function LearnScreen() {
  const t = useTranslations('childContent');
  const n = useNumber();
  const localize = useLocalized();
  const labels = useCodeLabels();
  const ageLabel = useAgeLabel();
  const canWrite = useCanWrite();
  const query = useChildItems(KINDS.learn.group);
  const [topic, setTopic] = useState('');
  const rows = (query.data ?? []).filter((r) => !topic || r.meta?.topic === topic);
  const range = (from: unknown, to: unknown) =>
    typeof from === 'number' && typeof to === 'number' ? t('ageRange', { from: ageLabel(from), to: ageLabel(to) }) : '—';

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('pages.learn')}
        meta={query.data ? <span>{t('itemsCount', { count: n(query.data.length) })}</span> : null}
        actions={
          canWrite ? (
            <Link href={itemHref('learn', null)} className="btn btn-primary btn-sm">
              <Icon name="plus" size={15} />
              {t('newLearn')}
            </Link>
          ) : null
        }
      />
      <ChildTabs active="learn" />
      <p className="field-hint m-0">{t('learnHint')}</p>
      {canWrite ? null : <ReadOnlyNotice />}
      <Panel bodyClassName="">
        <div className="filter-bar">
          <Select
            label={t('fields.topic')}
            className="w-48"
            value={topic}
            onChange={(e) => setTopic(e.target.value)}
            options={[{ value: '', label: t('allTopics') }, ...TOPICS.map((k) => ({ value: k, label: labels.topic(k) }))]}
          />
        </div>
        {query.error && !query.data ? (
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        ) : !query.data ? (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : rows.length === 0 ? (
          <p className="cell-empty m-0 p-8 text-center text-muted">{topic ? t('emptyFiltered') : t('emptyLearn')}</p>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <caption className="sr-only">{t('pages.learn')}</caption>
              <thead>
                <tr>
                  <th scope="col">{t('fields.title')}</th>
                  <th scope="col">{t('fields.topic')}</th>
                  <th scope="col">{t('ageBand')}</th>
                  <th scope="col">{t('review')}</th>
                  <th scope="col">
                    <span className="sr-only">{t('actions')}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id}>
                    <td className="cell-wrap">
                      <span className="flex flex-col gap-0.5">
                        <span className="flex flex-wrap items-center gap-2">
                          <strong>{localize(r.title) || r.code}</strong>
                          {r.meta?.featured === true ? <Badge tone="brand">{t('featured')}</Badge> : null}
                        </span>
                        {localize(r.body) ? <span className="text-xs text-muted">{excerpt(localize(r.body), 90)}</span> : null}
                        <span dir="ltr" className="cell-mono text-xs text-muted">
                          {r.code}
                        </span>
                      </span>
                    </td>
                    <td>{typeof r.meta?.topic === 'string' ? labels.topic(r.meta.topic) : '—'}</td>
                    <td className="whitespace-nowrap">{range(r.meta?.from_months, r.meta?.to_months)}</td>
                    <td>
                      <ReviewBadges kind="learn" row={r} />
                    </td>
                    <td className="cell-actions">
                      <ItemActions kind="learn" row={r} editHref={itemHref('learn', r.id)} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Panel>
    </div>
  );
}
