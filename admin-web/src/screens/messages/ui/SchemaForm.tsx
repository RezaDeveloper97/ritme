'use client';

import { useTranslations } from 'next-intl';

import { fieldError } from '@/shared/api';
import { cn, useNumber } from '@/shared/lib';
import { Button, Icon, Select, Switch, TextArea, TextInput } from '@/shared/ui';

import type { SchemaField } from '../api/messages';
import { emptyObject, type Draft } from '../lib/schema-form';

/**
 * A payload editor rendered from a registered item's schema (admin-api.md §13):
 * texts, text lists (one per line), integers, booleans, enums, nested objects and
 * object lists. Errors are read from the failed write as `payload.<path>`.
 */
export function SchemaForm({
  fields,
  value,
  onChange,
  error,
  dir,
  path = 'payload',
}: {
  fields: readonly SchemaField[];
  value: Draft;
  onChange: (next: Draft) => void;
  error: unknown;
  dir?: 'rtl' | 'ltr';
  path?: string;
}) {
  return (
    <div className="flex flex-col gap-4">
      {fields.map((f) => (
        <SchemaInput
          key={f.key}
          field={f}
          value={value[f.key]}
          onChange={(v) => onChange({ ...value, [f.key]: v })}
          error={error}
          dir={dir}
          path={`${path}.${f.key}`}
        />
      ))}
    </div>
  );
}

function SchemaInput({
  field,
  value,
  onChange,
  error,
  dir,
  path,
}: {
  field: SchemaField;
  value: unknown;
  onChange: (v: unknown) => void;
  error: unknown;
  dir?: 'rtl' | 'ltr';
  path: string;
}) {
  const t = useTranslations('smartMessages');
  const n = useNumber();
  const err = fieldError(error, path);
  const label = (
    <>
      <span dir="ltr">{field.key}</span>
      {field.nullable ? <span className="ms-2 font-normal text-muted">{t('optional')}</span> : null}
    </>
  );
  const str = typeof value === 'string' ? value : value === null || value === undefined ? '' : String(value);

  switch (field.kind) {
    case 'text': {
      const long = (field.max_length ?? 2000) > 255;
      return long ? (
        <TextArea
          label={label}
          dir={dir}
          rows={4}
          maxLength={field.max_length}
          value={str}
          onChange={(e) => onChange(e.target.value)}
          error={err}
        />
      ) : (
        <TextInput
          label={label}
          dir={dir}
          maxLength={field.max_length}
          value={str}
          onChange={(e) => onChange(e.target.value)}
          error={err}
        />
      );
    }
    case 'url':
      return (
        <TextInput
          label={label}
          dir="ltr"
          hint={t('urlHint')}
          maxLength={field.max_length}
          value={str}
          onChange={(e) => onChange(e.target.value)}
          error={err}
        />
      );
    case 'integer':
      return (
        <TextInput
          label={label}
          type="number"
          min={field.min}
          max={field.max}
          hint={field.min !== undefined && field.max !== undefined ? t('range', { min: n(field.min), max: n(field.max) }) : undefined}
          value={str}
          onChange={(e) => onChange(e.target.value)}
          error={err}
        />
      );
    case 'boolean':
      return <Switch label={field.key} checked={value === true} onChange={(v) => onChange(v)} />;
    case 'enum':
      return (
        <Select
          label={label}
          value={str}
          onChange={(e) => onChange(e.target.value === '' ? null : e.target.value)}
          options={[...(field.nullable ? [{ value: '', label: '—' }] : []), ...(field.values ?? []).map((v) => ({ value: v, label: v }))]}
          error={err}
        />
      );
    case 'enum_list': {
      const list = Array.isArray(value) ? (value as string[]) : [];
      return (
        <fieldset className="field m-0 border-0 p-0">
          <legend className="field-label mb-1.5 p-0">{label}</legend>
          <div className="flex flex-wrap gap-1.5">
            {(field.values ?? []).map((v) => {
              const on = list.includes(v);
              return (
                <button
                  key={v}
                  type="button"
                  aria-pressed={on}
                  className={cn('btn btn-sm', on ? 'btn-primary' : 'btn-ghost')}
                  onClick={() => onChange(on ? list.filter((x) => x !== v) : [...list, v])}
                  dir="ltr"
                >
                  {v}
                </button>
              );
            })}
          </div>
          {err ? <span className="field-error">{err}</span> : null}
        </fieldset>
      );
    }
    case 'text_list': {
      const list = Array.isArray(value) ? value.map((v) => (typeof v === 'string' ? v : String(v))) : [];
      return (
        <TextArea
          label={
            <>
              {label}
              <span className="ms-2 font-normal text-muted">{t('listHint')}</span>
            </>
          }
          dir={dir}
          rows={Math.max(3, list.length)}
          value={list.join('\n')}
          onChange={(e) => onChange(e.target.value.split(/\r\n|\r|\n/))}
          error={err ?? fieldError(error, `${path}.0`)}
        />
      );
    }
    case 'object': {
      const obj = value && typeof value === 'object' && !Array.isArray(value) ? (value as Draft) : {};
      return (
        <fieldset className="m-0 flex min-w-0 flex-col gap-3 rounded-xl border border-line p-3">
          <legend className="field-label px-1" dir="ltr">
            {field.key}
          </legend>
          <SchemaForm fields={field.fields ?? []} value={obj} onChange={onChange} error={error} dir={dir} path={path} />
        </fieldset>
      );
    }
    case 'object_list': {
      const list = Array.isArray(value) ? (value as Draft[]) : [];
      const max = field.max_items ?? 20;
      return (
        <fieldset className="m-0 flex min-w-0 flex-col gap-3 border-0 p-0">
          <legend className="field-label p-0">
            <span dir="ltr">{field.key}</span> <span className="text-xs text-muted">({t('countOf', { count: n(list.length), max: n(max) })})</span>
          </legend>
          {list.map((item, i) => (
            <div key={i} className="flex flex-col gap-3 rounded-xl border border-line p-3">
              <SchemaForm
                fields={field.fields ?? []}
                value={item}
                onChange={(next) => onChange(list.map((x, j) => (j === i ? next : x)))}
                error={error}
                dir={dir}
                path={`${path}.${i}`}
              />
              <div>
                <Button size="sm" variant="danger" onClick={() => onChange(list.filter((_, j) => j !== i))}>
                  {t('remove')}
                </Button>
              </div>
            </div>
          ))}
          {err ? <span className="field-error">{err}</span> : null}
          <div>
            <Button size="sm" disabled={list.length >= max} onClick={() => onChange([...list, emptyObject(field.fields ?? [])])}>
              <Icon name="plus" size={14} />
              {t('addItem')}
            </Button>
          </div>
        </fieldset>
      );
    }
  }
}
