'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { useFacts } from '@/features/rule-conditions';
import { cn, excerpt, useNumber } from '@/shared/lib';
import { Badge, ErrorState, Panel, SearchInput, Select, Skeleton } from '@/shared/ui';

import { useAtlasRules } from '../api/atlas';
import {
  bySurface,
  entryOfRule,
  filterEntries,
  groupEntries,
  MODES,
  whenOfRule,
  type AtlasEntry,
  type ConditionWords,
  type Mode,
  type Source,
} from '../lib/atlas';

/**
 * /messages/atlas — «نقشهٔ پیام‌ها»: what the app can say, to whom (mode tabs), where (one panel per surface) and
 * when (the rule's conditions in plain words, or the group's trigger). Read-only; «ویرایش» opens the source's own
 * screen. Step 1: engine rules + `/messages` copy groups.
 */
export function MessageAtlasScreen() {
  const t = useTranslations('messageAtlas');
  const tr = useTranslations('messageRules');
  const tg = useTranslations('smartMessages');
  const tc = useTranslations('ruleConditions');
  const locale = useLocale();
  const n = useNumber();
  const rules = useAtlasRules();
  const facts = useFacts();
  const [mode, setMode] = useState<Mode | 'all'>('all');
  const [source, setSource] = useState<Source | 'all'>('all');
  const [q, setQ] = useState('');

  const factIndex = useMemo(() => new Map((facts.data?.facts ?? []).map((f) => [f.name, f])), [facts.data]);
  const words: ConditionWords = useMemo(
    () => ({
      fact: (name) => factIndex.get(name)?.description[locale] || factIndex.get(name)?.description.en || name,
      op: (op) => (t.has(`ops.${op}` as 'ops.eq') ? t(`ops.${op}` as 'ops.eq') : op),
      yes: tc('yes'),
      no: tc('no'),
      listJoin: t('listJoin'),
      and: t('and'),
      or: t('or'),
      between: (a, b) => t('between', { a, b }),
      num: n,
    }),
    [factIndex, locale, n, t, tc],
  );

  const groupName = (g: string) => (tg.has(`groups.${g}` as 'groups.pattern') ? tg(`groups.${g}` as 'groups.pattern') : g);
  const entries = useMemo(
    () => [...(rules.data ?? []).map((r) => entryOfRule(r, (f) => factIndex.get(f)?.group)), ...groupEntries()],
    [rules.data, factIndex],
  );
  const searchText = (e: AtlasEntry) =>
    e.rule
      ? [e.rule.title, e.rule.rule_key ?? '', e.rule.content.a.title, e.rule.content.a.body, ...whenOfRule(e.rule, words).all].join(' ')
      : [groupName(e.name), e.name, e.group && t.has(`when.${e.group}` as 'when.pattern') ? t(`when.${e.group}` as 'when.pattern') : ''].join(' ');
  const visible = filterEntries(entries, { mode, source, q }, searchText);
  const buckets = bySurface(visible);
  const countOf = (m: Mode | 'all') => filterEntries(entries, { mode: m, source, q }, searchText).length;

  if (rules.error) return <ErrorState error={rules.error} onRetry={() => rules.refetch()} />;

  return (
    <div className="flex flex-col gap-4">
      <Panel title={t('title')}>
        <p className="field-hint mt-0">{t('intro')}</p>
        <p className="field-hint mt-0">{t('scopeNote')}</p>
        <div className="atlas-tabs" role="tablist" aria-label={t('modeLabel')}>
          {(['all', ...MODES] as const).map((m) => (
            <button
              key={m}
              type="button"
              role="tab"
              aria-selected={mode === m}
              className="atlas-tab"
              onClick={() => setMode(m)}
            >
              {t(`modes.${m}`)} <span className="atlas-tab-count">{n(countOf(m))}</span>
            </button>
          ))}
        </div>
        <div className="filter-bar px-0">
          <SearchInput value={q} onChange={setQ} placeholder={t('search')} />
          <Select
            label={t('source')}
            className="w-56"
            value={source}
            onChange={(e) => setSource(e.target.value as Source | 'all')}
            options={[
              { value: 'all', label: t('sources.all') },
              { value: 'rule', label: t('sources.rule') },
              { value: 'group', label: t('sources.group') },
            ]}
          />
        </div>
      </Panel>

      {rules.isPending ? <Skeleton className="h-40" /> : null}
      {!rules.isPending && buckets.length === 0 ? (
        <Panel>
          <p className="m-0 text-ink-3">{t('empty')}</p>
        </Panel>
      ) : null}

      {buckets.map((b) => (
        <Panel
          key={b.surface}
          title={`${t(`surfaces.${b.surface}.title`)} (${n(b.entries.length)})`}
          bodyClassName="flex flex-col divide-y divide-line"
        >
          <p className="atlas-where">{t(`surfaces.${b.surface}.where`)}</p>
          {b.entries.map((e) => (
            <article key={e.id} className="atlas-row">
              <div className="atlas-main">
                <div className="flex flex-wrap items-center gap-1.5">
                  <strong>{e.rule ? e.rule.title : groupName(e.name)}</strong>
                  <Badge tone={e.source === 'rule' ? 'brand' : 'data'}>{t(`sources.${e.source}`)}</Badge>
                  {e.rule ? (
                    <>
                      <Badge>{tr(`kinds.${e.rule.kind}` as 'kinds.safety')}</Badge>
                      <Badge tone={e.rule.status === 'approved' ? 'green' : 'amber'}>
                        {tr(`statuses.${e.rule.status}` as 'statuses.draft')}
                      </Badge>
                      {e.rule.is_active ? null : <Badge tone="red">{tr('inactive')}</Badge>}
                    </>
                  ) : null}
                  {e.modes.length === 0 ? (
                    <Badge>{t('modes.everyone')}</Badge>
                  ) : mode === 'all' ? (
                    e.modes.map((m) => <Badge key={m}>{t(`modes.${m}`)}</Badge>)
                  ) : null}
                </div>
                <When entry={e} words={words} />
                {e.rule ? (
                  <p className="atlas-sample">
                    <span className="atlas-label">{t('shows')}</span>
                    {e.rule.content.a.title ? <strong>{e.rule.content.a.title}</strong> : null}{' '}
                    <span>{excerpt(e.rule.content.a.body, 160)}</span>
                    {e.rule.content.a.cta_label ? <span className="atlas-cta">{e.rule.content.a.cta_label}</span> : null}
                  </p>
                ) : e.group && tg.has(`where.${e.group}` as 'where.pattern') ? (
                  <p className="atlas-sample">
                    <span className="atlas-label">{t('place')}</span>
                    <span>{tg(`where.${e.group}` as 'where.pattern')}</span>
                  </p>
                ) : null}
                {e.key ? (
                  <span dir="ltr" className={cn('cell-mono', 'text-xs text-muted')}>
                    {e.key}
                  </span>
                ) : null}
              </div>
              <Link href={e.href} className="btn btn-sm">
                {e.source === 'rule' ? t('openRule') : t('openGroup')}
              </Link>
            </article>
          ))}
        </Panel>
      ))}
    </div>
  );
}

/** «چه وقت»: a rule's conditions (and exceptions) in words, or the group's trigger sentence. */
function When({ entry, words }: { entry: AtlasEntry; words: ConditionWords }) {
  const t = useTranslations('messageAtlas');
  if (entry.rule) {
    const when = whenOfRule(entry.rule, words);
    return (
      <p className="atlas-when">
        <span className="atlas-label">{t('whenLabel')}</span>
        {when.all.length === 0 ? <span>{t('always')}</span> : <span>{when.all.join(words.and)}</span>}
        {when.except.length > 0 ? (
          <span className="text-ink-3">
            {' '}
            · {t('except')} {when.except.join(words.or)}
          </span>
        ) : null}
      </p>
    );
  }
  const key = `when.${entry.group}` as 'when.pattern';
  return (
    <p className="atlas-when">
      <span className="atlas-label">{t('whenLabel')}</span>
      <span>{t.has(key) ? t(key) : t('whenInCode')}</span>
    </p>
  );
}
