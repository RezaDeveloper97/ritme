'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { type Locale } from '@/shared/i18n';
import {
  diffInDays,
  formatJalaliMonthLabel,
  fromApiDate,
  jalaliMonthMatrix,
  shiftJalaliMonth,
  toApiDate,
  toJalali,
  today,
} from '@/shared/lib/date';
import { Icon, Sheet } from '@/shared/ui';

import { useDeletePeriod, useLogPeriodRange, useUpdatePeriod, type LoggedPeriod } from '../api/mutations';

const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

const PERIOD_COLOR = 'var(--brand)';
const PERIOD_BG = 'var(--pink-bg)';

/**
 * What just changed, so the caller can paint the calendar optimistically the
 * instant the sheet closes (before the backend recalculation lands).
 */
export interface PeriodSaveInfo {
  /** The freshly saved range (absent when the period was deleted). */
  saved?: { start: string; end: string };
  /** The range this edit replaced / removed, to un-paint. */
  cleared?: { start: string; end: string };
}

interface PeriodEditorProps {
  open: boolean;
  onClose: () => void;
  /** Gregorian `YYYY-MM-DD` to seed the view/selection with (the tapped day). */
  initialDateIso?: string;
  /** When set, the sheet edits this logged period (move / delete) instead of creating one. */
  editing?: LoggedPeriod | null;
  /** Fired after a successful save or delete, before the sheet closes. */
  onSaved?: (info: PeriodSaveInfo) => void;
}

interface Range {
  start: string;
  end: string;
}

/** Inclusive [start, end] from an anchor and the day being painted (ISO order). */
function rangeFrom(anchor: string, iso: string): Range {
  return anchor <= iso ? { start: anchor, end: iso } : { start: iso, end: anchor };
}

/** The visible range of a logged period; an open one paints just its start day. */
function rangeOf(period: LoggedPeriod): Range {
  return { start: period.period_start_date, end: period.period_end_date ?? period.period_start_date };
}

/**
 * "Mark period days" editor — a Jalali month the user paints (tap or drag) to
 * select the days they bled, then saves. In edit mode it opens pre-filled with
 * a logged period and can move or delete it; either way the backend re-anchors
 * the cycle and every prediction cascades. A range ending in the past is
 * recorded as finished (start+end); one ending today stays ongoing (start
 * only). Future days can't be bled, so they're disabled. Health data never
 * leaves this component's calls (CLAUDE.md §11); all copy is i18n'd and
 * RTL-safe (§6, §12).
 */
export function PeriodEditor({ open, onClose, initialDateIso, editing, onSaved }: PeriodEditorProps) {
  const t = useTranslations('logPeriod');
  const locale = useLocale() as Locale;
  const format = useFormatter();

  const [view, setView] = useState(() => {
    const j = toJalali(today());
    return { year: j.year, month: j.month };
  });
  const [range, setRange] = useState<Range | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const anchorRef = useRef<string | null>(null);
  const painting = useRef(false);

  const log = useLogPeriodRange();
  const update = useUpdatePeriod();
  const remove = useDeletePeriod();
  const todayIso = toApiDate(today());
  const isPending = log.isPending || update.isPending || remove.isPending;
  const isError = log.isError || update.isError || remove.isError;

  // Reset the view + selection each time the sheet opens: from the period being
  // edited, else from the tapped day (when it's not in the future).
  useEffect(() => {
    if (!open) return;
    const seedRange = editing
      ? rangeOf(editing)
      : initialDateIso && initialDateIso <= todayIso
        ? { start: initialDateIso, end: initialDateIso }
        : null;
    const j = toJalali(seedRange ? fromApiDate(seedRange.start) : today());
    setView({ year: j.year, month: j.month });
    setRange(seedRange);
    setConfirmingDelete(false);
    anchorRef.current = seedRange?.start ?? null;
    log.reset();
    update.reset();
    remove.reset();
    // Mutations are stable from react-query; only re-seed on open/target change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialDateIso, editing, todayIso]);

  const weeks = jalaliMonthMatrix(view.year, view.month);
  const goMonth = (delta: number) => setView((v) => shiftJalaliMonth(v.year, v.month, delta));

  const paintTo = (iso: string) => {
    if (iso > todayIso) return; // never select the future
    const anchor = anchorRef.current ?? iso;
    setRange(rangeFrom(anchor, iso));
  };

  const onDayDown = (iso: string) => {
    if (iso > todayIso) return;
    painting.current = true;
    anchorRef.current = iso;
    setRange({ start: iso, end: iso });
  };

  const onGridPointerMove = (e: React.PointerEvent) => {
    if (!painting.current) return;
    const el = document.elementFromPoint(e.clientX, e.clientY);
    const iso = (el?.closest('[data-day-iso]') as HTMLElement | null)?.dataset.dayIso;
    if (iso) paintTo(iso);
  };

  const endPaint = () => {
    painting.current = false;
  };

  const selectedCount = range
    ? diffInDays(fromApiDate(range.end), fromApiDate(range.start)) + 1
    : 0;

  const finish = (info: PeriodSaveInfo) => {
    onSaved?.(info);
    onClose();
  };

  const onSave = () => {
    if (!range || isPending) return;
    // A range ending before today is finished → send its end; one ending today
    // stays open so the period is still "ongoing".
    const finished = diffInDays(fromApiDate(range.end), today()) < 0;
    const info: PeriodSaveInfo = {
      saved: { ...range },
      cleared: editing ? rangeOf(editing) : undefined,
    };
    if (editing) {
      update.mutate(
        { id: editing.id, start: range.start, end: finished ? range.end : null },
        { onSuccess: () => finish(info) },
      );
    } else {
      log.mutate(
        { start: range.start, end: finished ? range.end : undefined },
        { onSuccess: () => finish(info) },
      );
    }
  };

  const onDelete = () => {
    if (!editing || isPending) return;
    remove.mutate(
      { id: editing.id },
      { onSuccess: () => finish({ cleared: rangeOf(editing) }) },
    );
  };

  return (
    <Sheet open={open} onClose={onClose} labelledBy="period-editor-title">
      <div className="ped-head">
        <div className="text-start">
          <div id="period-editor-title" className="ped-head-t">
            {editing ? t('editor.editTitle') : t('editor.title')}
          </div>
          <p className="sub ped-head-s">{t('editor.hint')}</p>
        </div>
        <button className="iconbtn" onClick={onClose} aria-label={t('editor.close')}>
          <Icon name="x" size={20} />
        </button>
      </div>

      {/* Month navigation. First flex child sits on the RIGHT in RTL = prev. */}
      <div className="ped-nav">
        <button className="iconbtn" onClick={() => goMonth(locale === 'fa' ? -1 : 1)} aria-label={locale === 'fa' ? 'ماه قبل' : 'Next month'}>
          <Icon name={locale === 'fa' ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <span className="ped-month">
          {formatJalaliMonthLabel(view.year, view.month, locale)}
        </span>
        <button className="iconbtn" onClick={() => goMonth(locale === 'fa' ? 1 : -1)} aria-label={locale === 'fa' ? 'ماه بعد' : 'Previous month'}>
          <Icon name={locale === 'fa' ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      <div className="cal-grid ped-weekdays">
        {WEEKDAY_KEYS.map((k) => (
          <span key={k} className="ped-weekday">
            {t(`editor.weekdays.${k}`)}
          </span>
        ))}
      </div>

      <div
        onPointerMove={onGridPointerMove}
        onPointerUp={endPaint}
        onPointerLeave={endPaint}
        className="ped-weeks"
      >
        {weeks.map((week, wi) => (
          <div key={wi} className="cal-grid">
            {week.map((cell, ci) => {
              if (!cell) return <span key={ci} />;
              const iso = toApiDate(cell.date);
              const isFuture = iso > todayIso;
              const selected = !!range && iso >= range.start && iso <= range.end;
              return (
                <button
                  key={ci}
                  data-day-iso={isFuture ? undefined : iso}
                  onPointerDown={() => onDayDown(iso)}
                  disabled={isFuture}
                  style={{
                    height: 40,
                    border: '2px solid transparent',
                    borderRadius: 14,
                    background: selected ? PERIOD_BG : 'transparent',
                    color: isFuture ? 'var(--line)' : selected ? PERIOD_COLOR : 'var(--ink)',
                    fontFamily: 'inherit',
                    fontSize: 14,
                    fontWeight: 700,
                    cursor: isFuture ? 'default' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    boxShadow: selected ? `inset 0 0 0 1.5px ${PERIOD_COLOR}` : undefined,
                    fontVariantNumeric: 'tabular-nums',
                    touchAction: 'none',
                  }}
                >
                  {format.number(cell.day)}
                </button>
              );
            })}
          </div>
        ))}
      </div>

      <div className="ped-summary">
        {range ? t('editor.count', { n: selectedCount }) : t('editor.selectPrompt')}
      </div>

      <button
        className="btn btn-primary ped-save"
        onClick={onSave}
        disabled={!range || isPending}
        aria-busy={isPending}
      >
        {isPending ? t('editor.saving') : editing ? t('editor.saveChanges') : t('editor.save')}
      </button>

      {/* Delete flow: one tap arms it, a second (clearly destructive) confirms. */}
      {editing && !confirmingDelete && (
        <button
          onClick={() => setConfirmingDelete(true)}
          disabled={isPending}
          className="ped-del-link"
        >
          {t('editor.delete')}
        </button>
      )}
      {editing && confirmingDelete && (
        <div className="ped-actions">
          <button
            className="btn ped-confirm"
            onClick={onDelete}
            disabled={isPending}
          >
            {remove.isPending ? t('editor.deleting') : t('editor.confirmDelete')}
          </button>
          <button
            className="btn ped-cancel"
            onClick={() => setConfirmingDelete(false)}
            disabled={isPending}
          >
            {t('editor.cancel')}
          </button>
        </div>
      )}

      {isError && (
        <p role="alert" className="ped-note is-error">
          {t('editor.error')}
        </p>
      )}
    </Sheet>
  );
}
