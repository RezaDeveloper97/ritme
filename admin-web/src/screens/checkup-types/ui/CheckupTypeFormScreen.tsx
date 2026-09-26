'use client';

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { useState, type ReactNode } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useContentLanguages, useLocalized, type ContentLanguage } from '@/shared/i18n';
import { blankToNull, cn, toIntOrNull } from '@/shared/lib';
import {
  Button,
  FormPage,
  Icon,
  LoadGate,
  PageHeader,
  Select,
  Switch,
  TextArea,
  TextInput,
  toast,
  useNotifyError,
  type Translations,
} from '@/shared/ui';

import {
  checkupTypesApi,
  type CheckupOptions,
  type CheckupType,
  type FindingOption,
  type GuideStep,
} from '../api/checkup-types';
import { checkupIcon } from '../lib/icon';
import { cleanTranslations, moveItem } from '../lib/payload';
import { toneClass } from '../lib/tone';
import { CheckupPreview } from './CheckupPreview';
import { useCheckupLabels } from './labels';

/** /checkup-types/new and /checkup-types/:id. */
export function CheckupTypeFormScreen({ id }: { id: number | null }) {
  const t = useTranslations('checkupTypes');
  const detail = checkupTypesApi.useDetail(id);
  const options = checkupTypesApi.useOptions();
  const languages = useContentLanguages();
  return (
    <LoadGate
      queries={id === null ? [options, languages] : [detail, options, languages]}
      header={<PageHeader title={id === null ? t('new') : t('edit')} backHref="/checkup-types" backLabel={t('backToList')} />}
    >
      {() =>
        options.data && languages.data ? (
          <CheckupTypeForm
            id={id}
            row={detail.data?.checkup_type ?? null}
            options={options.data}
            languages={[...languages.data.languages].sort((a, b) => Number(b.is_default) - Number(a.is_default))}
          />
        ) : null
      }
    </LoadGate>
  );
}

const num = (v: number | null | undefined) => (v === null || v === undefined ? '' : String(v));

function CheckupTypeForm({
  id,
  row,
  options,
  languages,
}: {
  id: number | null;
  row: CheckupType | null;
  options: CheckupOptions;
  languages: ContentLanguage[];
}) {
  const t = useTranslations('checkupTypes');
  const tc = useTranslations('crud');
  const router = useRouter();
  const localize = useLocalized();
  const { label } = useCheckupLabels();
  const notifyError = useNotifyError();
  const save = checkupTypesApi.useSave(id);

  const [lang, setLang] = useState(languages[0]?.code ?? 'fa');
  const [key, setKey] = useState(row?.key ?? '');
  const [title, setTitle] = useState<Translations>(row?.title ?? {});
  const [subtitle, setSubtitle] = useState<Translations>(row?.subtitle ?? {});
  const [why, setWhy] = useState<Translations>(row?.why ?? {});
  const [prep, setPrep] = useState<Translations[]>(row?.prep_steps ?? []);
  const [guide, setGuide] = useState<GuideStep[]>(row?.guide_steps ?? []);
  const [findings, setFindings] = useState<FindingOption[]>(row?.finding_options ?? []);
  const [category, setCategory] = useState(row?.category ?? options.categories[0] ?? 'annual');
  const [performedBy, setPerformedBy] = useState(row?.performed_by ?? options.performed_by[0] ?? 'doctor');
  const [icon, setIcon] = useState<string | null>(row?.icon ?? null);
  const [tone, setTone] = useState(row?.tone ?? options.default_tone);
  const [intervalMin, setIntervalMin] = useState(num(row?.interval_months ?? 12));
  const [intervalMax, setIntervalMax] = useState(num(row?.interval_months_max));
  const [ageMin, setAgeMin] = useState(num(row?.age_min));
  const [ageMax, setAgeMax] = useState(num(row?.age_max));
  const [cycleFrom, setCycleFrom] = useState(num(row?.cycle_day_from));
  const [cycleTo, setCycleTo] = useState(num(row?.cycle_day_to));
  const [remindLead, setRemindLead] = useState(num(row?.remind_lead_days ?? options.default_remind_lead_days));
  const [hideInPregnancy, setHideInPregnancy] = useState(row?.hide_in_pregnancy ?? false);
  const [active, setActive] = useState(row?.is_active ?? true);
  const [sortOrder, setSortOrder] = useState(num(row?.sort_order ?? options.next_sort_order));
  const [sourceNote, setSourceNote] = useState(row?.source_note ?? '');

  const errors = fieldErrorsOf(save.error) ?? {};
  const err = (name: string) => fieldError(save.error, name);
  const langErr = (name: string) => errors[`${name}.${lang}`]?.[0];
  const langHasError = (code: string) => Object.keys(errors).some((k) => k.endsWith(`.${code}`));
  const current = languages.find((l) => l.code === lang) ?? languages[0];
  const max = options.max_steps;

  const submit = () =>
    save.mutate(
      {
        ...(id === null ? { key: key.trim() } : {}),
        title,
        subtitle: cleanTranslations(subtitle),
        why: cleanTranslations(why),
        category,
        performed_by: performedBy,
        icon,
        tone,
        interval_months: toIntOrNull(intervalMin),
        interval_months_max: toIntOrNull(intervalMax),
        age_min: toIntOrNull(ageMin),
        age_max: toIntOrNull(ageMax),
        cycle_day_from: toIntOrNull(cycleFrom),
        cycle_day_to: toIntOrNull(cycleTo),
        remind_lead_days: toIntOrNull(remindLead),
        prep_steps: prep,
        guide_steps: guide,
        finding_options: findings.map((f) => ({ key: f.key.trim(), label: f.label, ...(f.exclusive ? { exclusive: true } : {}) })),
        hide_in_pregnancy: hideInPregnancy,
        is_active: active,
        sort_order: toIntOrNull(sortOrder) ?? undefined,
        source_note: blankToNull(sourceNote),
      },
      {
        onSuccess: () => {
          toast.success(id === null ? tc('created') : tc('saved'));
          router.push('/checkup-types');
        },
        onError: notifyError,
      },
    );

  const langInput = (name: string, value: Translations, onChange: (v: Translations) => void, opts: { area?: boolean; maxLength?: number; label: string; required?: boolean }) => {
    const Comp = opts.area ? TextArea : TextInput;
    return (
      <Comp
        label={opts.label}
        dir={current?.direction}
        lang={lang}
        required={opts.required}
        maxLength={opts.maxLength}
        value={value[lang] ?? ''}
        onChange={(e) => onChange({ ...value, [lang]: e.target.value })}
        error={langErr(name)}
      />
    );
  };

  return (
    <FormPage
      title={id === null ? t('new') : localize(row?.title) || t('edit')}
      backHref="/checkup-types"
      backLabel={t('backToList')}
      onSubmit={submit}
      submitLabel={id === null ? tc('create') : tc('saveChanges')}
      saving={save.isPending}
    >
      <CheckupPreview
        row={{
          icon,
          tone,
          performed_by: performedBy,
          interval_months: toIntOrNull(intervalMin) ?? 0,
          interval_months_max: toIntOrNull(intervalMax),
          age_min: toIntOrNull(ageMin),
          age_max: toIntOrNull(ageMax),
          cycle_day_from: toIntOrNull(cycleFrom),
          cycle_day_to: toIntOrNull(cycleTo),
        }}
        title={title[lang] || localize(title)}
        subtitle={subtitle[lang] || localize(subtitle)}
      />

      <TextInput
        label={t('key')}
        hint={id === null ? t('keyHint') : t('keyFixed')}
        value={key}
        onChange={(e) => setKey(e.target.value)}
        dir="ltr"
        maxLength={64}
        pattern="[a-z][a-z0-9_]*"
        required={id === null}
        disabled={id !== null}
        error={err('key')}
      />

      <Section title={t('texts')} hint={t('textsHint')}>
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
              {langHasError(l.code) ? <Icon name="alert" size={14} /> : null}
            </button>
          ))}
        </div>
        <div role="tabpanel" className="flex flex-col gap-4" dir={current?.direction}>
          {langInput('title', title, setTitle, { label: t('titleField'), required: true, maxLength: 255 })}
          {langInput('subtitle', subtitle, setSubtitle, { label: t('subtitle'), maxLength: 255 })}
          {langInput('why', why, setWhy, { label: t('why'), area: true, maxLength: 2000 })}

          <Repeat
            title={t('prepSteps')}
            count={prep.length}
            max={max}
            onAdd={() => setPrep([...prep, {}])}
            addLabel={t('addStep')}
            error={err('prep_steps')}
          >
            {prep.map((step, i) => (
              <ItemRow key={i} index={i} count={prep.length} onMove={(to) => setPrep(moveItem(prep, i, to))} onRemove={() => setPrep(prep.filter((_, j) => j !== i))}>
                {langInput(`prep_steps.${i}`, step, (v) => setPrep(prep.map((s, j) => (j === i ? v : s))), {
                  label: t('stepN', { n: i + 1 }),
                  maxLength: 500,
                })}
              </ItemRow>
            ))}
          </Repeat>

          <Repeat
            title={t('guideSteps')}
            count={guide.length}
            max={max}
            onAdd={() => setGuide([...guide, { title: {}, body: {} }])}
            addLabel={t('addStep')}
            error={err('guide_steps')}
          >
            {guide.map((step, i) => {
              const set = (patch: Partial<GuideStep>) => setGuide(guide.map((s, j) => (j === i ? { ...s, ...patch } : s)));
              return (
                <ItemRow key={i} index={i} count={guide.length} onMove={(to) => setGuide(moveItem(guide, i, to))} onRemove={() => setGuide(guide.filter((_, j) => j !== i))}>
                  {langInput(`guide_steps.${i}.title`, step.title, (v) => set({ title: v }), { label: t('stepTitle', { n: i + 1 }), maxLength: 120 })}
                  {langInput(`guide_steps.${i}.body`, step.body, (v) => set({ body: v }), { label: t('stepBody'), area: true, maxLength: 1000 })}
                </ItemRow>
              );
            })}
          </Repeat>

          <Repeat
            title={t('findings')}
            hint={t('findingsHint')}
            count={findings.length}
            max={max}
            onAdd={() => setFindings([...findings, { key: '', exclusive: false, label: {} }])}
            addLabel={t('addFinding')}
            error={err('finding_options')}
          >
            {findings.map((f, i) => {
              const set = (patch: Partial<FindingOption>) => setFindings(findings.map((s, j) => (j === i ? { ...s, ...patch } : s)));
              return (
                <ItemRow key={i} index={i} count={findings.length} onMove={(to) => setFindings(moveItem(findings, i, to))} onRemove={() => setFindings(findings.filter((_, j) => j !== i))}>
                  <div className="form-grid">
                    <TextInput
                      label={t('findingKey')}
                      value={f.key}
                      onChange={(e) => set({ key: e.target.value })}
                      dir="ltr"
                      maxLength={40}
                      required
                      error={err(`finding_options.${i}.key`)}
                    />
                    <Switch label={t('exclusive')} hint={t('exclusiveHint')} checked={f.exclusive} onChange={(v) => set({ exclusive: v })} />
                  </div>
                  {langInput(`finding_options.${i}.label`, f.label, (v) => set({ label: v }), { label: t('findingLabel'), maxLength: 120 })}
                </ItemRow>
              );
            })}
          </Repeat>
        </div>
      </Section>

      <Section title={t('appearance')}>
        <div className="form-grid">
          <Select
            label={t('category')}
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            options={options.categories.map((c) => ({ value: c, label: label('category', c) }))}
            required
            error={err('category')}
          />
          <Select
            label={t('performedBy')}
            value={performedBy}
            onChange={(e) => setPerformedBy(e.target.value)}
            options={options.performed_by.map((c) => ({ value: c, label: label('performer', c) }))}
            required
            error={err('performed_by')}
          />
        </div>
        <fieldset className="field m-0 border-0 p-0">
          <legend className="field-label mb-1.5 p-0">{t('icon')}</legend>
          <div className="flex flex-wrap gap-2">
            <Choice selected={icon === null} onClick={() => setIcon(null)} label={t('iconAuto')}>
              <span className="text-xs">{t('iconAuto')}</span>
            </Choice>
            {options.icons.map((name) => (
              <Choice key={name} selected={icon === name} onClick={() => setIcon(name)} label={name}>
                <Icon name={checkupIcon(name, performedBy)} size={20} />
              </Choice>
            ))}
          </div>
          {err('icon') ? <span className="field-error">{err('icon')}</span> : <span className="field-hint">{t('iconHint')}</span>}
        </fieldset>
        <fieldset className="field m-0 border-0 p-0">
          <legend className="field-label mb-1.5 p-0">{t('tone')}</legend>
          <div className="flex flex-wrap gap-2">
            {options.tones.map((name) => (
              <Choice key={name} selected={tone === name} onClick={() => setTone(name)} label={label('tone', name)} className={toneClass(name)}>
                <span className="text-xs font-semibold">{label('tone', name)}</span>
              </Choice>
            ))}
          </div>
          {err('tone') ? <span className="field-error">{err('tone')}</span> : null}
        </fieldset>
      </Section>

      <Section title={t('timing')}>
        <div className="form-grid">
          <TextInput label={t('intervalMonths')} type="number" min={1} max={options.max_interval_months} required value={intervalMin} onChange={(e) => setIntervalMin(e.target.value)} error={err('interval_months')} />
          <TextInput label={t('intervalMonthsMax')} hint={t('intervalMaxHint')} type="number" min={1} max={options.max_interval_months} value={intervalMax} onChange={(e) => setIntervalMax(e.target.value)} error={err('interval_months_max')} />
          <TextInput label={t('ageMin')} type="number" min={0} max={120} value={ageMin} onChange={(e) => setAgeMin(e.target.value)} error={err('age_min')} />
          <TextInput label={t('ageMax')} type="number" min={0} max={120} value={ageMax} onChange={(e) => setAgeMax(e.target.value)} error={err('age_max')} />
          <TextInput label={t('cycleDayFrom')} hint={t('cycleDayHint')} type="number" min={1} max={options.max_cycle_day} value={cycleFrom} onChange={(e) => setCycleFrom(e.target.value)} error={err('cycle_day_from')} />
          <TextInput label={t('cycleDayTo')} type="number" min={1} max={options.max_cycle_day} value={cycleTo} onChange={(e) => setCycleTo(e.target.value)} error={err('cycle_day_to')} />
          <TextInput label={t('remindLead')} hint={t('remindLeadHint')} type="number" min={0} max={365} value={remindLead} onChange={(e) => setRemindLead(e.target.value)} error={err('remind_lead_days')} />
          <TextInput label={tc('sortOrder')} hint={tc('sortOrderHint')} type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} error={err('sort_order')} />
        </div>
        <div className="form-grid">
          <Switch label={t('hideInPregnancy')} hint={t('hideInPregnancyHint')} checked={hideInPregnancy} onChange={setHideInPregnancy} />
          <Switch label={tc('isActive')} checked={active} onChange={setActive} />
        </div>
      </Section>

      <TextArea label={t('sourceNote')} hint={t('sourceNoteHint')} value={sourceNote} onChange={(e) => setSourceNote(e.target.value)} maxLength={1000} error={err('source_note')} />
    </FormPage>
  );
}

function Section({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <fieldset className="form-section m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
      <legend className="field-label p-0">{title}</legend>
      {hint ? <p className="field-hint m-0">{hint}</p> : null}
      {children}
    </fieldset>
  );
}

function Repeat({
  title,
  hint,
  count,
  max,
  onAdd,
  addLabel,
  error,
  children,
}: {
  title: string;
  hint?: string;
  count: number;
  max: number;
  onAdd: () => void;
  addLabel: string;
  error?: string;
  children: ReactNode;
}) {
  const t = useTranslations('checkupTypes');
  return (
    <div className="flex flex-col gap-2">
      <span className="field-label">
        {title} <span className="text-xs text-muted">({t('countOf', { count, max })})</span>
      </span>
      {hint ? <span className="field-hint">{hint}</span> : null}
      {children}
      {error ? <span className="field-error">{error}</span> : null}
      <div>
        <Button size="sm" onClick={onAdd} disabled={count >= max}>
          <Icon name="plus" size={14} />
          {addLabel}
        </Button>
      </div>
    </div>
  );
}

function ItemRow({
  index,
  count,
  onMove,
  onRemove,
  children,
}: {
  index: number;
  count: number;
  onMove: (to: number) => void;
  onRemove: () => void;
  children: ReactNode;
}) {
  const t = useTranslations('checkupTypes');
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-line p-3">
      {children}
      <div className="flex gap-1.5">
        <Button size="sm" variant="ghost" aria-label={t('moveUp')} disabled={index === 0} onClick={() => onMove(index - 1)}>
          ↑
        </Button>
        <Button size="sm" variant="ghost" aria-label={t('moveDown')} disabled={index === count - 1} onClick={() => onMove(index + 1)}>
          ↓
        </Button>
        <Button size="sm" variant="danger" onClick={onRemove}>
          {t('remove')}
        </Button>
      </div>
    </div>
  );
}

function Choice({
  selected,
  onClick,
  label,
  className,
  children,
}: {
  selected: boolean;
  onClick: () => void;
  label: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      aria-label={label}
      title={label}
      onClick={onClick}
      className={cn(
        'grid min-h-10 min-w-10 place-items-center rounded-xl border-2 px-2',
        selected ? 'border-[var(--brand)]' : 'border-line',
        className,
      )}
    >
      {children}
    </button>
  );
}
