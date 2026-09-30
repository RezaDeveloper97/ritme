'use client';

import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';
import { createPortal } from 'react-dom';

import { useSaveHealthLog } from '@/entities/health-log';
import { useUserProfile } from '@/entities/user';
import { useDirection, type Locale } from '@/shared/i18n';
import {
  addDays,
  formatDayMonth,
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
import { AppSheet } from '@/shared/sheet';
import { Icon, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { usePeriodHistory, useReconcilePeriods } from '../api/mutations';
import {
  focusRun,
  isoRange,
  runStarts,
  seedSelection,
  toggleDay,
  toSegments,
  type PeriodSelection,
} from '../model/selection';

// Fallback bleed length when the profile hasn't recorded one — matches the
// calendar so an open period pre-fills to the same span the user sees there.
const DEFAULT_PERIOD_DAYS = 5;

// How far back the month arrows go; older history is rarely edited here and a
// bound keeps the arrow from walking into years with no data.
const MONTHS_BACK = 24;

// One month past the current one stays reachable: a run that started near the
// end of the month may continue into it and must stay trimmable.
const MONTHS_FORWARD = 1;

const monthOrd = (y: number, m: number) => y * 12 + (m - 1);

interface PeriodDateEditorProps {
  open: boolean;
  onClose: () => void;
  /** Month to open on (locale's calendar) — e.g. the month the calendar is showing. */
  initialView?: { year: number; month: number };
  /** Fired after a successful save, before the editor closes (to show recalculating). */
  onSaved?: () => void;
  /**
   * `start` = «پریودت شروع شد؟» with the spotting escape; `edit` = «ویرایش پریود».
   * Default: `start` when today isn't already a logged period day and no month
   * was requested, otherwise `edit`.
   */
  intent?: 'start' | 'edit';
}

/**
 * Edit period (B-N1-08, `nbl_/nbd_Cycle_EditPeriod`). A bottom sheet with one
 * month grid: every logged bleeding day is a solid red circle, the days the app
 * suggests from the usual period length are dashed, and each day is an
 * independent toggle. Tapping an empty day that starts a new run suggests the
 * rest of the usual length; tapping a suggested day trims the run from there.
 * On save, solid + suggested days are reconciled against the logged history in
 * one action, so moving, extending, splitting or removing periods all work.
 * «هنوز نه، فقط لکه‌بینی است» logs spotting for today instead of a period.
 *
 * Health data never leaves this component's calls (CLAUDE.md §11); all copy is
 * i18n'd and every offset is logical (§6, §12).
 */
export function PeriodDateEditor({ open, onClose, initialView, onSaved, intent }: PeriodDateEditorProps) {
  const t = useTranslations('logPeriod');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const format = useFormatter();
  const todayIso = toApiDate(today());
  const currentJ = toParts(today(), locale);

  // The sheet backdrop is `position: absolute; inset: 0`, so it must live
  // directly under the phone frame — mounted in place it would resolve against
  // whatever container the caller sits in. Portalling to `.app-shell` makes it
  // rise over the whole screen from any call site. Resolved once on mount.
  const [host, setHost] = useState<HTMLElement | null>(null);
  useEffect(() => {
    setHost(document.querySelector<HTMLElement>('.app-shell') ?? document.body);
  }, []);

  const historyQuery = usePeriodHistory();
  const profileQuery = useUserProfile();
  const periodDuration = Math.max(1, profileQuery.data?.health?.periodDuration ?? DEFAULT_PERIOD_DAYS);
  const reconcile = useReconcilePeriods();
  const spotting = useSaveHealthLog();

  const seeded = useMemo(
    () => seedSelection(historyQuery.data ?? [], periodDuration, todayIso),
    [historyQuery.data, periodDuration, todayIso],
  );
  const [sel, setSel] = useState<PeriodSelection>(seeded);
  const [view, setView] = useState(initialView ?? currentJ);

  // Re-seed whenever the editor opens (dropping unsaved edits) or the logged
  // data changes; nothing renders while closed, so seeding then is free.
  useEffect(() => {
    setSel(seeded);
    if (open) {
      reconcile.reset();
      spotting.reset();
      setView(initialView ?? toParts(today(), locale));
    }
    // Mutations are stable from react-query; only re-seed on open/data change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, seeded]);

  const mode = intent ?? (initialView || seeded.selected.has(todayIso) ? 'edit' : 'start');
  const union = useMemo(() => new Set([...sel.selected, ...sel.suggested]), [sel]);
  const starts = useMemo(() => runStarts(union), [union]);
  const run = useMemo(() => focusRun(union, todayIso), [union, todayIso]);

  const ord = monthOrd(view.year, view.month);
  const currentOrd = monthOrd(currentJ.year, currentJ.month);
  const canPrev = ord > currentOrd - MONTHS_BACK;
  const canNext = ord < currentOrd + MONTHS_FORWARD;
  const weeks = useMemo(() => monthMatrix(view.year, view.month, locale), [view.year, view.month, locale]);

  const isReachable = (iso: string) => iso <= todayIso || union.has(iso) || union.has(toApiDate(addDays(fromApiDate(iso), -1)));

  const toggle = (iso: string) => {
    if (!isReachable(iso)) return;
    setSel((prev) => toggleDay(prev, iso, periodDuration));
  };

  const busy = reconcile.isPending || spotting.isPending;

  const onSave = () => {
    if (busy) return;
    reconcile.mutate(
      { segments: toSegments(union), existing: historyQuery.data ?? [], periodDuration },
      {
        onSuccess: () => {
          onSaved?.();
          onClose();
        },
      },
    );
  };

  const onSpotting = () => {
    if (busy) return;
    spotting.mutate({ log_date: todayIso, spotting: true }, { onSuccess: onClose });
  };

  if (!host) return null;

  const dayLabel = (iso: string) => formatDayMonth(fromApiDate(iso), locale);
  const startLabel = run ? (run.start === todayIso ? t('dateEditor.today', { date: dayLabel(run.start) }) : dayLabel(run.start)) : t('dateEditor.none');
  const endIsEstimate = run ? run.end >= todayIso || isoRange(run.start, run.end).some((d) => sel.suggested.has(d)) : true;
  const endLabel = run ? dayLabel(run.end) : t('dateEditor.none');
  const monthLabel = formatMonthLabel(view.year, view.month, locale);

  const renderDay = (cell: MonthCell | null, ci: number) => {
    if (!cell) return <span key={ci} aria-hidden />;
    const iso = toApiDate(cell.date);
    const isSelected = sel.selected.has(iso);
    const isSuggested = !isSelected && sel.suggested.has(iso);
    const locked = !isReachable(iso);
    const dayText = format.number(cell.day);
    return (
      <span key={ci} className="epd-cell">
        <button
          type="button"
          onClick={() => toggle(iso)}
          disabled={locked || busy}
          aria-pressed={isSelected || isSuggested}
          aria-current={iso === todayIso ? 'date' : undefined}
          aria-label={isSuggested ? t('dateEditor.daySuggested', { day: dayLabel(iso) }) : dayLabel(iso)}
          className={clsx(
            'epd-day',
            isSelected && 'is-selected',
            isSuggested && 'is-suggested',
            locked && 'is-locked',
            iso === todayIso && 'is-today',
          )}
        >
          {dayText}
          {isSelected && starts.has(iso) ? (
            <span className="epd-check" aria-hidden>
              <Icon name="check" size={10} strokeWidth={3} />
            </span>
          ) : null}
        </button>
      </span>
    );
  };

  return createPortal(
    <AppSheet
      open={open}
      onClose={onClose}
      size="half"
      title={<span className="epd-title">{mode === 'start' ? t('dateEditor.titleStart') : t('dateEditor.titleEdit')}</span>}
      className="epd-sheet"
      footer={
        <div className="epd-foot">
          {reconcile.isError || spotting.isError ? (
            <p className="epd-error" role="alert">
              {spotting.isError ? t('dateEditor.spottingError') : t('dateEditor.error')}
            </p>
          ) : null}
          <PrimaryButton onClick={onSave} loading={reconcile.isPending} disabled={busy}>
            {reconcile.isPending ? t('dateEditor.saving') : t('dateEditor.save')}
          </PrimaryButton>
          {mode === 'start' ? (
            <SecondaryButton variant="text" onClick={onSpotting} loading={spotting.isPending} disabled={busy}>
              {t('dateEditor.spotting')}
            </SecondaryButton>
          ) : null}
        </div>
      }
    >
      <div className="epd-body">
        <p className="epd-sub">{t('dateEditor.subtitle')}</p>

        <div className="epd-tiles">
          <div className="epd-tile is-start">
            <span className="epd-tile-k">{t('dateEditor.start')}</span>
            <b className="epd-tile-v">{startLabel}</b>
          </div>
          <div className="epd-tile">
            <span className="epd-tile-k">{endIsEstimate ? t('dateEditor.estimatedEnd') : t('dateEditor.end')}</span>
            <b className="epd-tile-v">{endLabel}</b>
          </div>
        </div>

        <div className="epd-month">
          <button
            type="button"
            className="epd-nav"
            onClick={() => setView(shiftMonth(view.year, view.month, -1))}
            disabled={!canPrev}
            aria-label={t('dateEditor.prevMonth')}
          >
            <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={20} />
          </button>
          <span className="epd-month-t" aria-live="polite">
            {monthLabel}
          </span>
          <button
            type="button"
            className="epd-nav"
            onClick={() => setView(shiftMonth(view.year, view.month, 1))}
            disabled={!canNext}
            aria-label={t('dateEditor.nextMonth')}
          >
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={20} />
          </button>
        </div>

        <div className="epd-grid" role="group" aria-label={t('dateEditor.gridLabel', { month: monthLabel })}>
          {weekdayKeys(locale).map((k) => (
            <span key={k} className="epd-wd" aria-hidden>
              {t(`editor.weekdays.${k}`)}
            </span>
          ))}
          {weeks.flatMap((week, wi) => week.map((cell, ci) => renderDay(cell, wi * 7 + ci)))}
        </div>

        <p className="epd-hint">{t('dateEditor.suggestHint', { days: periodDuration })}</p>
      </div>
    </AppSheet>,
    host,
  );
}
