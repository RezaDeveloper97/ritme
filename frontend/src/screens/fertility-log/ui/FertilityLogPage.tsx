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
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, Link, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import {
  Card,
  Icon,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

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
        <SkeletonGroup label={t('loading')}>
          <Skeleton shape="block" className="ttc-skel-chance" />
          <Skeleton shape="card" className="ttc-skel-tall" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </Shell>
    );
  }
  if (query.isError || !query.data) {
    return (
      <Shell day={null} date={logDate} onBack={null}>
        <LoadError message={t('loadError')} retryLabel={t('retry')} onRetry={() => void query.refetch()} />
      </Shell>
    );
  }
  return <LogForm key={query.data.date} day={query.data} date={logDate} focus={parseFocus(focus)} />;
}

/** Error card with a retry — the same shape on every TTC screen. */
function LoadError({ message, retryLabel, onRetry }: { message: string; retryLabel: string; onRetry: () => void }) {
  return (
    <Card className="ttc-state" role="alert">
      <span className="ttc-state-disc" aria-hidden>
        <Icon name="warning" size={24} />
      </span>
      <p className="ttc-state-text">{message}</p>
      <SecondaryButton icon="refresh" block={false} onClick={onRetry}>
        {retryLabel}
      </SecondaryButton>
    </Card>
  );
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
  const router = useRouter();
  const dateLabel = formatDayMonth(fromApiDate(date), locale);
  const cycleDay = day?.cycleDay ?? null;
  return (
    <div className="view fert-page">
      <div className="scroll ttc-screen">
        <SkyLayer />
        <ScreenHeader
          title={t('log.title')}
          subtitle={
            cycleDay === null
              ? dateLabel
              : t('log.subtitle', { date: dateLabel, day: formatNumber(cycleDay, locale) })
          }
          onBack={onBack ?? (() => router.push(BACK_HREF))}
          backLabel={t('back')}
        />
        <div className="ttc-body is-form">{children}</div>
      </div>
    </div>
  );
}

function ChanceCard({ chance, isToday }: { chance: FertilityChance; isToday: boolean }) {
  const t = useTranslations('fertility');
  // A past day's log must not say «امروز» (stage regression A, B-7).
  const title = isToday ? t('chance.today') : t('chance.day');
  const level = chance.level ?? 'unknown';
  const levelLabel = chance.label ?? t(`chance.levels.${level}`);
  const bars = Math.max(0, Math.min(CHANCE_MAX_BARS, chance.bars));
  return (
    <Link
      href={INSIGHTS_HREF}
      className="ttc-chance"
      aria-label={`${title}: ${levelLabel} · ${t('log.openInsights')}`}
    >
      <span className="ttc-chance-disc">
        <Icon name="target" size={24} />
      </span>
      <span className="ttc-chance-text">
        <span className="ttc-chance-over">{title}</span>
        <span className="ttc-chance-level">{levelLabel}</span>
      </span>
      <span className="ttc-chance-bars" aria-hidden>
        {BAR_HEIGHTS.map((h, i) => (
          <span key={h} className={clsx('ttc-chance-bar', h, i < bars && 'is-on')} />
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
      className={clsx('ttc-log-sect', highlight && 'is-highlight')}
    >
      <h2 id={`fertility-log-${id}-title`} className="ttc-log-title">
        {title}
      </h2>
      {children}
    </section>
  );
}

/**
 * `v19_TTC_Log` chip: 44 px (touch target), tinted `--brand-soft` + 1.5 px brand
 * outline when on — the soft look of the artboard, not the solid single chip.
 */
function Chip({ on, label, onClick }: { on: boolean; label: string; onClick: () => void }) {
  return (
    <PillChip pressed={on} onClick={onClick} className="ttc-chip">
      {label}
    </PillChip>
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
    <div className="nb-chips">
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
  const [saveError, setSaveError] = useState<string | null>(null);
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
    setSaveError(null);
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
        // 429 = the per-user write limit (60/min): say so, in the user's language,
        // instead of the generic failure (the API body is the framework's English).
        onError: (error) =>
          setSaveError(getApiErrorStatus(error) === 429 ? tl('tooManyRequests') : t('saveError')),
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
      {day.chance && <ChanceCard chance={day.chance} isToday={date === toApiDate(today())} />}

      <Card as="section" padding="lg" className="ttc-log-card" aria-label={tl('title')}>
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
          <div className="ttc-bbt-field">
            <label className="ttc-bbt-label">
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
                className="ttc-bbt-input"
              />
            </label>
            <span className="ttc-bbt-unit">{tl('bbt.unit')}</span>
            <button
              type="button"
              className="ttc-bbt-step"
              aria-label={tl('bbt.decrease')}
              onClick={() => onStep(-1)}
            >
              <Icon name="minus" size={20} strokeWidth={2.4} />
            </button>
            <button
              type="button"
              className="ttc-bbt-step"
              aria-label={tl('bbt.increase')}
              onClick={() => onStep(1)}
            >
              <Icon name="plus" size={20} strokeWidth={2.4} />
            </button>
          </div>
          {bbtError ? (
            <p id={bbtErrorId} role="alert" className="ttc-field-error">
              {bbtError}
            </p>
          ) : (
            <p id="fertility-log-bbt-hint" className="ttc-field-hint">
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
          <div className="nb-chips">
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
      </Card>

      {/* The note sits outside the card with its own label, above the button. */}
      <Section id="note" title={tl('note.label')} highlight={highlight === 'note'} sectionRef={refFor('note')}>
        <textarea
          aria-labelledby="fertility-log-note-title"
          rows={4}
          value={state.note}
          placeholder={tl('note.placeholder')}
          onChange={(e) => set('note', e.target.value)}
          className="ttc-note"
        />
      </Section>

      {saveError && (
        <p role="alert" className="ttc-save-error">
          {saveError}
        </p>
      )}

      <div className="ttc-footer">
        {toast && (
          <p role="status" className="ttc-toast">
            {toast}
          </p>
        )}
        <PrimaryButton loading={save.isPending} disabled={!bbtCheck.ok || toast !== null} onClick={onSave}>
          {save.isPending ? tl('saving') : tl('save')}
        </PrimaryButton>
      </div>
    </Shell>
  );
}
