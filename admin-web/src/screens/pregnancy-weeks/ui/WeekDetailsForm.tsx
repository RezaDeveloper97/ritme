'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { fieldError, fieldErrorsOf } from '@/shared/api';
import { useContentLanguages, type ContentLanguage } from '@/shared/i18n';
import { cn, formatNumber, useNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  FormPage,
  Icon,
  LoadGate,
  PageHeader,
  Select,
  TextArea,
  TextInput,
  toast,
  useNotifyError,
  type IconName,
  type Translations,
} from '@/shared/ui';

import {
  useSaveWeekDetails,
  useWeekDetails,
  useWeekDetailsOptions,
  type BodySymptom,
  type Highlight,
  type WeekDetails,
  type WeekDetailsOptions,
  type WeekSource,
  type WeekTask,
} from '../api/week-details';
import { blankOrNull, cleanTranslations, moveItem, nextTaskKey } from '../lib/details';

/** The «جزئیات ساختاریافته» tab of a week: the v2 hero, highlights, body, tasks, warning, review (admin-api.md §13). */
export function WeekDetailsEditor({ week, header }: { week: number; header: ReactNode }) {
  const t = useTranslations('pregnancyWeeks');
  const detail = useWeekDetails(week);
  const options = useWeekDetailsOptions();
  const languages = useContentLanguages();
  return (
    <LoadGate
      queries={[detail, options, languages]}
      header={header ?? <PageHeader title={t('details.title')} backHref="/pregnancy-weeks" backLabel={t('backToList')} />}
    >
      {() =>
        detail.data && options.data && languages.data ? (
          <DetailsForm
            key={week}
            row={detail.data.week_details}
            options={options.data}
            languages={[...languages.data.languages].sort((a, b) => Number(b.is_default) - Number(a.is_default))}
          />
        ) : null
      }
    </LoadGate>
  );
}

/** Admin icons standing in for the app's highlight icons (no admin glyph → sparkle). */
const HIGHLIGHT_ICON: Record<string, IconName> = {
  heart: 'heart',
  drop: 'drop',
  sparkle: 'sparkle',
  moon: 'moon',
  baby: 'baby',
};
const highlightIcon = (name: string | null): IconName => (name ? (HIGHLIGHT_ICON[name] ?? 'sparkle') : 'sparkle');
const TONE_CLASS: Record<string, string> = {
  brand: 'bg-[var(--pink-bg)] text-[var(--brand)]',
  pink: 'bg-[var(--danger-soft)] text-[var(--danger-deep)]',
  teal: 'bg-[var(--data-soft)] text-[var(--data-deep)]',
};
const toneClass = (tone: string | null) => TONE_CLASS[tone ?? 'brand'] ?? TONE_CLASS.brand;

function DetailsForm({ row, options, languages }: { row: WeekDetails; options: WeekDetailsOptions; languages: ContentLanguage[] }) {
  const t = useTranslations('pregnancyWeeks.details');
  const tw = useTranslations('pregnancyWeeks');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const save = useSaveWeekDetails(row.week_number);
  const week = row.week_number;

  const [lang, setLang] = useState(languages[0]?.code ?? 'fa');
  const [sizeLabel, setSizeLabel] = useState<Translations>(row.size_label);
  const [illustration, setIllustration] = useState<string | null>(row.illustration_key);
  const [lengthCm, setLengthCm] = useState(row.length_cm ?? '');
  const [weightG, setWeightG] = useState(row.weight_g ?? '');
  const [heartRate, setHeartRate] = useState(row.heart_rate ?? '');
  const [headline, setHeadline] = useState<Translations>(row.headline);
  const [highlights, setHighlights] = useState<Highlight[]>(row.highlights);
  const [symptoms, setSymptoms] = useState<BodySymptom[]>(row.body_symptoms);
  const [bodyText, setBodyText] = useState<Translations>(row.body_text);
  const [tasks, setTasks] = useState<WeekTask[]>(row.tasks);
  const [warning, setWarning] = useState<Translations>(row.warning);
  const [reviewer, setReviewer] = useState<Translations>(row.reviewer_name);
  const [reviewedAt, setReviewedAt] = useState(row.reviewed_at?.slice(0, 10) ?? '');
  const [sources, setSources] = useState<WeekSource[]>(row.sources);

  const errors = fieldErrorsOf(save.error) ?? {};
  const err = (name: string) => fieldError(save.error, name);
  const langErr = (name: string) => errors[`${name}.${lang}`]?.[0];
  const langHasError = (code: string) => Object.keys(errors).some((k) => k.endsWith(`.${code}`));
  const current = languages.find((l) => l.code === lang) ?? languages[0];
  const max = options.max_items;

  const submit = () =>
    save.mutate(
      {
        size_label: cleanTranslations(sizeLabel),
        illustration_key: illustration,
        length_cm: blankOrNull(lengthCm),
        weight_g: blankOrNull(weightG),
        heart_rate: blankOrNull(heartRate),
        headline: cleanTranslations(headline),
        highlights: highlights.map((h) => ({
          ...(h.icon ? { icon: h.icon } : {}),
          ...(h.tone ? { tone: h.tone } : {}),
          title: h.title,
          body: h.body,
        })),
        body_symptoms: symptoms.map((s) => ({
          key: s.key.trim(),
          label: s.label,
        })),
        body_text: cleanTranslations(bodyText),
        tasks: tasks.map((x) => ({ key: x.key, text: x.text })),
        warning: cleanTranslations(warning),
        reviewer_name: cleanTranslations(reviewer),
        reviewed_at: reviewedAt || null,
        sources: sources.map((s) => ({
          title: s.title,
          ...(s.url?.trim() ? { url: s.url.trim() } : {}),
        })),
      },
      { onSuccess: () => toast.success(tc('saved')), onError: notifyError },
    );

  const langInput = (
    name: string,
    value: Translations,
    onChange: (v: Translations) => void,
    opts: {
      label: string;
      area?: boolean;
      maxLength?: number;
      required?: boolean;
    },
  ) => {
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
  const pick = (v: Translations) => v[lang] ?? '';

  return (
    <FormPage
      title={tw('weekTitle', { week: formatNumber(week, locale) })}
      meta={row.exists ? <Badge tone="green">{t('saved')}</Badge> : <Badge tone="amber">{t('notSaved')}</Badge>}
      backHref="/pregnancy-weeks"
      backLabel={tw('backToList')}
      onSubmit={submit}
      submitLabel={tc('saveChanges')}
      saving={save.isPending}
    >
      <HeroPreview
        week={week}
        dir={current?.direction}
        illustration={illustration}
        sizeLabel={pick(sizeLabel)}
        headline={pick(headline)}
        lengthCm={lengthCm}
        weightG={weightG}
        heartRate={heartRate}
        highlights={highlights.map((h) => ({
          icon: h.icon,
          tone: h.tone,
          title: pick(h.title),
        }))}
      />

      <Section title={t('illustration')} hint={t('illustrationHint')}>
        <div className="flex flex-wrap gap-1.5">
          <Chip selected={illustration === null} onClick={() => setIllustration(null)}>
            {t('none')}
          </Chip>
          {options.illustration_keys.map((k) => (
            <Chip key={k} selected={illustration === k} onClick={() => setIllustration(k)}>
              <span dir="ltr">{k}</span>
            </Chip>
          ))}
        </div>
        {err('illustration_key') ? <span className="field-error">{err('illustration_key')}</span> : null}
      </Section>

      <Section title={t('measurements')} hint={t('measurementsHint')}>
        <div className="form-grid">
          <TextInput
            label={t('lengthCm')}
            dir="ltr"
            maxLength={20}
            placeholder="1.6"
            value={lengthCm}
            onChange={(e) => setLengthCm(e.target.value)}
            error={err('length_cm')}
          />
          <TextInput
            label={t('weightG')}
            dir="ltr"
            maxLength={20}
            placeholder="<1"
            value={weightG}
            onChange={(e) => setWeightG(e.target.value)}
            error={err('weight_g')}
          />
          <TextInput
            label={t('heartRate')}
            dir="ltr"
            maxLength={20}
            placeholder="150-170"
            value={heartRate}
            onChange={(e) => setHeartRate(e.target.value)}
            error={err('heart_rate')}
          />
        </div>
      </Section>

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
          {langInput('size_label', sizeLabel, setSizeLabel, {
            label: t('sizeLabel'),
            maxLength: 60,
          })}
          {langInput('headline', headline, setHeadline, {
            label: t('headline'),
            maxLength: 255,
          })}

          <Repeat
            title={t('highlights')}
            count={highlights.length}
            max={max}
            addLabel={t('add')}
            error={err('highlights')}
            onAdd={() => setHighlights([...highlights, { icon: null, tone: null, title: {}, body: {} }])}
          >
            {highlights.map((h, i) => {
              const set = (patch: Partial<Highlight>) => setHighlights(highlights.map((x, j) => (j === i ? { ...x, ...patch } : x)));
              return (
                <ItemRow
                  key={i}
                  index={i}
                  count={highlights.length}
                  onMove={(to) => setHighlights(moveItem(highlights, i, to))}
                  onRemove={() => setHighlights(highlights.filter((_, j) => j !== i))}
                >
                  <div className="form-grid">
                    <Select
                      label={t('icon')}
                      value={h.icon ?? ''}
                      onChange={(e) => set({ icon: e.target.value || null })}
                      options={[
                        { value: '', label: '—' },
                        ...options.highlight_icons.map((v) => ({
                          value: v,
                          label: v,
                        })),
                      ]}
                      error={err(`highlights.${i}.icon`)}
                    />
                    <Select
                      label={t('tone')}
                      value={h.tone ?? ''}
                      onChange={(e) => set({ tone: e.target.value || null })}
                      options={[
                        { value: '', label: '—' },
                        ...options.highlight_tones.map((v) => ({
                          value: v,
                          label: v,
                        })),
                      ]}
                      error={err(`highlights.${i}.tone`)}
                    />
                  </div>
                  {langInput(`highlights.${i}.title`, h.title, (v) => set({ title: v }), {
                    label: t('itemTitle'),
                    maxLength: 120,
                    required: true,
                  })}
                  {langInput(`highlights.${i}.body`, h.body, (v) => set({ body: v }), {
                    label: t('itemBody'),
                    area: true,
                    maxLength: 1000,
                    required: true,
                  })}
                </ItemRow>
              );
            })}
          </Repeat>

          <Repeat
            title={t('bodySymptoms')}
            hint={t('bodySymptomsHint', {
              keys: options.log_symptom_keys.join(', '),
            })}
            count={symptoms.length}
            max={max}
            addLabel={t('add')}
            error={err('body_symptoms')}
            onAdd={() => setSymptoms([...symptoms, { key: '', label: {} }])}
          >
            {symptoms.map((s, i) => {
              const set = (patch: Partial<BodySymptom>) => setSymptoms(symptoms.map((x, j) => (j === i ? { ...x, ...patch } : x)));
              return (
                <ItemRow
                  key={i}
                  index={i}
                  count={symptoms.length}
                  onMove={(to) => setSymptoms(moveItem(symptoms, i, to))}
                  onRemove={() => setSymptoms(symptoms.filter((_, j) => j !== i))}
                >
                  <TextInput
                    label={t('symptomKey')}
                    dir="ltr"
                    maxLength={40}
                    required
                    list="log-symptom-keys"
                    value={s.key}
                    onChange={(e) => set({ key: e.target.value })}
                    error={err(`body_symptoms.${i}.key`)}
                  />
                  {langInput(`body_symptoms.${i}.label`, s.label, (v) => set({ label: v }), {
                    label: t('symptomLabel'),
                    maxLength: 60,
                    required: true,
                  })}
                </ItemRow>
              );
            })}
          </Repeat>
          <datalist id="log-symptom-keys">
            {options.log_symptom_keys.map((k) => (
              <option key={k} value={k} />
            ))}
          </datalist>
          {langInput('body_text', bodyText, setBodyText, {
            label: t('bodyText'),
            area: true,
            maxLength: 2000,
          })}

          <Repeat
            title={t('tasks')}
            hint={t('tasksHint')}
            count={tasks.length}
            max={max}
            addLabel={t('add')}
            error={err('tasks')}
            onAdd={() =>
              setTasks([
                ...tasks,
                {
                  key: nextTaskKey(
                    week,
                    tasks.map((x) => x.key),
                  ),
                  text: {},
                },
              ])
            }
          >
            {tasks.map((task, i) => (
              <ItemRow
                key={task.key}
                index={i}
                count={tasks.length}
                onMove={(to) => setTasks(moveItem(tasks, i, to))}
                onRemove={() => setTasks(tasks.filter((_, j) => j !== i))}
              >
                <span className="text-xs text-muted" dir="ltr">
                  {task.key}
                </span>
                {langInput(`tasks.${i}.text`, task.text, (v) => setTasks(tasks.map((x, j) => (j === i ? { ...x, text: v } : x))), {
                  label: t('taskText'),
                  maxLength: 255,
                  required: true,
                })}
                {err(`tasks.${i}.key`) ? <span className="field-error">{err(`tasks.${i}.key`)}</span> : null}
              </ItemRow>
            ))}
          </Repeat>

          {langInput('warning', warning, setWarning, {
            label: t('warning'),
            area: true,
            maxLength: 2000,
          })}
          {langInput('reviewer_name', reviewer, setReviewer, {
            label: t('reviewerName'),
            maxLength: 255,
          })}

          <Repeat
            title={t('sources')}
            count={sources.length}
            max={max}
            addLabel={t('add')}
            error={err('sources')}
            onAdd={() => setSources([...sources, { title: {}, url: null }])}
          >
            {sources.map((s, i) => {
              const set = (patch: Partial<WeekSource>) => setSources(sources.map((x, j) => (j === i ? { ...x, ...patch } : x)));
              return (
                <ItemRow
                  key={i}
                  index={i}
                  count={sources.length}
                  onMove={(to) => setSources(moveItem(sources, i, to))}
                  onRemove={() => setSources(sources.filter((_, j) => j !== i))}
                >
                  {langInput(`sources.${i}.title`, s.title, (v) => set({ title: v }), {
                    label: t('sourceTitle'),
                    maxLength: 255,
                    required: true,
                  })}
                  <TextInput
                    label={t('sourceUrl')}
                    dir="ltr"
                    maxLength={1000}
                    value={s.url ?? ''}
                    onChange={(e) => set({ url: e.target.value })}
                    error={err(`sources.${i}.url`)}
                  />
                </ItemRow>
              );
            })}
          </Repeat>
        </div>
      </Section>

      <TextInput
        label={t('reviewedAt')}
        hint={t('reviewedAtHint')}
        type="date"
        dir="ltr"
        value={reviewedAt}
        onChange={(e) => setReviewedAt(e.target.value)}
        error={err('reviewed_at')}
      />
    </FormPage>
  );
}

/** Mimics the app's week hero: illustration tile, size, measurements, headline and highlight chips. */
function HeroPreview({
  week,
  dir,
  illustration,
  sizeLabel,
  headline,
  lengthCm,
  weightG,
  heartRate,
  highlights,
}: {
  week: number;
  dir?: 'rtl' | 'ltr';
  illustration: string | null;
  sizeLabel: string;
  headline: string;
  lengthCm: string;
  weightG: string;
  heartRate: string;
  highlights: { icon: string | null; tone: string | null; title: string }[];
}) {
  const t = useTranslations('pregnancyWeeks.details');
  const locale = useLocale();
  const stats = [
    [t('lengthCm'), lengthCm],
    [t('weightG'), weightG],
    [t('heartRate'), heartRate],
  ].filter(([, v]) => v);
  return (
    <section className="flex flex-col gap-2" aria-label={t('preview')}>
      <span className="field-label">{t('preview')}</span>
      <p className="field-hint m-0">{t('previewHint')}</p>
      <div className="grid gap-3 md:grid-cols-2">
        {(['light', 'dark'] as const).map((theme) => (
          <div key={theme} data-theme={theme} className="rounded-xl border border-line bg-[var(--page)] p-3 text-[var(--ink)]">
            <div dir={dir} className="flex flex-col gap-3 rounded-2xl bg-[var(--surface)] p-4">
              <div className="flex items-center gap-3">
                <span
                  className="grid size-16 shrink-0 place-items-center rounded-full bg-[var(--pink-bg)] text-[var(--brand)]"
                  title={illustration ?? ''}
                >
                  <Icon name="baby" size={30} />
                </span>
                <span className="flex min-w-0 flex-col">
                  <span className="text-xs text-[var(--muted)]">{t('previewWeek', { week: formatNumber(week, locale) })}</span>
                  <strong className="text-[15px]">{sizeLabel || t('previewNoSize')}</strong>
                  {illustration ? (
                    <span dir="ltr" className="text-[11px] text-[var(--muted)]">
                      {illustration}
                    </span>
                  ) : null}
                </span>
              </div>
              {stats.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {stats.map(([label, value]) => (
                    <span key={label} className="flex flex-col rounded-lg bg-[var(--data-soft)] px-2.5 py-1 text-[var(--data-deep)]">
                      <span className="text-[10px]">{label}</span>
                      <strong dir="ltr" className="text-xs">
                        {value}
                      </strong>
                    </span>
                  ))}
                </div>
              ) : null}
              {headline ? <p className="m-0 text-sm font-semibold">{headline}</p> : null}
              {highlights.length > 0 ? (
                <div className="flex flex-wrap gap-1.5">
                  {highlights.map((h, i) => (
                    <span key={i} className={cn('flex items-center gap-1 rounded-full px-2 py-1 text-xs', toneClass(h.tone))}>
                      <Icon name={highlightIcon(h.icon)} size={12} />
                      {h.title || '…'}
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

function Section({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <fieldset className="form-section m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
      <legend className="field-label p-0">{title}</legend>
      {hint ? <p className="field-hint m-0">{hint}</p> : null}
      {children}
    </fieldset>
  );
}

function Chip({ selected, onClick, children }: { selected: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" aria-pressed={selected} onClick={onClick} className={cn('btn btn-sm', selected ? 'btn-primary' : 'btn-ghost')}>
      {children}
    </button>
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
  const t = useTranslations('pregnancyWeeks.details');
  const n = useNumber();
  return (
    <div className="flex flex-col gap-2">
      <span className="field-label">
        {title} <span className="text-xs text-muted">({t('countOf', { count: n(count), max: n(max) })})</span>
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
  const t = useTranslations('pregnancyWeeks.details');
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
