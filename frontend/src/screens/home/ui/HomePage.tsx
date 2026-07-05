'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  deriveCyclePredictions,
  useCycleToday,
  type CyclePredictions,
} from '@/entities/cycle';
import { useDailyMessage, type DailyMessage } from '@/entities/message';
import type { Locale } from '@/shared/i18n';
import {
  addDays,
  formatJalaliDayMonth,
  formatJalaliMonthLabel,
  jalaliMonthMatrix,
  today,
  todayJalali,
  type JalaliMonthCell,
} from '@/shared/lib/date';
import { DropSolid, Icon, StatusBar } from '@/shared/ui';
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
function WeekRow({
  days, todayDay, loc,
}: {
  days: (JalaliMonthCell | null)[];
  todayDay: number;
  loc: Locale;
}) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
      {days.map((cell, i) => {
        const isToday = cell?.day === todayDay;
        return (
          <div key={i} style={{ display: 'flex', justifyContent: 'center', flex: 1 }}>
            <span style={{
              width: 34, height: 34, borderRadius: '50%',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 14, fontVariantNumeric: 'tabular-nums',
              ...(isToday
                ? { background: 'var(--brand)', color: '#fff', fontWeight: 700, boxShadow: '0 8px 16px -6px rgba(233,30,99,.6)' }
                : cell ? { background: '#F9FAFB', color: '#364153', fontWeight: 600 } : {}),
            }}>
              {cell ? localizeNum(cell.day, loc) : ''}
            </span>
          </div>
        );
      })}
    </div>
  );
}

function WeekStrip({
  calOpen, onToggle, loc, t,
}: {
  calOpen: boolean;
  onToggle: () => void;
  loc: Locale;
  t: T;
}) {
  const tj = todayJalali();
  const weeks = jalaliMonthMatrix(tj.year, tj.month);
  const monthLabel = formatJalaliMonthLabel(tj.year, tj.month, loc);
  const currentIdx = Math.max(0, weeks.findIndex(w => w.some(c => c?.day === tj.day)));

  return (
    <div style={{ padding: '2px 16px 0' }}>
      <div style={{ background: '#fff', borderRadius: 12, padding: '12px 14px' }}>
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
              <WeekRow key={i} days={week} todayDay={tj.day} loc={loc} />
            ))}
          </div>
        ) : (
          <WeekRow days={weeks[currentIdx] ?? []} todayDay={tj.day} loc={loc} />
        )}
      </div>
    </div>
  );
}

// ── Next period hero card ──────────────────────────────────────
function NextPeriodCard({
  t, pred, nextPeriodDate, phaseLabel, phaseDesc,
}: {
  t: T;
  pred: CyclePredictions | null;
  nextPeriodDate: string | null;
  phaseLabel: string;
  phaseDesc: string;
}) {
  const [expanded, setExpanded] = useState(true);
  const daysValue = pred ? t('days', { n: pred.daysUntilNextPeriod }) : t('unavailable');
  const fertility = pred ? t('percent', { n: pred.fertilityPercent }) : t('unavailable');
  // cycle progress for the single bar (Figma Frame 65)
  const pct = pred ? Math.min(100, Math.round((pred.cycleDay / pred.cycleLength) * 100)) : 0;
  return (
    <div style={{
      margin: '16px 16px 0', borderRadius: 12,
      background: 'linear-gradient(135deg,#E91E63 0%,#F06292 100%)',
      color: '#fff', boxShadow: '0 18px 34px -16px rgba(233,30,99,.7)',
      padding: 12,
    }}>
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

// ── Start period button ────────────────────────────────────────
function StartPeriodBtn({ label }: { label: string }) {
  return (
    <div style={{ padding: '16px 16px 0' }}>
      <button className="btn" style={{ height: 40, borderRadius: 14, gap: 8, fontSize: 14, background: '#fff', border: '1px solid #FCE7F3', color: 'var(--brand)', fontWeight: 600 }}>
        <Icon name="drop" size={16} fill="var(--brand)" strokeWidth={0} /> {label}
      </button>
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

// ── Mini stat cards ────────────────────────────────────────────
function MiniCards({ t, pred }: { t: T; pred: CyclePredictions | null }) {
  const dash = t('unavailable');
  const cards = [
    { l: t('phases.nextPeriod'), v: pred ? t('days', { n: pred.daysUntilNextPeriod }) : dash, c: '#FB64B6', bg: '#FFEBF5' },
    { l: t('phases.ovulation'),  v: pred ? t('days', { n: Math.max(0, pred.daysUntilOvulation) }) : dash, c: '#34C77B', bg: '#E7F8EF' },
    { l: t('phases.window'),     v: pred ? t('days', { n: Math.max(0, pred.daysUntilFertileWindow) }) : dash, c: '#F5A623', bg: '#FFF3DF' },
  ];
  // Figma mini «Card»: 143px, icon bubble 44 (r20), value inside a tinted chip.
  return (
    <div className="scroll-x" style={{ padding: '16px 16px 0' }}>
      <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-start', minWidth: 'min-content' }}>
        {cards.map(c => (
          <div key={c.l} style={{ width: 143, flex: '0 0 143px', background: '#fff', border: '1px solid #EEEEEE', borderRadius: 16, padding: 16, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12 }}>
            <span className="dot" style={{ width: 44, height: 44, borderRadius: 20, background: c.bg, color: c.c }}>
              <DropSolid size={22} color={c.c} />
            </span>
            <span style={{ fontSize: 13, color: '#6C6C6C', fontWeight: 700 }}>{c.l}</span>
            <span style={{ background: c.bg, borderRadius: 12, padding: '6px 14px', fontSize: 15, fontWeight: 700, color: '#4C5853', fontVariantNumeric: 'tabular-nums' }}>{c.v}</span>
          </div>
        ))}
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

  const phaseLabel = pred ? t(`phaseLabel.${pred.phase}`) : '';
  const message: DailyMessage | undefined = daily;
  const phaseDesc = message?.primary.longMessage || t('nextPeriod.phaseDesc');
  const smartTipBody = message?.primary.longMessage || t('smartTip.body');
  const smartTipQuote = message?.primary.actionSuggestion || t('smartTip.quote');
  const dos = message?.primary.dos ?? [];

  return (
    <div className="view">
      {/* Full-page gradient backdrop (Figma: #FFE5EA → #CFF9EB) */}
      <div className="home-grad" style={{ position: 'absolute', inset: 0 }} />

      <div style={{ position: 'relative', zIndex: 1 }}>
        <StatusBar />
      </div>

      <div className="scroll" style={{ position: 'relative', zIndex: 1 }}>
        <HomeHeader tagline={t('tagline')} />
        {todayData?.isRecalculating && (
          <div style={{ textAlign: 'center', fontSize: 11, color: 'var(--muted)', padding: '0 16px 4px' }}>
            {t('updating')}
          </div>
        )}
        <WeekStrip calOpen={calOpen} onToggle={() => setCalOpen(v => !v)} loc={loc} t={t} />
        <NextPeriodCard t={t} pred={pred} nextPeriodDate={nextPeriodDate} phaseLabel={phaseLabel} phaseDesc={phaseDesc} />
        <StartPeriodBtn label={t('startPeriod')} />
        <PhaseRows t={t} windowDate={windowDate} ovulationDate={ovulationDate} nextPeriodDate={nextPeriodDate} />
        <MiniCards t={t} pred={pred} />
        <Recommendations t={t} dos={dos} />
        <TodayTasks t={t} />
        <Challenge t={t} />
        <ReminderCards t={t} />
        <SmartTip t={t} body={smartTipBody} quote={smartTipQuote} />
        <WeekSummary t={t} />
        <TodayStatus t={t} />
        <Articles t={t} />
        <MyCycles t={t} pred={pred} cycleStartDate={cycleStartDate} />
        <CycleSummary t={t} pred={pred} />
        <div style={{ height: 26 }} />
      </div>

      <BottomNav />
    </div>
  );
}
