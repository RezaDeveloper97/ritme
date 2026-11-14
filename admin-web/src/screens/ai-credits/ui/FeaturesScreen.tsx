'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useLocalized } from '@/shared/i18n';
import { formatNumber, toIntOrNull } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  FormDialog,
  Panel,
  SearchInput,
  Select,
  Switch,
  TextInput,
  toast,
  TranslatableField,
  useNotifyError,
  type Column,
  type Translations,
} from '@/shared/ui';

import { useFeatures, useSaveFeature, type AiFeature, type AiModelOption } from '../api/ai-credits';
import { FEATURE_FILTERS, filterFeatures, missingCapabilities, modelChoices, type FeatureFilter } from '../lib/features';
import { CapabilityBadges, useCanManage } from './parts';

/**
 * /ai/features — the AI feature slots (declared in code): which model serves each, its fallback, how many credits
 * one use costs, and whether it is on. Built for ~100 rows: client-side search + status filter, edit in a dialog.
 */
export function FeaturesScreen() {
  const t = useTranslations('aiCredits.features');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const localize = useLocalized();
  const canManage = useCanManage();
  const query = useFeatures();
  const [q, setQ] = useState('');
  const [filter, setFilter] = useState<FeatureFilter>('all');
  const [editing, setEditing] = useState<AiFeature | null>(null);

  const items = query.data?.items;
  const shown = useMemo(() => (items ? filterFeatures(items, q, filter) : undefined), [items, q, filter]);
  const problems = useMemo(() => (items ? filterFeatures(items, '', 'problems').length : 0), [items]);

  const modelLabel = (m: AiFeature['model']) =>
    m ? (
      <span className="flex flex-col items-start">
        <span dir="auto">{m.title}</span>
        {!m.is_active ? <Badge tone="amber">{tc('inactive')}</Badge> : null}
      </span>
    ) : (
      <span className="text-muted">—</span>
    );

  const columns: Column<AiFeature>[] = [
    {
      key: 'feature',
      header: t('feature'),
      cell: (f) => (
        <span className="flex flex-col items-start gap-1">
          <span className="font-semibold" dir="auto">
            {localize(f.title) || f.key}
          </span>
          <code dir="ltr" className="text-xs text-muted">
            {f.key}
          </code>
          <span className="flex flex-wrap gap-1">
            {!f.declared ? <Badge tone="neutral">{t('undeclared')}</Badge> : null}
            {f.declared && f.is_active && !f.usable ? <Badge tone="red">{t('noUsableModel')}</Badge> : null}
          </span>
        </span>
      ),
    },
    { key: 'needs', header: t('needs'), cell: (f) => <CapabilityBadges caps={f.needs} /> },
    { key: 'model', header: t('model'), cell: (f) => modelLabel(f.model) },
    { key: 'fallback', header: t('fallback'), cell: (f) => modelLabel(f.fallback_model) },
    {
      key: 'cost',
      header: t('cost'),
      cell: (f) => t('credits', { count: f.credit_cost, n: formatNumber(f.credit_cost, locale) }),
      className: 'cell-num whitespace-nowrap',
    },
    {
      key: 'status',
      header: tc('status'),
      cell: (f) => <Badge tone={f.is_active ? 'green' : 'neutral'}>{f.is_active ? tc('active') : tc('inactive')}</Badge>,
    },
    {
      key: 'actions',
      header: <span className="sr-only">{tc('actions')}</span>,
      cell: (f) =>
        canManage ? (
          <Button
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              setEditing(f);
            }}
          >
            {tc('edit')}
          </Button>
        ) : null,
      className: 'cell-actions',
    },
  ];

  return (
    <Panel title={t('title')} bodyClassName="">
      <p className="panel-note">{t('intro')}</p>
      <div className="flex flex-wrap items-end gap-3 px-4 pb-3 sm:px-5">
        <div className="min-w-56 flex-1">
          <SearchInput value={q} onChange={setQ} placeholder={t('search')} delay={150} />
        </div>
        <Select
          label={t('filter')}
          value={filter}
          onChange={(e) => setFilter(e.target.value as FeatureFilter)}
          options={FEATURE_FILTERS.map((f) => ({ value: f, label: t(`filters.${f}`) }))}
        />
        {problems > 0 ? <Badge tone="red">{t('problemsCount', { count: problems })}</Badge> : null}
      </div>
      <DataTable
        columns={columns}
        items={shown}
        rowKey={(f) => f.id}
        loading={query.isPending}
        error={query.error}
        onRetry={() => query.refetch()}
        onRowClick={canManage ? (f) => setEditing(f) : undefined}
        emptyText={q || filter !== 'all' ? t('noMatch') : t('empty')}
        caption={t('title')}
        skeletonRows={8}
      />
      {editing && query.data ? (
        <FeatureDialog
          key={editing.id}
          feature={editing}
          models={query.data.models}
          maxCost={query.data.limits.max_credit_cost}
          onClose={() => setEditing(null)}
        />
      ) : null}
    </Panel>
  );
}

function FeatureDialog({
  feature,
  models,
  maxCost,
  onClose,
}: {
  feature: AiFeature;
  models: AiModelOption[];
  maxCost: number;
  onClose: () => void;
}) {
  const t = useTranslations('aiCredits.features');
  const tc = useTranslations('crud');
  const notifyError = useNotifyError();
  const save = useSaveFeature();
  const [title, setTitle] = useState<Translations>(feature.title);
  const [modelId, setModelId] = useState(feature.model_id ? String(feature.model_id) : '');
  const [fallbackId, setFallbackId] = useState(feature.fallback_model_id ? String(feature.fallback_model_id) : '');
  const [cost, setCost] = useState(String(feature.credit_cost));
  const [active, setActive] = useState(feature.is_active);

  const options = (current: number | null) => [
    { value: '', label: t('noModel') },
    ...modelChoices(models, feature.needs, current).map((m) => {
      const missing = missingCapabilities(m, feature.needs);
      const flags = [
        !m.is_active ? tc('inactive') : null,
        missing.length > 0 ? t('lacks', { caps: missing.join(', ') }) : null,
        !m.configured ? t('notConfigured') : null,
      ].filter(Boolean);
      return { value: String(m.id), label: `${m.title} (${m.provider})${flags.length ? ` — ${flags.join('، ')}` : ''}` };
    }),
  ];

  const submit = () =>
    save.mutate(
      {
        id: feature.id,
        body: {
          title,
          model_id: modelId ? Number(modelId) : null,
          fallback_model_id: fallbackId ? Number(fallbackId) : null,
          credit_cost: toIntOrNull(cost),
          is_active: active,
        },
      },
      {
        onSuccess: () => {
          toast.success(tc('saved'));
          onClose();
        },
        onError: notifyError,
      },
    );

  return (
    <FormDialog open title={t('editTitle', { key: feature.key })} onClose={onClose} onSubmit={submit} submitLabel={tc('saveChanges')} saving={save.isPending}>
      <div className="flex flex-col gap-1 text-sm text-ink-3">
        <span>{t('needsHint')}</span>
        <CapabilityBadges caps={feature.needs} />
        {!feature.declared ? <span>{t('undeclaredHint')}</span> : null}
      </div>
      <TranslatableField name="title" label={t('titleField')} value={title} onChange={setTitle} maxLength={128} errors={fieldErrorsOf(save.error)} />
      <Select
        label={t('model')}
        hint={t('modelHint')}
        value={modelId}
        onChange={(e) => setModelId(e.target.value)}
        options={options(feature.model_id)}
        error={fieldError(save.error, 'model_id')}
      />
      <Select
        label={t('fallback')}
        hint={t('fallbackHint')}
        value={fallbackId}
        onChange={(e) => setFallbackId(e.target.value)}
        options={options(feature.fallback_model_id).filter((o) => o.value === '' || o.value !== modelId)}
        error={fieldError(save.error, 'fallback_model_id')}
      />
      <TextInput
        label={t('cost')}
        hint={t('costHint', { max: maxCost })}
        type="number"
        min={0}
        max={maxCost}
        value={cost}
        onChange={(e) => setCost(e.target.value)}
        required
        error={fieldError(save.error, 'credit_cost')}
      />
      <Switch label={tc('isActive')} hint={t('activeHint')} checked={active} onChange={setActive} />
    </FormDialog>
  );
}
