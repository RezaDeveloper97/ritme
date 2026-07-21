'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import { type Locale } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatJalaliMonthLabel,
  fromApiDate,
  jalaliMonthMatrix,
  shiftJalaliMonth,
  toApiDate,
  toJalali,
  today,
} from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { useDeletePeriod, useLogPeriodRange, useUpdatePeriod, type LoggedPeriod } from '../api/mutations';
import { type PeriodSaveInfo } from './PeriodEditor';

const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

const PERIOD_COLOR = '#E91E63';

interface PeriodDateEditorProps {
  open: boolean;
  onClose: () => void;
  /** Gregorian `YYYY-MM-DD` to seed the visible month with (usually the selected day). */
  initialDateIso?: string;
  /** When set, the editor opens pre-filled with this logged period's days. */
  editing?: LoggedPeriod | null;
  /** Fired after a successful save/delete, before the editor closes. */
  onSaved?: (info: PeriodSaveInfo) => void;
}

/** The visible range of a logged period; an open one covers just its start day. */
function rangeOf(period: LoggedPeriod): { start: string; end: string } {
  return { start: period.period_start_date, end: period.period_end_date ?? period.period_start_date };
}

/** Every ISO day in the inclusive [start, end] range. */
function isoRange(start: string, end: string): string[] {
  const out: string[] = [];
  const first = fromApiDate(start);
  const count = diffInDays(fromApiDate(end), first) + 1;
  for (let i = 0; i < count; i += 1) out.push(toApiDate(addDays(first, i)));
  return out;
}

/**
 * Full-screen "Edit Period Date" editor — the calendar's top-button entry. Each
 * day is an independent toggle: tapping a selected day clears it, tapping an
 * empty one selects it (the numbered badge shows its order). A menstrual period
 * is contiguous bleeding, and the backend stores it as a single start/end range,
 * so on save the selection persists as the range spanning its earliest→latest
 * day; clearing everything (in edit mode) deletes the period. Future days can't
 * be bled, so they're disabled. Health data never leaves this component's calls
 * (CLAUDE.md §11); all copy is i18n'd and RTL-safe (§6, §12).
 */
export function PeriodDateEditor({ open, onClose, initialDateIso, editing, onSaved }: PeriodDateEditorProps) {
  const t = useTranslations('logPeriod');
  const locale = useLocale() as Locale;
  const format = useFormatter();
  const isRtl = locale === 'fa';
  const todayIso = toApiDate(today());

  const [view, setView] = useState(() => {
    const j = toJalali(today());
    return { year: j.year, month: j.month };
  });
  const [selected, setSelected] = useState<Set<string>>(() => new Set());

  const log = useLogPeriodRange();
  const update = useUpdatePeriod();
  const remove = useDeletePeriod();
  const isPending = log.isPending || update.isPending || remove.isPending;
  const isError = log.isError || update.isError || remove.isError;

  // Reset the view + selection each time the editor opens: from the period being
  // edited, else from the tapped day (when it's not in the future).
  useEffect(() => {
    if (!open) return;
    const seed = editing
      ? isoRange(editing.period_start_date, editing.period_end_date ?? editing.period_start_date)
      : [];
    const anchor = editing?.period_start_date ?? (initialDateIso && initialDateIso <= todayIso ? initialDateIso : undefined);
    const j = toJalali(anchor ? fromApiDate(anchor) : today());
    setView({ year: j.year, month: j.month });
    setSelected(new Set(seed));
    log.reset();
    update.reset();
    remove.reset();
    // Mutations are stable from react-query; only re-seed on open/target change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialDateIso, editing, todayIso]);

  const weeks = useMemo(() => jalaliMonthMatrix(view.year, view.month), [view]);
  const goMonth = (delta: number) => setView((v) => shiftJalaliMonth(v.year, v.month, delta));

  // Order badge for each selected day: its position in the full chronological
  // selection (1-based), so numbering stays stable across month navigation.
  const orderOf = useMemo(() => {
    const map = new Map<string, number>();
    [...selected].sort().forEach((iso, i) => map.set(iso, i + 1));
    return map;
  }, [selected]);

  const toggle = (iso: string) => {
    if (iso > todayIso) return; // never select the future
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(iso)) next.delete(iso);
      else next.add(iso);
      return next;
    });
  };

  const finish = (info: PeriodSaveInfo) => {
    onSaved?.(info);
    onClose();
  };

  const onSave = () => {
    if (isPending) return;
    const isos = [...selected].sort();
    // Nothing selected: in edit mode this removes the period; otherwise there's
    // simply nothing to save.
    if (isos.length === 0) {
      if (editing) remove.mutate({ id: editing.id }, { onSuccess: () => finish({ cleared: rangeOf(editing) }) });
      else onClose();
      return;
    }
    const start = isos[0]!;
    const end = isos[isos.length - 1]!;
    // A range ending before today is finished → send its end; one ending today
    // stays open so the period is still "ongoing".
    const finished = diffInDays(fromApiDate(end), today()) < 0;
    const info: PeriodSaveInfo = { saved: { start, end }, cleared: editing ? rangeOf(editing) : undefined };
    if (editing) {
      update.mutate({ id: editing.id, start, end: finished ? end : null }, { onSuccess: () => finish(info) });
    } else {
      log.mutate({ start, end: finished ? end : undefined }, { onSuccess: () => finish(info) });
    }
  };

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="period-date-editor-title"
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 60,
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--surface)',
      }}
    >
      {/* Pink header: close on the inline-start, centered title, weekday row. */}
      <div
        style={{
          background: 'linear-gradient(180deg, var(--brand-deep), var(--brand))',
          color: '#fff',
          padding: '14px 16px 0',
          flexShrink: 0,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, minHeight: 44 }}>
          <button
            onClick={onClose}
            aria-label={t('dateEditor.close')}
            style={{
              width: 40,
              height: 40,
              borderRadius: '50%',
              border: 'none',
              background: 'rgba(0,0,0,.18)',
              color: '#fff',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              flexShrink: 0,
            }}
          >
            <Icon name="x" size={20} />
          </button>
          <div
            id="period-date-editor-title"
            style={{ flex: 1, textAlign: 'center', fontSize: 18, fontWeight: 800, marginInlineEnd: 40 }}
          >
            {t('dateEditor.title')}
          </div>
        </div>

        <div className="cal-grid" style={{ padding: '12px 0 10px' }}>
          {WEEKDAY_KEYS.map((k) => (
            <span key={k} style={{ fontSize: 11, fontWeight: 700, textAlign: 'center', opacity: 0.8 }}>
              {t(`editor.weekdays.${k}`)}
            </span>
          ))}
        </div>
      </div>

      {/* Scrollable month body. */}
      <div style={{ flex: 1, overflowY: 'auto', background: 'var(--surface)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '18px 16px 14px',
            background: 'var(--line)',
          }}
        >
          {/* First flex child sits on the RIGHT in RTL = previous month. */}
          <button
            className="iconbtn"
            onClick={() => goMonth(isRtl ? -1 : 1)}
            aria-label={isRtl ? 'ماه قبل' : 'Next month'}
          >
            <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
          </button>
          <span style={{ fontSize: 17, fontWeight: 800, color: 'var(--ink)' }}>
            {formatJalaliMonthLabel(view.year, view.month, locale)}
          </span>
          <button
            className="iconbtn"
            onClick={() => goMonth(isRtl ? 1 : -1)}
            aria-label={isRtl ? 'ماه بعد' : 'Previous month'}
          >
            <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
          </button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: '18px 14px' }}>
          {weeks.map((week, wi) => (
            <div key={wi} className="cal-grid" style={{ gap: '12px 2px' }}>
              {week.map((cell, ci) => {
                if (!cell) return <span key={ci} />;
                const iso = toApiDate(cell.date);
                const isFuture = iso > todayIso;
                const isSelected = selected.has(iso);
                const order = orderOf.get(iso);
                return (
                  <div key={ci} style={{ display: 'flex', justifyContent: 'center' }}>
                    <button
                      onClick={() => toggle(iso)}
                      disabled={isFuture}
                      aria-pressed={isSelected}
                      aria-label={format.number(cell.day)}
                      style={{
                        position: 'relative',
                        width: 52,
                        height: 52,
                        borderRadius: '50%',
                        border: isSelected ? `1.5px dashed ${PERIOD_COLOR}` : '1.5px solid var(--line)',
                        background: 'transparent',
                        color: isFuture ? 'var(--line)' : isSelected ? PERIOD_COLOR : 'var(--ink)',
                        fontFamily: 'inherit',
                        fontSize: 15,
                        fontWeight: 700,
                        cursor: isFuture ? 'default' : 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontVariantNumeric: 'tabular-nums',
                      }}
                    >
                      {format.number(cell.day)}
                      {/* Order badge: filled + numbered when selected, a hollow ring
                          otherwise (matches the mockup). Hidden for future days. */}
                      {!isFuture && (
                        <span
                          aria-hidden
                          style={{
                            position: 'absolute',
                            top: -2,
                            insetInlineEnd: -2,
                            width: 20,
                            height: 20,
                            borderRadius: '50%',
                            display: 'inline-flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: 11,
                            fontWeight: 800,
                            color: isSelected ? '#fff' : 'transparent',
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
              })}
            </div>
          ))}
        </div>
      </div>

      {/* Footer: hint + Cancel / Save. */}
      <div style={{ flexShrink: 0, background: 'var(--surface)', borderTop: '1px solid var(--line)' }}>
        <div
          style={{
            background: 'var(--surface-2)',
            color: 'var(--brand)',
            textAlign: 'center',
            fontSize: 13,
            fontWeight: 700,
            padding: '12px 16px',
          }}
        >
          {isError ? t('dateEditor.error') : t('dateEditor.hint')}
        </div>
        <div style={{ display: 'flex', gap: 12, padding: '14px 16px', paddingBottom: 'calc(14px + env(safe-area-inset-bottom))' }}>
          <button
            onClick={onClose}
            disabled={isPending}
            style={{
              flex: 1,
              height: 54,
              borderRadius: 999,
              border: `1.5px solid ${PERIOD_COLOR}`,
              background: 'transparent',
              color: PERIOD_COLOR,
              fontFamily: 'inherit',
              fontSize: 16,
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
              height: 54,
              borderRadius: 999,
              border: 'none',
              background: 'var(--green)',
              color: '#fff',
              fontFamily: 'inherit',
              fontSize: 16,
              fontWeight: 800,
              cursor: isPending ? 'default' : 'pointer',
              opacity: isPending ? 0.7 : 1,
            }}
          >
            {isPending ? t('dateEditor.saving') : t('dateEditor.save')}
          </button>
        </div>
      </div>
    </div>
  );
}
