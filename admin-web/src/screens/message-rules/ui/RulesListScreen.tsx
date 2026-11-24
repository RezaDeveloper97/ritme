'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { formatNumber, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, PageHeader, Pagination, Panel, Select, type BadgeTone, type Column } from '@/shared/ui';

import { useRuleList, type ListRule } from '../api/rules';
import { groupByVersion } from '../lib/ctr';
import { INBOX_SLOT, RULE_KEY_PREFIXES, matchesPrefix } from '../lib/inbox';
import { summarize } from '../lib/rule-form';

const KIND_TONE: Record<string, BadgeTone> = { safety: 'red', engine: 'data', clinical: 'amber', product: 'neutral' };
const LEVEL_TONE: Record<string, BadgeTone> = { info: 'data', suggestion: 'brand', worth_followup: 'amber', follow_soon: 'red' };
const LEVELS = ['info', 'suggestion', 'worth_followup', 'follow_soon'] as const;
const KINDS = ['safety', 'engine', 'clinical', 'product'] as const;

/** /messages/rules — the rule-based messages and their 7-day numbers (Admin_Messages artboard). */
export function RulesListScreen() {
  const t = useTranslations('messageRules');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const list = useListParams({ placement: '', kind: '', status: '', rule_key: '' });
  const prefix = list.params.filters.rule_key ?? '';
  // The list API has no rule_key filter: a prefix reads the biggest page and filters here.
  const query = useRuleList(prefix ? { ...list.query, per_page: 100 } : list.query);
  const items = query.data ? matchesPrefix(query.data.items, prefix) : undefined;
  const enums = query.data?.enums;
  const num = (n: number) => formatNumber(n, locale);

  const columns: Column<ListRule>[] = [
    {
      key: 'title',
      header: t('message'),
      cell: (r) => (
        <Link href={`/messages/rules/${r.id}`} className="font-medium">
          {r.title}
        </Link>
      ),
      className: 'cell-wrap',
    },
    {
      key: 'placement',
      header: t('placement'),
      cell: (r) => (
        <span className="flex flex-col gap-0.5">
          <span>{t(`placements.${r.placement}` as 'placements.smart_tip')}</span>
          {t.has(`placementWhere.${r.placement}` as 'placementWhere.push') ? (
            <span className="text-xs text-muted">{t(`placementWhere.${r.placement}` as 'placementWhere.push')}</span>
          ) : null}
        </span>
      ),
      className: 'cell-wrap',
    },
    { key: 'kind', header: t('kind'), cell: (r) => <Badge tone={KIND_TONE[r.kind]}>{t(`kinds.${r.kind}` as 'kinds.safety')}</Badge> },
    { key: 'level', header: t('level'), cell: (r) => <Badge tone={LEVEL_TONE[r.level]}>{t(`levels.${r.level}` as 'levels.info')}</Badge> },
    {
      key: 'rule',
      header: t('ruleSummary'),
      cell: (r) => (
        <span dir="ltr" className="text-ink-3">
          {summarize(r.conditions) || '—'}
        </span>
      ),
      className: 'cell-wrap cell-mono',
    },
    { key: 'priority', header: t('priority'), cell: (r) => num(r.priority), className: 'num' },
    {
      key: 'status',
      header: tc('status'),
      cell: (r) => (
        <span className="flex flex-wrap gap-1">
          {r.is_active ? <Badge tone="green">{t('active')}</Badge> : <Badge>{t('inactive')}</Badge>}
          {r.variants_enabled ? <Badge tone="data">A/B</Badge> : null}
          {r.locked ? <Badge tone="red">{t('locked')}</Badge> : null}
          {r.needs_medical_approval && r.status !== 'approved' ? (
            <Badge tone="amber">{t(`statuses.${r.status}` as 'statuses.draft')}</Badge>
          ) : null}
        </span>
      ),
    },
    { key: 'impressions', header: t('impressions7d'), cell: (r) => num(r.stats_7d.impressions), className: 'num' },
    {
      key: 'ctr',
      header: t('ctr'),
      cell: (r) => <CtrCell rule={r} num={num} />,
      className: 'cell-wrap',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('title')}
        backHref="/messages"
        backLabel={t('backToContent')}
        meta={t('subtitle')}
        actions={
          <>
            <Link href="/messages/rules/engine" className="btn">
              {t('engineLink')}
            </Link>
            <Link href="/messages/rules/new" className="btn btn-primary">
              {t('new')}
            </Link>
          </>
        }
      />
      <Panel title={t('levelsTitle')}>
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {LEVELS.map((l) => (
            <li key={l} className="tone-card" data-tone={LEVEL_TONE[l]}>
              <strong>
                <span className="tone-dot" aria-hidden="true" />
                {t(`levels.${l}`)}
              </strong>
              <span className="text-[13px] text-ink-3">{t(`levelHints.${l}`)}</span>
            </li>
          ))}
        </ul>
      </Panel>
      <Panel title={t('bandsTitle')}>
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {KINDS.map((k) => {
            const b = enums?.bands[k];
            return (
              <li key={k} className="tone-card" data-tone={KIND_TONE[k] === 'data' ? 'brand' : KIND_TONE[k] === 'neutral' ? 'neutral' : KIND_TONE[k]}>
                <strong>
                  {t(`kinds.${k}`)}
                  {b ? ` · ${num(b.min)}${b.max !== b.min ? `–${num(b.max)}` : ''}` : ''}
                </strong>
                <span className="text-[13px] text-ink-3">{t(`bandHints.${k}`)}</span>
              </li>
            );
          })}
        </ul>
      </Panel>
      <Panel title={t('listTitle')} bodyClassName="">
        <div className="filter-bar">
          <Select
            label={t('placement')}
            className="min-w-44"
            value={list.params.filters.placement}
            onChange={(e) => list.setFilter('placement', e.target.value)}
            options={[
              { value: '', label: t('all') },
              ...(enums?.placements ?? []).map((p) => ({ value: p, label: t(`placements.${p}` as 'placements.smart_tip') })),
            ]}
          />
          <Select
            label={t('kind')}
            className="w-44"
            value={list.params.filters.kind}
            onChange={(e) => list.setFilter('kind', e.target.value)}
            options={[{ value: '', label: t('all') }, ...KINDS.map((k) => ({ value: k, label: t(`kinds.${k}`) }))]}
          />
          <Select
            label={tc('status')}
            className="w-52"
            value={list.params.filters.status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[
              { value: '', label: t('all') },
              ...(['draft', 'pending_approval', 'approved'] as const).map((s) => ({ value: s, label: t(`statuses.${s}`) })),
            ]}
          />
          <Select
            label={t('ruleKeyPrefix')}
            className="w-44"
            value={prefix}
            onChange={(e) => list.setFilter('rule_key', e.target.value)}
            options={[{ value: '', label: t('all') }, ...RULE_KEY_PREFIXES.map((p) => ({ value: p, label: `${p}*` }))]}
          />
          <Button
            size="sm"
            variant={list.params.filters.placement === INBOX_SLOT ? 'primary' : 'default'}
            aria-pressed={list.params.filters.placement === INBOX_SLOT}
            onClick={() => list.setFilter('placement', list.params.filters.placement === INBOX_SLOT ? '' : INBOX_SLOT)}
          >
            {t('inboxRules')}
          </Button>
        </div>
        <DataTable
          columns={columns}
          items={items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('empty')}
          caption={t('listTitle')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

/** CTR of the last 7 days; the details open the per-version and per-variant numbers (T-M10-06). */
function CtrCell({ rule, num }: { rule: ListRule; num: (n: number) => string }) {
  const t = useTranslations('messageRules');
  const pct = (v: number | null) => (v === null ? '—' : `${num(Math.round(v * 100))}٪`);
  const versions = groupByVersion(rule.stats_by_version);
  return (
    <div className="flex flex-col gap-1">
      <span className="num">{rule.stats_7d.impressions > 0 ? pct(rule.stats_7d.ctr) : '—'}</span>
      {versions.length > 0 ? (
        <details className="text-[12px]">
          <summary className="cursor-pointer text-ink-3">{t('ctrByVersion')}</summary>
          <ul className="m-0 mt-1 flex list-none flex-col gap-1 p-0">
            {versions.map((v) => (
              <li key={v.version}>
                <strong>{v.version === 0 ? t('versionUnknown') : t('version', { n: num(v.version) })}</strong> · {pct(v.ctr)} ({num(v.impressions)})
                <ul className="m-0 list-none ps-3 text-ink-3">
                  {v.variants.map((x) => (
                    <li key={x.variant}>
                      {x.variant.toUpperCase()}: {pct(x.ctr)} ({num(x.clicks)}/{num(x.impressions)})
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        </details>
      ) : null}
    </div>
  );
}
