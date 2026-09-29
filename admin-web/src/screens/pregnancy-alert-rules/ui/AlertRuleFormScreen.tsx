'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useContentLanguages, type ContentLanguage } from '@/shared/i18n';
import { cn, useNumber } from '@/shared/lib';
import { Button, FormPage, Icon, LoadGate, PageHeader, Select, Switch, TextArea, TextInput, toast, useNotifyError } from '@/shared/ui';

import {
  useAlertOptions,
  useAlertRule,
  useSaveAlertRule,
  type AlertAction,
  type AlertOptions,
  type AlertRule,
  type AlertTexts,
  type ParamField,
} from '../api/alert-rules';
import { fillSample, levelClass } from '../lib/level';
import { paramsBody } from '../lib/params';
import { useAlertLabels } from './labels';

/** /pregnancy-alert-rules/:key — behaviour (all locales) + texts per locale + a sample card. */
export function AlertRuleFormScreen({ ruleKey }: { ruleKey: string }) {
  const t = useTranslations('pregnancyAlertRules');
  const detail = useAlertRule(ruleKey);
  const options = useAlertOptions();
  const languages = useContentLanguages();
  return (
    <LoadGate
      queries={[detail, options, languages]}
      header={<PageHeader title={t('edit')} backHref="/pregnancy-alert-rules" backLabel={t('backToList')} />}
    >
      {() =>
        detail.data && options.data && languages.data ? (
          <RuleForm
            rule={detail.data.rule}
            options={options.data}
            languages={[...languages.data.languages].sort((a, b) => Number(b.is_default) - Number(a.is_default))}
          />
        ) : null
      }
    </LoadGate>
  );
}

const MAX_ACTIONS = 4;
const emptyTexts = (): AlertTexts => ({
  title: '',
  what_we_saw: '',
  how_sure: '',
  advice: '',
  actions: [],
  contact: null,
});

function RuleForm({ rule, options, languages }: { rule: AlertRule; options: AlertOptions; languages: ContentLanguage[] }) {
  const t = useTranslations('pregnancyAlertRules');
  const tc = useTranslations('crud');
  const n = useNumber();
  const router = useRouter();
  const labels = useAlertLabels();
  const notifyError = useNotifyError();
  const save = useSaveAlertRule(rule.key);

  const [enabled, setEnabled] = useState(rule.enabled ?? false);
  const [level, setLevel] = useState(rule.level ?? options.levels[0] ?? 'info');
  const [windowDays, setWindowDays] = useState(String(rule.window_days ?? 7));
  const [params, setParams] = useState<Record<string, unknown>>(rule.params);
  const [texts, setTexts] = useState<Record<string, AlertTexts>>(() =>
    Object.fromEntries(languages.map((l) => [l.code, rule.texts[l.code] ?? emptyTexts()])),
  );
  const [lang, setLang] = useState(languages[0]?.code ?? 'fa');
  const errors = fieldErrorsOf(save.error) ?? {};
  const err = (name: string) => fieldError(save.error, name);
  const current = languages.find((l) => l.code === lang) ?? languages[0];
  const cur = texts[lang] ?? emptyTexts();
  const setCur = (patch: Partial<AlertTexts>) => setTexts((all) => ({ ...all, [lang]: { ...cur, ...patch } }));
  const touched = (x: AlertTexts) => Boolean(x.title || x.what_we_saw || x.how_sure || x.advice || x.contact || x.actions.length);
  const hasRow = (code: string) => Boolean(rule.texts[code]);

  const submit = () => {
    const body: Record<string, unknown> = {
      enabled,
      level,
      window_days: Number(windowDays),
      params: paramsBody(rule.params_schema, params),
      texts: Object.fromEntries(
        languages
          .filter((l) => hasRow(l.code) || touched(texts[l.code] ?? emptyTexts()))
          .map((l) => {
            const x = texts[l.code] ?? emptyTexts();
            return [
              l.code,
              {
                title: x.title ?? '',
                what_we_saw: x.what_we_saw ?? '',
                how_sure: x.how_sure ?? '',
                advice: x.advice ?? '',
                actions: x.actions.map((a) => ({ key: a.key, label: a.label })),
                contact: x.contact?.trim() ? x.contact : null,
              },
            ];
          }),
      ),
    };
    save.mutate(body, {
      onSuccess: () => {
        toast.success(tc('saved'));
        router.push('/pregnancy-alert-rules');
      },
      onError: notifyError,
    });
  };

  const textInput = (name: keyof Omit<AlertTexts, 'actions'>, label: string, max: number, area = false) => {
    const Comp = area ? TextArea : TextInput;
    return (
      <Comp
        label={label}
        dir={current?.direction}
        lang={lang}
        maxLength={max}
        required={name !== 'contact' && (hasRow(lang) || touched(cur))}
        value={cur[name] ?? ''}
        onChange={(e) => setCur({ [name]: e.target.value })}
        error={errors[`texts.${lang}.${name}`]?.[0]}
      />
    );
  };

  return (
    <FormPage
      title={labels.rule(rule.key)}
      backHref="/pregnancy-alert-rules"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={tc('saveChanges')}
      saving={save.isPending}
    >
      <AlertPreview level={level} texts={cur} dir={current?.direction} locale={current?.code ?? lang} />

      <fieldset className="form-section m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
        <legend className="field-label p-0">{t('behaviour')}</legend>
        <p className="field-hint m-0">{t('behaviourHint')}</p>
        <Switch label={t('enabled')} checked={enabled} onChange={setEnabled} />
        <fieldset className="field m-0 border-0 p-0">
          <legend className="field-label mb-1.5 p-0">{t('level')}</legend>
          <div className="flex flex-wrap gap-2">
            {options.levels.map((l) => (
              <button
                key={l}
                type="button"
                aria-pressed={level === l}
                onClick={() => setLevel(l)}
                className={cn(
                  'rounded-xl border-2 px-3 py-1.5 text-sm font-semibold',
                  levelClass(l, 'chip'),
                  level === l ? 'border-current' : 'border-transparent',
                )}
              >
                {labels.level(l)}
              </button>
            ))}
          </div>
          {err('level') ? <span className="field-error">{err('level')}</span> : null}
        </fieldset>
        <div className="form-grid">
          <TextInput
            label={t('windowDays')}
            hint={t('windowDaysHint')}
            type="number"
            min={options.min_window_days}
            max={options.max_window_days}
            required
            value={windowDays}
            onChange={(e) => setWindowDays(e.target.value)}
            error={err('window_days')}
          />
          {rule.params_schema
            .filter((f) => f.kind === 'integer')
            .map((f) => (
              <TextInput
                key={f.key}
                label={labels.param(f.key)}
                hint={f.min !== undefined && f.max !== undefined ? t('range', { min: n(f.min), max: n(f.max) }) : undefined}
                type="number"
                min={f.min}
                max={f.max}
                required={!f.nullable}
                value={params[f.key] === null || params[f.key] === undefined ? '' : String(params[f.key])}
                onChange={(e) => setParams((p) => ({ ...p, [f.key]: e.target.value }))}
                error={err(`params.${f.key}`)}
              />
            ))}
        </div>
        {rule.params_schema
          .filter((f) => f.kind !== 'integer')
          .map((f) => (
            <ParamInput
              key={f.key}
              field={f}
              value={params[f.key]}
              onChange={(v) => setParams((p) => ({ ...p, [f.key]: v }))}
              error={err(`params.${f.key}`)}
            />
          ))}
      </fieldset>

      <fieldset className="form-section m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
        <legend className="field-label p-0">{t('texts')}</legend>
        <p className="field-hint m-0">
          {t('textsHint')}
          {rule.placeholders.length > 0 ? (
            <>
              {' '}
              {t('placeholders')}: <span dir="ltr">{rule.placeholders.map((p) => `{${p}}`).join(' ')}</span>
            </>
          ) : null}
        </p>
        <div role="tablist" aria-label={t('languages')} className="flex flex-wrap gap-1.5">
          {languages.map((l) => (
            <button
              key={l.code}
              type="button"
              role="tab"
              aria-selected={l.code === lang}
              className={cn('btn btn-sm', l.code === lang ? 'btn-primary' : 'btn-ghost')}
              onClick={() => setLang(l.code)}
            >
              {l.name}
              <span className="lang-code">{l.code}</span>
              {Object.keys(errors).some((k) => k.startsWith(`texts.${l.code}.`)) || rule.missing_locales.includes(l.code) ? (
                <Icon name="alert" size={14} />
              ) : null}
            </button>
          ))}
        </div>
        <div role="tabpanel" className="flex flex-col gap-4" dir={current?.direction}>
          {rule.missing_locales.includes(lang) ? <p className="field-hint m-0">{t('missingHint')}</p> : null}
          {textInput('title', t('fields.title'), 255)}
          {textInput('what_we_saw', t('fields.what_we_saw'), 500, true)}
          {textInput('how_sure', t('fields.how_sure'), 500, true)}
          {textInput('advice', t('fields.advice'), 2000, true)}
          {textInput('contact', t('fields.contact'), 500)}
          <ActionsEditor
            actions={cur.actions}
            keys={options.actions}
            onChange={(actions) => setCur({ actions })}
            errorOf={(path) => errors[`texts.${lang}.actions${path}`]?.[0]}
          />
        </div>
      </fieldset>
    </FormPage>
  );
}

function ParamInput({
  field,
  value,
  onChange,
  error,
}: {
  field: ParamField;
  value: unknown;
  onChange: (v: unknown) => void;
  error?: string;
}) {
  const labels = useAlertLabels();
  if (field.kind === 'boolean') return <Switch label={labels.param(field.key)} checked={value === true} onChange={onChange} />;
  if (field.kind === 'enum') {
    return (
      <Select
        label={labels.param(field.key)}
        value={typeof value === 'string' ? value : ''}
        onChange={(e) => onChange(e.target.value)}
        options={(field.values ?? []).map((v) => ({ value: v, label: v }))}
        error={error}
      />
    );
  }
  const list = Array.isArray(value) ? (value as string[]) : [];
  return (
    <fieldset className="field m-0 border-0 p-0">
      <legend className="field-label mb-1.5 p-0">{labels.param(field.key)}</legend>
      <div className="flex flex-wrap gap-1.5">
        {(field.values ?? []).map((v) => {
          const on = list.includes(v);
          return (
            <button
              key={v}
              type="button"
              dir="ltr"
              aria-pressed={on}
              className={cn('btn btn-sm', on ? 'btn-primary' : 'btn-ghost')}
              onClick={() => onChange(on ? list.filter((x) => x !== v) : [...list, v])}
            >
              {v}
            </button>
          );
        })}
      </div>
      {error ? <span className="field-error">{error}</span> : null}
    </fieldset>
  );
}

function ActionsEditor({
  actions,
  keys,
  onChange,
  errorOf,
}: {
  actions: AlertAction[];
  keys: string[];
  onChange: (next: AlertAction[]) => void;
  errorOf: (path: string) => string | undefined;
}) {
  const t = useTranslations('pregnancyAlertRules');
  const n = useNumber();
  const labels = useAlertLabels();
  const free = keys.filter((k) => !actions.some((a) => a.key === k));
  return (
    <div className="flex flex-col gap-2">
      <span className="field-label">
        {t('fields.actions')} <span className="text-xs text-muted">({t('countOf', { count: n(actions.length), max: n(MAX_ACTIONS) })})</span>
      </span>
      {actions.map((a, i) => (
        <div key={i} className="flex flex-wrap items-end gap-2 rounded-xl border border-line p-3">
          <Select
            label={t('actionKey')}
            value={a.key}
            onChange={(e) => onChange(actions.map((x, j) => (j === i ? { ...x, key: e.target.value } : x)))}
            options={keys.map((k) => ({ value: k, label: labels.action(k) }))}
            error={errorOf(`.${i}.key`)}
          />
          <TextInput
            label={t('actionLabel')}
            maxLength={60}
            required
            value={a.label}
            onChange={(e) => onChange(actions.map((x, j) => (j === i ? { ...x, label: e.target.value } : x)))}
            error={errorOf(`.${i}.label`)}
          />
          <Button size="sm" variant="danger" onClick={() => onChange(actions.filter((_, j) => j !== i))}>
            {t('remove')}
          </Button>
        </div>
      ))}
      {errorOf('') ? <span className="field-error">{errorOf('')}</span> : null}
      <div>
        <Button
          size="sm"
          disabled={actions.length >= MAX_ACTIONS || free.length === 0}
          onClick={() => onChange([...actions, { key: free[0] ?? 'ack', label: '' }])}
        >
          <Icon name="plus" size={14} />
          {t('addAction')}
        </Button>
      </div>
    </div>
  );
}

/** A sample rendering of the app's alert card (placeholders filled with sample values). */
function AlertPreview({ level, texts, dir, locale }: { level: string; texts: AlertTexts; dir?: 'rtl' | 'ltr'; locale: string }) {
  const t = useTranslations('pregnancyAlertRules');
  const labels = useAlertLabels();
  return (
    <section className="flex flex-col gap-2" aria-label={t('preview')}>
      <span className="field-label">{t('preview')}</span>
      <p className="field-hint m-0">{t('previewHint')}</p>
      <div className="grid gap-3 md:grid-cols-2">
        {(['light', 'dark'] as const).map((theme) => (
          <div key={theme} data-theme={theme} className="rounded-xl border border-line bg-[var(--page)] p-3 text-[var(--ink)]">
            <div dir={dir} className={cn('flex flex-col gap-2 rounded-xl border-s-4 p-3', levelClass(level, 'card'))}>
              <span className={cn('self-start rounded-full px-2 py-0.5 text-xs font-semibold', levelClass(level, 'chip'))}>
                {labels.level(level)}
              </span>
              <strong className="text-[14px]">{fillSample(texts.title ?? '', locale) || t('previewUntitled')}</strong>
              {texts.what_we_saw ? <p className="m-0 text-xs text-[var(--ink-3)]">{fillSample(texts.what_we_saw, locale)}</p> : null}
              {texts.how_sure ? <p className="m-0 text-xs text-[var(--muted)]">{fillSample(texts.how_sure, locale)}</p> : null}
              {texts.advice ? <p className="m-0 text-sm">{fillSample(texts.advice, locale)}</p> : null}
              {level === 'urgent' && texts.contact ? <p className="m-0 text-sm font-semibold">{fillSample(texts.contact, locale)}</p> : null}
              {texts.actions.length > 0 ? (
                <div className="flex flex-wrap gap-1.5">
                  {texts.actions.map((a, i) => (
                    <span key={i} className="rounded-full bg-[var(--surface)] px-3 py-1 text-xs font-semibold">
                      {a.label || labels.action(a.key)}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
