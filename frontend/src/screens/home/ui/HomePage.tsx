'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useMemo, useState, type ReactNode } from 'react';

import {
  articleCategoryLabel,
  type CategoryTranslator,
  useCycleArticles,
} from '@/entities/article';
import { useBannersSettled } from '@/entities/banner';
import {
  CycleValuesCard,
  cycleDayMarkerAt,
  cycleScheduleFor,
  daysUntilNextPeriod,
  deriveCycleSchedule,
  deriveCyclePredictions,
  hasFertileWindow,
  useCycleForDate,
  useCycleMonth,
  useCycleOverview,
  useCycleStatus,
  useCycleToday,
  type CycleCalculation,
  type CycleDailyTip,
  type CycleDayMarker,
  type CyclePhase,
  type CycleSchedule,
} from '@/entities/cycle';
import { useFertilityToday } from '@/entities/fertility';
import { LossCareReturn } from '@/entities/loss';
import { useDailyMessage, useUserMode, type DailyMessage } from '@/entities/message';
import {
  readLifeModeHint,
  useLifeStage,
  useUserProfile,
  writeLifeModeHint,
  type LifeMode,
} from '@/entities/user';
import { QuickEditSheet } from '@/features/edit-profile';
import { PeriodDateEditor, useEndPeriod, useStartPeriod } from '@/features/log-period';
import { getApiErrorCode } from '@/shared/api';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import {
  addDays,
  currentHour,
  diffInDays,
  formatDayMonth,
  formatNumber,
  formatWeekdayDayMonth,
  fromApiDate,
  toApiDate,
  toParts,
  today,
  weekdayLabels,
  weekOf,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  Card,
  Icon,
  IconCircle,
  PrimaryButton,
  Skeleton,
  SkeletonGroup,
  StatusPill,
  type IconName,
  type Tone,
} from '@/shared/ui';
import { BannerSlideshow } from '@/widgets/banner-slideshow';
import { BottomNav } from '@/widgets/bottom-nav';
import { CheckupsCard } from '@/widgets/checkups-card';
import { LinkedTeenCards } from '@/widgets/linked-teen-card';
import {
  FertilityChanceCard,
  FertilityTiles,
  LhTipCard,
  TtcPhasePills,
  ttcPhase,
} from '@/widgets/fertility-tiles';
import { TeenHome } from '@/widgets/teen-home';
import { TodayChallengeCard } from '@/widgets/today-challenge';
import { TodayRemindersCard } from '@/widgets/today-reminders';

import { needsPeriodData } from '../model/cycle-data';
import { heroState, phaseFromMainPhase } from '../model/hero-state';
import { dismissPrompt, isPromptDismissed } from '../model/prompt-dismiss';
import { readTtcHint, writeTtcHint } from '../model/ttc-hint';

import { PmsInsightCard, PredictionsCard, type PredictionSlot } from './PredictionsCard';
import { MenopauseHome } from './MenopauseHome';
import { PlusTrialOffer } from './PlusTrialOffer';
import { TodayLogCard } from './TodayLogCard';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);
const localizeNum = (n: string | number, loc: Locale) => (loc === 'fa' ? faNum(n) : String(n));

type T = ReturnType<typeof useTranslations>;

// ── Header: today's date + a time-of-day greeting, notification & cycle settings ──
function greetingKey(hour: number): 'morning' | 'noon' | 'evening' | 'night' {
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 16) return 'noon';
  if (hour >= 16 && hour < 20) return 'evening';
  return 'night';
}

function HomeHeader({ t, loc }: { t: T; loc: Locale }) {
  const tSearch = useTranslations('search');
  return (
    <header className="home-hdr">
      <div>
        <div className="home-hdr-date">{formatWeekdayDayMonth(today(), loc)}</div>
        <div className="home-hdr-greet">{t(`greeting.${greetingKey(currentHour())}`)}</div>
      </div>
      <div className="home-hdr-actions">
        {/* CB-NAV-02: global search (nbd_Nav_Today header). */}
        <Link href="/search" className="home-hdr-btn" aria-label={tSearch('open')}>
          <Icon name="search" size={20} strokeWidth={1.8} />
        </Link>
        <button
          type="button"
          className="home-hdr-btn"
          onClick={() => openSheet('notifications')}
          aria-label={t('header.notifications')}
        >
          <Icon name="bellPlain" size={20} strokeWidth={1.8} />
        </button>
        {/* B-N1-09: the cog opens the cycle settings screen (nbl_Cycle_Home → nbl_Cycle_Settings). */}
        <Link href="/cycle/settings" className="home-hdr-btn" aria-label={t('header.cycleSettings')}>
          <Icon name="cog" size={20} strokeWidth={1.8} />
        </Link>
      </div>
    </header>
  );
}

// ── Day markers ────────────────────────────────────────────────

// Fallbacks when neither the engine nor the profile has a value (they match
// the backend defaults).
const DEFAULT_PERIOD_DAYS = 5;
const DEFAULT_CYCLE_DAYS = 28;

/** Gregorian year/month a date falls in, parsed from its API serialization (§7). */
function gregYearMonth(date: Date): { year: number; month: number } {
  const [year, month] = toApiDate(date).split('-');
  return { year: Number(year), month: Number(month) };
}

/**
 * Cycle marker per day, from the same `cycle/month` cache the calendar screen
 * reads — so editing a period there (which invalidates `cycleKeys.all`)
 * repaints the week strip and the ring too. The days asked for (this week plus
 * the ring's cycle) span at most three Gregorian months, so the first, middle
 * and last month are fetched; a day none of them covers yet falls back to the
 * schedule's own reading of it. From the current cycle on, the fertile window
 * and ovulation come from the anchored `schedule` (task.md §19) — the same days
 * the calendar, the timeline bar and `/fertility/*` show.
 */
function useDayMarks(
  dates: Date[],
  schedule: CycleSchedule | null,
  periodLength: number,
): (date: Date) => CycleDayMarker | null {
  const sorted = [...dates].sort((a, b) => a.getTime() - b.getTime());
  const first = sorted[0] ?? today();
  const last = sorted[sorted.length - 1] ?? first;
  const middle = addDays(first, Math.round(diffInDays(last, first) / 2));
  const gA = gregYearMonth(first);
  const gM = gregYearMonth(middle);
  const gB = gregYearMonth(last);
  const monthA = useCycleMonth(gA.year, gA.month);
  const monthM = useCycleMonth(gM.year, gM.month);
  const monthB = useCycleMonth(gB.year, gB.month);

  const calcMap = useMemo(() => {
    const map = new Map<string, CycleCalculation>();
    for (const month of [monthA.data, monthM.data, monthB.data]) {
      for (const c of month?.calculations ?? []) map.set(c.calculationDate, c);
    }
    return map;
  }, [monthA.data, monthM.data, monthB.data]);

  return (date: Date) =>
    cycleDayMarkerAt(date, calcMap.get(toApiDate(date)), schedule, periodLength);
}

// ── Week strip ─────────────────────────────────────────────────
// The current week in the locale's grid order, each day a tall tile with its
// cycle-marker dot. Tapping a day points the ring and the phase card at it.
function WeekStrip({
  days, loc, selectedIso, onSelect, markOf,
}: {
  days: Date[];
  loc: Locale;
  selectedIso: string;
  onSelect: (date: Date) => void;
  markOf: (date: Date) => CycleDayMarker | null;
}) {
  const todayIso = toApiDate(today());
  const letters = weekdayLabels(loc);
  return (
    <div className="home-week">
      {days.map((date, i) => {
        const iso = toApiDate(date);
        const isToday = iso === todayIso;
        const isSelected = iso === selectedIso && !isToday;
        return (
          <button
            key={iso}
            type="button"
            className={clsx('home-wday', isToday && 'is-today', isSelected && 'is-selected')}
            onClick={() => onSelect(date)}
            aria-pressed={iso === selectedIso}
            aria-label={formatWeekdayDayMonth(date, loc)}
          >
            <span className="home-wday-name" aria-hidden>{letters[i]}</span>
            <span className="home-wday-cell" aria-hidden>
              {formatNumber(toParts(date, loc).day, loc)}
              <span className={clsx('home-wday-dot', `is-${markOf(date) ?? 'none'}`)} />
            </span>
          </button>
        );
      })}
    </div>
  );
}

// ── Cycle ring ─────────────────────────────────────────────────
// One dot per day of the cycle, clockwise from the top: day 1 at 12 o'clock.

const RING_CENTER = 150;
const RING_RADIUS = 128;
/** Dot radius per marker — the notable days sit larger than the quiet ones. */
const DOT_RADIUS: Record<CycleDayMarker | 'none' | 'ahead', number> = {
  period: 7,
  fertile: 7,
  ovulation: 8,
  pms: 6,
  none: 5,
  ahead: 6,
};

interface RingDay {
  marker: CycleDayMarker | null;
  /** The day is still ahead of today — drawn hollow, as a prediction. */
  ahead: boolean;
}

function CycleRing({
  days, nowIndex, label, children,
}: {
  days: RingDay[];
  /** Index of the selected day, or null when the cycle can't be placed. */
  nowIndex: number | null;
  label: string;
  children: ReactNode;
}) {
  // useId() may contain characters a `url(#…)` reference can't carry.
  const glowId = `ring-glow-${useId().replace(/[^\w-]/g, '')}`;
  const count = Math.max(1, days.length);
  // Long cycles pack the dots closer — shrink them before they touch.
  const step = (2 * Math.PI * RING_RADIUS) / count;
  const scale = Math.min(1, step / 18);
  const at = (i: number) => {
    const angle = -Math.PI / 2 + (i * 2 * Math.PI) / count;
    return {
      cx: (RING_CENTER + RING_RADIUS * Math.cos(angle)).toFixed(1),
      cy: (RING_CENTER + RING_RADIUS * Math.sin(angle)).toFixed(1),
    };
  };
  const now = nowIndex != null ? days[nowIndex] : undefined;
  const nowKind = now?.marker ?? 'none';

  return (
    <>
      <svg className="home-ring-svg" viewBox="0 0 300 300" role="img" aria-label={label}>
        <defs>
          <filter id={glowId}>
            <feGaussianBlur stdDeviation="3" />
          </filter>
        </defs>
        <circle className="home-ring-track" cx={RING_CENTER} cy={RING_CENTER} r={RING_RADIUS} />
        {days.map((day, i) => {
          if (i === nowIndex) return null;
          const hollow = day.ahead && day.marker !== null;
          const kind = hollow ? 'ahead' : (day.marker ?? 'none');
          return (
            <circle
              key={i}
              {...at(i)}
              r={DOT_RADIUS[kind] * scale}
              className={clsx('home-ring-dot', day.marker && `is-${day.marker}`, hollow && 'is-ahead')}
            />
          );
        })}
        {nowIndex != null && (
          <>
            <circle {...at(nowIndex)} r={16 * scale} className={clsx('home-ring-halo', `is-${nowKind}`)} filter={`url(#${glowId})`} />
            <circle {...at(nowIndex)} r={10 * scale} className={clsx('home-ring-now', `is-${nowKind}`)} />
          </>
        )}
      </svg>
      <div className="home-ring-center">{children}</div>
    </>
  );
}

// ── Phase card ─────────────────────────────────────────────────
function PhaseCard({
  t, title, dotKind, fertilityLabel, description, showMore, loading, action,
}: {
  t: T;
  title: string;
  dotKind: CyclePhase | null;
  /** Engine fertility read-out for the day — informational, never diagnostic (§11). */
  fertilityLabel: string | null;
  description: string;
  /** The phase sheet reads today's live phase, so it's offered for today only. */
  showMore: boolean;
  loading: boolean;
  /** A call to action under the text (the empty state's «ثبت آخرین پریود»). */
  action?: { label: string; onClick: () => void };
}) {
  return (
    <section className={clsx('home-phase', loading && 'is-loading')}>
      <div className="home-phase-top">
        <div className="home-phase-name">
          {/* No phase to mark in the empty state — a brand dot would read as one. */}
          {!action && <span className={clsx('home-phase-dot', dotKind && `is-${dotKind}`)} />}
          <b>{title}</b>
        </div>
        {!loading && fertilityLabel && (
          <span className="home-phase-fert">
            {t('fertility.caption')}
            <span className="home-phase-fert-pill">{fertilityLabel}</span>
          </span>
        )}
      </div>
      <p className="home-phase-text">{loading ? t('unavailable') : description}</p>
      {showMore && (
        // The sheet reads the phase from live cycle data, never from the URL (§11).
        <button type="button" className="home-phase-more" onClick={() => openSheet('phase')}>
          {t('phaseCard.more')}
          <Icon name="chevronLeft" size={15} strokeWidth={2.2} className="home-phase-more-chev" />
        </button>
      )}
      {action && (
        <button type="button" className="home-phase-more" onClick={action.onClick}>
          {action.label}
          <Icon name="chevronLeft" size={15} strokeWidth={2.2} className="home-phase-more-chev" />
        </button>
      )}
    </section>
  );
}

// ── Daily recommendations ──────────────────────────────────────
/**
 * Category icon per engine tip `type`. Unknown/new backend categories fall back
 * to `sparkle` rather than rendering nothing, so a server-side addition can't
 * blank a row.
 */
const TIP_ICONS: Record<string, IconName> = {
  nutrition: 'apple',
  hydration: 'glass',
  warmth: 'flame',
  rest: 'moon',
  energy: 'zap',
  exercise: 'walk',
  fertility: 'heart',
  pms: 'drop',
  mood: 'smile',
  mental_health: 'brain',
  pain_relief: 'pill',
  sleep: 'moon',
  digestion: 'thermo',
};

/**
 * "توصیه‌های امروز" — the admin-managed recommendations the engine resolved for
 * today, carried on the cycle calculation we already fetch (no extra request).
 * Each tip shows its heading and the localized advice below it. When the engine
 * has no tips (incomplete profile), the short "do" suggestions from the daily
 * message stand in; with neither, the section renders nothing rather than
 * showing invented advice.
 */
/** Tint per row, in the artboard's order (data → warm → brand). */
const REC_TONES: Tone[] = ['data', 'warm', 'brand', 'bloom'];

function Recommendations({
  t, tips, dos, phase,
}: {
  t: T;
  tips: CycleDailyTip[];
  dos: string[];
  /** Today's phase — the pill beside the title (none while unknown). */
  phase: CyclePhase | null;
}) {
  const items = tips.length > 0
    ? tips.slice(0, 4).map(tip => ({
        // `hasOwn`, not `in`: `in` also matches Object.prototype keys, so a
        // category named `toString` would hand <Icon> a function and then ask
        // for a message key that doesn't exist.
        icon: Object.hasOwn(TIP_ICONS, tip.type) ? TIP_ICONS[tip.type] : ('sparkle' as IconName),
        // The backend resolves the heading (an admin's per-recommendation
        // override, else the category label), so a text edit in the panel shows
        // up without a client release. Only when it sends none do we fall back
        // to our own translation — and only for categories we ship a key for,
        // so a new backend category can't raise a missing-key error.
        title: tip.title
          ?? (Object.hasOwn(TIP_ICONS, tip.type)
            ? t(`recommendations.types.${tip.type}` as 'recommendations.types.nutrition')
            : t('recommendations.fallbackTitle')),
        desc: tip.text as string | undefined,
      }))
    : dos.slice(0, 4).map(text => ({
        icon: 'check' as IconName,
        title: text,
        desc: undefined as string | undefined,
      }));

  if (items.length === 0) return null;

  return (
    <Card as="section" className="ch-rec" aria-labelledby="ch-rec-title">
      <div className="ch-card-head">
        <h2 id="ch-rec-title" className="ch-card-title">{t('recommendations.title')}</h2>
        {phase && (
          <StatusPill tone={phase === 'period' ? 'period' : 'brand'} className="ch-rec-pill">
            {t(`nb.phasePill.${phase}`)}
          </StatusPill>
        )}
      </div>
      {items.map((item, i) => (
        <div key={i} className="ch-rec-row">
          <IconCircle icon={item.icon} tone={REC_TONES[i % REC_TONES.length]} />
          <div className="ch-rec-body">
            <b className="ch-rec-name">{item.title}</b>
            {item.desc && <p className="ch-rec-desc">{item.desc}</p>}
          </div>
        </div>
      ))}
    </Card>
  );
}

// ── Cycle-based articles (published from the admin panel) ─────────
// Figma «Frame 36»: white card, 18px title, 162px blog cards, primary CTA.
// The list comes from `GET /home/sections/articles`, which returns what an
// admin tagged for the phase the user is in today (plus general articles), so
// nothing here is hardcoded. The whole section disappears when there is
// nothing published for this phase.
function Articles({ t, locale }: { t: T; locale: Locale }) {
  const tArticles = useTranslations('articles');
  // `categories.<slug>` keys are data-driven, so the literal-key typing can't see them.
  const categoryLabel = (name: string): string =>
    articleCategoryLabel(name, tArticles as unknown as CategoryTranslator);
  const { data, isPending } = useCycleArticles();
  const articles = data?.articles ?? [];

  if (isPending) return <ArticlesSkeleton t={t} />;
  if (articles.length === 0) return null;

  return (
    <div className="sec">
      <div className="home-articles">
        {/* The backend words and localizes the heading; the message file is
            the fallback for when the section omits it. */}
        <div className="home-articles-title">{data?.title ?? t('articles.title')}</div>
        <div className="scroll-x">
          <div className="home-articles-track">
            {articles.map(article => (
              // The card opens the article sheet; the slug is public content, so
              // unlike the cycle phase it is safe to carry in the URL (§11).
              <button
                key={article.id}
                type="button"
                onClick={() => openSheet('article', article.slug)}
                className="home-article"
              >
                <div className="home-article-cover">
                  {article.imageUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={article.imageUrl}
                      alt=""
                      loading="lazy"
                      draggable={false}
                      className="home-article-img"
                    />
                  ) : (
                    <Icon name="bookOpen" size={30} stroke="currentColor" />
                  )}
                </div>
                <div className="home-article-name">{article.title}</div>
                {(article.readTimeMinutes !== null || article.category) && (
                  <div className="home-article-meta">
                    <Icon name="bookOpen" size={16} stroke="currentColor" />
                    {article.readTimeMinutes !== null
                      ? t('articles.min', { n: localizeNum(article.readTimeMinutes, locale) })
                      : categoryLabel(article.category ?? '')}
                  </div>
                )}
              </button>
            ))}
          </div>
        </div>
        <button
          type="button"
          onClick={() => openSheet('articles')}
          className="btn btn-primary home-articles-cta"
        >
          {t('articles.readMore')}
        </button>
      </div>
    </div>
  );
}

/** Placeholder cards while the articles load — same rhythm as the real row. */
function ArticlesSkeleton({ t }: { t: T }) {
  return (
    <div className="sec">
      <div className="home-articles" aria-hidden>
        <div className="home-articles-title">{t('articles.title')}</div>
        <div className="scroll-x">
          <div className="home-articles-track">
            {[0, 1, 2].map(i => (
              <div key={i} className="home-article">
                <span className="skeleton-line home-article-skel" />
                {['90%', '55%'].map(width => (
                  <span key={width} className="skeleton-line" style={{ width }} />
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

/** One upcoming (or running) cycle event: its dates and the countdown to its start. */
interface TimelineSlot {
  start: Date;
  end: Date;
  days: number;
}

// ── Cycle-engine home ──────────────────────────────────────────
/**
 * The cycle-engine home: cycle, TTC (`v19_Main`), teen (simplified) and the
 * interim postpartum home. `lifeMode` is the effective life-stage mode
 * (B-N2-03); `null` = unknown / no life-stage route → the legacy layout.
 */
function CycleHome({ lifeMode }: { lifeMode: LifeMode | null }) {
  const t = useTranslations('home');
  // Teen (gaps.md #3): simplified cycle home — no ads (banners), no fertility
  // read-outs, no PMS insight. Postpartum runs the cycle home until B-N5 with a
  // "coming soon" note on top (QUESTIONS #60).
  const teen = lifeMode === 'teen';
  const postpartum = lifeMode === 'postpartum';
  const ttcAllowed = lifeMode === null || lifeMode === 'ttc' || lifeMode === 'cycle';
  // Chance labels and the TTC home copy live with the fertility screens, so the
  // home and «ثبت روز» say the same word for the same level (audit #1).
  const tf = useTranslations('fertility');
  const loc = useLocale() as Locale;
  // This route is statically prerendered, so anything derived from "now" would
  // be frozen at build time in the server HTML and disagree with the client on
  // the next day — a hydration text mismatch (React #418). The whole screen is
  // date-driven, so it renders only after mount; its data is client-fetched
  // anyway, so nothing meaningful is lost from the prerendered HTML.
  const mounted = useMounted();
  const [dateEditorOpen, setDateEditorOpen] = useState(false);
  // The §12 sync nudge edits the profile in place, in the same bottom sheet the
  // profile screen uses for cycle length — no detour through /profile/health.
  const [cycleSheetOpen, setCycleSheetOpen] = useState(false);
  // The day the user tapped in the week strip (defaults to today). Only the
  // ring and the phase card reflect it; the rest of the page stays on today.
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());

  // Server state (§8) — cycle math + personalized message for today.
  const todayQuery = useCycleToday();
  const todayData = todayQuery.data;
  const { data: daily } = useDailyMessage();
  const profileQuery = useUserProfile();
  // The layout the last fresh profile decided (read after mount: localStorage is
  // client-only). Lets a cold load skip waiting on `/profile` (T-M5-12).
  const ttcHint = useMemo(() => (mounted ? readTtcHint() : null), [mounted]);
  // TTC home (`v19_Main`): only while trying to conceive; pregnancy mode never matches.
  // The fresh profile wins; until it lands, the remembered layout stands in.
  const isTtc =
    ttcAllowed &&
    (profileQuery.data ? profileQuery.data.health?.pregnancyIntention === 'trying' : ttcHint === true);
  useEffect(() => {
    if (profileQuery.data) writeTtcHint(isTtc);
  }, [profileQuery.data, isTtc]);
  // Same cache as the tiles (no extra request): the LH tip hides once today's test is logged.
  const fertilityToday = useFertilityToday();

  const calc = todayData?.calculation ?? null;

  // While the backend recalculates, poll status and refetch today's calc once it
  // settles, so the page reflects the fresh result without a manual reload.
  const recalculating = Boolean(todayData?.isRecalculating);
  // Only asked while a recalculation is in flight — on a normal cold load the
  // status is never read, and skipping it saves a request + CORS preflight.
  const { data: cycleStatus } = useCycleStatus({ poll: recalculating, enabled: recalculating });
  const settled = cycleStatus ? !cycleStatus.is_processing : false;
  useEffect(() => {
    if (recalculating && settled) {
      void todayQuery.refetch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recalculating, settled]);

  const base = today();
  // The cycle's own calendar (engine anchors → absolute dates). Everything the
  // page shows about upcoming events hangs off this, so the dates stay tied to
  // the period start rather than being re-measured from today each render.
  const schedule = deriveCycleSchedule(todayData?.cycleView ?? null, calc);
  // Calendar formatting happens only here, at the display boundary (§7).
  const fmt = (date: Date) => formatDayMonth(date, loc);
  // «۱۷ تا ۲۱ مهر»: a range inside one month names the month once.
  const range = (from: Date, to: Date) => {
    const a = toParts(from, loc);
    const b = toParts(to, loc);
    const sameMonth = a.year === b.year && a.month === b.month;
    return t('dateRange', { from: sameMonth ? formatNumber(a.day, loc) : fmt(from), to: fmt(to) });
  };
  // Every prediction row describes the occurrence the user is actually waiting on:
  // an event whose window is already running is reported as in-progress, and one
  // that is wholly behind us rolls forward a cycle.
  const slotFor = (start: Date, end: Date): TimelineSlot | null => {
    if (!schedule) return null;
    let from = start;
    let to = end;
    while (diffInDays(to, base) < 0) {
      from = addDays(from, schedule.cycleLength);
      to = addDays(to, schedule.cycleLength);
    }
    return { start: from, end: to, days: diffInDays(from, base) };
  };
  // `/home/cycle-overview` (B-N1-06): the engine's period range, PMS window,
  // streak and cycle-length summary. Absent on a backend without the route —
  // the schedule derived from `cycle_view` stands in.
  const overviewQuery = useCycleOverview();
  const overview = overviewQuery.data ?? null;
  const periodDays =
    todayData?.cycleView?.effectiveValues.periodDuration ??
    profileQuery.data?.health?.periodDuration ??
    DEFAULT_PERIOD_DAYS;
  const nextPeriodSlot = overview?.nextPeriod
    ? slotFor(fromApiDate(overview.nextPeriod.start), fromApiDate(overview.nextPeriod.end))
    : schedule
      ? slotFor(schedule.nextPeriodStart, addDays(schedule.nextPeriodStart, periodDays - 1))
      : null;
  const ovulationSlot = schedule ? slotFor(schedule.ovulation, schedule.ovulation) : null;
  // The §19 display window — the same days `/fertility/bbt` and the insights
  // screen mark (none when a long period swallows it).
  const windowSlot =
    schedule && hasFertileWindow(schedule)
      ? slotFor(schedule.fertileStart, schedule.fertileEnd)
      : null;
  const pmsSlot = overview?.pmsWindow
    ? slotFor(fromApiDate(overview.pmsWindow.start), fromApiDate(overview.pmsWindow.end))
    : schedule
      ? slotFor(schedule.pmsStart, schedule.pmsEnd)
      : null;
  const nextPeriodDate = nextPeriodSlot ? fmt(nextPeriodSlot.start) : null;
  const ovulationDate = ovulationSlot ? fmt(ovulationSlot.start) : null;
  const toPrediction = (slot: TimelineSlot | null, single = false): PredictionSlot | null =>
    slot && {
      text: single ? fmt(slot.start) : range(slot.start, slot.end),
      startIn: slot.days,
      endIn: diffInDays(slot.end, base),
    };

  const message: DailyMessage | null = daily ?? null;
  const dos = message?.primary.dos ?? [];

  // ── Selected-day info for the ring and the phase card ──
  const selApiDate = toApiDate(selectedDate);
  const isToday = selApiDate === toApiDate(base);

  // A tapped past/future day loads its own calculation + message; today reuses
  // the queries above (same cache keys — no extra fetch).
  const { data: dateData, isFetching: dateFetching } = useCycleForDate(selApiDate, !isToday);
  const { data: infoMessage } = useDailyMessage(isToday ? undefined : selApiDate);

  const infoCalc = (isToday ? todayData : dateData)?.calculation ?? null;
  const infoPred = infoCalc ? deriveCyclePredictions(infoCalc) : null;

  // The countdown measures the selected day against *its own* cycle's predicted
  // start: the engine's anchors for that day once they load (a past day then
  // shows the period it actually led to), and until then today's schedule rolled
  // whole cycles — so scrubbing counts down smoothly instead of jumping.
  const selectedSchedule: CycleSchedule | null =
    (isToday ? null : deriveCycleSchedule(dateData?.cycleView ?? null, dateData?.calculation ?? null)) ??
    (schedule ? cycleScheduleFor(schedule, selectedDate) : null);
  // The engine's own read-out for the selected day — the very same `cycle_view`
  // that renders the day-status card — so the ring can never disagree with it.
  const infoView = (isToday ? todayData : dateData)?.cycleView ?? null;
  // The engine's main phase wins over the legacy calculation's phase.
  const infoPhase: CyclePhase | null = phaseFromMainPhase(infoView?.mainPhase) ?? infoPred?.phase ?? null;
  const infoDaysUntilNextPeriod =
    infoView?.daysToPeriod ??
    (selectedSchedule ? daysUntilNextPeriod(selectedSchedule, selectedDate) : null);
  // Day X of N, again straight from the engine (falls back to the local derivation).
  const infoCycleDay = infoView?.cycleDay ?? infoPred?.cycleDay ?? null;
  const infoCycleLength =
    infoView?.effectiveValues.cycleLength ?? infoPred?.cycleLength ?? null;
  // The line under the phase title is the engine's own day-status copy — the
  // exact subtitle the day-status card shows (§19) — so the two never tell the
  // user different things about the same day. Behind it: the personalized
  // message for today, then a tense-neutral phase blurb (a tapped day must
  // never say «امروز»).
  // The v1.1 top-level level (§26) — what `/fertility/*` reads too — not the
  // daily card's legacy calendar mapping; `unknown` shows no pill (audit #1).
  const infoFertilityLevel = infoView?.fertilityLevel ?? null;
  const infoFertilityLabel =
    !teen && infoFertilityLevel && infoFertilityLevel !== 'unknown'
      ? tf(`chance.levels.${infoFertilityLevel}`)
      : null;
  const infoPhaseDesc =
    infoView?.dailyCard?.subtitle ||
    (isToday
      ? (infoMessage?.primary.shortMessage || t('nextPeriod.phaseDesc'))
      : (infoPhase ? t(`phaseDescription.${infoPhase}`) : t('nextPeriod.phaseDesc')));
  // Selecting a past/future day fetches its data; dim the ring and the card
  // meanwhile so the placeholder values read as "loading", not as a broken
  // empty state (§ loading).
  const infoLoading = !isToday && dateFetching && !dateData;
  // First load: until today's calculation (and the banner slot above the
  // timeline) has settled, the ring shows its loading state and nothing is
  // rendered below it. Content that appears is not a layout shift; content
  // that is already on screen and gets pushed down by the ring growing or a
  // banner landing is — that was the home CLS (perf baseline §1.4, §3 #5).
  // It also lets the below-the-fold reads (challenge, articles) start after
  // the critical ones instead of competing with them (§3 #8).
  const bannersSettled = useBannersSettled();
  // Only while a request is actually in flight: a disabled or offline-paused
  // query must never hold the page back.
  // The profile decides whether this is the TTC home (tiles, chance card, ring
  // centre), so it is part of the boot too: the ring and phase card hold their
  // loading state instead of the TTC blocks landing late and pushing the page
  // down (audit #27). It is already fetched in parallel with today's cycle.
  // With a remembered layout the page doesn't wait on it at all: a slow `/profile`
  // (seen at ~5 s on stage) then only refreshes the hint in the background.
  const profileBooting =
    profileQuery.isPending && profileQuery.fetchStatus === 'fetching' && ttcHint === null;
  const booting =
    (todayQuery.isPending && todayQuery.fetchStatus === 'fetching') ||
    !bannersSettled ||
    profileBooting;
  const loadingInfo = infoLoading || booting;
  // No period on record (e.g. right after leaving pregnancy mode): ask for the
  // last period instead of a ring and a phase card full of «—». The daily
  // message is null in this state (the API's 400, see entities/message).
  const noPeriodData =
    !booting && todayQuery.isSuccess && needsPeriodData(todayData?.cycleView ?? null, calc);

  // ── The ring: the selected day's own cycle, day 1 at the top ──
  const periodLength =
    infoView?.effectiveValues.periodDuration ??
    profileQuery.data?.health?.periodDuration ??
    DEFAULT_PERIOD_DAYS;
  const ringCycleDay =
    infoCycleDay ??
    (selectedSchedule ? diffInDays(selectedDate, selectedSchedule.cycleStart) + 1 : null);
  // A late period runs past the expected length — the ring grows to hold it.
  const ringLength = Math.max(
    infoCycleLength ?? selectedSchedule?.cycleLength ?? DEFAULT_CYCLE_DAYS,
    ringCycleDay ?? 1,
  );
  const ringStart = ringCycleDay != null ? addDays(selectedDate, -(ringCycleDay - 1)) : null;
  const weekDays = weekOf(base, loc);
  const ringDates = ringStart
    ? Array.from({ length: ringLength }, (_, i) => addDays(ringStart, i))
    : [];
  // Today's anchored schedule (not the tapped day's): the window it gives the
  // current and predicted cycles must not move when another day is selected.
  const rawMarkOf = useDayMarks([...weekDays, ...ringDates], schedule ?? selectedSchedule, periodLength);
  // Teen (B-N2-10 audit): no fertile-window / ovulation dots on the ring or the week strip either.
  const markOf = (date: Date): CycleDayMarker | null => {
    const mark = rawMarkOf(date);
    return teen && (mark === 'fertile' || mark === 'ovulation') ? null : mark;
  };
  const ringDays = ringDates.map(date => ({
    marker: markOf(date),
    ahead: diffInDays(date, base) > 0,
  }));
  const ringNowIndex = ringStart && !loadingInfo ? diffInDays(selectedDate, ringStart) : null;

  const inPeriod = infoPhase === 'period';
  const daysLeft = infoDaysUntilNextPeriod;
  const dash = t('unavailable');

  // ── Hero variant (nbl_Cycle_Home / _Near / _During) ──
  // The prompts only act on today; «هنوز نه» / «هنوز ادامه دارد» hide them until tomorrow.
  const todayIso = toApiDate(base);
  const [dismissTick, setDismissTick] = useState(0);
  const startDismissed = mounted && dismissTick >= 0 && isPromptDismissed('start', todayIso);
  const endDismissed = mounted && dismissTick >= 0 && isPromptDismissed('end', todayIso);
  const dismiss = (kind: 'start' | 'end') => {
    dismissPrompt(kind, todayIso);
    setDismissTick((n) => n + 1);
  };
  const hero = heroState({
    isToday: isToday && !loadingInfo && !noPeriodData,
    inPeriod,
    daysLeft,
    daysLate: infoView?.daysLate ?? null,
    periodOpen: infoView?.anchors?.currentPeriodEndIsConfirmed === false,
    cycleDay: infoCycleDay,
    periodLength,
    startDismissed,
    endDismissed,
  });
  const startPeriod = useStartPeriod();
  const endPeriod = useEndPeriod();
  const daysLate = infoView?.daysLate ?? 0;

  // TTC ring centre (`v19_Main`): «تخمک‌گذاری تا N روز» while ovulation is still
  // ahead in this cycle; once it has passed (or during the period) the ring
  // counts down to the next period like every other cycle home.
  const daysToOvulation = infoView?.daysToOvulation ?? null;
  const ttcRing =
    isTtc && !noPeriodData && !loadingInfo && !inPeriod && daysToOvulation != null && daysToOvulation >= 0;
  const ringOverline = noPeriodData
    ? t('ring.noData')
    : ttcRing
      ? tf('home.ovulationIn')
      : inPeriod && !loadingInfo
        ? t('ring.period')
        : hero.variant === 'late'
          ? t('nb.hero.late')
          : hero.variant === 'near' && daysLeft !== 0
            ? t('nb.hero.periodIn')
            : t('ring.nextPeriod');
  const ringNumber = loadingInfo
    ? dash
    : ttcRing
      ? daysToOvulation === 0
        ? t('ring.today')
        : formatNumber(daysToOvulation, loc)
      : inPeriod && infoCycleDay != null
        ? t('ring.periodDay', { n: infoCycleDay })
        : hero.variant === 'late'
          ? formatNumber(daysLate, loc)
          : daysLeft == null
            ? dash
            : daysLeft === 0
              ? t('ring.today')
              : formatNumber(daysLeft, loc);
  const ringUnit = loadingInfo
    ? null
    : ttcRing
      ? daysToOvulation > 0
        ? tf('home.days')
        : null
      : hero.variant === 'late'
        ? t('nb.hero.lateUnit')
        : !inPeriod && daysLeft != null && daysLeft > 0
          ? t('ring.daysLeft', { n: daysLeft })
          : null;
  const ringSub = loadingInfo
    ? null
    : inPeriod
      ? t('ring.usually', { n: periodLength })
      : infoCycleDay != null
        ? isTtc && infoFertilityLabel
          ? tf('home.ringCaption', { day: formatNumber(infoCycleDay, loc), level: infoFertilityLabel })
          : t('ring.cycleDay', { n: infoCycleDay })
        : null;
  // «ثبت امروز» opens the day log for the selected day (never a future one).
  const logHref = selApiDate < toApiDate(base) ? `/fertility/log?date=${selApiDate}` : '/fertility/log';
  const ringLabel = ringCycleDay != null
    ? t('ring.label', { day: ringCycleDay, length: ringLength })
    : t('ring.labelEmpty');

  // ── The phase card ──
  const nearPeriod = hero.variant === 'near' || hero.variant === 'late';
  const phaseTitle = loadingInfo || !infoPhase
    ? dash
    : nearPeriod
      ? t('phaseCard.nearPeriod')
      : t(`phaseCard.title.${infoPhase}`, { n: infoCycleDay ?? infoPred?.cycleDay ?? 1 });
  const phaseDot: CyclePhase | null = loadingInfo || !infoPhase ? null : nearPeriod ? 'period' : infoPhase;
  const phaseText =
    !isToday || loadingInfo
      ? infoPhaseDesc
      : hero.variant === 'late'
        ? t('nb.lateText', { n: formatNumber(daysLate, loc) })
        : hero.variant === 'near' && daysLeft != null
          ? t('nb.nearText', { days: daysLeft, n: formatNumber(daysLeft, loc) })
          : infoPhaseDesc;

  // ── TTC blocks (`v19_Main`) ──
  const chanceTitle = isToday
    ? tf('home.chanceCard.title')
    : tf('home.chanceCard.titleDay', { date: fmt(selectedDate) });
  // In the window the line names the days the ring counts down to (ovulation
  // in N days), never a fixed «۲ روز» that only fits one day of the window.
  const chanceDesc =
    isToday && infoView?.mainPhase === 'fertile' && daysToOvulation != null && daysToOvulation >= 0
      ? tf('home.chanceCard.inWindow', { days: daysToOvulation, n: formatNumber(daysToOvulation, loc) })
      : infoPhaseDesc;
  // «امروز تست LH بزن»: today is in the fertile window and no LH test is logged yet.
  const showLhTip =
    isTtc &&
    todayData?.cycleView?.mainPhase === 'fertile' &&
    fertilityToday.data != null &&
    fertilityToday.data.lh.value === null;

  // Today's data failed and nothing is cached: an error card with a retry
  // instead of a ring and cards full of «—».
  const loadFailed = !booting && todayQuery.isError && !todayData;
  const todayPred = calc ? deriveCyclePredictions(calc) : null;
  const todayPhase: CyclePhase | null =
    phaseFromMainPhase(todayData?.cycleView?.mainPhase) ?? todayPred?.phase ?? null;

  // «طول سیکل (میانه N سیکل)» — the engine's summary; without the overview
  // route the effective length alone stands in.
  const cycleLengthSummary = overview
    ? {
        median: overview.cycleLength.median,
        basedOnCycles: overview.cycleLength.basedOnCycles,
        spreadDays: overview.cycleLength.spreadDays,
        regularity: overview.cycleLength.regularity,
      }
    : todayData?.cycleView?.effectiveValues.cycleLength != null
      ? {
          median: todayData.cycleView.effectiveValues.cycleLength,
          basedOnCycles: null,
          spreadDays: null,
          regularity: null,
        }
      : null;

  // The §12 sync nudge: only when recent cycles disagree with the profile value.
  const view = todayData?.cycleView ?? null;
  const suggested = view?.calculatedValues.cycleLength ?? null;
  const logged = view?.profileValues.cycleLength ?? null;
  const syncNudge =
    view && suggested != null && logged != null && suggested !== logged ? (
      <CycleValuesCard
        title={t('values.title')}
        loggedLabel={t('values.logged')}
        loggedValue={t('days', { n: logged })}
        suggestion={{
          text: t('values.suggestion', { n: suggested }),
          ctaLabel: t('values.syncCta', { n: suggested }),
          onSync: () => setCycleSheetOpen(true),
        }}
        basedOnText={t('values.basedOn', {
          source: t(
            view.effectiveValues.source === 'recent_valid_cycles'
              ? 'values.source.recent_valid_cycles'
              : view.effectiveValues.source === 'profile'
                ? 'values.source.profile'
                : 'values.source.default',
          ),
        })}
      />
    ) : null;

  // Server pass / first client render: backdrop only, so both sides match.
  if (!mounted) {
    return (
      <div className="view">
        <div className="home-grad home-grad-fill" />
      </div>
    );
  }

  return (
    <div className="view">
      {/* Full-page gradient backdrop (lavender → soft turquoise, §10.2) */}
      <div className="home-grad home-grad-fill" />

      <div className="scroll page-scroll">
        <div className="home-top">
          <HomeHeader t={t} loc={loc} />
          {recalculating && (
            <div className="page-updating">
              {t('updating')}
            </div>
          )}
          <WeekStrip
            days={weekDays}
            loc={loc}
            selectedIso={selApiDate}
            onSelect={setSelectedDate}
            markOf={markOf}
          />
          <section className={clsx('home-ring', loadingInfo && 'is-loading')}>
            <div
              className={clsx('home-ring-glow', ttcRing && 'is-ttc', !ttcRing && hero.variant !== 'normal' && 'is-period')}
              aria-hidden
            />
            <CycleRing days={ringDays} nowIndex={ringNowIndex} label={ringLabel}>
              {/* `--period` red means menstruation (§10.2); "nothing logged yet"
                  is not a period, so the empty state takes the neutral caption. */}
              <span className={noPeriodData ? 'home-ring-sub' : clsx('home-ring-over', ttcRing && 'is-ttc')}>
                {ringOverline}
              </span>
              {!noPeriodData && (
                <div className="home-ring-big">
                  <span className="home-ring-num">{ringNumber}</span>
                  {ringUnit && <span className="home-ring-unit">{ringUnit}</span>}
                </div>
              )}
              {ringSub && !noPeriodData && <span className="home-ring-sub">{ringSub}</span>}
              {ttcRing ? (
                <Link href={logHref} className="home-ring-edit is-ink">
                  <Icon name="plus" size={16} strokeWidth={2.4} />
                  {tf('home.logToday')}
                </Link>
              ) : hero.startPrompt ? (
                // «پریودم شروع شد / هنوز نه» (nbl_Cycle_Home_Near): logs today as day 1.
                <>
                  <button
                    type="button"
                    className="home-ring-cta is-period"
                    onClick={() => startPeriod.mutate()}
                    disabled={startPeriod.isPending}
                  >
                    <Icon name="drop" size={16} strokeWidth={2.2} />
                    {t('nb.hero.started')}
                  </button>
                  <button type="button" className="home-ring-later" onClick={() => dismiss('start')}>
                    {t('nb.hero.notYet')}
                  </button>
                </>
              ) : hero.endPrompt ? (
                // «پریودم تموم شد / هنوز ادامه دارد» (nbl_Cycle_Home_During): closes the open period today.
                <>
                  <button
                    type="button"
                    className="home-ring-cta"
                    onClick={() => endPeriod.mutate()}
                    disabled={endPeriod.isPending}
                  >
                    <Icon name="check" size={16} strokeWidth={2.4} />
                    {t('nb.hero.ended')}
                  </button>
                  <button type="button" className="home-ring-later" onClick={() => dismiss('end')}>
                    {t('nb.hero.stillGoing')}
                  </button>
                </>
              ) : (
                // The date editor rises over the home screen itself (§4.1).
                <button type="button" className="home-ring-edit" onClick={() => setDateEditorOpen(true)}>
                  <Icon name="pen" size={16} strokeWidth={2.2} />
                  {noPeriodData ? t('ring.logPeriod') : t('ring.editPeriod')}
                </button>
              )}
            </CycleRing>
          </section>
          {hero.endMissing && (
            <div role="status" className="ch-alert">
              <IconCircle icon="info" tone="warm" size="sm" />
              <span className="ch-alert-body">
                <b className="ch-alert-title">{t('nb.endMissing.title')}</b>
                <span className="ch-alert-text">{t('nb.endMissing.text')}</span>
              </span>
            </div>
          )}
          {loadFailed ? (
            <Card as="section" className="ch-error" role="alert">
              <IconCircle icon="warning" tone="danger" />
              <p className="ch-error-text">{t('nb.loadError')}</p>
              <PrimaryButton block={false} className="ch-error-retry" onClick={() => void todayQuery.refetch()}>
                {t('nb.retry')}
              </PrimaryButton>
            </Card>
          ) : noPeriodData ? (
            <PhaseCard
              t={t}
              title={t('phaseCard.noData.title')}
              dotKind={null}
              fertilityLabel={null}
              description={t('phaseCard.noData.text')}
              showMore={false}
              loading={false}
              action={{ label: t('phaseCard.noData.cta'), onClick: () => setDateEditorOpen(true) }}
            />
          ) : isTtc ? (
            <>
              <TtcPhasePills active={loadingInfo ? null : ttcPhase(infoView?.mainPhase ?? null)} />
              <FertilityChanceCard
                title={chanceTitle}
                level={loadingInfo ? null : infoFertilityLevel}
                levelLabel={loadingInfo ? dash : (infoFertilityLabel ?? tf('chance.levels.unknown'))}
                description={loadingInfo ? t('unavailable') : chanceDesc}
                windowStart={windowSlot ? fmt(windowSlot.start) : null}
                ovulation={ovulationDate}
                nextPeriod={nextPeriodDate}
                loading={loadingInfo}
              />
            </>
          ) : (
            <PhaseCard
              t={t}
              title={phaseTitle}
              dotKind={phaseDot}
              fertilityLabel={infoFertilityLabel}
              description={phaseText}
              showMore={isToday && Boolean(todayData?.cycleView?.subphase)}
              loading={loadingInfo}
            />
          )}
          {/* TTC quick tiles (M5) — under the chance card, then the LH tip (`v19_Main`). */}
          {isTtc && !booting && <FertilityTiles />}
          {showLhTip && !booting && <LhTipCard />}
        </div>
        {booting ? (
          // First load: card-shaped placeholders in the feed's rhythm (no layout jump when it lands).
          <SkeletonGroup label={t('nb.loading')} className="ch-feed">
            <Skeleton shape="card" />
            <Skeleton shape="card" />
            <Skeleton shape="card" />
          </SkeletonGroup>
        ) : (
          <div className="ch-feed">
            {/* Admin-managed promo slot — renders nothing until a banner is active */}
            {postpartum && <PostpartumNotice />}
            {/* CB-LOSS-03b: a quiet, dismissible way back into /loss/care for 60 days after a loss. */}
            {!teen && <LossCareReturn variant="home" />}
            {/* CB-TEEN-03: a parent's read-only teen cards (nothing for everyone else). */}
            {!teen && <LinkedTeenCards />}
            {!teen && <BannerSlideshow position="home_top" />}
            {/* «یادآورهای امروز» — today's doses + next appointment (M3, /care/today),
                right under the hero like the reminders artboard (T-M3-10 audit L-6). */}
            <TodayRemindersCard />
            {/* «ثبت امروز» + streak (nbl_Cycle_Home). The TTC home logs through its tiles. */}
            {!noPeriodData && !loadFailed && !isTtc && <TodayLogCard streakDays={overview?.streakDays ?? null} />}
            {!noPeriodData && !loadFailed && (
              <PredictionsCard
                nextPeriod={toPrediction(nextPeriodSlot)}
                pms={toPrediction(pmsSlot)}
                window={teen ? null : toPrediction(windowSlot)}
                ovulation={teen ? null : ovulationDate}
                hideFertility={teen}
                length={cycleLengthSummary}
                footer={syncNudge}
              />
            )}
            {/* «چکاپ‌های دوره‌ای» (M4, /checkups/home) — after the cycle predictions. */}
            <CheckupsCard />
            <Recommendations
              t={t}
              tips={calc?.dailyTips ?? []}
              dos={dos}
              phase={todayPhase}
            />
            {!teen && <BannerSlideshow position="home_middle" />}
            <TodayChallengeCard />
            {!noPeriodData && !loadFailed && !teen && (
              <PmsInsightCard pms={toPrediction(pmsSlot)} basedOnCycles={overview?.cycleLength.basedOnCycles ?? null} />
            )}
            <Articles t={t} locale={loc} />
            {!teen && <BannerSlideshow position="home_bottom" />}
          </div>
        )}
        <div className="page-tail" />
      </div>

      {/* Seeded with what recent cycles measured, so the wheel opens on the
          value the nudge proposes and saving is a single tap. */}
      <QuickEditSheet
        field={cycleSheetOpen ? 'cycleDuration' : null}
        values={{
          cycleDuration:
            todayData?.cycleView?.calculatedValues.cycleLength ??
            todayData?.cycleView?.profileValues.cycleLength ??
            null,
        }}
        onClose={() => setCycleSheetOpen(false)}
      />

      {/* It portals to `.app-shell`, so mounting it here still covers the
          whole screen. */}
      <PeriodDateEditor open={dateEditorOpen} onClose={() => setDateEditorOpen(false)} />

      {/* B-N2-08: trial-offer banner + sheet — never for teen (QUESTIONS #70/#72). */}
      {!teen && <PlusTrialOffer />}
      <BottomNav />
    </div>
  );
}

/** Interim postpartum note (QUESTIONS #60): the B-N5 home replaces this whole layout. */
function PostpartumNotice() {
  const t = useTranslations('home.life.postpartum');
  return (
    <Card as="section" className="life-note" aria-labelledby="life-pp-title">
      <IconCircle icon="heart" tone="data" size="lg" />
      <span className="life-note-text">
        <b id="life-pp-title" className="life-note-title">
          {t('title')}
        </b>
        <span className="life-note-body">{t('body')}</span>
        <Link href="/profile/mode" className="life-note-link">
          {t('action')}
        </Link>
      </span>
    </Card>
  );
}

function LifeHomeHeader() {
  const t = useTranslations('home');
  const loc = useLocale() as Locale;
  return <HomeHeader t={t} loc={loc} />;
}

/** The legacy mode where `/profile/life-stage` is missing (production on Laravel). */
function legacyLifeMode(mode: string | undefined, isTtc: boolean | undefined): LifeMode {
  if (mode === 'pregnancy' || mode === 'postpartum') return mode;
  return isTtc ? 'ttc' : 'cycle';
}

// ── Main export ────────────────────────────────────────────────
/**
 * `/home` — mode-aware (B-N2-03): the effective life-stage mode picks the home.
 * Pregnancy never shows the cycle home — a deep link to `/home` goes to
 * `/pregnancy` (stage smoke B-2); menopause gets its own minimal home; every
 * other mode runs the cycle-engine home. A remembered mode (wiped at sign-out)
 * lets a cold load start on the right layout; the fresh value always wins.
 */
export function HomePage() {
  const router = useRouter();
  const mounted = useMounted();
  const life = useLifeStage();
  const legacy = useUserMode();
  const hint = useMemo(() => (mounted ? readLifeModeHint() : null), [mounted]);
  const fresh: LifeMode | null =
    life.data?.mode ?? (life.isError && legacy.data ? legacyLifeMode(legacy.data.mode, legacy.data.isTtc) : null);
  // Both reads settled without an answer: don't trust the hint (a stale
  // «pregnancy» would hold the backdrop forever) — fall back to the cycle home.
  const unresolved = life.isError && !legacy.data && !(legacy.isPending && legacy.fetchStatus !== 'idle');
  const mode = fresh ?? (unresolved ? null : hint);
  // CB-TEEN-04b: with no remembered mode, wait for `/profile/life-stage` instead of mounting the cycle home
  // for a moment — it fires the cycle-only reads (`/messages/daily`, which 400s without period data) at a
  // teen whose home never shows them.
  const waiting = !fresh && !hint && life.isPending && life.fetchStatus !== 'idle';
  // B-N4-05 (N2 stage bug B-1): a male companion account has no cycle home —
  // `/home*` answers 409 `companion_account` — so it goes to the companion panel.
  const companion = Boolean(life.data?.companion) || getApiErrorCode(legacy.error) === 'companion_account';
  // CB-IVF-02: TTC with bloom's «IVF/IUI» switch on → the IVF home is «امروز».
  const ivf = fresh === 'ttc' && Boolean(life.data?.ivfIui);

  useEffect(() => {
    if (companion) {
      router.replace('/companion');
      return;
    }
    if (fresh) writeLifeModeHint(fresh);
    if (fresh === 'pregnancy') router.replace('/pregnancy');
    // B-N5-04 (N3 stage bug B-4): postpartum has its own home — never the cycle home.
    if (fresh === 'postpartum') router.replace('/postpartum');
    if (ivf) router.replace('/ivf');
  }, [companion, fresh, ivf, router]);

  // Server pass, first client render, an unknown mode, and the pregnancy / postpartum / IVF / companion
  // hand-off: backdrop only.
  if (!mounted || waiting || mode === 'pregnancy' || mode === 'postpartum' || ivf || companion) {
    return (
      <div className="view">
        <div className="home-grad home-grad-fill" />
      </div>
    );
  }
  if (mode === 'menopause') return <MenopauseHome header={<LifeHomeHeader />} />;
  // CB-TEEN-02: the canvas teen home (nbl_Teen_Home) replaces the simplified cycle home.
  if (mode === 'teen') {
    return (
      <div className="view">
        <div className="home-grad home-grad-fill" />
        <div className="scroll page-scroll">
          <TeenHome />
          <div className="page-tail" />
        </div>
        <BottomNav />
      </div>
    );
  }
  return <CycleHome lifeMode={mode} />;
}
