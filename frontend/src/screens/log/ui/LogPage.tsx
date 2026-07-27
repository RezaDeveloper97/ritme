'use client';

import clsx from 'clsx';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  LOG_CATEGORIES,
  useHealthLog,
  useHealthLogEnums,
  useSaveHealthLog,
  type CategoryDef,
  type HealthLogField,
  type HealthLogInput,
} from '@/entities/health-log';
import { wellbeingKeys } from '@/entities/wellbeing';
import type { Locale } from '@/shared/i18n';
import { addDays, diffInDays, formatJalaliDayMonth, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { Icon, type IconName } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { DayTasks } from '@/widgets/day-tasks';

import { CategorySheet } from './CategorySheet';

// Presentation-only mapping (icon + accent) for each category. Colors are a UI
// concern, so they live here rather than in the entity's model config.
const CATEGORY_STYLE: Record<string, { icon: IconName; color: string; soft: string }> = {
  bleeding: { icon: 'drop', color: 'var(--brand)', soft: 'var(--pink-bg)' },
  pain: { icon: 'zap', color: 'var(--amber)', soft: 'var(--amber-soft)' },
  digestion: { icon: 'glass', color: 'var(--green)', soft: 'var(--green-tint)' },
  mood: { icon: 'smile', color: 'var(--violet)', soft: 'var(--violet-soft)' },
  sleep: { icon: 'moon', color: 'var(--indigo-deep)', soft: 'var(--indigo-soft)' },
  body: { icon: 'sparkle', color: 'var(--orange)', soft: 'var(--orange-soft)' },
  discharge: { icon: 'drop', color: 'var(--blue)', soft: 'var(--blue-soft)' },
  intimate: { icon: 'shield', color: 'var(--teal)', soft: 'var(--teal-soft)' },
  sexual: { icon: 'heart', color: 'var(--rose)', soft: 'var(--pink-bg)' },
  measure: { icon: 'thermo', color: 'var(--muted)', soft: 'var(--line-2)' },
  notes: { icon: 'pencil', color: 'var(--muted)', soft: 'var(--line)' },
};

type T = ReturnType<typeof useTranslations>;

/** Count of non-empty draft fields in a category → drives the card's summary. */
function filledCount(category: CategoryDef, draft: HealthLogInput): number {
  return category.fields.reduce((n, f) => {
    const v = draft[f.key];
    if (v === undefined || v === null) return n;
    if (Array.isArray(v) && v.length === 0) return n;
    if (typeof v === 'string' && v.trim() === '') return n;
    return n + 1;
  }, 0);
}

// ── Day switcher ───────────────────────────────────────────────
interface DaySwitcherProps {
  t: T;
  locale: Locale;
  date: Date;
  isRtl: boolean;
  onShift: (delta: number) => void;
  canGoNext: boolean;
}

function DaySwitcher({ t, locale, date, isRtl, onShift, canGoNext }: DaySwitcherProps) {
  const isToday = diffInDays(date, today()) === 0;
  const prev = (
    <button className="iconbtn" onClick={() => onShift(-1)} aria-label={t('prevDay')}>
      <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
    </button>
  );
  const next = (
    <button
      className="iconbtn"
      onClick={() => canGoNext && onShift(1)}
      disabled={!canGoNext}
      aria-label={t('nextDay')}
    >
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
    </button>
  );

  return (
    <div className="card log-daynav">
      {/* First child sits at the inline-start (right in RTL) */}
      {isRtl ? prev : next}
      <div className="log-daynav-mid">
        <Icon name="calendar" size={16} />
        <span className="log-daynav-date">
          {formatJalaliDayMonth(date, locale)}
        </span>
        {isToday && (
          <span className="log-today">
            {t('today')}
          </span>
        )}
      </div>
      {isRtl ? next : prev}
    </div>
  );
}

// ── Category card ──────────────────────────────────────────────
interface CategoryCardProps {
  t: T;
  category: CategoryDef;
  count: number;
  isRtl: boolean;
  onOpen: () => void;
}

function CategoryCard({ t, category, count, isRtl, onOpen }: CategoryCardProps) {
  const style = CATEGORY_STYLE[category.key];
  return (
    <button
      onClick={onOpen}
      className="card log-cat"
    >
      <span className="dot log-cat-dot" style={{ background: style.soft, color: style.color }}>
        <Icon name={style.icon} size={20} />
      </span>
      <div className="log-cat-body">
        <div className="log-cat-t">
          {t(`categories.${category.key}`)}
        </div>
        <div style={{ fontSize: 12, color: count ? style.color : 'var(--muted)', marginTop: 3, fontWeight: count ? 700 : 500 }}>
          {count ? t('selected', { count }) : t(`categoryHint.${category.key}`)}
        </div>
      </div>
      {/* Disclosure chevron points toward the reading-end (left in RTL). */}
      <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={18} className="log-cat-chev" />
    </button>
  );
}

// ── Main export ────────────────────────────────────────────────
export function LogPage() {
  const t = useTranslations('log');
  const locale = useLocale() as Locale;
  const isRtl = locale === 'fa';

  // Opened from the calendar with `?date=YYYY-MM-DD` to edit a specific day;
  // absent (or a future date) falls back to today. Only the initial value is
  // read — after mount the in-screen day switcher owns the date.
  const searchParams = useSearchParams();
  const [date, setDate] = useState<Date>(() => {
    const param = searchParams.get('date');
    if (!param) return today();
    const parsed = fromApiDate(param);
    return diffInDays(parsed, today()) > 0 ? today() : parsed;
  });
  const apiDate = useMemo(() => toApiDate(date), [date]);

  const enumsQuery = useHealthLogEnums();
  const logQuery = useHealthLog(apiDate);
  const save = useSaveHealthLog();
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState<HealthLogInput>(() => ({ log_date: apiDate }));
  const [openKey, setOpenKey] = useState<string | null>(null);

  // Reset to an empty draft the moment the day changes; the prefill effect below
  // repopulates it once that day's saved log settles.
  useEffect(() => {
    setDraft({ log_date: apiDate });
  }, [apiDate]);

  // Adopt the saved log for the day (kept per-date in the query cache).
  useEffect(() => {
    if (logQuery.data) setDraft(logQuery.data);
  }, [logQuery.data]);

  // ── Auto-save ────────────────────────────────────────────────
  // Every edit persists on its own — there is no manual save. Rapid changes
  // (wheel scrubbing, typing) are coalesced into one per-field patch, so we hit
  // the upsert endpoint once the user pauses rather than on every keystroke.
  const pendingRef = useRef<Record<string, unknown>>({});
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveRef = useRef(save);
  saveRef.current = save;

  // The home "خلاصه هفته" card is scored server-side from these logs, so a
  // saved patch makes it stale. Composing the two entities is this screen's job
  // — the health-log slice stays free of sibling coupling (§3.3).
  const invalidateWeeklyWellbeing = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: wellbeingKeys.all });
  }, [queryClient]);
  const invalidateRef = useRef(invalidateWeeklyWellbeing);
  invalidateRef.current = invalidateWeeklyWellbeing;

  const flush = useCallback((forDate: string) => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    const patch = pendingRef.current;
    if (Object.keys(patch).length === 0) return;
    pendingRef.current = {};
    // Deselected fields carry `null` so the upsert clears them server-side.
    saveRef.current.mutate(
      { log_date: forDate, ...patch },
      { onSuccess: () => invalidateRef.current() },
    );
  }, []);

  const handleChange = (key: HealthLogField, value: unknown) => {
    setDraft((d) => {
      const next = { ...d };
      if (value === undefined) delete next[key];
      else (next as Record<string, unknown>)[key] = value;
      return next;
    });
    pendingRef.current[key] = value === undefined ? null : value;
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => flush(apiDate), 350);
  };

  // Never lose a pending patch: flush before the day changes and on unmount.
  useEffect(() => () => flush(apiDate), [apiDate, flush]);

  const shiftDay = (delta: number) => {
    flush(apiDate);
    setDate((d) => addDays(d, delta));
  };
  const canGoNext = diffInDays(date, today()) < 0;

  const openCategory = openKey ? LOG_CATEGORIES.find((c) => c.key === openKey) ?? null : null;

  const status = save.isPending ? t('saving') : save.isError ? t('saveError') : save.isSuccess ? t('autoSaved') : null;

  return (
    <div className="view log-page">

      <div className="scroll">
        <div className="log-hdr">
          <div className="log-hdr-txt">
            <div className="titr">{t('title')}</div>
            <p className="sub log-hdr-sub">{t('subtitle')}</p>
          </div>
          {status && (
            <span className={clsx('log-status', save.isError && 'is-error')}>
              {save.isSuccess && !save.isError ? <Icon name="check" size={13} /> : null}
              {status}
            </span>
          )}
        </div>

        <div className="sec-tight">
          <DaySwitcher t={t} locale={locale} date={date} isRtl={isRtl} onShift={shiftDay} canGoNext={canGoNext} />
        </div>

        <div className="log-cats">
          {LOG_CATEGORIES.map((category) => (
            <CategoryCard
              key={category.key}
              t={t}
              category={category}
              count={filledCount(category, draft)}
              isRtl={isRtl}
              onOpen={() => setOpenKey(category.key)}
            />
          ))}
        </div>

        {/* Doctor / medication reminders and to-dos set for the day being edited. */}
        <DayTasks date={date} />

        <div className="log-tail" />
      </div>

      <BottomNav />

      {openCategory && (
        <CategorySheet
          category={openCategory}
          enums={enumsQuery.data}
          draft={draft}
          onChange={handleChange}
          onClose={() => setOpenKey(null)}
        />
      )}
    </div>
  );
}
