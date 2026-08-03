'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

import { useUserProfile } from '@/entities/user';
import { type Locale } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatMonthLabel,
  fromApiDate,
  monthMatrix,
  shiftMonth,
  toApiDate,
  toParts,
  today,
  weekdayKeys,
  type MonthCell,
} from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { usePeriodHistory, useReconcilePeriods, type PeriodSegment } from '../api/mutations';

const PERIOD_COLOR = 'var(--brand)';

// Fallback bleed length when the profile hasn't recorded one — matches the
// calendar so an open period pre-fills to the same span the user sees there.
const DEFAULT_PERIOD_DAYS = 5;

// How many months of history to show above the opened month, so the user can
// scroll back to older periods without a month-picker.
const MONTHS_BACK = 12;

interface PeriodDateEditorProps {
  open: boolean;
  onClose: () => void;
  /** Month to open on (locale's calendar) — the month the calendar is showing. */
  initialView?: { year: number; month: number };
  /** Fired after a successful save, before the editor closes (to show recalculating). */
  onSaved?: () => void;
}

/** Serialize a month to a stable key / ordinal (both calendars have 12 months). */
const monthKey = (y: number, m: number) => `${y}-${m}`;
const monthOrd = (y: number, m: number) => y * 12 + (m - 1);

/** Every ISO day in the inclusive [start, end] range. */
function isoRange(start: string, end: string): string[] {
  const out: string[] = [];
  const first = fromApiDate(start);
  const count = diffInDays(fromApiDate(end), first) + 1;
  for (let i = 0; i < count; i += 1) out.push(toApiDate(addDays(first, i)));
  return out;
}

/** Group a set of ISO days into contiguous [start, end] segments (chronological). */
function toSegments(selected: Set<string>): PeriodSegment[] {
  const days = [...selected].sort();
  const segments: PeriodSegment[] = [];
  for (const iso of days) {
    const last = segments[segments.length - 1];
    const prevIso = last ? toApiDate(addDays(fromApiDate(last.end), 1)) : null;
    if (last && prevIso === iso) last.end = iso;
    else segments.push({ start: iso, end: iso });
  }
  return segments;
}

/**
 * Full-screen "Edit Period Date" editor — the calendar's top-button entry. Months
 * are stacked in one vertical scroll (opened on the calendar's month), so a period
 * spanning a month boundary reads as one continuous run. Every logged period is
 * pre-selected; each day is an independent toggle (tap a filled day to clear it, an
 * empty one to select it). The badge counts the day's position *within its own
 * contiguous run* — a one-day gap restarts the count at 1 — matching how the user
 * thinks of "day N of this period". On save the whole selection is reconciled
 * against the logged history in one action, so moving, extending, splitting, merging
 * or removing periods all just work. Future days can't be bled, so they're disabled.
 * Health data never leaves this component's calls (CLAUDE.md §11); all copy is
 * i18n'd and RTL-safe (§6, §12).
 */
export function PeriodDateEditor({ open, onClose, initialView, onSaved }: PeriodDateEditorProps) {
  const t = useTranslations('logPeriod');
  const locale = useLocale() as Locale;
  const format = useFormatter();
  const todayIso = toApiDate(today());
  const currentJ = toParts(today(), locale);

  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const monthRefs = useRef<Map<string, HTMLDivElement | null>>(new Map());

  // The editor is `position: absolute; inset: 0`, so it must live directly under
  // the phone frame — mounted in place it would resolve against whatever scroll
  // container the caller sits in (the cycle screen's `.scroll`) and render off
  // screen. Portalling to `.app-shell` makes it full-screen from any call site.
  const [host, setHost] = useState<HTMLElement | null>(null);
  useEffect(() => {
    if (!open) return;
    setHost(document.querySelector<HTMLElement>('.app-shell') ?? document.body);
  }, [open]);

  const historyQuery = usePeriodHistory();
  const profileQuery = useUserProfile();
  const periodDuration = profileQuery.data?.health?.periodDuration ?? DEFAULT_PERIOD_DAYS;
  const reconcile = useReconcilePeriods();
  const isPending = reconcile.isPending;
  const isError = reconcile.isError;

  // The months to render: from a bit before the opened month (and at least
  // MONTHS_BACK before today) up to the current month — never a full future one.
  const openMonth = initialView ?? currentJ;
  const months = useMemo(() => {
    const prevOfOpen = shiftMonth(openMonth.year, openMonth.month, -1);
    const back = shiftMonth(currentJ.year, currentJ.month, -MONTHS_BACK);
    const startOrd = Math.min(monthOrd(prevOfOpen.year, prevOfOpen.month), monthOrd(back.year, back.month));
    const endOrd = monthOrd(currentJ.year, currentJ.month);
    const list: { year: number; month: number; weeks: (MonthCell | null)[][] }[] = [];
    for (let o = startOrd; o <= endOrd; o += 1) {
      const year = Math.floor(o / 12);
      const month = (o % 12) + 1;
      list.push({ year, month, weeks: monthMatrix(year, month, locale) });
    }
    return list;
     
  }, [openMonth.year, openMonth.month, currentJ.year, currentJ.month, locale]);

  // Every logged period day, so the editor opens showing what's already recorded.
  // An open (ongoing) period only stores its start, so it fills to the profile's
  // usual bleed length (never past today) — matching how the calendar renders it.
  const loggedDays = useMemo(() => {
    const set = new Set<string>();
    for (const p of historyQuery.data ?? []) {
      const rawEnd =
        p.period_end_date ?? toApiDate(addDays(fromApiDate(p.period_start_date), Math.max(0, periodDuration - 1)));
      const end = rawEnd > todayIso ? todayIso : rawEnd;
      for (const iso of isoRange(p.period_start_date, end)) set.add(iso);
    }
    return set;
  }, [historyQuery.data, periodDuration, todayIso]);

  // Re-seed the selection each time the editor opens (or the logged data changes).
  useEffect(() => {
    if (!open) return;
    setSelected(new Set(loggedDays));
    reconcile.reset();
    // Mutations are stable from react-query; only re-seed on open/data change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, loggedDays]);

  // Jump the scroll to the opened month so the user lands where the calendar was.
  useEffect(() => {
    if (!open || !host) return;
    monthRefs.current.get(monthKey(openMonth.year, openMonth.month))?.scrollIntoView({ block: 'start' });

  }, [open, host, openMonth.year, openMonth.month]);

  // Badge value per day: its 1-based position within its own contiguous run, so a
  // one-day gap restarts the numbering (day N *of that period*).
  const orderOf = useMemo(() => {
    const map = new Map<string, number>();
    const days = [...selected].sort();
    let n = 0;
    let prev: string | null = null;
    for (const iso of days) {
      const expected = prev ? toApiDate(addDays(fromApiDate(prev), 1)) : null;
      n = prev && iso === expected ? n + 1 : 1;
      map.set(iso, n);
      prev = iso;
    }
    return map;
  }, [selected]);

  // Tapping an empty day that starts a NEW run pre-fills the user's usual bleed
  // length forward from it (day 1 → `periodDuration` days), because a period is
  // a range, not a single day — the common case is then only trimming the tail.
  // Extending an existing run (tapping a day touching one) stays a single-day
  // toggle, so manual fine-tuning is never overwritten. The fill stops at today
  // and at any already-selected day, so it can't reach into the future or
  // silently swallow the gap before another period.
  const toggle = (iso: string) => {
    if (iso > todayIso) return; // never select the future
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(iso)) {
        next.delete(iso);
        return next;
      }
      const date = fromApiDate(iso);
      const touchesRun =
        prev.has(toApiDate(addDays(date, -1))) || prev.has(toApiDate(addDays(date, 1)));
      next.add(iso);
      if (touchesRun) return next;
      for (let i = 1; i < Math.max(1, periodDuration); i += 1) {
        const dayIso = toApiDate(addDays(date, i));
        if (dayIso > todayIso || prev.has(dayIso)) break;
        next.add(dayIso);
      }
      return next;
    });
  };

  const onSave = () => {
    if (isPending) return;
    reconcile.mutate(
      { segments: toSegments(selected), existing: historyQuery.data ?? [], periodDuration },
      {
        onSuccess: () => {
          onSaved?.();
          onClose();
        },
      },
    );
  };

  if (!open || !host) return null;

  const renderDay = (cell: MonthCell | null, ci: number) => {
    if (!cell) return <span key={ci} />;
    const iso = toApiDate(cell.date);
    const isFuture = iso > todayIso;
    const isSelected = selected.has(iso);
    const isToday = iso === todayIso;
    const order = orderOf.get(iso);
    const border = isSelected
      ? `1.5px dashed ${PERIOD_COLOR}`
      : isToday
        ? '1.5px dashed var(--ink)'
        : '1.5px solid var(--line)';
    return (
      <div key={ci} className="pde-daycell">
        <button
          onClick={() => toggle(iso)}
          disabled={isFuture}
          aria-pressed={isSelected}
          aria-label={format.number(cell.day)}
          style={{
            position: 'relative',
            width: '100%',
            maxWidth: 52,
            aspectRatio: '1 / 1',
            borderRadius: '50%',
            border,
            background: 'transparent',
            color: isFuture ? 'var(--line)' : isSelected ? PERIOD_COLOR : 'var(--ink)',
            fontFamily: 'inherit',
            fontSize: 'clamp(13px, 3.6vw, 15px)',
            fontWeight: 700,
            cursor: isFuture ? 'default' : 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontVariantNumeric: 'tabular-nums',
          }}
        >
          {format.number(cell.day)}
          {/* Order badge: filled + numbered when selected, a hollow ring otherwise
              (matches the mockup). Hidden for future days. */}
          {!isFuture && (
            <span
              aria-hidden
              style={{
                position: 'absolute',
                top: -1,
                insetInlineEnd: -1,
                width: 'clamp(15px, 4.4vw, 19px)',
                height: 'clamp(15px, 4.4vw, 19px)',
                borderRadius: '50%',
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 'clamp(9px, 2.6vw, 11px)',
                fontWeight: 800,
                color: isSelected ? 'var(--on-accent)' : 'transparent',
                background: isSelected ? PERIOD_COLOR : 'var(--surface)',
                border: isSelected ? 'none' : '1.5px solid var(--line)',
              }}
            >
              {order ?? ''}
            </span>
          )}
        </button>
      </div>
    );
  };

  return createPortal(
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="period-date-editor-title"
      className="pde"
    >
      {/* Pink header: close on the inline-start, centered title, weekday row. */}
      <div className="pde-head">
        <div className="pde-head-row">
          <button
            onClick={onClose}
            aria-label={t('dateEditor.close')}
            className="pde-close"
          >
            <Icon name="x" size={20} />
          </button>
          <div
            id="period-date-editor-title"
            className="pde-title"
          >
            {t('dateEditor.title')}
          </div>
        </div>

        <div className="cal-grid pde-weekdays">
          {weekdayKeys(locale).map((k) => (
            <span key={k} className="pde-weekday">
              {t(`editor.weekdays.${k}`)}
            </span>
          ))}
        </div>
      </div>

      {/* One continuous vertical scroll of months (newest at the bottom). */}
      <div className="pde-scroll">
        {months.map((m) => (
          <div
            key={monthKey(m.year, m.month)}
            ref={(el) => {
              monthRefs.current.set(monthKey(m.year, m.month), el);
            }}
            className="pde-month"
          >
            <div className="pde-month-t">
              {formatMonthLabel(m.year, m.month, locale)}
            </div>
            <div className="pde-weeks">
              {m.weeks.map((week, wi) => (
                <div key={wi} className="cal-grid pde-week">
                  {week.map((cell, ci) => renderDay(cell, ci))}
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* Footer: hint + Cancel / Save. */}
      <div className="pde-foot">
        <div className="pde-hint">
          {isError ? t('dateEditor.error') : t('dateEditor.hint', { days: periodDuration })}
        </div>
        <div className="pde-actions">
          <button
            onClick={onClose}
            disabled={isPending}
            style={{
              flex: 1,
              height: 52,
              borderRadius: 999,
              border: `1.5px solid ${PERIOD_COLOR}`,
              background: 'transparent',
              color: PERIOD_COLOR,
              fontFamily: 'inherit',
              fontSize: 'clamp(14px, 4vw, 16px)',
              fontWeight: 800,
              cursor: 'pointer',
            }}
          >
            {t('dateEditor.cancel')}
          </button>
          <button
            onClick={onSave}
            disabled={isPending}
            aria-busy={isPending}
            style={{
              flex: 1,
              height: 52,
              borderRadius: 999,
              border: 'none',
              background: 'var(--gradient-brand)',
              color: 'var(--on-accent)',
              fontFamily: 'inherit',
              fontSize: 'clamp(14px, 4vw, 16px)',
              fontWeight: 800,
              cursor: isPending ? 'default' : 'pointer',
              opacity: isPending ? 0.7 : 1,
            }}
          >
            {isPending ? t('dateEditor.saving') : t('dateEditor.save')}
          </button>
        </div>
      </div>
    </div>,
    host,
  );
}
