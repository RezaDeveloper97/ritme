'use client';

import clsx from 'clsx';
import type { ReactNode } from 'react';
import { useFormatter, useTranslations } from 'next-intl';

import { LOG_CATEGORIES, useHealthLog, type FieldDef, type HealthLogInput } from '@/entities/health-log';
import { useFertilityDay, type ChanceLevel } from '@/entities/fertility';
import type { Locale } from '@/shared/i18n';
import { formatDayMonth, toApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { Icon, Skeleton, SkeletonGroup, type IconName } from '@/shared/ui';

type TLog = ReturnType<typeof useTranslations<'log'>>;
type Format = ReturnType<typeof useFormatter>;
// `log.enums.*` keys mirror the API enums, so the typed translator can't see them.
type DynT = (key: string) => string;

/** Chip accent + glyph per log category (artboard: soft tint, 1.5px tone border, icon disc). */
const CATEGORY_CHIP: Record<string, { tone: string; icon: IconName }> = {
  bleeding: { tone: 'period', icon: 'drop' },
  pain: { tone: 'warm', icon: 'flame' },
  digestion: { tone: 'warm', icon: 'sparkle' },
  mood: { tone: 'brand', icon: 'smile' },
  sleep: { tone: 'brand', icon: 'moon' },
  body: { tone: 'bloom', icon: 'heart' },
  discharge: { tone: 'data', icon: 'drop' },
  intimate: { tone: 'bloom', icon: 'heart' },
  sexual: { tone: 'bloom', icon: 'heart' },
  weight: { tone: 'data', icon: 'scale' },
  temperature: { tone: 'data', icon: 'thermo' },
  notes: { tone: 'neutral', icon: 'note' },
};

const MAX_CHIPS = 8;

interface LogChip {
  key: string;
  category: string;
  text: string;
}

/** One chip text per recorded value (symptoms read by name, choices by value). */
function chipTexts(field: FieldDef, raw: unknown, tLog: TLog, format: Format): string[] {
  if (raw === undefined || raw === null || raw === false || raw === '') return [];
  const label = tLog(`fields.${field.key}` as never);
  const dyn = tLog as unknown as DynT;
  const control = field.control;
  switch (control.kind) {
    case 'bool':
    case 'degree':
      return raw === true || control.kind === 'degree' ? [label] : [];
    case 'chips':
      return [dyn(`enums.${control.enumKey}.${String(raw)}`)];
    case 'multi':
      return (Array.isArray(raw) ? (raw as string[]) : []).map((v) => dyn(`enums.${control.enumKey}.${v}`));
    case 'measure':
      return [`${label} ${format.number(raw as number)} ${dyn(`units.${control.unit}`)}`];
    case 'note':
      return String(raw).trim() ? [label] : [];
    default:
      return [];
  }
}

function logChips(log: HealthLogInput, tLog: TLog, format: Format): LogChip[] {
  const chips: LogChip[] = [];
  for (const category of LOG_CATEGORIES) {
    for (const field of category.fields) {
      chipTexts(field, log[field.key], tLog, format).forEach((text, i) =>
        chips.push({ key: `${field.key}-${i}`, category: category.key, text }),
      );
    }
  }
  return chips;
}

/** Recorded items of the day as tone chips (read-only; editing goes to the log screen). */
function LoggedChips({ date }: { date: Date }) {
  const t = useTranslations('calendar.nb');
  const tLog = useTranslations('log');
  const format = useFormatter();
  const mounted = useMounted();
  const logQuery = useHealthLog(toApiDate(date));

  if (!mounted || logQuery.isLoading) {
    return (
      <SkeletonGroup label={t('loadingDay')} className="cc-chips">
        <Skeleton shape="block" width="short" />
        <Skeleton shape="block" width="short" />
      </SkeletonGroup>
    );
  }
  const chips = logQuery.data ? logChips(logQuery.data, tLog, format) : [];
  if (chips.length === 0) return <p className="cc-day-empty">{t('noLogs')}</p>;
  const shown = chips.slice(0, MAX_CHIPS);
  return (
    <ul className="cc-chips" aria-label={t('loggedLabel')}>
      {shown.map((chip) => {
        const style = CATEGORY_CHIP[chip.category] ?? CATEGORY_CHIP.notes;
        return (
          <li key={chip.key} className={clsx('cc-chip', `nb-tone-${style.tone}`)}>
            <span className="cc-chip-ic" aria-hidden>
              <Icon name={style.icon} size={15} />
            </span>
            {chip.text}
          </li>
        );
      })}
      {chips.length > MAX_CHIPS ? (
        <li className="cc-chip nb-tone-neutral">{t('moreLogs', { n: format.number(chips.length - MAX_CHIPS) })}</li>
      ) : null}
    </ul>
  );
}

const CHANCE_TONE: Record<ChanceLevel, string> = {
  none: 'is-low',
  low: 'is-low',
  medium: 'is-mid',
  high: 'is-high',
  peak: 'is-high',
};

interface DayCardProps {
  date: Date;
  locale: Locale;
  cycleDay: number | null;
  /** Short phase name for the pill («لوتئال»), or null when unknown. */
  phaseLabel: string | null;
  /** TTC variant: chance of pregnancy headline + hint instead of the phase pill. */
  ttc: boolean;
  /** Future days have nothing to show or log. */
  isFuture: boolean;
  onLog: () => void;
  /** Period range actions (edit / extend / start / remove), rendered under the log button. */
  actions?: ReactNode;
}

/**
 * Selected-day card (`Cycle_Calendar` / `TTC_Calendar`): «۱۲ مهر · روز ۲۵ سیکل»,
 * phase pill or chance of pregnancy, the day's logged items and «ثبت جزئیات».
 */
export function DayCard({ date, locale, cycleDay, phaseLabel, ttc, isFuture, onLog, actions }: DayCardProps) {
  const t = useTranslations('calendar.nb');
  const tCal = useTranslations('calendar');
  const format = useFormatter();
  const iso = toApiDate(date);
  // Same `/fertility/days/{date}` read (and cache entry) the day log shows (§26).
  const chance = useFertilityDay(ttc ? iso : '').data?.chance ?? null;
  const dateLine = formatDayMonth(date, locale);
  const head =
    cycleDay != null ? t('dayHead', { date: dateLine, n: format.number(cycleDay) }) : dateLine;

  return (
    <section className="nb-card cc-daycard" aria-live="polite" aria-labelledby="cc-day-title">
      {ttc ? (
        <div className="cc-day-top">
          <div className="cc-day-titles">
            <h2 id="cc-day-title" className="cc-day-cap">
              {head}
            </h2>
            <p className="cc-chance">
              {t('chanceTitle')}{' '}
              <span className={clsx('cc-chance-v', chance?.level ? CHANCE_TONE[chance.level] : 'is-low')}>
                {chance ? (chance.label ?? tCal(`chance.${chance.level ?? 'unknown'}`)) : '—'}
              </span>
            </p>
          </div>
          <span className="cc-chance-disc" aria-hidden>
            <Icon name="target" size={24} />
          </span>
        </div>
      ) : (
        <div className="cc-day-top">
          <h2 id="cc-day-title" className="cc-day-title">
            {head}
          </h2>
          {phaseLabel ? <span className="cc-phase-pill">{phaseLabel}</span> : null}
        </div>
      )}

      {ttc ? (
        <p className="cc-day-hint">
          {chance?.level === 'high' || chance?.level === 'peak' ? t('ttcHintHigh') : t('ttcHint')}
        </p>
      ) : null}

      {!isFuture && !ttc ? <LoggedChips date={date} /> : null}

      {!isFuture ? (
        <button type="button" className={clsx('cc-log-btn', !ttc && 'is-inline')} onClick={onLog}>
          <Icon name="plus" size={18} strokeWidth={2.2} />
          {ttc ? t('logDayDetails') : t('logDetails')}
        </button>
      ) : (
        <p className="cc-day-empty">{t('futureDay')}</p>
      )}

      {actions}
    </section>
  );
}
