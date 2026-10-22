'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { useCan, useCurrentAdmin } from '@/features/auth';
import { fieldError } from '@/shared/api';
import { cn, formatDate, formatNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  Icon,
  Panel,
  Select,
  TextInput,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import {
  CONFIG_GROUPS,
  numbersOf,
  rowsOf,
  useClinicalConfig,
  useDecide,
  usePropose,
  type ConfigGroup,
  type ConfigKey,
  type ConfigVersion,
} from '../api/clinical-config';
import { formatValue, parseValue } from '../lib/format';
import { diffRows, parseDraft, toDraft, type DraftRow } from '../lib/rows';
import { RowsEditor } from './RowsEditor';

/**
 * /clinical-config «آستانه‌ها و پیکربندی» (nbl_Admin_Config): every clinical key with the version in force today,
 * its source, effective date and approvers; a change is a proposal that needs two distinct medical advisors'
 * approvals and applies from its date (admin-api.md §23).
 */
export function ClinicalConfigScreen() {
  const t = useTranslations('clinicalConfig');
  const query = useClinicalConfig();
  const canPropose = useCan('config', 'manage');
  const [group, setGroup] = useState<ConfigGroup | ''>('');
  const [proposing, setProposing] = useState<string | null>(null);

  const keys = useMemo(
    () => (query.data?.keys ?? []).filter((k) => group === '' || k.group === group),
    [query.data, group],
  );
  const pending = (query.data?.keys ?? []).filter((k) => k.pending);

  return (
    <div className="flex flex-col gap-4">
      <div className="page-head">
        <div className="flex min-w-0 flex-col gap-1">
          <h1 className="page-title">{t('title')}</h1>
          <p className="text-ink-3 text-sm">{t('subtitle')}</p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex flex-1 flex-wrap gap-2" role="group" aria-label={t('groupsLabel')}>
          {(['', ...CONFIG_GROUPS] as const).map((g) => (
            <button
              key={g || 'all'}
              type="button"
              aria-pressed={group === g}
              onClick={() => setGroup(g)}
              className={cn('btn btn-sm min-h-11', group === g ? 'btn-primary' : 'btn-ghost')}
            >
              {t(`groups.${g || 'all'}`)}
            </button>
          ))}
        </div>
        {canPropose ? (
          <Button variant="primary" onClick={() => setProposing(keys[0]?.key ?? query.data?.keys[0]?.key ?? null)}>
            <Icon name="plus" size={16} />
            {t('propose')}
          </Button>
        ) : null}
      </div>

      {proposing && query.data ? (
        <ProposalForm
          keys={query.data.keys}
          initialKey={proposing}
          today={query.data.today}
          onClose={() => setProposing(null)}
        />
      ) : null}

      {pending.length > 0 && query.data ? (
        <PendingPanel items={pending} canApprove={query.data.can_approve} />
      ) : null}

      <Panel bodyClassName="">
        <ConfigTable
          keys={keys}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onPropose={canPropose ? setProposing : undefined}
        />
      </Panel>

      <div className="panel flex items-start gap-3 p-4 text-sm">
        <Icon name="alert" size={18} className="text-ink-3 mt-0.5 shrink-0" />
        <p className="text-ink-2">{t('note')}</p>
      </div>
    </div>
  );
}

/** day / score / week / month are words; mmHg, mg/dL, bpm, ng/mL stay as printed. */
function unitLabel(t: ReturnType<typeof useTranslations<'clinicalConfig'>>, unit: string): string {
  if (unit === 'day') return t('units.day');
  if (unit === 'score') return t('units.score');
  if (unit === 'week') return t('units.week');
  if (unit === 'month') return t('units.month');
  return unit;
}

function useNum() {
  const locale = useLocale();
  return (n: number) => formatNumber(n, locale);
}

/** The value cell: «180 / 120» for a numbers key, «6 rows» for a rows key. */
function useValueText() {
  const t = useTranslations('clinicalConfig');
  const num = useNum();
  return (k: ConfigKey, v: ConfigVersion) =>
    k.kind === 'rows'
      ? t('rows.count', { count: rowsOf(v.value).length, n: num(rowsOf(v.value).length) })
      : formatValue(k.fields, numbersOf(v.value), num);
}

function approverNames(v: ConfigVersion): string {
  return v.approvals.map((a) => a.name ?? `#${a.id}`).join(' · ');
}

function ConfigTable({
  keys,
  loading,
  error,
  onRetry,
  onPropose,
}: {
  keys: ConfigKey[];
  loading: boolean;
  error: unknown;
  onRetry: () => void;
  onPropose?: (key: string) => void;
}) {
  const t = useTranslations('clinicalConfig');
  const locale = useLocale();
  const num = useNum();
  const text = useValueText();

  const columns: Column<ConfigKey>[] = [
    {
      key: 'key',
      header: t('key'),
      cell: (k) => (
        <span className="font-mono text-xs" dir="ltr">
          {k.key}
        </span>
      ),
    },
    {
      key: 'value',
      header: t('value'),
      cell: (k) => (
        <span className="flex flex-col gap-0.5">
          <span className="font-semibold" dir="ltr">
            {text(k, k.current)}
          </span>
          {k.upcoming ? (
            <span className="text-ink-3 text-xs">
              {t('upcoming', {
                value: text(k, k.upcoming),
                date: formatDate(k.upcoming.effective_from, locale),
              })}
            </span>
          ) : null}
        </span>
      ),
    },
    { key: 'unit', header: t('unit'), cell: (k) => <span dir="ltr">{unitLabel(t, k.unit)}</span> },
    { key: 'source', header: t('source'), cell: (k) => <span className="text-ink-2">{k.current.source}</span> },
    {
      key: 'version',
      header: t('version'),
      cell: (k) => <Badge tone="brand">{`v${num(k.current.version)}`}</Badge>,
    },
    {
      key: 'effective',
      header: t('effectiveFrom'),
      cell: (k) => (k.current.effective_from ? formatDate(k.current.effective_from, locale) : '—'),
      className: 'cell-num whitespace-nowrap',
    },
    {
      key: 'approvers',
      header: t('approvers'),
      cell: (k) => (k.current.version === 1 ? <span className="text-ink-3">{t('codeDefault')}</span> : approverNames(k.current)),
    },
    ...(onPropose
      ? [
          {
            key: 'actions',
            header: <span className="sr-only">{t('actions')}</span>,
            cell: (k: ConfigKey) =>
              k.pending ? (
                <Badge tone="amber">{t('pendingBadge')}</Badge>
              ) : (
                <Button size="sm" variant="ghost" onClick={() => onPropose(k.key)}>
                  {t('change')}
                </Button>
              ),
            className: 'cell-actions',
          },
        ]
      : []),
  ];

  return (
    <DataTable
      columns={columns}
      items={keys}
      rowKey={(k) => k.key}
      loading={loading}
      error={error}
      onRetry={onRetry}
      emptyText={t('empty')}
      caption={t('title')}
    />
  );
}

function PendingPanel({ items, canApprove }: { items: ConfigKey[]; canApprove: boolean }) {
  const t = useTranslations('clinicalConfig');
  const locale = useLocale();
  const num = useNum();
  const me = useCurrentAdmin();
  const decide = useDecide();
  const notifyError = useNotifyError();
  const text = useValueText();

  const act = (id: number, action: 'approve' | 'reject') =>
    decide.mutate(
      { id, action },
      { onSuccess: () => toast.success(t(action === 'approve' ? 'approved' : 'rejected')), onError: notifyError },
    );

  return (
    <Panel title={t('pendingTitle')} bodyClassName="flex flex-col divide-y divide-line">
      {items.map((k) => {
        const p = k.pending!;
        const mine = p.proposed_by?.id === me?.id;
        const approvedByMe = p.approvals.some((a) => a.id === me?.id);
        return (
          <div key={k.key} className="flex flex-wrap items-center gap-3 p-4">
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <span className="flex flex-wrap items-center gap-2">
                <span className="font-mono text-xs" dir="ltr">
                  {k.key}
                </span>
                <Badge tone="amber">{`v${num(p.version)}`}</Badge>
              </span>
              <span className="text-sm">
                <span dir="ltr" className="text-ink-3 line-through">
                  {text(k, k.current)}
                </span>{' '}
                ←{' '}
                <span dir="ltr" className="font-semibold">
                  {text(k, p)}
                </span>
                {k.kind === 'rows' && k.schema ? <PendingRowsDiff k={k} p={p} /> : null}
              </span>
              <span className="text-ink-3 text-xs">
                {t('pendingMeta', {
                  date: formatDate(p.effective_from, locale),
                  by: p.proposed_by?.name ?? '—',
                  count: num(p.approvals.length),
                  required: num(p.required_approvals),
                })}
                {p.approvals.length > 0 ? ` · ${approverNames(p)}` : ''}
              </span>
              <span className="text-ink-3 text-xs">{t('sourceOf', { source: p.source })}</span>
              {p.note ? <span className="text-ink-2 text-xs">{p.note}</span> : null}
            </div>
            <span className="flex flex-wrap gap-2">
              {canApprove && !mine && !approvedByMe ? (
                <Button size="sm" variant="primary" onClick={() => act(p.id!, 'approve')} disabled={decide.isPending}>
                  {t('approve')}
                </Button>
              ) : null}
              {canApprove || mine ? (
                <Button size="sm" variant="ghost" onClick={() => act(p.id!, 'reject')} disabled={decide.isPending}>
                  {mine ? t('withdraw') : t('reject')}
                </Button>
              ) : null}
              {mine && canApprove ? <span className="text-ink-3 self-center text-xs">{t('selfHint')}</span> : null}
            </span>
          </div>
        );
      })}
    </Panel>
  );
}

function ProposalForm({
  keys,
  initialKey,
  today,
  onClose,
}: {
  keys: ConfigKey[];
  initialKey: string;
  today: string;
  onClose: () => void;
}) {
  const t = useTranslations('clinicalConfig');
  const save = usePropose();
  const num = useNum();
  const notifyError = useNotifyError();
  const [keyName, setKeyName] = useState(initialKey);
  const def = keys.find((k) => k.key === keyName) ?? keys[0]!;
  const initial = (k: ConfigKey) =>
    Object.fromEntries(k.fields.map((f) => [f.name, String(numbersOf(k.current.value)[f.name] ?? f.default)]));
  const initialDraft = (k: ConfigKey) => (k.schema ? toDraft(k.schema, rowsOf(k.current.value)) : []);
  const [raw, setRaw] = useState<Record<string, string>>(() => initial(def));
  const [draft, setDraft] = useState<DraftRow[]>(() => initialDraft(def));
  const parsed = def.schema ? parseDraft(def.schema, draft) : null;
  const [effective, setEffective] = useState(today);
  const [source, setSource] = useState(def.current.source);
  const [note, setNote] = useState('');
  const err = (f: string) => fieldError(save.error, f);

  return (
    <Panel
      title={t('proposeTitle')}
      actions={
        <Button size="sm" variant="ghost" onClick={onClose} aria-label={t('close')}>
          <Icon name="close" size={16} />
        </Button>
      }
    >
      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={(e) => {
          e.preventDefault();
          const value = parsed ? (parsed.rows ? { rows: parsed.rows } : null) : parseValue(def.fields, raw);
          if (!value) return;
          save.mutate(
            { key: def.key, value, effective_from: effective, source, note },
            {
              onSuccess: () => {
                toast.success(t('proposed'));
                onClose();
              },
              onError: (e2) => {
                if (!['value', 'effective_from', 'source', 'note'].some((f) => fieldError(e2, f))) notifyError(e2);
              },
            },
          );
        }}
      >
        <Select
          label={t('key')}
          value={def.key}
          onChange={(e) => {
            const next = keys.find((k) => k.key === e.target.value);
            if (!next) return;
            setKeyName(next.key);
            setRaw(initial(next));
            setDraft(initialDraft(next));
            setSource(next.current.source);
          }}
          options={keys.filter((k) => !k.pending).map((k) => ({ value: k.key, label: k.key }))}
        />
        <TextInput
          label={t('effectiveFrom')}
          type="date"
          required
          min={today}
          dir="ltr"
          value={effective}
          onChange={(e) => setEffective(e.target.value)}
          error={err('effective_from')}
        />
        {def.kind === 'rows' && def.schema && parsed ? (
          <RowsEditor
            table={def.schema}
            draft={draft}
            parsed={parsed}
            current={rowsOf(def.current.value)}
            onChange={setDraft}
          />
        ) : null}
        {def.fields.map((f) => (
          <TextInput
            key={f.name}
            label={`${f.name} (${unitLabel(t, def.unit)})`}
            hint={t('range', { min: num(f.min), max: num(f.max), def: num(f.default) })}
            type="number"
            inputMode="decimal"
            step="any"
            required
            min={f.min}
            max={f.max}
            dir="ltr"
            value={raw[f.name] ?? ''}
            onChange={(e) => setRaw((r) => ({ ...r, [f.name]: e.target.value }))}
          />
        ))}
        {err('value') ? <p className="text-danger text-sm sm:col-span-2">{err('value')}</p> : null}
        <TextInput
          className="sm:col-span-2"
          label={t('source')}
          maxLength={255}
          value={source}
          onChange={(e) => setSource(e.target.value)}
          error={err('source')}
        />
        <TextInput
          className="sm:col-span-2"
          label={t('noteLabel')}
          maxLength={500}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          error={err('note')}
        />
        <p className="text-ink-3 text-xs sm:col-span-2">{t('twoApprovals')}</p>
        <div className="flex gap-2 sm:col-span-2">
          <Button type="submit" variant="primary" loading={save.isPending} disabled={parsed ? !parsed.rows : false}>
            {t('submit')}
          </Button>
          <Button type="button" variant="ghost" onClick={onClose}>
            {t('cancel')}
          </Button>
        </div>
      </form>
    </Panel>
  );
}

/** The changed / added / removed rows of a pending rows proposal against the version in force. */
function PendingRowsDiff({ k, p }: { k: ConfigKey; p: ConfigVersion }) {
  const t = useTranslations('clinicalConfig');
  const num = useNum();
  const d = diffRows(k.schema!, rowsOf(k.current.value), rowsOf(p.value));
  const count = (s: 'added' | 'changed') => d.rows.filter((r) => r.status === s).length;
  return (
    <span className="text-ink-3 block text-xs">
      {t('rows.diff', { changed: num(count('changed')), added: num(count('added')), removed: num(d.removed) })}
    </span>
  );
}
