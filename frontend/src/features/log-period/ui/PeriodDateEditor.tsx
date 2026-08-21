'use client';

import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

import { useUserProfile } from '@/entities/user';
import { type Locale } from '@/shared/i18n';
import { AppSheet } from '@/shared/sheet';
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

import { usePeriodHistory, useReconcilePeriods, type PeriodSegment } from '../api/mutations';

// Fallback bleed length when the profile hasn't recorded one — matches the
// calendar so an open period pre-fills to the same span the user sees there.
const DEFAULT_PERIOD_DAYS = 5;

// How many months of history to show above the opened month, so the user can
// scroll back to older periods without a month-picker.
const MONTHS_BACK = 12;

// How many months past the current one to render, greyed out. Days there can't
// start a period, but a run reaching today may extend into them, so the month
// has to be on screen for the user to see and trim it.
const MONTHS_FORWARD = 1;

// Height of the sticky weekday ribbon the opened month must clear (mirrors
// `.pde-month { scroll-margin-top }` in globals.css).
const MONTH_SCROLL_OFFSET = 44;

// How long to keep re-trying the opening scroll (~1s at 60fps) while the sheet
// mounts and animates in.
const SCROLL_RETRY_FRAMES = 60;

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

/** Same set of days? Used to skip a no-op re-seed (and the re-render it costs). */
function sameDays(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false;
  for (const iso of a) if (!b.has(iso)) return false;
  return true;
}

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
 * "Edit Period Date" editor — the calendar's top-button entry. It enters the way
 * every other secondary screen in this app does: as an `AppSheet` rising from the
 * bottom over whatever the user was looking at (CLAUDE.md §4.1), `full` size
 * because a year of months is exactly the long-scroll case that size exists for.
 * Months
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

  // The sheet backdrop is `position: absolute; inset: 0`, so it must live
  // directly under the phone frame — mounted in place it would resolve against
  // whatever container the caller sits in (a card inside the cycle screen's
  // `.scroll`) and cover only that. Portalling to `.app-shell` makes it rise
  // over the whole screen from any call site.
  // Resolved once on mount, not on open: looking it up only when `open` flips
  // cost the first open a whole extra render round-trip (state set in an effect
  // → re-render → portal mounts → sheet starts its enter animation), which read
  // as a hitch the first time and never again. Nothing is rendered until `open`,
  // so holding the node early is free.
  const [host, setHost] = useState<HTMLElement | null>(null);
  useEffect(() => {
    setHost(document.querySelector<HTMLElement>('.app-shell') ?? document.body);
  }, []);

  const historyQuery = usePeriodHistory();
  const profileQuery = useUserProfile();
  const periodDuration = profileQuery.data?.health?.periodDuration ?? DEFAULT_PERIOD_DAYS;
  const reconcile = useReconcilePeriods();
  const isPending = reconcile.isPending;
  const isError = reconcile.isError;

  // The months to render: from a bit before the opened month (and at least
  // MONTHS_BACK before today) up to MONTHS_FORWARD past the current one — the
  // future months read as greyed-out context.
  const openMonth = initialView ?? currentJ;
  const months = useMemo(() => {
    const prevOfOpen = shiftMonth(openMonth.year, openMonth.month, -1);
    const back = shiftMonth(currentJ.year, currentJ.month, -MONTHS_BACK);
    const startOrd = Math.min(monthOrd(prevOfOpen.year, prevOfOpen.month), monthOrd(back.year, back.month));
    const endOrd = monthOrd(currentJ.year, currentJ.month) + MONTHS_FORWARD;
    const currentOrd = monthOrd(currentJ.year, currentJ.month);
    const list: { year: number; month: number; isFuture: boolean; weeks: (MonthCell | null)[][] }[] = [];
    for (let o = startOrd; o <= endOrd; o += 1) {
      const year = Math.floor(o / 12);
      const month = (o % 12) + 1;
      list.push({ year, month, isFuture: o > currentOrd, weeks: monthMatrix(year, month, locale) });
    }
    return list;

  }, [openMonth.year, openMonth.month, currentJ.year, currentJ.month, locale]);

  // Every logged period day, so the editor opens showing what's already recorded.
  // An open (ongoing) period only stores its start, so it fills to the profile's
  // usual bleed length (never past today) — matching how the calendar renders it.
  const loggedDays = useMemo(() => {
    const set = new Set<string>();
    for (const p of historyQuery.data ?? []) {
      // An open period fills to the usual bleed length even when that reaches
      // into the future — those days read as already logged (and stay editable).
      const end =
        p.period_end_date ?? toApiDate(addDays(fromApiDate(p.period_start_date), Math.max(0, periodDuration - 1)));
      for (const iso of isoRange(p.period_start_date, end)) set.add(iso);
    }
    return set;
  }, [historyQuery.data, periodDuration]);

  // Re-seed the selection whenever the editor opens or closes (dropping any
  // unsaved edits) and whenever the logged data changes. It runs while closed
  // too — nothing is rendered then, so the seeding is free and the grid is
  // already correct on open. Re-seeding to an identical set is skipped, so the
  // open commit isn't followed by a second render of ~500 day cells mid-animation.
  useEffect(() => {
    setSelected((prev) => (sameDays(prev, loggedDays) ? prev : new Set(loggedDays)));
    if (open) reconcile.reset();
    // Mutations are stable from react-query; only re-seed on open/data change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, loggedDays]);

  // Jump the scroll to the opened month, so the user lands on the days they
  // already logged instead of a year in the past.
  //
  // Retried across frames rather than done once: the sheet mounts its body a
  // commit *after* `open` flips (AppSheet keeps its own `mounted` state) and
  // then rises with a transform, so on the first frame the month is either not
  // in the DOM at all or sits in a panel that isn't laid out yet — and the jump
  // silently clamped to the top, which is exactly what shipped. Each attempt
  // verifies the scroll actually landed and stops as soon as it did.
  useEffect(() => {
    if (!open || !host) return;
    const key = monthKey(openMonth.year, openMonth.month);
    let frames = 0;
    let raf = 0;
    const jump = () => {
      const el = monthRefs.current.get(key);
      const box = el?.closest<HTMLElement>('.osheet-body') ?? null;
      if (el && box && box.scrollHeight > box.clientHeight) {
        const delta =
          el.getBoundingClientRect().top - box.getBoundingClientRect().top - MONTH_SCROLL_OFFSET;
        if (Math.abs(delta) > 1) box.scrollTop += delta;
        // Landed (or clamped at the end of the scroll, which is the same view).
        if (Math.abs(delta) <= 1 || box.scrollTop + box.clientHeight >= box.scrollHeight - 1) return;
      }
      if (frames++ < SCROLL_RETRY_FRAMES) raf = requestAnimationFrame(jump);
    };
    raf = requestAnimationFrame(jump);
    return () => cancelAnimationFrame(raf);
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

  // A future day is editable only when it continues a selected run — i.e. the
  // day before it is selected, or it already is (so it can be trimmed back off).
  // Everything further out stays inert grey context.
  const isReachableFuture = (iso: string) =>
    selected.has(iso) || selected.has(toApiDate(addDays(fromApiDate(iso), -1)));

  // Tapping an empty day that starts a NEW run pre-fills the user's usual bleed
  // length forward from it (day 1 → `periodDuration` days), because a period is
  // a range, not a single day — the common case is then only trimming the tail.
  // The fill may run past today (marking a period that started two days ago fills
  // the days still to come), which the backend now stores as a real end date.
  // Extending an existing run (tapping a day touching one) stays a single-day
  // toggle, so manual fine-tuning is never overwritten. The fill stops at any
  // already-selected day, so it can't silently swallow the gap before another
  // period.

  const toggle = (iso: string) => {
    // A future day can only ever be part of a run that already started (today or
    // earlier); it can never open one on its own.
    if (iso > todayIso && !isReachableFuture(iso)) return;
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
        if (prev.has(dayIso)) break;
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

  // `open` is handed to the sheet rather than short-circuiting here, so it can
  // play its exit animation on close instead of vanishing.
  if (!host) return null;

  const renderDay = (cell: MonthCell | null, ci: number) => {
    if (!cell) return <span key={ci} />;
    const iso = toApiDate(cell.date);
    const isSelected = selected.has(iso);
    const isFuture = iso > todayIso;
    // Grey-but-tappable: a future day continuing a selected run stays editable.
    const isLocked = isFuture && !isReachableFuture(iso);
    const isToday = iso === todayIso;
    const order = orderOf.get(iso);
    return (
      <div key={ci} className="pde-daycell">
        <button
          onClick={() => toggle(iso)}
          disabled={isLocked}
          aria-pressed={isSelected}
          aria-current={isToday ? 'date' : undefined}
          aria-label={format.number(cell.day)}
          className={clsx(
            'pde-day',
            isToday && 'is-today',
            isSelected && 'is-selected',
            isFuture && 'is-future',
            isLocked && 'is-locked',
          )}
        >
          {format.number(cell.day)}
          {/* Hidden for future days outside any run — nothing to count there. */}
          {!isLocked && (
            <span aria-hidden className="pde-badge">
              {order ?? ''}
            </span>
          )}
        </button>
      </div>
    );
  };

  return createPortal(
    <AppSheet
      open={open}
      onClose={onClose}
      size="full"
      title={t('dateEditor.title')}
      className="pde-sheet"
      footer={
        <div className="pde-foot">
          <div className="pde-hint">
            {isError ? t('dateEditor.error') : t('dateEditor.hint', { days: periodDuration })}
          </div>
          <div className="pde-actions">
            <button
              type="button"
              className="btn btn-ghost"
              onClick={onClose}
              disabled={isPending}
            >
              {t('dateEditor.cancel')}
            </button>
            <button
              type="button"
              className="btn btn-primary"
              onClick={onSave}
              disabled={isPending}
              aria-busy={isPending}
            >
              {isPending ? t('dateEditor.saving') : t('dateEditor.save')}
            </button>
          </div>
        </div>
      }
    >
      {/* Weekday row sticks to the top of the sheet's scroll, so the columns stay
          labelled however far back the user scrolls. */}
      <div className="cal-grid pde-weekdays">
        {weekdayKeys(locale).map((k) => (
          <span key={k} className="pde-weekday">
            {t(`editor.weekdays.${k}`)}
          </span>
        ))}
      </div>

      {/* One continuous vertical scroll of months (newest at the bottom). */}
      {months.map((m) => (
        <div
          key={monthKey(m.year, m.month)}
          ref={(el) => {
            monthRefs.current.set(monthKey(m.year, m.month), el);
          }}
          className={clsx('pde-month', m.isFuture && 'is-future')}
        >
          <div className="pde-month-t">{formatMonthLabel(m.year, m.month, locale)}</div>
          <div className="pde-weeks">
            {m.weeks.map((week, wi) => (
              <div key={wi} className="cal-grid pde-week">
                {week.map((cell, ci) => renderDay(cell, ci))}
              </div>
            ))}
          </div>
        </div>
      ))}
    </AppSheet>,
    host,
  );
}
