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
/** `v19_TTC_Log` chance bars: 10 → 30 px in 5 px steps. */
const BAR_HEIGHTS = ['h-2.5', 'h-[15px]', 'h-5', 'h-[25px]', 'h-7.5'] as const;

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
        <header className="rmd-hdr">
          <button
            type="button"
            className="rmd-hdr-btn"
            aria-label={t('back')}
            onClick={onBack ?? (() => router.push(BACK_HREF))}
          >
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </button>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{t('log.title')}</h1>
            <p className="rmd-hdr-sub">
              {cycleDay === null
                ? dateLabel
                : t('log.subtitle', { date: dateLabel, day: formatNumber(cycleDay, locale) })}
            </p>
          </div>
          <span className="rmd-hdr-btn invisible" aria-hidden />
        </header>
        <div className="flex flex-col gap-4.5 px-4 pt-1 pb-32">{children}</div>
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
      className="flex items-center gap-3.5 rounded-3xl border border-(--fert-chance-line) bg-linear-to-l from-(--fert-chance-from) to-(--fert-chance-to) p-4"
      aria-label={`${t('chance.today')}: ${levelLabel} · ${t('log.openInsights')}`}
    >
      <span className="grid size-12 shrink-0 place-items-center rounded-full bg-(--fert-chance-disc) text-(--fert-amber)">
        <Icon name="target" size={24} />
      </span>
      <span className="flex min-w-0 flex-1 flex-col text-start">
        <span className="text-[11px] font-extrabold text-(--fert-amber)">{t('chance.today')}</span>
        <span className="text-[20px] font-extrabold text-(--ink)">{levelLabel}</span>
      </span>
      <span className="flex items-end gap-1" aria-hidden>
        {BAR_HEIGHTS.map((h, i) => (
          <span
            key={h}
            className={clsx('w-2 rounded-sm', h, i < bars ? 'bg-(--fert-amber)' : 'bg-(--fert-bar-off)')}
          />
        ))}
      </span>
    </Link>
  );
}

/** One group inside the shared log card (`v19_TTC_Log`: title + control, 22 px apart). */
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
        'flex scroll-m-24 flex-col gap-2.5 rounded-2xl transition-shadow duration-500',
        highlight && 'ring-2 ring-(--fert-chip-on-line) ring-offset-4 ring-offset-(--surface)',
      )}
    >
      <h2 id={`fertility-log-${id}-title`} className="text-start text-[14.5px] font-extrabold text-(--ink)">
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
        'h-11 rounded-full border-[1.5px] px-4 text-[13.5px] font-bold transition-colors',
        on
          ? 'border-(--fert-chip-on-line) bg-(--fert-chip-on-bg) text-(--ink)'
          : 'border-(--line) bg-transparent text-(--ink-3)',
      )}
    >
      {label}
    </button>
  );
}

/**
 * One-of chips. With `noneLabel` a «ثبت نشه» chip clears the value (LH,
 * intercourse, as drawn); without it (mucus, audit #25) tapping the selected
 * chip again clears it.
 */
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
  noneLabel?: string;
  onChange: (v: T | null) => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      {noneLabel !== undefined && <Chip on={value === null} label={noneLabel} onClick={() => onChange(null)} />}
      {values.map((v) => (
        <Chip
          key={v}
          on={value === v}
          label={labelOf(v)}
          onClick={() => onChange(noneLabel === undefined && value === v ? null : v)}
        />
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

      <div className="fert-card flex flex-col gap-5.5">
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
          {/* One bordered field: the value at the start, then the unit and both
              steppers together at the end (`v19_TTC_Log`). Empty = not logged. */}
          <div className="flex items-center gap-2 rounded-[20px] border border-(--line) bg-(--fert-field) py-1.5 ps-4.5 pe-1.5">
            <label className="flex min-w-0 flex-1 items-center">
              <span className="sr-only">{tl('bbt.label')}</span>
              {/* Body font, not Lalezar: an input can't set «٫» apart, and Lalezar
                  draws it like «/» (audit #23). */}
              <input
                inputMode="decimal"
                dir="ltr"
                value={state.bbtText}
                placeholder={formatBbt(36.5, locale)}
                aria-invalid={bbtError !== null}
                aria-describedby={bbtError ? bbtErrorId : 'fertility-log-bbt-hint'}
                onChange={(e) => set('bbtText', e.target.value)}
                className="w-full min-w-0 bg-transparent text-start text-[28px] leading-none font-extrabold text-(--ink) outline-none placeholder:text-(--muted)"
              />
            </label>
            <span className="text-[13px] font-bold text-(--ink-3)">{tl('bbt.unit')}</span>
            <button
              type="button"
              className="grid size-11 shrink-0 place-items-center rounded-[14px] bg-(--fert-chip-on-bg) text-[22px] font-extrabold text-(--ink)"
              aria-label={tl('bbt.decrease')}
              onClick={() => onStep(-1)}
            >
              −
            </button>
            <button
              type="button"
              className="grid size-11 shrink-0 place-items-center rounded-[14px] bg-(--fert-chip-on-bg) text-[22px] font-extrabold text-(--ink)"
              aria-label={tl('bbt.increase')}
              onClick={() => onStep(1)}
            >
              +
            </button>
          </div>
          {bbtError ? (
            <p id={bbtErrorId} role="alert" className="text-start text-[12px] font-bold text-(--danger-deep)">
              {bbtError}
            </p>
          ) : (
            <p id="fertility-log-bbt-hint" className="text-start text-[11.5px] font-semibold text-(--muted)">
              {tl('bbt.hint')}
            </p>
          )}
        </Section>

        <Section id="mucus" title={tl('mucus.title')} highlight={highlight === 'mucus'} sectionRef={refFor('mucus')}>
          <SingleChips
            values={CERVICAL_MUCUS}
            value={state.mucus}
            labelOf={(v) => tl(`mucus.options.${v}`)}
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
      </div>

      {/* The note sits outside the card with its own label, above the button. */}
      <Section id="note" title={tl('note.label')} highlight={highlight === 'note'} sectionRef={refFor('note')}>
        <textarea
          aria-labelledby="fertility-log-note-title"
          rows={4}
          value={state.note}
          placeholder={tl('note.placeholder')}
          onChange={(e) => set('note', e.target.value)}
          className="w-full resize-none rounded-[20px] border border-(--line) bg-(--fert-field) p-4 text-start text-[14px] text-(--ink) outline-none placeholder:text-(--muted) focus-visible:shadow-(--ring)"
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
