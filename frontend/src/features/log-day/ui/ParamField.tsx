'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import type { LogOption, LogParam, LogParamValue } from '@/entities/health-log';
import { Link, type Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';
import { ChipGroup, Icon, PillChip, StatusPill, Switch, type Tone } from '@/shared/ui';

import {
  asBool,
  asItems,
  asList,
  asNumber,
  asString,
  asTextItems,
  commonLevel,
  defaultLevel,
  isGraded,
  normalizeNumber,
  parseDecimal,
  pickableOptions,
  setAllLevels,
  toggleInList,
  toggleItem,
} from '../model/draft';
import { linkHref } from '../model/presentation';

export interface ParamFieldProps {
  param: LogParam;
  value: LogParamValue | undefined;
  onChange: (value: LogParamValue | null) => void;
  /** The taxonomy mode (option availability). */
  mode: string | null;
  tone: Tone;
  locale: Locale;
  /** Items the taxonomy can't list: the user's custom items, care-reminder medications. */
  extraOptions?: readonly LogOption[];
  /** Hide the param's own label (a category with one param already says it). */
  hideLabel?: boolean;
  /** An exclusive chip shown first in a chip row («بدون درد», «لکه‌بینی»). */
  leading?: { label: string; pressed: boolean; onToggle: () => void };
  /** Shown instead of an empty chip row (meds without reminders). */
  emptyText?: string;
}

/**
 * One taxonomy param, rendered from its type — single / multi / items (yes-no or graded level) / number /
 * integer / text / text_items / bool / link. The log sheet never hardcodes a category's controls; a param
 * added to the registry tomorrow renders here without a frontend change.
 */
export function ParamField(props: ParamFieldProps) {
  const { param, hideLabel } = props;
  const labelId = useId();
  const showLabel = !hideLabel && param.type !== 'bool' && param.type !== 'link';
  return (
    <div className="lday-param">
      {showLabel ? (
        <div id={labelId} className="lday-plabel">
          {param.label}
        </div>
      ) : null}
      <Control {...props} labelId={showLabel ? labelId : undefined} />
    </div>
  );
}

function Control(props: ParamFieldProps & { labelId?: string }) {
  switch (props.param.type) {
    case 'single':
      return <SingleField {...props} />;
    case 'multi':
      return <MultiField {...props} />;
    case 'items':
      return <ItemsField {...props} />;
    case 'number':
    case 'integer':
      return <NumberField {...props} />;
    case 'text':
      return <TextField {...props} />;
    case 'text_items':
      return <TextItemsField {...props} />;
    case 'bool':
      return <BoolField {...props} />;
    case 'link':
      return <LinkField {...props} />;
    default:
      return null;
  }
}

function options(props: ParamFieldProps): LogOption[] {
  return [...pickableOptions(props.param, props.mode), ...(props.extraOptions ?? [])];
}

function Leading({ leading, tone }: Pick<ParamFieldProps, 'leading' | 'tone'>) {
  if (!leading) return null;
  return (
    <PillChip pressed={leading.pressed} mode="multi" tone={tone} onPressedChange={leading.onToggle}>
      {leading.label}
    </PillChip>
  );
}

function SingleField(props: ParamFieldProps) {
  const current = asString(props.value);
  return (
    <ChipGroup label={props.param.label}>
      <Leading leading={props.leading} tone={props.tone} />
      {options(props).map((o) => (
        <PillChip
          key={o.value}
          pressed={current === o.value}
          mode="multi"
          tone={props.tone}
          onPressedChange={(on) => props.onChange(on ? o.value : null)}
        >
          {o.label}
        </PillChip>
      ))}
    </ChipGroup>
  );
}

function MultiField(props: ParamFieldProps) {
  const list = asList(props.value);
  return (
    <ChipGroup label={props.param.label}>
      {options(props).map((o) => (
        <PillChip
          key={o.value}
          pressed={list.includes(o.value)}
          mode="multi"
          tone={props.tone}
          onPressedChange={() => props.onChange(toggleInList(list, o.value))}
        >
          {o.label}
        </PillChip>
      ))}
    </ChipGroup>
  );
}

function ItemsField(props: ParamFieldProps) {
  const t = useTranslations('logSheet');
  const { param } = props;
  const items = asItems(props.value);
  const graded = isGraded(param);
  const list = options(props);
  const picked = Object.entries(items).filter(([, v]) => v.level !== 'no');
  const level = commonLevel(Object.fromEntries(picked));
  if (!list.length && !props.leading) {
    return props.emptyText ? <p className="lday-empty-note">{props.emptyText}</p> : null;
  }
  return (
    <>
      <ChipGroup label={param.label}>
        <Leading leading={props.leading} tone={props.tone} />
        {list.map((o) => (
          <PillChip
            key={o.value}
            pressed={Boolean(items[o.value]) && items[o.value].level !== 'no'}
            mode="multi"
            tone={props.tone}
            onPressedChange={() => props.onChange(toggleItem(items, o.value, level ?? defaultLevel(param)))}
          >
            {o.label}
          </PillChip>
        ))}
      </ChipGroup>
      {graded && picked.length ? (
        <div className="lday-param">
          <div className="lday-plabel">{t('severity')}</div>
          <ChipGroup label={t('severity')} layout="fill">
            {param.levels.map((l) => (
              <PillChip
                key={l.value}
                shape="square"
                mode="multi"
                tone={props.tone}
                pressed={level === l.value}
                onPressedChange={() => props.onChange(setAllLevels(items, l.value))}
              >
                {l.label}
              </PillChip>
            ))}
          </ChipGroup>
        </div>
      ) : null}
    </>
  );
}

function NumberField(props: ParamFieldProps & { labelId?: string }) {
  const t = useTranslations('logSheet');
  const { param, locale } = props;
  const current = asNumber(props.value);
  const shown = current === null ? '' : formatDecimal(current, locale);
  const [text, setText] = useState(shown);
  // Re-sync when the value changes from outside (a panel stepper, another date).
  useEffect(() => setText(shown), [shown]);
  const commit = (raw: string) => {
    if (raw.trim() === '') {
      props.onChange(null);
      return;
    }
    const n = parseDecimal(raw);
    if (n === null) {
      setText(shown);
      return;
    }
    const v = normalizeNumber(param, n);
    props.onChange(v);
    setText(formatDecimal(v, locale));
  };
  return (
    <div className="lday-num">
      <input
        className="lday-num-input"
        type="text"
        inputMode={param.type === 'integer' ? 'numeric' : 'decimal'}
        autoComplete="off"
        aria-labelledby={props.labelId}
        aria-label={props.labelId ? undefined : param.label}
        placeholder={t('number.placeholder')}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onBlur={(e) => commit(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') commit((e.target as HTMLInputElement).value);
        }}
      />
      {param.unit ? <span className="lday-num-unit">{param.unit.label}</span> : null}
      {current !== null ? (
        <button type="button" className="lday-num-clear" aria-label={t('number.clear')} onClick={() => props.onChange(null)}>
          <Icon name="x" size={16} strokeWidth={2.2} />
        </button>
      ) : null}
    </div>
  );
}

function TextField(props: ParamFieldProps & { labelId?: string }) {
  const t = useTranslations('logSheet');
  const { param, locale } = props;
  const value = asString(props.value) ?? '';
  const max = param.maxLength ?? 2000;
  return (
    <div className="lday-text">
      <textarea
        className="lday-textarea"
        aria-labelledby={props.labelId}
        aria-label={props.labelId ? undefined : param.label}
        placeholder={t('text.placeholder')}
        maxLength={max}
        rows={4}
        value={value}
        onChange={(e) => props.onChange(e.target.value === '' ? null : e.target.value)}
      />
      <span className="lday-text-count" aria-hidden>
        {t('text.count', { count: formatDecimal(value.length, locale), max: formatDecimal(max, locale) })}
      </span>
    </div>
  );
}

function TextItemsField(props: ParamFieldProps) {
  const t = useTranslations('logSheet');
  const { param } = props;
  const items = asTextItems(props.value);
  const list = options(props);
  const max = param.maxLength ?? 255;
  const set = (code: string, text: string | null) => {
    const next = { ...items };
    if (text === null) delete next[code];
    else next[code] = text;
    props.onChange(next);
  };
  return (
    <>
      <ChipGroup label={param.label}>
        {list.map((o) => (
          <PillChip
            key={o.value}
            pressed={o.value in items}
            mode="multi"
            tone={props.tone}
            onPressedChange={(on) => set(o.value, on ? '' : null)}
          >
            {o.label}
          </PillChip>
        ))}
      </ChipGroup>
      {Object.keys(items).map((code) => {
        const label = list.find((o) => o.value === code)?.label ?? code;
        return (
          <input
            key={code}
            className="lday-item-input"
            type="text"
            maxLength={max}
            aria-label={label}
            placeholder={`${label} · ${t('text.itemPlaceholder')}`}
            value={items[code]}
            onChange={(e) => set(code, e.target.value)}
          />
        );
      })}
    </>
  );
}

function BoolField(props: ParamFieldProps) {
  const id = useId();
  const on = asBool(props.value) === true;
  return (
    <div className="lday-bool">
      <span id={id} className="lday-bool-label">
        {props.param.label}
      </span>
      <Switch checked={on} onCheckedChange={(next) => props.onChange(next ? true : null)} labelledBy={id} />
    </div>
  );
}

function LinkField({ param }: ParamFieldProps) {
  const t = useTranslations('logSheet');
  const href = linkHref(param.source);
  return (
    <div className={clsx('lday-link', !href && 'is-soon')}>
      <span className="lday-link-label">{param.label}</span>
      {href ? (
        <Link href={href} className="lday-link-cta">
          {t('link.open')}
          <Icon name="chevronDown" size={16} className="lday-link-chev" />
        </Link>
      ) : (
        <StatusPill tone="neutral">{t('link.soon')}</StatusPill>
      )}
    </div>
  );
}
