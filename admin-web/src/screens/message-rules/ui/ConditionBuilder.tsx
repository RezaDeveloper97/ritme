'use client';

import { useTranslations } from 'next-intl';

import { Button, Panel, Select, TextInput } from '@/shared/ui';

import type { CondForm, GroupForm } from '../lib/rule-form';

const emptyCond = (): CondForm => ({ fact: '', op: 'eq', value: '' });

function CondRow({
  c,
  ops,
  onChange,
  onRemove,
}: {
  c: CondForm;
  ops: readonly string[];
  onChange: (c: CondForm) => void;
  onRemove: () => void;
}) {
  const t = useTranslations('messageRules');
  return (
    <div className="grid grid-cols-[1fr_9rem_1fr_auto] items-end gap-2 max-sm:grid-cols-1">
      <TextInput
        label={t('fact')}
        dir="ltr"
        value={c.fact}
        onChange={(e) => onChange({ ...c, fact: e.target.value })}
        placeholder="days_to_period"
      />
      <Select
        label={t('operator')}
        value={c.op}
        onChange={(e) => onChange({ ...c, op: e.target.value })}
        options={ops.map((o) => ({ value: o, label: t(`ops.${o}` as 'ops.eq') }))}
      />
      <TextInput
        label={t('value')}
        dir="ltr"
        value={c.value}
        onChange={(e) => onChange({ ...c, value: e.target.value })}
        hint={c.op === 'in' || c.op === 'nin' || c.op === 'between' ? t('valueListHint') : undefined}
      />
      <Button size="sm" variant="ghost" onClick={onRemove} aria-label={t('removeCondition')}>
        ×
      </Button>
    </div>
  );
}

/**
 * Condition groups: groups are joined by AND; each group is «all of» (AND) or «any of» (OR). Exceptions remove the
 * message when any of them holds (Admin_MessageEditor section 2).
 */
export function ConditionBuilder({
  groups,
  exceptions,
  ops,
  onGroups,
  onExceptions,
  error,
}: {
  groups: GroupForm[];
  exceptions: CondForm[];
  ops: readonly string[];
  onGroups: (g: GroupForm[]) => void;
  onExceptions: (e: CondForm[]) => void;
  error?: string;
}) {
  const t = useTranslations('messageRules');
  const setGroup = (i: number, g: GroupForm) => onGroups(groups.map((x, j) => (j === i ? g : x)));
  return (
    <Panel
      title={t('ruleTitle')}
      actions={
        <Button size="sm" onClick={() => onGroups([...groups, { op: 'or', items: [emptyCond()] }])}>
          {t('addOrGroup')}
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        {groups.map((g, gi) => (
          <fieldset key={gi} className="flex flex-col gap-3 rounded-lg border border-line p-3">
            <legend className="flex items-center gap-2 px-1 text-[13px]">
              <Select
                label={t('groupMode')}
                value={g.op}
                onChange={(e) => setGroup(gi, { ...g, op: e.target.value as 'and' | 'or' })}
                options={[
                  { value: 'and', label: t('groupAnd') },
                  { value: 'or', label: t('groupOr') },
                ]}
              />
            </legend>
            {g.items.map((c, ci) => (
              <CondRow
                key={ci}
                c={c}
                ops={ops}
                onChange={(n) => setGroup(gi, { ...g, items: g.items.map((x, j) => (j === ci ? n : x)) })}
                onRemove={() => {
                  const items = g.items.filter((_, j) => j !== ci);
                  if (items.length === 0 && groups.length > 1) onGroups(groups.filter((_, j) => j !== gi));
                  else setGroup(gi, { ...g, items });
                }}
              />
            ))}
            <div>
              <Button size="sm" onClick={() => setGroup(gi, { ...g, items: [...g.items, emptyCond()] })}>
                {t('addCondition')}
              </Button>
            </div>
          </fieldset>
        ))}
        {error ? (
          <span className="field-error" role="alert">
            {error}
          </span>
        ) : null}
        <div className="flex flex-col gap-3">
          <h3 className="m-0 text-[14px] font-semibold">{t('exceptions')}</h3>
          <p className="m-0 text-[13px] text-ink-3">{t('exceptionsHint')}</p>
          {exceptions.map((c, i) => (
            <CondRow
              key={i}
              c={c}
              ops={ops}
              onChange={(n) => onExceptions(exceptions.map((x, j) => (j === i ? n : x)))}
              onRemove={() => onExceptions(exceptions.filter((_, j) => j !== i))}
            />
          ))}
          <div>
            <Button size="sm" onClick={() => onExceptions([...exceptions, emptyCond()])}>
              {t('addException')}
            </Button>
          </div>
        </div>
      </div>
    </Panel>
  );
}
