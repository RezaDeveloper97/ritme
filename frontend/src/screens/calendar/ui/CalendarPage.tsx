'use client';

import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { cycleDayInfo, type CyclePhase } from '@/entities/cycle';
import type { Locale } from '@/shared/i18n';
import {
  addDays,
  diffInDays,
  formatJalaliDayMonth,
  formatJalaliMonthLabel,
  jalaliMonthMatrix,
  shiftJalaliMonth,
  today,
  todayJalali,
  type JalaliMonthCell,
} from '@/shared/lib/date';
import { Icon } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

// Demo cycle so the calendar renders meaningful phases until it is wired to the
// backend (cycle group in §8.1). Cycle started 3 days ago → «today» is cycle
// day 4, matching the Figma reference. Real config will come from the API.
const DEMO_CYCLE = {
  startDate: addDays(today(), -3),
  cycleLength: 28,
  periodLength: 5,
};

const WEEKDAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

// Only the "active" phases get a colored cell; follicular/luteal read as neutral.
const PHASE_STYLE: Partial<Record<CyclePhase, { bg: string; color: string }>> = {
  period: { bg: '#FCE7F3', color: '#E91E63' },
  fertile: { bg: '#FEF3C6', color: '#F5A623' },
  ovulation: { bg: '#E7F8EF', color: '#22B07D' },
};

const MOOD_KEYS = ['happy', 'calm', 'energetic', 'tired', 'sad', 'irritable'] as const;

const isSameDay = (a: Date, b: Date) => diffInDays(a, b) === 0;

type T = ReturnType<typeof useTranslations>;

// ── Month navigation bar ───────────────────────────────────────
interface MonthNavProps {
  label: string;
  isRtl: boolean;
  onPrev: () => void;
  onNext: () => void;
  fullMonth: boolean;
  onToggle: () => void;
  toggleLabel: string;
}

function MonthNav({ label, isRtl, onPrev, onNext, fullMonth, onToggle, toggleLabel }: MonthNavProps) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 4px 14px' }}>
      {/* First child → RIGHT in RTL = previous month */}
      <button className="iconbtn" onClick={isRtl ? onPrev : onNext} aria-label={isRtl ? 'ماه قبل' : 'Next month'}>
        <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
      </button>

      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <span style={{ fontSize: 16, fontWeight: 800, color: 'var(--ink)' }}>{label}</span>
        <button
          onClick={onToggle}
          className={`chip${fullMonth ? ' on' : ''}`}
          style={{ padding: '5px 12px', fontSize: 12 }}
        >
          <Icon name={fullMonth ? 'grid' : 'calendar'} size={13} />
          {toggleLabel}
        </button>
      </div>

      {/* Last child → LEFT in RTL = next month */}
      <button className="iconbtn" onClick={isRtl ? onNext : onPrev} aria-label={isRtl ? 'ماه بعد' : 'Previous month'}>
        <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
      </button>
    </div>
  );
}

// ── One day cell ───────────────────────────────────────────────
interface DayCellProps {
  cell: JalaliMonthCell | null;
  selectedDate: Date;
  onSelect: (date: Date) => void;
  dayNumber: string;
}

function DayCell({ cell, selectedDate, onSelect, dayNumber }: DayCellProps) {
  if (!cell) return <span />;

  const info = cycleDayInfo(cell.date, DEMO_CYCLE);
  const phaseStyle = PHASE_STYLE[info.phase];
  const selected = isSameDay(cell.date, selectedDate);
  const isToday = isSameDay(cell.date, today());
  // A period day still in the future is a prediction → hollow ring, not fill.
  const isPredicted = info.phase === 'period' && diffInDays(cell.date, today()) > 0;

  return (
    <button
      onClick={() => onSelect(cell.date)}
      style={{
        position: 'relative',
        height: 40,
        border: selected ? '2px solid var(--brand)' : '2px solid transparent',
        borderRadius: 14,
        background: phaseStyle && !isPredicted ? phaseStyle.bg : 'transparent',
        color: phaseStyle ? phaseStyle.color : 'var(--ink)',
        fontFamily: 'inherit',
        fontSize: 14,
        fontWeight: 700,
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxShadow: isPredicted && phaseStyle ? `inset 0 0 0 1.5px ${phaseStyle.color}` : undefined,
        fontVariantNumeric: 'tabular-nums',
      }}
    >
      {dayNumber}
      {isToday && (
        <span
          style={{
            position: 'absolute',
            bottom: 4,
            width: 4,
            height: 4,
            borderRadius: '50%',
            background: phaseStyle ? phaseStyle.color : 'var(--brand)',
          }}
        />
      )}
    </button>
  );
}

// ── Legend ─────────────────────────────────────────────────────
function Legend({ t }: { t: T }) {
  const items = [
    { key: 'period', color: '#E91E63' },
    { key: 'fertile', color: '#F5A623' },
    { key: 'ovulation', color: '#22B07D' },
  ] as const;
  return (
    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', padding: '4px 4px 0' }}>
      {items.map((it) => (
        <span key={it.key} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <span style={{ width: 10, height: 10, borderRadius: '50%', background: it.color }} />
          <span style={{ fontSize: 11.5, fontWeight: 600, color: 'var(--muted)' }}>{t(`legend.${it.key}`)}</span>
        </span>
      ))}
    </div>
  );
}

// ── Selected-day detail card ───────────────────────────────────
interface DayDetailProps {
  t: T;
  locale: Locale;
  selectedDate: Date;
  onLog: () => void;
}

function DayDetail({ t, locale, selectedDate, onLog }: DayDetailProps) {
  const info = cycleDayInfo(selectedDate, DEMO_CYCLE);
  const phaseStyle = PHASE_STYLE[info.phase] ?? { bg: '#F3F0FF', color: '#7C7CF0' };
  const isToday = isSameDay(selectedDate, today());

  return (
    <div className="card" style={{ padding: '16px 14px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span className="dot" style={{ width: 42, height: 42, background: phaseStyle.bg, color: phaseStyle.color }}>
            <Icon name="drop" size={20} fill="currentColor" strokeWidth={0} />
          </span>
          <div style={{ textAlign: 'start' }}>
            <div style={{ fontSize: 15, fontWeight: 800, color: 'var(--ink)' }}>
              {formatJalaliDayMonth(selectedDate, locale)}
            </div>
            <div style={{ fontSize: 12, color: 'var(--muted)', marginTop: 2 }}>
              {t('day.cycleDay', { n: info.cycleDay })}
            </div>
          </div>
        </div>
        {isToday && (
          <span style={{ fontSize: 11, fontWeight: 700, color: 'var(--brand)', background: '#FFF1F7', borderRadius: 20, padding: '4px 12px' }}>
            {t('today')}
          </span>
        )}
      </div>

      <div style={{ display: 'flex', gap: 10 }}>
        <div style={{ flex: 1, background: phaseStyle.bg, borderRadius: 12, padding: '10px 12px', textAlign: 'start' }}>
          <div style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 600 }}>{t('phaseLabel')}</div>
          <div style={{ fontSize: 13, fontWeight: 800, color: phaseStyle.color, marginTop: 3 }}>{t(`phase.${info.phase}`)}</div>
        </div>
        <div style={{ flex: 1, background: 'var(--line)', borderRadius: 12, padding: '10px 12px', textAlign: 'start' }}>
          <div style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 600 }}>{t('day.chanceLabel')}</div>
          <div style={{ fontSize: 13, fontWeight: 800, color: 'var(--steel)', marginTop: 3 }}>{t(`chance.${info.pregnancyChance}`)}</div>
        </div>
      </div>

      <button className="btn btn-ghost" onClick={onLog} style={{ height: 46, borderRadius: 14, gap: 8, fontSize: 14, marginTop: 14 }}>
        <Icon name="plus" size={16} /> {t('day.logMood')}
      </button>
    </div>
  );
}

// ── Smart tip (educational, non-diagnostic — §11) ──────────────
function SmartTip({ t }: { t: T }) {
  return (
    <div className="card" style={{ padding: '14px 12px' }}>
      <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 10 }}>
        {t('smartTip.title')}
      </div>
      <p className="sub" style={{ textAlign: 'start', margin: '0 2px 14px' }}>{t('smartTip.body')}</p>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, background: 'linear-gradient(90deg,#FFF0F7,#F3F0FF)', borderRadius: 12, padding: 12 }}>
        <span style={{ color: 'var(--ritme-pink)' }}>
          <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
        </span>
        <span style={{ flex: 1, textAlign: 'start', fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>
          {t('smartTip.quote')}
        </span>
      </div>
    </div>
  );
}

// ── Log-mood bottom sheet ──────────────────────────────────────
interface MoodSheetProps {
  t: T;
  locale: Locale;
  selectedDate: Date;
  onClose: () => void;
}

function MoodSheet({ t, locale, selectedDate, onClose }: MoodSheetProps) {
  const [mood, setMood] = useState<string | null>(null);

  return (
    <div className="sheet-backdrop" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-grip" />
        <div style={{ textAlign: 'start', marginBottom: 4 }}>
          <div style={{ fontSize: 17, fontWeight: 800, color: 'var(--ink)' }}>{t('sheet.title')}</div>
          <p className="sub" style={{ margin: '6px 0 0' }}>{t('sheet.subtitle')}</p>
        </div>

        <div style={{ fontSize: 12, fontWeight: 700, color: 'var(--muted)', textAlign: 'start', margin: '18px 2px 10px' }}>
          {t('sheet.moodTitle')} · {formatJalaliDayMonth(selectedDate, locale)}
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          {MOOD_KEYS.map((m) => (
            <button key={m} className={`chip${mood === m ? ' on' : ''}`} onClick={() => setMood((v) => (v === m ? null : m))}>
              {t(`sheet.moods.${m}`)}
            </button>
          ))}
        </div>

        <button className="btn btn-primary" onClick={onClose} style={{ borderRadius: 14, marginTop: 22 }}>
          {t('sheet.save')}
        </button>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
export function CalendarPage() {
  const t = useTranslations('calendar');
  const locale = useLocale() as Locale;
  const format = useFormatter();
  const isRtl = locale === 'fa';

  const [{ year, month }, setView] = useState(() => {
    const j = todayJalali();
    return { year: j.year, month: j.month };
  });
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());
  const [fullMonth, setFullMonth] = useState(false);
  const [sheetOpen, setSheetOpen] = useState(false);

  const weeks = useMemo(() => jalaliMonthMatrix(year, month), [year, month]);

  const displayWeeks = useMemo(() => {
    if (fullMonth) return weeks;
    const withSelected = weeks.find((w) => w.some((c) => c && isSameDay(c.date, selectedDate)));
    return [withSelected ?? weeks[0]];
  }, [weeks, fullMonth, selectedDate]);

  const goMonth = (delta: number) => setView((v) => shiftJalaliMonth(v.year, v.month, delta));

  return (
    <div className="view" style={{ background: 'var(--page)' }}>
      <div className="home-grad" style={{ position: 'absolute', top: 0, insetInlineStart: 0, insetInlineEnd: 0, height: 260 }} />

      <div style={{ position: 'relative', zIndex: 1 }}>
      </div>

      <div className="scroll" style={{ position: 'relative', zIndex: 1 }}>
        <div style={{ padding: '6px 20px 0', textAlign: 'start' }}>
          <div className="titr">{t('title')}</div>
          <p className="sub" style={{ margin: '6px 0 0' }}>{t('subtitle')}</p>
        </div>

        <div style={{ padding: '14px 16px 0' }}>
          <div className="card" style={{ padding: '14px 14px 16px' }}>
            <MonthNav
              label={formatJalaliMonthLabel(year, month, locale)}
              isRtl={isRtl}
              onPrev={() => goMonth(-1)}
              onNext={() => goMonth(1)}
              fullMonth={fullMonth}
              onToggle={() => setFullMonth((v) => !v)}
              toggleLabel={fullMonth ? t('weekView') : t('fullMonth')}
            />

            <div className="cal-grid" style={{ marginBottom: 6 }}>
              {WEEKDAY_KEYS.map((k) => (
                <span key={k} style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 700, textAlign: 'center' }}>
                  {t(`weekdays.${k}`)}
                </span>
              ))}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {displayWeeks.map((week, wi) => (
                <div key={wi} className="cal-grid">
                  {week.map((cell, ci) => (
                    <DayCell
                      key={ci}
                      cell={cell}
                      selectedDate={selectedDate}
                      onSelect={setSelectedDate}
                      dayNumber={cell ? format.number(cell.day) : ''}
                    />
                  ))}
                </div>
              ))}
            </div>

            <div style={{ height: 1, background: 'var(--line)', margin: '14px 0' }} />
            <Legend t={t} />
          </div>
        </div>

        <div style={{ padding: '14px 16px 0' }}>
          <DayDetail t={t} locale={locale} selectedDate={selectedDate} onLog={() => setSheetOpen(true)} />
        </div>

        <div style={{ padding: '14px 16px 0' }}>
          <SmartTip t={t} />
        </div>

        <div style={{ height: 26 }} />
      </div>

      <BottomNav />

      {sheetOpen && (
        <MoodSheet t={t} locale={locale} selectedDate={selectedDate} onClose={() => setSheetOpen(false)} />
      )}
    </div>
  );
}
