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

import { usePeriodHistory, useReconcilePeriods, type PeriodSegment } from '../api/mutations';

const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

const PERIOD_COLOR = '#E91E63';

interface PeriodDateEditorProps {
  open: boolean;
  onClose: () => void;
  /** Gregorian `YYYY-MM-DD` to seed the visible month with (usually the selected day). */
  initialDateIso?: string;
  /** Fired after a successful save, before the editor closes (to show recalculating). */
  onSaved?: () => void;
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
 * Full-screen "Edit Period Date" editor — the calendar's top-button entry. Every
 * logged period is pre-selected; each day is an independent toggle (tap a filled
 * day to clear it, an empty one to select it, its badge shows its order). On save
 * the whole selection is reconciled against the logged history in one action, so
 * moving, extending, splitting, merging or removing periods all just work. A
 * menstrual period is contiguous bleeding, so contiguous runs each persist as one
 * period (backend stores start/end). Future days can't be bled, so they're
 * disabled. Health data never leaves this component's calls (CLAUDE.md §11); all
 * copy is i18n'd and RTL-safe (§6, §12).
 */
export function PeriodDateEditor({ open, onClose, initialDateIso, onSaved }: PeriodDateEditorProps) {
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

  const historyQuery = usePeriodHistory();
  const reconcile = useReconcilePeriods();
  const isPending = reconcile.isPending;
  const isError = reconcile.isError;

  // Every logged period day, so the editor opens showing what's already recorded.
  const loggedDays = useMemo(() => {
    const set = new Set<string>();
    for (const p of historyQuery.data ?? []) {
      for (const iso of isoRange(p.period_start_date, p.period_end_date ?? p.period_start_date)) set.add(iso);
    }
    return set;
  }, [historyQuery.data]);

  // Reset the view + selection each time the editor opens: pre-fill the logged
  // periods and jump to the tapped day's month (when it isn't in the future).
  useEffect(() => {
    if (!open) return;
    const anchor = initialDateIso && initialDateIso <= todayIso ? initialDateIso : undefined;
    const j = toJalali(anchor ? fromApiDate(anchor) : today());
    setView({ year: j.year, month: j.month });
    setSelected(new Set(loggedDays));
    reconcile.reset();
    // Mutations are stable from react-query; only re-seed on open/data change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialDateIso, todayIso, loggedDays]);

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

  const onSave = () => {
    if (isPending) return;
    reconcile.mutate(
      { segments: toSegments(selected), existing: historyQuery.data ?? [] },
      {
        onSuccess: () => {
          onSaved?.();
          onClose();
        },
      },
    );
  };

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="period-date-editor-title"
      style={{
        position: 'absolute',
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
          padding: '12px clamp(12px, 4vw, 18px) 0',
          flexShrink: 0,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, minHeight: 40 }}>
          <button
            onClick={onClose}
            aria-label={t('dateEditor.close')}
            style={{
              width: 38,
              height: 38,
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
            style={{ flex: 1, textAlign: 'center', fontSize: 'clamp(15px, 4.4vw, 18px)', fontWeight: 800, marginInlineEnd: 38 }}
          >
            {t('dateEditor.title')}
          </div>
        </div>

        <div className="cal-grid" style={{ padding: '10px 0 9px' }}>
          {WEEKDAY_KEYS.map((k) => (
            <span key={k} style={{ fontSize: 'clamp(9px, 2.6vw, 11px)', fontWeight: 700, textAlign: 'center', opacity: 0.85 }}>
              {t(`editor.weekdays.${k}`)}
            </span>
          ))}
        </div>
      </div>

      {/* Scrollable month body. */}
      <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', background: 'var(--surface)' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '14px clamp(12px, 4vw, 18px)',
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
          <span style={{ fontSize: 'clamp(15px, 4.2vw, 17px)', fontWeight: 800, color: 'var(--ink)' }}>
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

        <div style={{ display: 'flex', flexDirection: 'column', gap: 'clamp(8px, 2.4vw, 14px)', padding: 'clamp(12px, 4vw, 20px) clamp(8px, 3vw, 16px)' }}>
          {weeks.map((week, wi) => (
            <div key={wi} className="cal-grid" style={{ gap: 'clamp(6px, 2vw, 12px) 2px' }}>
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
                        width: '100%',
                        maxWidth: 52,
                        aspectRatio: '1 / 1',
                        borderRadius: '50%',
                        border: isSelected ? `1.5px dashed ${PERIOD_COLOR}` : '1.5px solid var(--line)',
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
                      {/* Order badge: filled + numbered when selected, a hollow ring
                          otherwise (matches the mockup). Hidden for future days. */}
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
            fontSize: 'clamp(12px, 3.2vw, 13px)',
            fontWeight: 700,
            padding: '11px 16px',
          }}
        >
          {isError ? t('dateEditor.error') : t('dateEditor.hint')}
        </div>
        <div style={{ display: 'flex', gap: 'clamp(8px, 3vw, 12px)', padding: '12px clamp(12px, 4vw, 16px)', paddingBottom: 'calc(12px + env(safe-area-inset-bottom))' }}>
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
              background: 'var(--green)',
              color: '#fff',
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
    </div>
  );
}
