'use client';

import { useLocale, useTranslations } from 'next-intl';

import { Button, Field, Select, TextInput } from '@/shared/ui';

import type { FactDef } from '../api/facts';
import { emptyCond, inputKind, onFactChange, opsFor, type CondForm, type ConditionsForm } from '../model/conditions';

function ValueInput({ def, c, onChange }: { def: FactDef | undefined; c: CondForm; onChange: (v: string) => void }) {
  const t = useTranslations('ruleConditions');
  const kind = inputKind(def, c.op);
  const unit = def?.unit ?? undefined;
  switch (kind) {
    case 'number':
      return <TextInput label={t('value')} dir="ltr" type="number" value={c.value} onChange={(e) => onChange(e.target.value)} hint={unit} />;
    case 'interval': {
      const [lo = '', hi = ''] = c.value.split(',').map((s) => s.trim());
      return (
        <div className="grid grid-cols-2 gap-2">
          <TextInput label={t('from')} dir="ltr" type="number" value={lo} onChange={(e) => onChange(`${e.target.value}, ${hi}`)} hint={unit} />
          <TextInput label={t('to')} dir="ltr" type="number" value={hi} onChange={(e) => onChange(`${lo}, ${e.target.value}`)} />
        </div>
      );
    }
    case 'bool':
      return (
        <Select
          label={t('value')}
          value={c.value === 'true' || c.value === 'false' ? c.value : ''}
          onChange={(e) => onChange(e.target.value)}
          options={[
            { value: '', label: '—' },
            { value: 'true', label: t('yes') },
            { value: 'false', label: t('no') },
          ]}
        />
      );
    case 'date':
      return <TextInput label={t('value')} dir="ltr" type="date" value={c.value} onChange={(e) => onChange(e.target.value)} />;
    case 'enum-single':
      return (
        <Select
          label={t('value')}
          dir="ltr"
          value={c.value}
          onChange={(e) => onChange(e.target.value)}
          options={[{ value: '', label: '—' }, ...(def?.enum ?? []).map((v) => ({ value: v, label: v }))]}
        />
      );
    case 'enum-multi': {
      const picked = new Set(c.value.split(',').map((s) => s.trim()).filter(Boolean));
      const toggle = (v: string, on: boolean) => {
        const next = new Set(picked);
        if (on) next.add(v);
        else next.delete(v);
        onChange((def?.enum ?? []).filter((x) => next.has(x)).join(', '));
      };
      return (
        <Field label={t('values')}>
          <div className="flex flex-wrap gap-x-3 gap-y-1" dir="ltr">
            {(def?.enum ?? []).map((v) => (
              <label key={v} className="flex items-center gap-1 text-[13px]">
                <input type="checkbox" checked={picked.has(v)} onChange={(e) => toggle(v, e.target.checked)} />
                {v}
              </label>
            ))}
          </div>
        </Field>
      );
    }
    default:
      return (
        <TextInput
          label={t('value')}
          dir="ltr"
          value={c.value}
          onChange={(e) => onChange(e.target.value)}
          hint={c.op === 'in' || c.op === 'nin' || c.op === 'between' ? t('listHint') : undefined}
        />
      );
  }
}

function CondRow({
  c,
  facts,
  onChange,
  onRemove,
}: {
  c: CondForm;
  facts: readonly FactDef[];
  onChange: (c: CondForm) => void;
  onRemove: () => void;
}) {
  const t = useTranslations('ruleConditions');
  const locale = useLocale();
  const def = facts.find((f) => f.name === c.fact);
  const groups = Array.from(new Set(facts.map((f) => f.group)));
  const description = def ? (def.description[locale] ?? def.description.en ?? '') : '';
  return (
    <div className="flex flex-col gap-1">
      <div className="grid grid-cols-[minmax(0,1.3fr)_8rem_minmax(0,1.2fr)_auto] items-end gap-2 max-md:grid-cols-1">
        <Field label={t('fact')}>
          <select className="input" dir="ltr" value={c.fact} onChange={(e) => onChange(onFactChange(c, e.target.value, facts))}>
            <option value="">—</option>
            {c.fact && !def ? <option value={c.fact}>{c.fact}</option> : null}
            {groups.map((g) => (
              <optgroup key={g} label={g}>
                {facts
                  .filter((f) => f.group === g)
                  .map((f) => (
                    <option key={f.name} value={f.name} disabled={f.pending !== null}>
                      {f.name}
                      {f.pending ? ` (${t('pending', { task: f.pending })})` : ''}
                    </option>
                  ))}
              </optgroup>
            ))}
          </select>
        </Field>
        <Select
          label={t('operator')}
          value={c.op}
          onChange={(e) => onChange({ ...c, op: e.target.value, value: '' })}
          options={opsFor(def?.type).map((o) => ({ value: o, label: t(`ops.${o}` as 'ops.eq') }))}
        />
        <ValueInput def={def} c={c} onChange={(value) => onChange({ ...c, value })} />
        <Button size="sm" variant="ghost" onClick={onRemove} aria-label={t('remove')}>
          ×
        </Button>
      </div>
      {def ? (
        <p className="m-0 text-[12px] text-ink-3">
          <span dir="ltr">{def.type}</span>
          {def.unit ? ` · ${def.unit}` : ''}
          {description ? ` — ${description}` : ''}
        </p>
      ) : null}
    </div>
  );
}

function Section({
  title,
  hint,
  rows,
  facts,
  onChange,
  onRemoveSection,
}: {
  title: string;
  hint: string;
  rows: CondForm[];
  facts: readonly FactDef[];
  onChange: (rows: CondForm[]) => void;
  onRemoveSection?: () => void;
}) {
  const t = useTranslations('ruleConditions');
  return (
    <fieldset className="flex flex-col gap-3 rounded-lg border border-line p-3">
      <legend className="px-1 text-[14px] font-semibold">{title}</legend>
      <p className="m-0 text-[13px] text-ink-3">{hint}</p>
      {rows.map((c, i) => (
        <CondRow
          key={i}
          c={c}
          facts={facts}
          onChange={(n) => onChange(rows.map((x, j) => (j === i ? n : x)))}
          onRemove={() => onChange(rows.filter((_, j) => j !== i))}
        />
      ))}
      <div className="flex gap-2">
        <Button size="sm" onClick={() => onChange([...rows, emptyCond()])}>
          {t('add')}
        </Button>
        {onRemoveSection ? (
          <Button size="sm" variant="ghost" onClick={onRemoveSection}>
            {t('removeSection')}
          </Button>
        ) : null}
      </div>
    </fieldset>
  );
}

/** Condition builder: «all of» (AND), «any of» (OR, several groups joined by AND) and «exclude» (exceptions). */
export function ConditionBuilder({
  value,
  onChange,
  facts,
  error,
}: {
  value: ConditionsForm;
  onChange: (v: ConditionsForm) => void;
  facts: readonly FactDef[];
  error?: string;
}) {
  const t = useTranslations('ruleConditions');
  return (
    <div className="flex flex-col gap-4">
      <Section title={t('allOf')} hint={t('allOfHint')} rows={value.allOf} facts={facts} onChange={(allOf) => onChange({ ...value, allOf })} />
      {value.anyOf.map((g, gi) => (
        <Section
          key={gi}
          title={t('anyOf')}
          hint={t('anyOfHint')}
          rows={g}
          facts={facts}
          onChange={(rows) => onChange({ ...value, anyOf: value.anyOf.map((x, j) => (j === gi ? rows : x)) })}
          onRemoveSection={() => onChange({ ...value, anyOf: value.anyOf.filter((_, j) => j !== gi) })}
        />
      ))}
      <div>
        <Button size="sm" onClick={() => onChange({ ...value, anyOf: [...value.anyOf, [emptyCond()]] })}>
          {t('addAnyOf')}
        </Button>
      </div>
      <Section title={t('exclude')} hint={t('excludeHint')} rows={value.exclude} facts={facts} onChange={(exclude) => onChange({ ...value, exclude })} />
      {error ? (
        <span className="field-error" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
