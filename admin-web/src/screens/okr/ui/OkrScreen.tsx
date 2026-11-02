'use client';

import { useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { useCan } from '@/features/auth';
import { fieldError, isApiError } from '@/shared/api';
import { formatDate, formatNumber } from '@/shared/lib';
import { Badge, Button, confirm, ErrorState, PageHeader, Panel, Select, Skeleton, TextInput, toast, useNotifyError, type BadgeTone } from '@/shared/ui';

import {
  useDeleteKeyResult,
  useDeleteObjective,
  useOkr,
  useSaveKeyResult,
  useSaveObjective,
  type KeyResult,
  type KrStatus,
  type Objective,
  type Okr,
} from '../api/okr';
import { elapsedPct, statusHint } from '../lib/status';

const TONE: Record<KrStatus, BadgeTone> = { on_track: 'data', at_risk: 'amber', done: 'green', no_data: 'neutral' };
const FILL: Record<KrStatus, string> = { on_track: 'bg-data-deep', at_risk: 'bg-amber-deep', done: 'bg-green-deep', no_data: 'bg-track' };

/** «OKR» (B-N9-09): objectives and key results; a bound key result follows a live aggregate metric. */
export function OkrScreen() {
  const t = useTranslations('okr');
  const locale = useLocale();
  const canWrite = useCan('dashboard', 'manage');
  const [period, setPeriod] = useState('');
  const [editing, setEditing] = useState<Objective | 'new' | null>(null);
  const q = useOkr(period);
  const d = q.data;
  const num = (n: number) => formatNumber(n, locale);

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('title')}
        meta={t('subtitle')}
        actions={canWrite ? <Button variant="primary" onClick={() => setEditing('new')}>{t('newObjective')}</Button> : undefined}
      />
      {q.error && !d ? (
        <Panel>
          <ErrorState error={q.error} onRetry={() => q.refetch()} />
        </Panel>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-3">
            <div className="w-56">
              <Select
                aria-label={t('period')}
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
                options={[{ value: '', label: t('allPeriods') }, ...(d?.periods ?? []).map((p) => ({ value: p, label: p }))]}
              />
            </div>
            {d?.summary ? (
              <p className="m-0 flex flex-wrap items-center gap-2 text-ink-3" aria-live="polite">
                <span>{t('summary', { n: num(d.summary.on_track + d.summary.done), of: num(d.summary.key_results) })}</span>
                {d.summary.at_risk > 0 ? <Badge tone="amber">{t('status.at_risk')} {num(d.summary.at_risk)}</Badge> : null}
                {d.summary.no_data > 0 ? <Badge>{t('status.no_data')} {num(d.summary.no_data)}</Badge> : null}
              </p>
            ) : null}
          </div>
          {editing && d ? <ObjectiveForm key={editing === 'new' ? 'new' : editing.id} objective={editing === 'new' ? null : editing} periods={d.periods} onClose={() => setEditing(null)} /> : null}
          {!d ? (
            <Skeleton className="h-48 w-full" />
          ) : d.objectives.length === 0 ? (
            <Panel>
              <p className="m-0 text-ink-3">{t('empty')}</p>
            </Panel>
          ) : (
            d.objectives.map((o) => <ObjectiveCard key={o.id} o={o} okr={d} canWrite={canWrite} onEdit={() => setEditing(o)} />)
          )}
          {d ? <p className="m-0 text-xs text-muted">{t('liveNote')}</p> : null}
        </>
      )}
    </div>
  );
}

function fmt(v: number | null, unit: string, locale: string): string {
  if (v == null) return '—';
  const n = formatNumber(v, locale);
  return unit === 'pct' ? `${n}٪` : n;
}

function StatusBadge({ s }: { s: KrStatus }) {
  const t = useTranslations('okr');
  return <Badge tone={TONE[s]}>{t(`status.${s}`)}</Badge>;
}

function ObjectiveCard({ o, okr, canWrite, onEdit }: { o: Objective; okr: Okr; canWrite: boolean; onEdit: () => void }) {
  const t = useTranslations('okr');
  const locale = useLocale();
  const [adding, setAdding] = useState(false);
  const del = useDeleteObjective();
  const notifyError = useNotifyError();
  const elapsed = elapsedPct(o.starts_on, o.ends_on, new Date());
  return (
    <Panel
      title={
        <span className="flex flex-wrap items-center gap-2">
          {o.title} <StatusBadge s={o.status} />
        </span>
      }
      actions={
        canWrite ? (
          <div className="flex gap-1">
            <Button size="sm" variant="ghost" onClick={() => setAdding(true)}>{t('addKr')}</Button>
            <Button size="sm" variant="ghost" onClick={onEdit}>{t('edit')}</Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                if (await confirm({ message: t('confirmDeleteObjective'), tone: 'danger', confirmLabel: t('delete') })) del.mutate(o.id, { onError: notifyError });
              }}
            >
              {t('delete')}
            </Button>
          </div>
        ) : undefined
      }
    >
      <p className="m-0 mb-3 text-[13px] text-ink-3">
        {o.owner ? `${t('owner')}: ${o.owner} · ` : ''}
        {o.period} · {formatDate(o.starts_on, locale)} – {formatDate(o.ends_on, locale)}
        {o.progress_pct != null ? ` · ${formatNumber(o.progress_pct, locale)}٪` : ''}
      </p>
      <p className="m-0 mb-3 text-xs text-muted">{elapsed == null ? t('paceRule', { slack: formatNumber(10, locale) }) : t('paceNote', { elapsed: formatNumber(elapsed, locale), slack: formatNumber(10, locale) })}</p>
      <ul className="m-0 flex list-none flex-col gap-3 p-0">
        {o.key_results.map((kr) => (
          <KrRow key={kr.id} kr={kr} canWrite={canWrite} okr={okr} elapsed={elapsed} />
        ))}
      </ul>
      {adding ? <KrForm objectiveId={o.id} kr={null} okr={okr} onClose={() => setAdding(false)} /> : null}
    </Panel>
  );
}

function KrRow({ kr, canWrite, okr, elapsed }: { kr: KeyResult; canWrite: boolean; okr: Okr; elapsed: number | null }) {
  const t = useTranslations('okr');
  const locale = useLocale();
  const [edit, setEdit] = useState(false);
  const del = useDeleteKeyResult();
  const notifyError = useNotifyError();
  const pct = kr.progress_pct ?? 0;
  const hint = statusHint(kr.status, kr.progress_pct, elapsed);
  const hintText = t(`statusHint.${hint.key}`, { elapsed: formatNumber(hint.elapsed ?? 0, locale), floor: formatNumber(hint.floor ?? 0, locale), progress: formatNumber(kr.progress_pct ?? 0, locale) });
  if (edit) return <li><KrForm objectiveId={kr.objective_id} kr={kr} okr={okr} onClose={() => setEdit(false)} /></li>;
  return (
    <li className="grid items-center gap-2 sm:grid-cols-[1fr_16rem_auto]">
      <div>
        <div className="font-semibold text-ink">{kr.title}</div>
        <div className="text-xs text-muted">
          {kr.metric_key ? t(`metric.${kr.metric_key}` as 'metric.dau') : t('manual')}
          {kr.owner ? ` · ${kr.owner}` : ''} · {fmt(kr.baseline, kr.unit, locale)} → {fmt(kr.target, kr.unit, locale)}
        </div>
        <div className="text-xs text-muted">{hintText}</div>
      </div>
      <div className="flex items-center gap-2">
        <div
          className="h-2.5 flex-1 overflow-hidden rounded bg-track"
          role="progressbar"
          aria-label={kr.title}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={kr.progress_pct ?? undefined}
        >
          {kr.progress_pct != null ? <div className={`h-full w-(--w) rounded ${FILL[kr.status]}`} style={{ '--w': `${Math.max(2, pct)}%` } as React.CSSProperties} /> : null}
        </div>
        <span className="w-14 text-end font-semibold text-ink" dir="ltr">{fmt(kr.current, kr.unit, locale)}</span>
      </div>
      <div className="flex items-center gap-1">
        <StatusBadge s={kr.status} />
        {canWrite ? (
          <>
            <Button size="sm" variant="ghost" onClick={() => setEdit(true)}>{t('edit')}</Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                if (await confirm({ message: t('confirmDeleteKr'), tone: 'danger', confirmLabel: t('delete') })) del.mutate(kr.id, { onError: notifyError });
              }}
            >
              {t('delete')}
            </Button>
          </>
        ) : null}
      </div>
    </li>
  );
}

function ObjectiveForm({ objective, periods, onClose }: { objective: Objective | null; periods: string[]; onClose: () => void }) {
  const t = useTranslations('okr');
  const save = useSaveObjective();
  const [f, setF] = useState({
    title: objective?.title ?? '',
    owner: objective?.owner ?? '',
    period: objective?.period ?? periods[0] ?? '',
    starts_on: objective?.starts_on ?? '',
    ends_on: objective?.ends_on ?? '',
  });
  const set = (k: keyof typeof f) => (e: React.ChangeEvent<HTMLInputElement>) => setF({ ...f, [k]: e.target.value });
  return (
    <Panel title={objective ? t('editObjective') : t('newObjective')}>
      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate({ id: objective?.id, input: f }, { onSuccess: () => { toast.success(t('saved')); onClose(); } });
        }}
      >
        <TextInput className="sm:col-span-2" label={t('objectiveTitle')} required value={f.title} onChange={set('title')} error={fieldError(save.error, 'title')} />
        <TextInput label={t('owner')} value={f.owner} onChange={set('owner')} error={fieldError(save.error, 'owner')} />
        <TextInput label={t('period')} required value={f.period} onChange={set('period')} error={fieldError(save.error, 'period')} />
        <TextInput label={t('startsOn')} type="date" required value={f.starts_on} onChange={set('starts_on')} error={fieldError(save.error, 'starts_on')} />
        <TextInput label={t('endsOn')} type="date" required value={f.ends_on} onChange={set('ends_on')} error={fieldError(save.error, 'ends_on')} />
        <FormError error={save.error} />
        <div className="flex gap-2 sm:col-span-2">
          <Button type="submit" variant="primary" loading={save.isPending}>{t('save')}</Button>
          <Button variant="ghost" onClick={onClose}>{t('cancel')}</Button>
        </div>
      </form>
    </Panel>
  );
}

function KrForm({ objectiveId, kr, okr, onClose }: { objectiveId: number; kr: KeyResult | null; okr: Okr; onClose: () => void }) {
  const t = useTranslations('okr');
  const save = useSaveKeyResult();
  const [f, setF] = useState({
    title: kr?.title ?? '',
    metric_key: kr?.metric_key ?? '',
    unit: kr?.unit ?? 'count',
    baseline: kr ? String(kr.baseline) : '0',
    target: kr ? String(kr.target) : '',
    manual_value: kr?.manual_value != null ? String(kr.manual_value) : '',
    owner: kr?.owner ?? '',
  });
  const bound = f.metric_key !== '';
  const set = (k: keyof typeof f) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setF({ ...f, [k]: e.target.value });
  return (
    <form
      className="mt-3 grid gap-3 rounded border border-line p-3 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate({ objectiveId, id: kr?.id, input: f }, { onSuccess: () => { toast.success(t('saved')); onClose(); } });
      }}
    >
      <TextInput className="sm:col-span-2" label={t('krTitle')} required value={f.title} onChange={set('title')} error={fieldError(save.error, 'title')} />
      <Select
        label={t('bindTo')}
        value={f.metric_key}
        onChange={set('metric_key')}
        error={fieldError(save.error, 'metric_key')}
        options={[{ value: '', label: t('manual') }, ...okr.metrics.map((m) => ({ value: m.key, label: t(`metric.${m.key}` as 'metric.dau') }))]}
      />
      {bound ? (
        <p className="m-0 self-end text-xs text-muted">{t('boundHint')}</p>
      ) : (
        <Select label={t('unit')} value={f.unit} onChange={set('unit')} options={okr.units.map((u) => ({ value: u, label: t(`units.${u}` as 'units.count') }))} />
      )}
      <TextInput label={t('baseline')} type="number" step="any" required value={f.baseline} onChange={set('baseline')} error={fieldError(save.error, 'baseline')} />
      <TextInput label={t('target')} type="number" step="any" required value={f.target} onChange={set('target')} error={fieldError(save.error, 'target')} />
      {bound ? null : <TextInput label={t('manualValue')} type="number" step="any" value={f.manual_value} onChange={set('manual_value')} error={fieldError(save.error, 'manual_value')} />}
      <TextInput label={t('owner')} value={f.owner} onChange={set('owner')} error={fieldError(save.error, 'owner')} />
      <FormError error={save.error} />
      <div className="flex gap-2 sm:col-span-2">
        <Button type="submit" variant="primary" loading={save.isPending}>{t('save')}</Button>
        <Button variant="ghost" onClick={onClose}>{t('cancel')}</Button>
      </div>
    </form>
  );
}

function FormError({ error }: { error: unknown }) {
  if (!error || (isApiError(error) && error.status === 422)) return null;
  return <p role="alert" className="m-0 text-danger-deep sm:col-span-2">{isApiError(error) ? error.message : String(error)}</p>;
}
