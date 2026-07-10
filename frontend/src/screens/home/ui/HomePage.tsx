'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  deriveCyclePredictions,
  useCycleForDate,
  useCycleToday,
  type CyclePredictions,
} from '@/entities/cycle';
import { useDailyMessage, type DailyMessage } from '@/entities/message';
import { StartPeriodButton } from '@/features/log-period';
import type { Locale } from '@/shared/i18n';
import {
  addDays,
  formatJalaliDayMonth,
  formatJalaliMonthLabel,
  jalaliMonthMatrix,
  toApiDate,
  toJalali,
  today,
  todayJalali,
  type JalaliMonthCell,
} from '@/shared/lib/date';
import { DropSolid, Icon } from '@/shared/ui';
import { BannerSlideshow } from '@/widgets/banner-slideshow';
import { BottomNav } from '@/widgets/bottom-nav';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);
const localizeNum = (n: string | number, loc: Locale) => (loc === 'fa' ? faNum(n) : String(n));

type T = ReturnType<typeof useTranslations>;

// ── Section header ────────────────────────────────────────────
function SectionHead({ title, action }: { title: string; action?: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', margin: '0 2px 12px' }}>
      <span style={{ fontSize: 16, fontWeight: 800, color: 'var(--ink)' }}>{title}</span>
      {action
        ? <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--brand)', cursor: 'pointer' }}>{action}</span>
        : <span />}
    </div>
  );
}

// ── App header (Figma: bare icons — stars on the start side, bell on the end) ──
function HomeHeader({ tagline }: { tagline: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '8px 16px 10px' }}>
      <button className="iconbtn" style={{ color: 'var(--brand)' }}>
        <Icon name="sparkle" size={22} fill="currentColor" strokeWidth={0} />
      </button>
      <div style={{ textAlign: 'center' }}>
        <div style={{ fontWeight: 800, fontSize: 20, color: 'var(--brand)' }}>ریـــــتمی</div>
        <div style={{ fontSize: 11, color: '#6A7282', fontWeight: 600, marginTop: 2 }}>{tagline}</div>
      </div>
      <button className="iconbtn" style={{ color: '#6A7282' }}>
        <Icon name="bell" size={21} />
      </button>
    </div>
  );
}

// ── Week strip ─────────────────────────────────────────────────
// The month grid comes from the centralized Jalali date layer (§7) — no
// hardcoded month/day. Day names come from i18n.
const WEEK_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;

// Figma «TodayCalender»: white card (r12). Closed → weekday names + the
// current week only. Open → the whole month grid, rows in calendar order.
// Tapping a day selects it so the connected info card below shows that day.
function WeekRow({
  days, todayDay, selectedDay, loc, onSelect,
}: {
  days: (JalaliMonthCell | null)[];
  todayDay: number;
  selectedDay: number | null;
  loc: Locale;
  onSelect: (cell: JalaliMonthCell) => void;
}) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
      {days.map((cell, i) => {
        const isToday = cell?.day === todayDay;
        const isSelected = cell != null && cell.day === selectedDay && !isToday;
        return (
          <div key={i} style={{ display: 'flex', justifyContent: 'center', flex: 1 }}>
            <button
              type="button"
              disabled={!cell}
              onClick={cell ? () => onSelect(cell) : undefined}
              aria-pressed={isToday || isSelected}
              style={{
                width: 34, height: 34, borderRadius: '50%', padding: 0,
                border: isSelected ? '2px solid var(--brand)' : '2px solid transparent',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 14, fontFamily: 'inherit', fontVariantNumeric: 'tabular-nums',
                cursor: cell ? 'pointer' : 'default',
                ...(isToday
                  ? { background: 'var(--brand)', color: '#fff', fontWeight: 700, boxShadow: '0 8px 16px -6px rgba(233,30,99,.6)' }
                  : isSelected
                    ? { background: '#FFF0F6', color: 'var(--brand)', fontWeight: 700 }
                    : cell ? { background: '#F9FAFB', color: '#364153', fontWeight: 600 } : { background: 'transparent' }),
              }}
            >
              {cell ? localizeNum(cell.day, loc) : ''}
            </button>
          </div>
        );
      })}
    </div>
  );
}

function WeekStrip({
  calOpen, onToggle, loc, t, selectedDay, onSelect,
}: {
  calOpen: boolean;
  onToggle: () => void;
  loc: Locale;
  t: T;
  selectedDay: number | null;
  onSelect: (cell: JalaliMonthCell) => void;
}) {
  const tj = todayJalali();
  const weeks = jalaliMonthMatrix(tj.year, tj.month);
  const monthLabel = formatJalaliMonthLabel(tj.year, tj.month, loc);
  // Show the week that contains the selected day (falls back to today's week).
  const selIdx = weeks.findIndex(w => w.some(c => c?.day === selectedDay));
  const todayIdx = weeks.findIndex(w => w.some(c => c?.day === tj.day));
  const currentIdx = Math.max(0, selIdx >= 0 ? selIdx : todayIdx);

  return (
    // Top half of the connected calendar↔info card unit — flat bottom so it
    // butts against the pink info panel below (no own margin/radius).
    <div style={{ background: '#fff', padding: '12px 14px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
          <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--brand)' }}>{monthLabel}</span>
          <button
            onClick={onToggle}
            style={{
              display: 'flex', alignItems: 'center', gap: 4,
              background: 'transparent', border: 0, cursor: 'pointer',
              padding: '2px 4px', borderRadius: 8,
            }}
          >
            <span style={{ fontSize: 13, fontWeight: 600, color: '#6A7282' }}>
              {calOpen ? t('week.close') : t('week.fullMonth')}
            </span>
            <Icon
              name="chevronDown"
              size={14}
              style={{
                color: '#6A7282',
                transition: 'transform .28s cubic-bezier(.22,.61,.36,1)',
                transform: calOpen ? 'rotate(180deg)' : undefined,
              }}
            />
          </button>
        </div>

        {/* weekday names — one header row for both states */}
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
          {WEEK_KEYS.map(k => (
            <span key={k} style={{ flex: 1, textAlign: 'center', fontSize: 11, color: '#6A7282', fontWeight: 600 }}>
              {t(`week.${k}`)}
            </span>
          ))}
        </div>

        {calOpen ? (
          <div className="cal-drop" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {weeks.map((week, i) => (
              <WeekRow key={i} days={week} todayDay={tj.day} selectedDay={selectedDay} loc={loc} onSelect={onSelect} />
            ))}
          </div>
        ) : (
          <WeekRow days={weeks[currentIdx] ?? []} todayDay={tj.day} selectedDay={selectedDay} loc={loc} onSelect={onSelect} />
        )}
    </div>
  );
}

// ── Next period hero card ──────────────────────────────────────
function NextPeriodCard({
  t, pred, nextPeriodDate, phaseLabel, phaseDesc, isToday, selectedDateLabel,
}: {
  t: T;
  pred: CyclePredictions | null;
  nextPeriodDate: string | null;
  phaseLabel: string;
  phaseDesc: string;
  isToday: boolean;
  selectedDateLabel: string;
}) {
  const [expanded, setExpanded] = useState(true);
  const daysValue = pred ? t('days', { n: pred.daysUntilNextPeriod }) : t('unavailable');
  const fertility = pred ? t('percent', { n: pred.fertilityPercent }) : t('unavailable');
  // cycle progress for the single bar (Figma Frame 65)
  const pct = pred ? Math.min(100, Math.round((pred.cycleDay / pred.cycleLength) * 100)) : 0;
  return (
    // Bottom half of the connected calendar↔info unit — fills the wrapper
    // (no own margin/radius); the wrapper owns the rounding + shadow.
    <div style={{
      background: 'linear-gradient(135deg,#E91E63 0%,#F06292 100%)',
      color: '#fff', padding: 12,
    }}>
      {/* Which day the info below reflects — updates as the user taps a day. */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '2px 4px 8px', fontSize: 12, fontWeight: 700, opacity: .95 }}>
        <Icon name="calendar" size={14} stroke="#fff" />
        {isToday ? t('selectedDay.today') : t('selectedDay.date', { date: selectedDateLabel })}
      </div>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 10, padding: '4px 4px 0' }}>
        <div style={{ textAlign: 'start', flex: 1, paddingTop: 4 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 15, fontWeight: 700, opacity: .92 }}>
            <Icon name="calendar" size={18} stroke="#fff" />
            {t('nextPeriod.label')}
          </div>
          <div style={{ fontSize: 20, fontWeight: 800, margin: '10px 10px 0' }}>{daysValue}</div>
          {nextPeriodDate && (
            <div style={{ fontSize: 12, fontWeight: 700, opacity: .9, margin: '10px 8px 0' }}>
              {t('nextPeriod.startDate', { date: nextPeriodDate })}
            </div>
          )}
        </div>

        {/* fertility ring (Figma: 72px circle, white ring) */}
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
          <div style={{ fontSize: 11.5, fontWeight: 700, opacity: .95 }}>
            {pred ? t('nextPeriod.currentPhase', { phase: phaseLabel }) : t('nextPeriod.phase')}
          </div>
          <div style={{
            width: 72, height: 72, borderRadius: '50%', border: '3px solid #fff',
            display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
          }}>
            <span style={{ fontSize: 17, fontWeight: 800 }}>{fertility}</span>
            <span style={{ fontSize: 10.5, opacity: .9, fontWeight: 700 }}>{t('nextPeriod.fertility')}</span>
          </div>
        </div>
      </div>

      {/* single cycle-progress bar (Figma Frame 65) */}
      <div style={{ height: 10, borderRadius: 99, background: 'rgba(255,255,255,.3)', border: '1px solid rgba(255,255,255,.55)', margin: '14px 4px 12px', overflow: 'hidden' }}>
        <div style={{ height: '100%', width: `${pct}%`, borderRadius: 99, background: '#fff', marginInlineStart: 'auto' }} />
      </div>

      {/* phase explanation box (Figma Frame 66 — #F96C9C) */}
      {expanded && (
        <div style={{ background: '#F96C9C', borderRadius: 8, padding: '8px 10px', fontSize: 11, lineHeight: 1.7, textAlign: 'start', fontWeight: 600 }}>
          {phaseDesc}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'center', marginTop: 10 }}>
        <button
          onClick={() => setExpanded(v => !v)}
          style={{ background: 'transparent', border: 0, color: '#FAFCFC', padding: '2px 8px', fontFamily: 'inherit', fontSize: 12, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4 }}
        >
          {expanded ? t('nextPeriod.close') : t('nextPeriod.showMore')}
          <Icon name="chevronDown" size={14} style={{ transform: expanded ? undefined : 'rotate(180deg)' }} />
        </button>
      </div>
    </div>
  );
}

// ── Phase rows ─────────────────────────────────────────────────
function PhaseRows({
  t, windowDate, ovulationDate, nextPeriodDate,
}: {
  t: T;
  windowDate: string | null;
  ovulationDate: string | null;
  nextPeriodDate: string | null;
}) {
  const dash = t('unavailable');
  const rows = [
    { l: t('phases.window'),     d: windowDate ?? dash,     c: '#F5A623', bg: '#FEF3C6' },
    { l: t('phases.ovulation'),  d: ovulationDate ?? dash,  c: '#34C77B', bg: '#F0FDFA' },
    { l: t('phases.nextPeriod'), d: nextPeriodDate ?? dash, c: '#FB64B6', bg: '#FCE7F3' },
  ];
  // Figma «Card»: label + 30px icon bubble on the start side, the date inside
  // a chip tinted with the same phase color on the end side. No dividers.
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: '#fff', borderRadius: 16, padding: '16px' }}>
        {rows.map((r, i) => (
          <div key={r.l} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: i > 0 ? 14 : 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <span className="dot" style={{ background: r.bg, color: r.c }}>
                <DropSolid size={16} color={r.c} />
              </span>
              <span style={{ fontSize: 14, fontWeight: 700, color: '#707983' }}>{r.l}</span>
            </div>
            <span style={{ background: r.bg, borderRadius: 12, padding: '8px 14px', fontSize: 13, fontWeight: 700, color: '#58636E', fontVariantNumeric: 'tabular-nums', minWidth: 76, textAlign: 'center' }}>
              {r.d}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── PMS window box ─────────────────────────────────────────────
// Shows the predicted PMS days (the short run before the next period) as a
// Jalali date range, e.g. «۷ تا ۱۰ اسفند». Derived from predictions (§7 —
// formatted at the display boundary), no extra API call.
function PmsBox({ t, dateRange }: { t: T; dateRange: string | null }) {
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: '#fff', borderRadius: 16, padding: '14px 16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span className="dot" style={{ width: 40, height: 40, borderRadius: 20, background: '#F3E8FF', color: '#A91EE9' }}>
            <DropSolid size={18} color="#A91EE9" />
          </span>
          <span style={{ fontSize: 14, fontWeight: 700, color: '#707983' }}>{t('pms.label')}</span>
        </div>
        <span style={{ background: '#F3E8FF', borderRadius: 12, padding: '8px 14px', fontSize: 13, fontWeight: 700, color: '#7E22CE', fontVariantNumeric: 'tabular-nums', textAlign: 'center' }}>
          {dateRange ?? t('unavailable')}
        </span>
      </div>
    </div>
  );
}

// ── Daily recommendations (from /messages/daily) ───────────────
function Recommendations({ t, dos }: { t: T; dos: string[] }) {
  // Prefer the backend's personalized "do" suggestions; fall back to the
  // static defaults until the message service has data for the day.
  const items = dos.length > 0
    ? dos.slice(0, 3).map(text => ({ title: text, desc: undefined as string | undefined }))
    : [
        { title: t('recommendations.iron'), desc: t('recommendations.ironDesc') },
        { title: t('recommendations.omega'), desc: t('recommendations.omegaDesc') },
      ];
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ fontSize: 16, fontWeight: 800, color: 'var(--ritme-pink)', textAlign: 'start', marginBottom: 12 }}>
          {t('recommendations.title')}
        </div>
        {/* Figma item: soft green→pink gradient, white circular check badge */}
        {items.map((item, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 12, background: 'linear-gradient(90deg,#FDF2F8,#F7FFF9)', borderRadius: 8, padding: 12, marginBottom: 8 }}>
            <span style={{ width: 36, height: 36, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', flex: '0 0 auto', color: '#66BB6A' }}>
              <Icon name="check" size={18} stroke="currentColor" />
            </span>
            <div style={{ flex: 1, textAlign: 'start' }}>
              <div style={{ fontSize: 15, fontWeight: 600, color: '#2D2D2D' }}>{item.title}</div>
              {item.desc && <div style={{ fontSize: 12.5, color: '#8A9098', marginTop: 4 }}>{item.desc}</div>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Today tasks (local UI state — no API counterpart in the spec) ──
const INITIAL_TASKS = [
  { done: true },
  { done: true },
  { done: false },
  { done: false },
];

function TodayTasks({ t }: { t: T }) {
  const LABELS = [t('tasks.temp'), t('tasks.water'), t('tasks.vitamin'), t('tasks.walk')];
  const [tasks, setTasks] = useState(INITIAL_TASKS.map((x, i) => ({ ...x, label: LABELS[i] })));

  const toggle = (i: number) => setTasks(prev => prev.map((tk, k) => k === i ? { ...tk, done: !tk.done } : tk));
  const done = tasks.filter(tk => tk.done).length;
  const pct = Math.round((done / tasks.length) * 100);

  return (
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
          <span style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>{t('tasks.title')}</span>
          <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, fontWeight: 700, color: 'var(--steel)' }}>
            {faNum(done)}/{faNum(tasks.length)}
            <Icon name="checkCircle" size={16} stroke="var(--green)" />
          </span>
        </div>

        {tasks.map((tk, i) => (
          <div key={i} onClick={() => toggle(i)} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, padding: '9px 2px', cursor: 'pointer' }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: tk.done ? '#AEB6BF' : 'var(--ink)', textDecoration: tk.done ? 'line-through' : undefined }}>{tk.label}</span>
            <span className={`cbx${tk.done ? ' on' : ''}`}>
              <Icon name="check" size={14} stroke="#fff" />
            </span>
          </div>
        ))}

        <div style={{ height: 1, background: 'var(--line)', margin: '14px 0 12px' }} />
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
          <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--muted)' }}>{t('tasks.progress')}</span>
          <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--steel)' }}>{t('tasks.percent', { n: faNum(pct) })}</span>
        </div>
        <div className="bar"><i style={{ width: `${pct}%` }} /></div>
      </div>
    </div>
  );
}

// ── Smart tip (from /messages/daily) ───────────────────────────
function SmartTip({ t, body, quote }: { t: T; body: string; quote: string }) {
  return (
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)', textAlign: 'start', marginBottom: 10 }}>
          {t('smartTip.title')}
        </div>
        <p className="sub" style={{ textAlign: 'start', margin: '0 2px 14px' }}>{body}</p>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, background: 'linear-gradient(90deg,#FFF0F7,#F3F0FF)', borderRadius: 12, padding: 12 }}>
          <span style={{ color: 'var(--ritme-pink)' }}>
            <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
          </span>
          <span style={{ flex: 1, textAlign: 'start', fontSize: 12.5, fontWeight: 700, color: 'var(--ink)' }}>
            {quote}
          </span>
        </div>
      </div>
    </div>
  );
}

// ── Week summary (no aggregate endpoint in the spec — static) ───
function WeekSummary({ t }: { t: T }) {
  const cards = [
    { l: t('weekSummary.mood'),   v: '۸۱٪',                       icon: 'smile' as const, c: '#F06292' },
    { l: t('weekSummary.sleep'),  v: t('weekSummary.sleepHours'), icon: 'moon'  as const, c: '#7C7CF0' },
    { l: t('weekSummary.energy'), v: '۷۳٪',                       icon: 'zap'   as const, c: '#F5A623' },
  ];
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: '#fff', border: '1px solid #F0FDFA', borderRadius: 12, padding: '14px 12px' }}>
      <SectionHead title={t('weekSummary.title')} action={t('weekSummary.viewAll')} />
      {/* Figma tile: #F9FAFB, icon in a white circle, label above value */}
      <div style={{ display: 'flex', gap: 12 }}>
        {cards.map(c => (
          <div key={c.l} style={{ flex: 1, background: '#F9FAFB', borderRadius: 12, padding: '12px 8px', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
            <span style={{ width: 36, height: 36, borderRadius: '50%', background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', color: c.c }}>
              <Icon name={c.icon} size={20} stroke="currentColor" />
            </span>
            <span style={{ fontSize: 12, color: 'var(--ink)', fontWeight: 700 }}>{c.l}</span>
            <span style={{ fontSize: 14, fontWeight: 600, color: '#2D2D2D', fontVariantNumeric: 'tabular-nums' }}>{c.v}</span>
          </div>
        ))}
      </div>
      <div style={{ marginTop: 12, background: 'linear-gradient(90deg,#FDF2F8,#FAF5FF)', borderRadius: 12, padding: '10px 12px', textAlign: 'center', fontSize: 12, fontWeight: 600, color: '#404D58' }}>
        ✨ {t('weekSummary.desc')}
      </div>
      </div>
    </div>
  );
}

// ── Cycle summary (length from /cycle/today) ───────────────────
function CycleSummary({ t, pred }: { t: T; pred: CyclePredictions | null }) {
  const rows: [string, string][] = [
    [t('cycleSummary.length'), pred ? t('days', { n: pred.cycleLength }) : t('unavailable')],
    [t('cycleSummary.duration'), '۵ روز'],
    [t('cycleSummary.variation'), '۳ روز'],
  ];
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: '#fff', borderRadius: 12, padding: '16px 14px' }}>
        <SectionHead title={t('cycleSummary.title')} />
        {/* Figma Frame 41: rows live inside a bordered box */}
        <div style={{ border: '1px solid var(--line)', borderRadius: 12, padding: '6px 14px' }}>
          {rows.map(([label, val], i) => (
            <div key={label} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 2px', ...(i < 2 ? { borderBottom: '1px solid var(--line)' } : {}) }}>
              <span style={{ fontSize: 14, color: '#6C6C6C', fontWeight: 700 }}>{label}</span>
              <span style={{ fontSize: 13.5, fontWeight: 600, color: '#949494', fontVariantNumeric: 'tabular-nums' }}>{val}</span>
            </div>
          ))}
        </div>
        <div style={{ paddingTop: 14 }}>
          <button className="btn btn-primary" style={{ height: 40 }}>{t('cycleSummary.viewMore')}</button>
        </div>
      </div>
    </div>
  );
}

// ── Today challenge (no API counterpart — static) ──────────────
function Challenge({ t }: { t: T }) {
  return (
    <div style={{ padding: '14px 16px 0' }}>
      <div className="card" style={{ padding: '14px 12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
          <span style={{ color: 'var(--amber)' }}><Icon name="flame" size={18} stroke="currentColor" /></span>
          <span style={{ fontSize: 14, fontWeight: 800, color: 'var(--ink)' }}>{t('challenge.title')}</span>
        </div>
        {/* Figma: the challenge item has a 2px green outline */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', border: '2px solid #007C53', borderRadius: 8, padding: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ color: 'var(--brand)' }}><Icon name="thermo" size={18} stroke="currentColor" /></span>
            <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--ink)' }}>{t('challenge.item')}</span>
          </div>
          <span className="cbx" />
        </div>
        <div style={{ marginTop: 10, background: 'linear-gradient(90deg,#FDF2F8,#FAF5FF)', borderRadius: 12, padding: '8px 12px', textAlign: 'center', fontSize: 12, fontWeight: 600, color: '#404D58' }}>
          ✨ {t('challenge.desc')}
        </div>
      </div>
    </div>
  );
}

// ── Reminder cards (no API counterpart — static) ───────────────
function ReminderCards({ t }: { t: T }) {
  const items = [
    { icon: 'stetho' as const, title: t('reminder.doctor'), sub: `${t('reminder.doctorName')} · ${t('reminder.specialty')}`, c: '#7C7CF0', bg: '#F3F0FF' },
    { icon: 'pill'   as const, title: t('reminder.medicine'), sub: undefined, c: 'var(--brand)', bg: '#FFEBF5' },
  ];
  return (
    <div style={{ padding: '14px 16px 0', display: 'flex', flexDirection: 'column', gap: 10 }}>
      {items.map(it => (
        <div key={it.title} className="card" style={{ padding: '12px 14px', display: 'flex', alignItems: 'center', gap: 12 }}>
          <span className="dot" style={{ width: 40, height: 40, background: it.bg, color: it.c }}>
            <Icon name={it.icon} size={20} stroke="currentColor" />
          </span>
          <div style={{ flex: 1, textAlign: 'start' }}>
            <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--ink)' }}>{it.title}</div>
            {it.sub && <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 3 }}>{it.sub}</div>}
          </div>
          <span style={{ color: 'var(--muted)' }}><Icon name="alarm" size={18} stroke="currentColor" /></span>
        </div>
      ))}
    </div>
  );
}

// ── Cycle-based articles (no article endpoint in the spec — static) ──
function Articles({ t }: { t: T }) {
  // Figma «Frame 36»: white card, 18px title, 162px blog cards, primary CTA.
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: '#fff', borderRadius: 12, padding: '20px 16px' }}>
        <div style={{ fontSize: 17, fontWeight: 700, color: 'var(--ink)', textAlign: 'start', marginBottom: 16 }}>
          {t('articles.title')}
        </div>
        <div className="scroll-x">
          <div style={{ display: 'flex', gap: 16, minWidth: 'min-content' }}>
            {[0, 1, 2].map(i => (
              <div key={i} style={{ width: 162, flex: '0 0 162px', display: 'flex', flexDirection: 'column', gap: 4 }}>
                <div style={{ height: 130, borderRadius: 12, background: 'linear-gradient(135deg,#FFF0F7,#F3F0FF)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--ritme-pink)' }}>
                  <Icon name="bookOpen" size={30} stroke="currentColor" />
                </div>
                <div style={{ fontSize: 13.5, fontWeight: 700, color: '#000', textAlign: 'start', lineHeight: 1.55, padding: '8px 0 0' }}>{t('articles.article1')}</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 5, fontSize: 12.5, fontWeight: 600, color: '#3A3A3A' }}>
                  <Icon name="bookOpen" size={16} stroke="currentColor" />
                  {t('articles.min', { n: faNum(9) })}
                </div>
              </div>
            ))}
          </div>
        </div>
        <button className="btn btn-primary" style={{ marginTop: 18, height: 40 }}>{t('articles.readMore')}</button>
      </div>
    </div>
  );
}

// ── Today status (vitals not in the spec — static) ─────────────
function TodayStatus({ t }: { t: T }) {
  const items = [
    { icon: 'heart' as const, label: t('todayStatus.bp'),      c: '#F06292' },
    { icon: 'drop'  as const, label: t('todayStatus.glucose'), c: '#7C7CF0' },
    { icon: 'zap'   as const, label: t('todayStatus.hr'),      c: '#F5A623' },
  ];
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div style={{ background: '#fff', border: '1px solid #F0FDFA', borderRadius: 12, padding: '14px 12px' }}>
        <SectionHead title={t('todayStatus.title')} action={t('todayStatus.viewAll')} />
        <div style={{ display: 'flex', gap: 12 }}>
          {items.map(it => (
            <div key={it.label} style={{ flex: 1, background: '#fff', border: '1px solid #EEEEEE', borderRadius: 12, padding: '10px 8px', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
              <span style={{ color: it.c }}><Icon name={it.icon} size={22} stroke="currentColor" /></span>
              <span style={{ fontSize: 12, color: 'var(--ink)', fontWeight: 700 }}>{it.label}</span>
              <span style={{ fontSize: 10.5, fontWeight: 700, color: '#478F96', background: '#D0FBFF', borderRadius: 4, padding: '2px 8px' }}>{t('todayStatus.normal')}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ── My cycles (current day & start from /cycle/today) ──────────
function MyCycles({
  t, pred, cycleStartDate,
}: {
  t: T;
  pred: CyclePredictions | null;
  cycleStartDate: string | null;
}) {
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div className="card" style={{ padding: '16px 14px' }}>
        <SectionHead title={t('cycles.title')} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '4px 2px 12px' }}>
          <span className="dot" style={{ width: 40, height: 40, background: '#FCE7F3', color: 'var(--brand)' }}>
            <DropSolid size={18} color="var(--brand)" />
          </span>
          <div style={{ textAlign: 'start' }}>
            <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--ink)' }}>
              {t('cycles.current')}: {pred ? t('cycles.dayN', { n: pred.cycleDay }) : t('cycles.day')}
            </div>
            {cycleStartDate && (
              <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 3 }}>
                {t('cycles.startedOn')} {cycleStartDate}
              </div>
            )}
          </div>
        </div>
        {/* Figma Frame 21: divider + pink inline action with add-circle icon */}
        <div style={{ height: 1, background: 'var(--line)', margin: '2px 0 12px' }} />
        <div role="button" tabIndex={0} style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, cursor: 'pointer', color: '#F6339A', fontSize: 14, fontWeight: 700, padding: '2px 0' }}>
          <Icon name="plus" size={18} stroke="currentColor" /> {t('cycles.addPrevious')}
        </div>
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
export function HomePage() {
  const t = useTranslations('home');
  const loc = useLocale() as Locale;
  const [calOpen, setCalOpen] = useState(false);
  // The day the user tapped in the mini calendar (defaults to today). Only the
  // connected info card reflects it; the rest of the page stays on today.
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());

  // Server state (§8) — cycle math + personalized message for today.
  const { data: todayData } = useCycleToday();
  const { data: daily } = useDailyMessage();

  const calc = todayData?.calculation ?? null;
  const pred = calc ? deriveCyclePredictions(calc) : null;

  // Turn day-offsets into Jalali dates only here, at the display boundary (§7).
  const base = today();
  const fmt = (offset: number) => formatJalaliDayMonth(addDays(base, offset), loc);
  const nextPeriodDate = pred ? fmt(pred.daysUntilNextPeriod) : null;
  const ovulationDate = pred ? fmt(Math.max(0, pred.daysUntilOvulation)) : null;
  const windowDate = pred ? fmt(Math.max(0, pred.daysUntilFertileWindow)) : null;
  const cycleStartDate = pred ? fmt(-(pred.cycleDay - 1)) : null;
  const pmsRange = pred
    ? t('pms.range', { from: fmt(pred.daysUntilPmsStart), to: fmt(pred.daysUntilPmsEnd) })
    : null;

  const message: DailyMessage | undefined = daily;
  const smartTipBody = message?.primary.longMessage || t('smartTip.body');
  const smartTipQuote = message?.primary.actionSuggestion || t('smartTip.quote');
  const dos = message?.primary.dos ?? [];

  // ── Selected-day info for the connected calendar↔info card ──
  const selApiDate = toApiDate(selectedDate);
  const isToday = selApiDate === toApiDate(base);
  const selJalali = toJalali(selectedDate);
  const monthJalali = todayJalali();
  const selectedDay =
    selJalali.year === monthJalali.year && selJalali.month === monthJalali.month
      ? selJalali.day
      : null;

  // A tapped past/future day loads its own calculation + message; today reuses
  // the queries above (same cache keys — no extra fetch).
  const { data: dateData } = useCycleForDate(selApiDate, !isToday);
  const { data: infoMessage } = useDailyMessage(isToday ? undefined : selApiDate);

  const infoCalc = (isToday ? todayData : dateData)?.calculation ?? null;
  const infoPred = infoCalc ? deriveCyclePredictions(infoCalc) : null;
  const infoFmt = (offset: number) => formatJalaliDayMonth(addDays(selectedDate, offset), loc);
  const infoNextPeriodDate = infoPred ? infoFmt(infoPred.daysUntilNextPeriod) : null;
  const infoPhaseLabel = infoPred ? t(`phaseLabel.${infoPred.phase}`) : '';
  const infoPhaseDesc = infoMessage?.primary.longMessage || t('nextPeriod.phaseDesc');
  const selectedDateLabel = formatJalaliDayMonth(selectedDate, loc);

  return (
    <div className="view">
      {/* Full-page gradient backdrop (Figma: #FFE5EA → #CFF9EB) */}
      <div className="home-grad" style={{ position: 'absolute', inset: 0 }} />

      <div style={{ position: 'relative', zIndex: 1 }}>
      </div>

      <div className="scroll" style={{ position: 'relative', zIndex: 1 }}>
        <HomeHeader tagline={t('tagline')} />
        {todayData?.isRecalculating && (
          <div style={{ textAlign: 'center', fontSize: 11, color: 'var(--muted)', padding: '0 16px 4px' }}>
            {t('updating')}
          </div>
        )}
        {/* Connected unit: the mini calendar butts directly against the info
            card below it, and tapping a day updates that card (§ home request). */}
        <div style={{ padding: '2px 16px 0' }}>
          <div style={{ borderRadius: 16, overflow: 'hidden', boxShadow: '0 18px 34px -18px rgba(233,30,99,.55)' }}>
            <WeekStrip
              calOpen={calOpen}
              onToggle={() => setCalOpen(v => !v)}
              loc={loc}
              t={t}
              selectedDay={selectedDay}
              onSelect={(cell) => setSelectedDate(cell.date)}
            />
            <NextPeriodCard
              t={t}
              pred={infoPred}
              nextPeriodDate={infoNextPeriodDate}
              phaseLabel={infoPhaseLabel}
              phaseDesc={infoPhaseDesc}
              isToday={isToday}
              selectedDateLabel={selectedDateLabel}
            />
          </div>
        </div>
        {/* Admin-managed promo slot — renders nothing until a banner is active */}
        <BannerSlideshow position="home_top" />
        <StartPeriodButton />
        <PmsBox t={t} dateRange={pmsRange} />
        <PhaseRows t={t} windowDate={windowDate} ovulationDate={ovulationDate} nextPeriodDate={nextPeriodDate} />
        <Recommendations t={t} dos={dos} />
        <BannerSlideshow position="home_middle" />
        <TodayTasks t={t} />
        <Challenge t={t} />
        <ReminderCards t={t} />
        <SmartTip t={t} body={smartTipBody} quote={smartTipQuote} />
        <WeekSummary t={t} />
        <TodayStatus t={t} />
        <Articles t={t} />
        <MyCycles t={t} pred={pred} cycleStartDate={cycleStartDate} />
        <BannerSlideshow position="home_bottom" />
        <CycleSummary t={t} pred={pred} />
        <div style={{ height: 26 }} />
      </div>

      <BottomNav />
    </div>
  );
}
