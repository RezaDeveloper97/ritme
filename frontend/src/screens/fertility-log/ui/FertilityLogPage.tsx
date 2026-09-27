'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type ReactNode, useEffect, useRef, useState } from 'react';

import {
  BBT_MAX,
  BBT_MIN,
  CERVICAL_MUCUS,
  CHANCE_MAX_BARS,
  type FertilityChance,
  type FertilityDay,
  FERTILITY_SYMPTOMS,
  formatBbt,
  INTERCOURSE_TYPES,
  LH_RESULTS,
  stepBbt,
  useFertilityDay,
} from '@/entities/fertility';
import { useSaveFertilityDay } from '@/features/log-fertility-day';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import {
  changedInput,
  checkBbt,
  fromDay,
  isDirty,
  type LogFormState,
  type LogSection,
  parseFocus,
  parseLogDate,
  toggleSymptom,
} from '../model/form';

const INSIGHTS_HREF = '/fertility/insights';
const BAR_HEIGHTS = ['h-2', 'h-3', 'h-4', 'h-5', 'h-6'] as const;

/** Back never walks history (CLAUDE.md §4.2): the log always returns to home. */
const BACK_HREF = '/home';

/**
 * «ثبت روز» — the TTC day log (`v19_TTC_Log` / `nb2_TTC_Log`), route
 * `/fertility/log?date=Y-m-d&focus=lh|bbt|mucus|intercourse|symptoms|note`.
 *
 * Privacy (§11): the day is health data — shown and sent, never logged.
 */
export function FertilityLogPage({ date, focus }: { date?: string; focus?: string }) {
  const t = useTranslations('fertility');
  const logDate = parseLogDate(date, toApiDate(today()));
  const query = useFertilityDay(logDate);

  if (query.isPending) {
    return (
      <Shell day={null} date={logDate} onBack={null}>
        <p className="py-10 text-center text-[14px] text-(--muted)">{t('loading')}</p>
      </Shell>
    );
  }
  if (query.isError || !query.data) {
    return (
      <Shell day={null} date={logDate} onBack={null}>
        <p className="py-10 text-center text-[14px] text-(--muted)">{t('loadError')}</p>
        <button
          type="button"
          className="mx-auto rounded-full bg-(--fert-chip-on-bg) px-5 py-2 text-[14px] font-bold text-(--brand)"
          onClick={() => void query.refetch()}
        >
          {t('retry')}
        </button>
      </Shell>
    );
  }
  return <LogForm key={query.data.date} day={query.data} date={logDate} focus={parseFocus(focus)} />;
}

function Shell({
  day,
  date,
  onBack,
  children,
}: {
  day: FertilityDay | null;
  date: string;
  onBack: (() => void) | null;
  children: ReactNode;
}) {
  const t = useTranslations('fertility');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const dateLabel = formatDayMonth(fromApiDate(date), locale);
  const cycleDay = day?.cycleDay ?? null;
  return (
    <div className="view">
      <div className="scroll">
        <header className="flex items-center gap-3 px-4 pt-4 pb-3">
          <button
            type="button"
            className="iconbtn"
            aria-label={t('back')}
            onClick={onBack ?? (() => router.push(BACK_HREF))}
          >
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={22} />
          </button>
          <div className="flex min-w-0 flex-1 flex-col text-start">
            <h1 className="text-[18px] font-extrabold text-(--ink)">{t('log.title')}</h1>
            <p className="text-[13px] text-(--muted)">
              {cycleDay === null
                ? dateLabel
                : t('log.subtitle', { date: dateLabel, day: formatNumber(cycleDay, locale) })}
            </p>
          </div>
        </header>
        <div className="flex flex-col gap-3 px-4 pb-32">{children}</div>
      </div>
    </div>
  );
}

function ChanceCard({ chance }: { chance: FertilityChance }) {
  const t = useTranslations('fertility');
  const level = chance.level ?? 'unknown';
  const levelLabel = chance.label ?? t(`chance.levels.${level}`);
  const bars = Math.max(0, Math.min(CHANCE_MAX_BARS, chance.bars));
  return (
    <Link
      href={INSIGHTS_HREF}
      className="flex items-center gap-3 rounded-[20px] border border-(--fert-chance-line) bg-linear-to-l from-(--fert-chance-from) to-(--fert-chance-to) p-4"
      aria-label={`${t('chance.today')}: ${levelLabel} · ${t('log.openInsights')}`}
    >
      <span className="grid size-11 shrink-0 place-items-center rounded-full bg-(--fert-chance-disc) text-(--fert-amber)">
        <Icon name="target" size={22} />
      </span>
      <span className="flex min-w-0 flex-1 flex-col text-start">
        <span className="text-[12px] text-(--muted)">{t('chance.today')}</span>
        <span className="text-[17px] font-extrabold text-(--fert-amber)">{levelLabel}</span>
      </span>
      <span className="flex items-end gap-1" aria-hidden>
        {BAR_HEIGHTS.map((h, i) => (
          <span
            key={h}
            className={clsx('w-1.5 rounded-full', h, i < bars ? 'bg-(--fert-amber)' : 'bg-(--fert-bar-off)')}
          />
        ))}
      </span>
    </Link>
  );
}

function Section({
  id,
  title,
  highlight,
  sectionRef,
  children,
}: {
  id: LogSection;
  title: string;
  highlight: boolean;
  sectionRef?: (el: HTMLElement | null) => void;
  children: ReactNode;
}) {
  return (
    <section
      id={`fertility-log-${id}`}
      ref={sectionRef}
      aria-labelledby={`fertility-log-${id}-title`}
      className={clsx(
        'card flex flex-col gap-3 transition-shadow duration-500 p-4',
        highlight && 'ring-2 ring-(--fert-chip-on-line)',
      )}
    >
      <h2 id={`fertility-log-${id}-title`} className="text-start text-[15px] font-bold text-(--ink)">
        {title}
      </h2>
      {children}
    </section>
  );
}

function Chip({ on, label, onClick }: { on: boolean; label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      aria-pressed={on}
      onClick={onClick}
      className={clsx(
        'rounded-full border px-3.5 py-2 text-[13px] font-bold transition-colors',
        on
          ? 'border-(--fert-chip-on-line) bg-(--fert-chip-on-bg) text-(--brand)'
          : 'border-(--line) bg-(--surface) text-(--ink-3)',
      )}
    >
      {label}
    </button>
  );
}

function SingleChips<T extends string>({
  values,
  value,
  labelOf,
  noneLabel,
  onChange,
}: {
  values: readonly T[];
  value: T | null;
  labelOf: (v: T) => string;
  noneLabel: string;
  onChange: (v: T | null) => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <Chip on={value === null} label={noneLabel} onClick={() => onChange(null)} />
      {values.map((v) => (
        <Chip key={v} on={value === v} label={labelOf(v)} onClick={() => onChange(v)} />
      ))}
    </div>
  );
}

function LogForm({ day, date, focus }: { day: FertilityDay; date: string; focus: LogSection | null }) {
  const t = useTranslations('fertility');
  const tl = useTranslations('fertility.log');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const save = useSaveFertilityDay();

  const fmt = (v: number) => formatBbt(v, locale);
  const [state, setState] = useState<LogFormState>(() => fromDay(day, fmt));
  const [highlight, setHighlight] = useState<LogSection | null>(focus);
  const [toast, setToast] = useState<string | null>(null);
  const [saveError, setSaveError] = useState(false);
  const focusRef = useRef<HTMLElement | null>(null);

  const dirty = isDirty(day, state);
  const bbtCheck = checkBbt(state.bbtText);
  const leavingRef = useRef(false);

  // `focus` scrolls to and briefly highlights that section.
  useEffect(() => {
    if (!focus) return;
    focusRef.current?.scrollIntoView({ block: 'center', behavior: 'smooth' });
    const timer = window.setTimeout(() => setHighlight(null), 2000);
    return () => window.clearTimeout(timer);
  }, [focus]);

  // Dirty-state guard on reload / tab close.
  useEffect(() => {
    if (!dirty) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      if (leavingRef.current) return;
      e.preventDefault();
    };
    window.addEventListener('beforeunload', onBeforeUnload);
    return () => window.removeEventListener('beforeunload', onBeforeUnload);
  }, [dirty]);

  const set = <K extends keyof LogFormState>(key: K, value: LogFormState[K]) =>
    setState((s) => ({ ...s, [key]: value }));

  const onBack = () => {
    if (dirty && !window.confirm(tl('discardConfirm'))) return;
    leavingRef.current = true;
    router.push(BACK_HREF);
  };

  const onStep = (steps: number) => {
    const current = checkBbt(state.bbtText);
    const base = current.ok ? current.value : null;
    set('bbtText', fmt(stepBbt(base, steps)));
  };

  const onSave = () => {
    const input = changedInput(day, state);
    if (input === null || save.isPending) return;
    setSaveError(false);
    if (Object.keys(input).length === 0) {
      router.push(BACK_HREF);
      return;
    }
    save.mutate(
      { date, input },
      {
        onSuccess: () => {
          leavingRef.current = true;
          setToast(tl('saved'));
          window.setTimeout(() => router.push(BACK_HREF), 900);
        },
        onError: () => setSaveError(true),
      },
    );
  };

  const refFor = (id: LogSection) =>
    id === focus
      ? (el: HTMLElement | null) => {
          focusRef.current = el;
        }
      : undefined;

  const bbtErrorId = 'fertility-log-bbt-error';
  const bbtError = !bbtCheck.ok
    ? bbtCheck.reason === 'range'
      ? tl('bbt.outOfRange', { min: formatBbt(BBT_MIN, locale), max: formatBbt(BBT_MAX, locale) })
      : tl('bbt.invalid')
    : null;

  return (
    <Shell day={day} date={date} onBack={onBack}>
      {day.chance && <ChanceCard chance={day.chance} />}

      <Section id="lh" title={tl('lh.title')} highlight={highlight === 'lh'} sectionRef={refFor('lh')}>
        <SingleChips
          values={LH_RESULTS}
          value={state.lh}
          labelOf={(v) => tl(`lh.options.${v}`)}
          noneLabel={tl('lh.options.none')}
          onChange={(v) => set('lh', v)}
        />
      </Section>

      <Section id="bbt" title={tl('bbt.label')} highlight={highlight === 'bbt'} sectionRef={refFor('bbt')}>
        <div className="flex items-center gap-3">
          <button
            type="button"
            className="grid size-11 shrink-0 place-items-center rounded-full bg-(--fert-chip-on-bg) text-[22px] font-bold text-(--brand)"
            aria-label={tl('bbt.decrease')}
            onClick={() => onStep(-1)}
          >
            −
          </button>
          <label className="flex min-w-0 flex-1 items-baseline justify-center gap-1 rounded-2xl bg-(--fert-field) px-3 py-2">
            <span className="sr-only">{tl('bbt.label')}</span>
            <input
              inputMode="decimal"
              dir="ltr"
              value={state.bbtText}
              placeholder={formatBbt(36.5, locale)}
              aria-invalid={bbtError !== null}
              aria-describedby={bbtError ? bbtErrorId : 'fertility-log-bbt-hint'}
              onChange={(e) => set('bbtText', e.target.value)}
              className="w-28 bg-transparent text-center font-['Lalezar',var(--font-sans)] text-[34px] leading-none text-(--ink) outline-none placeholder:text-(--muted)"
            />
            <span className="text-[14px] font-bold text-(--muted)">{tl('bbt.unit')}</span>
          </label>
          <button
            type="button"
            className="grid size-11 shrink-0 place-items-center rounded-full bg-(--fert-chip-on-bg) text-[22px] font-bold text-(--brand)"
            aria-label={tl('bbt.increase')}
            onClick={() => onStep(1)}
          >
            +
          </button>
        </div>
        {state.bbtText !== '' && (
          <button
            type="button"
            className="self-start text-[12px] font-bold text-(--brand)"
            onClick={() => set('bbtText', '')}
          >
            {tl('bbt.clear')}
          </button>
        )}
        {bbtError ? (
          <p id={bbtErrorId} role="alert" className="text-start text-[12px] font-bold text-(--danger-deep)">
            {bbtError}
          </p>
        ) : (
          <p id="fertility-log-bbt-hint" className="text-start text-[12px] text-(--muted)">
            {tl('bbt.hint')}
          </p>
        )}
      </Section>

      <Section id="mucus" title={tl('mucus.title')} highlight={highlight === 'mucus'} sectionRef={refFor('mucus')}>
        <SingleChips
          values={CERVICAL_MUCUS}
          value={state.mucus}
          labelOf={(v) => tl(`mucus.options.${v}`)}
          noneLabel={tl('mucus.options.none')}
          onChange={(v) => set('mucus', v)}
        />
      </Section>

      <Section
        id="intercourse"
        title={tl('intercourse.title')}
        highlight={highlight === 'intercourse'}
        sectionRef={refFor('intercourse')}
      >
        <SingleChips
          values={INTERCOURSE_TYPES}
          value={state.intercourse}
          labelOf={(v) => tl(`intercourse.options.${v}`)}
          noneLabel={tl('intercourse.options.none')}
          onChange={(v) => set('intercourse', v)}
        />
      </Section>

      <Section
        id="symptoms"
        title={tl('symptoms.title')}
        highlight={highlight === 'symptoms'}
        sectionRef={refFor('symptoms')}
      >
        <div className="flex flex-wrap gap-2">
          {FERTILITY_SYMPTOMS.map((s) => (
            <Chip
              key={s}
              on={state.symptoms.includes(s)}
              label={tl(`symptoms.options.${s}`)}
              onClick={() => set('symptoms', toggleSymptom(state.symptoms, s))}
            />
          ))}
        </div>
      </Section>

      <Section id="note" title={tl('note.label')} highlight={highlight === 'note'} sectionRef={refFor('note')}>
        <textarea
          aria-label={tl('note.label')}
          rows={3}
          value={state.note}
          placeholder={tl('note.placeholder')}
          onChange={(e) => set('note', e.target.value)}
          className="w-full resize-none rounded-2xl bg-(--fert-field) p-3 text-start text-[14px] text-(--ink) outline-none placeholder:text-(--muted)"
        />
      </Section>

      {saveError && (
        <p role="alert" className="text-center text-[13px] font-bold text-(--danger-deep)">
          {t('saveError')}
        </p>
      )}

      <div className="fixed inset-x-0 bottom-0 z-10 mx-auto flex max-w-[480px] flex-col items-center gap-2 bg-(--page) px-4 pt-3 pb-[calc(16px+env(safe-area-inset-bottom))]">
        {toast && (
          <p role="status" className="rounded-full bg-(--ink) px-4 py-2 text-[13px] font-bold text-(--surface)">
            {toast}
          </p>
        )}
        <button
          type="button"
          className="btn btn-primary w-full"
          disabled={!bbtCheck.ok || save.isPending || toast !== null}
          onClick={onSave}
        >
          {save.isPending ? tl('saving') : tl('save')}
        </button>
      </div>
    </Shell>
  );
}
