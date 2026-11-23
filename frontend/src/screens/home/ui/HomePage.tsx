'use client';

import clsx from 'clsx';
import dynamic from 'next/dynamic';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useMemo, useState, type ReactNode } from 'react';

import {
  articleCategoryLabel,
  type CategoryTranslator,
  useCycleArticles,
} from '@/entities/article';
import { useAppModules } from '@/entities/app-module';
import { useBannersSettled } from '@/entities/banner';
import {
  cycleDayMarkerAt,
  cycleScheduleFor,
  daysUntilNextPeriod,
  deriveCycleSchedule,
  deriveCyclePredictions,
  fertileLevelAt,
  futureCycleDayMap,
  monthDayPaint,
  useCycleForDate,
  useCycleMonths,
  homeMonthSet,
  useCycleStatus,
  useCycleToday,
  type CycleCalculation,
  type CycleDailyTip,
  type CycleFutureCycle,
  type CycleMonthDay,
  type CycleDayMarker,
  type CyclePhase,
  type FertileLevel,
  type CycleSchedule,
} from '@/entities/cycle';
import { useFertilityToday } from '@/entities/fertility';
import { HomeConditionNudge } from '@/entities/condition';
import { LossCareReturn } from '@/entities/loss';
import {
  dailyAdvice,
  feedLinkTarget,
  feedState,
  localizeFeedText,
  slotItems,
  useAckAlert,
  useDailyMessage,
  useMessageEvent,
  useMessageFeed,
  useUserMode,
  type DailyMessage,
  type FeedItem,
  type FeedLinkTarget,
} from '@/entities/message';
import {
  readLifeModeHint,
  useLifeStage,
  useUserProfile,
  writeLifeModeHint,
  type LifeMode,
} from '@/entities/user';
import { PeriodDateEditor, useEndPeriod, useStartPeriod } from '@/features/log-period';
import { canOfferSkip, SkipPeriodSheet, SkipUndo } from '@/features/skip-period';
import { getApiErrorCode } from '@/shared/api';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import {
  addDays,
  currentHour,
  diffInDays,
  formatNumber,
  formatWeekdayDayMonth,
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
import { AdSlot } from '@/widgets/ad-slot';
import { BannerSlideshow } from '@/widgets/banner-slideshow';
import { BottomNav } from '@/widgets/bottom-nav';
import { ForToday } from '@/widgets/for-today';
import { HeaderActions } from '@/widgets/notification-bell';
import { CheckupsCard } from '@/widgets/checkups-card';
import {
  needsPeriodData,
  PmsInsightCard,
  usePredictionSlots,
} from '@/widgets/cycle-predictions';
import { CompanionInviteBanner } from '@/widgets/companion-invite-banner';
import { LearnHomeCard } from '@/widgets/learn-home-card';
import {
  FertilityChanceCard,
  FertilityTiles,
  LhTipCard,
  TtcPhasePills,
  ttcPhase,
} from '@/widgets/fertility-tiles';
import { FeedSmartTipCard } from '@/widgets/smart-tip';
import { TodayChallengeCard } from '@/widgets/today-challenge';
import { TodayRemindersCard } from '@/widgets/today-reminders';

import { feedItemKey, firstVisible, legacyDailyEnabled, usableLink } from '../model/feed-home';
import { phaseFromMainPhase } from '../model/hero-state';
import { ringNextMilestone } from '../model/ring-next';
import { dismissPrompt, isPromptDismissed } from '../model/prompt-dismiss';
import { readTtcHint, writeTtcHint } from '../model/ttc-hint';
import { isEnginePayload, todayHero } from '../model/today-ui';

import { PregnancyPromptCard } from './PregnancyPromptCard';
import { FeedBanner, TodayBanner } from './TodayBanner';
import { InAppModal, UrgentModal } from './FeedModals';
import { PlusTrialOffer } from './PlusTrialOffer';
import { TodayLogCard } from './TodayLogCard';
import '@/shared/ui/styles/btn.css';
import '@/shared/ui/styles/home.css';
import '@/shared/ui/styles/skeleton.css';
import '@/shared/ui/styles/page.css';
import '@/shared/ui/styles/log.css';
import '@/shared/ui/styles/ch.css';
import '@/shared/ui/styles/life.css';

// T-M12-28: mode-specific branches, the card widgets only some users get and the editor sheets nobody opens on
// most visits are separate chunks. Each fallback keeps the layout: the backdrop for a whole home, nothing for a
// card that renders nothing while its data loads, nothing for a sheet that is closed.
const HomeBackdrop = () => (
  <div className="view">
    <div className="home-grad home-grad-fill" />
  </div>
);
const MenopauseHome = dynamic(() => import('./MenopauseHome').then((m) => m.MenopauseHome), { loading: HomeBackdrop });
const TeenHome = dynamic(() => import('@/widgets/teen-home').then((m) => m.TeenHome), { loading: HomeBackdrop });
const LinkedTeenCards = dynamic(() => import('@/widgets/linked-teen-card').then((m) => m.LinkedTeenCards), { loading: () => null });
const HomeLabCards = dynamic(() => import('@/widgets/home-lab-cards').then((m) => m.HomeLabCards), { loading: () => null });
// PeriodDateEditor / SkipPeriodSheet are lazy wrappers exported by their feature barrels (T-M12-27b).

/** True from the first time `open` was true — a lazy sheet is not fetched before it is first needed, and stays mounted for its close animation. */
function useEverOpened(open: boolean): boolean {
  const [ever, setEver] = useState(open);
  if (open && !ever) setEver(true);
  return ever;
}

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);
const localizeNum = (n: string | number, loc: Locale) => (loc === 'fa' ? faNum(n) : String(n));

type T = ReturnType<typeof useTranslations>;

// ── Header: today's date + a time-of-day greeting, notifications & light/dark ──
function greetingKey(hour: number): 'morning' | 'noon' | 'evening' | 'night' {
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 16) return 'noon';
  if (hour >= 16 && hour < 20) return 'evening';
  return 'night';
}

function HomeHeader({ t, loc }: { t: T; loc: Locale }) {
  return (
    <header className="home-hdr">
      <div>
        <div className="home-hdr-date">{formatWeekdayDayMonth(today(), loc)}</div>
        <div className="home-hdr-greet">{t(`greeting.${greetingKey(currentHour())}`)}</div>
      </div>
      <div className="home-hdr-actions">
        <HeaderActions className="home-hdr-btn" />
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
  engine: boolean,
  suppressed: boolean,
): (date: Date) => CycleDayMarker | null {
  // prev/current/next month start at mount, in parallel with /cycle/today (T-M12-33); a day of `dates`
  // outside them (long cycle, late period) adds its own month — the fallback.
  const cur = gregYearMonth(today());
  const monthQueries = useCycleMonths(homeMonthSet(cur, dates.map(toApiDate)));
  // The set length varies only in the fallback case, so memo on a stable signature, not a spread array.
  const monthSig = monthQueries.map(q => `${q.dataUpdatedAt}:${q.data ? 1 : 0}`).join('|');

  const calcMap = useMemo(() => {
    const map = new Map<string, CycleCalculation>();
    for (const month of monthQueries.map(q => q.data)) {
      for (const c of month?.calculations ?? []) map.set(c.calculationDate, c);
    }
    return map;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [monthSig]);

  // v6: the server's per-day fields and `future_cycles` paint the dots — no browser cycle math.
  const v6 = useMemo(() => {
    const days = new Map<string, CycleMonthDay>();
    let future: CycleFutureCycle[] = [];
    let freshest = -1;
    for (const q of monthQueries) {
      if (!q.data) continue;
      for (const d of q.data.days) days.set(d.date, d);
      if (q.dataUpdatedAt > freshest && q.data.days.length > 0) {
        freshest = q.dataUpdatedAt;
        future = q.data.futureCycles;
      }
    }
    return { days, futureMap: futureCycleDayMap(future, suppressed) };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [monthSig, suppressed]);

  if (engine) {
    return (date: Date) => {
      const iso = toApiDate(date);
      const marker = monthDayPaint(v6.days.get(iso), iso, v6.futureMap, suppressed)?.marker ?? null;
      return suppressed && (marker === 'fertile' || marker === 'ovulation') ? null : marker;
    };
  }
  return (date: Date) =>
    cycleDayMarkerAt(date, calcMap.get(toApiDate(date)), schedule, periodLength);
}

// ── Week strip ─────────────────────────────────────────────────
// The current week in the locale's grid order, each day a tall tile with its
// cycle-marker dot. Tapping a day points the ring and the phase card at it.
function WeekStrip({
  days, loc, selectedIso, onSelect, markOf, levelOf,
}: {
  days: Date[];
  loc: Locale;
  selectedIso: string;
  onSelect: (date: Date) => void;
  markOf: (date: Date) => CycleDayMarker | null;
  /** Chance step of a fertile-window day — the green deepens toward ovulation. */
  levelOf: (date: Date) => FertileLevel | null;
}) {
  const todayIso = toApiDate(today());
  const letters = weekdayLabels(loc);
  return (
    <div className="home-week">
      {days.map((date, i) => {
        const iso = toApiDate(date);
        const isToday = iso === todayIso;
        const isSelected = iso === selectedIso && !isToday;
        const level = levelOf(date);
        return (
          <button
            key={iso}
            type="button"
            className={clsx('home-wday', isToday && 'is-today', isSelected && 'is-selected')}
            onClick={() => onSelect(date)}
            aria-pressed={iso === selectedIso}
            {...pilot(`home.day:${iso}`)}
            aria-label={formatWeekdayDayMonth(date, loc)}
          >
            <span className="home-wday-name" aria-hidden>{letters[i]}</span>
            <span className="home-wday-cell" aria-hidden>
              {formatNumber(toParts(date, loc).day, loc)}
              <span className={clsx('home-wday-dot', `is-${markOf(date) ?? 'none'}`, level && `is-lv${level}`)} />
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
const DOT_RADIUS: Record<CycleDayMarker | 'none', number> = {
  period: 7,
  fertile: 7,
  ovulation: 8,
  pms: 7,
  none: 5,
};

interface RingDay {
  marker: CycleDayMarker | null;
  /** Chance step of a fertile-window day (null elsewhere) — picks its green. */
  level: FertileLevel | null;
}

function CycleRing({
  days, nowIndex, label, compact = false, children,
}: {
  days: RingDay[];
  /** Index of the selected day, or null when the cycle can't be placed. */
  nowIndex: number | null;
  label: string;
  /** Three actions under the number: tighten the stack so it stays inside the dot track. */
  compact?: boolean;
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
  const nowLevel = now?.level ? `is-lv${now.level}` : null;

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
          // Every marked day is a solid dot in its state's colour — predicted or not.
          return (
            <circle
              key={i}
              {...at(i)}
              r={DOT_RADIUS[day.marker ?? 'none'] * scale}
              className={clsx('home-ring-dot', day.marker && `is-${day.marker}`, day.level && `is-lv${day.level}`)}
            />
          );
        })}
        {nowIndex != null && (
          <>
            <circle {...at(nowIndex)} r={19 * scale} className={clsx('home-ring-halo', `is-${nowKind}`, nowLevel)} filter={`url(#${glowId})`} />
            <circle {...at(nowIndex)} r={12.5 * scale} className={clsx('home-ring-now', `is-${nowKind}`, nowLevel)} />
          </>
        )}
      </svg>
      <div className={clsx('home-ring-center', compact && 'is-compact')}>{children}</div>
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
        <button type="button" className="home-phase-more" onClick={() => openSheet('phase')} {...pilot('home.phase-more')}>
          {t('phaseCard.more')}
          <Icon name="chevronLeft" size={15} strokeWidth={2.2} className="home-phase-more-chev" />
        </button>
      )}
      {action && (
        <button type="button" className="home-phase-more" onClick={action.onClick} {...pilot('home.phase-action')}>
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
  t, tips, dos, phase, advice, loc,
}: {
  t: T;
  tips: CycleDailyTip[];
  dos: string[];
  /** The message engine's `daily_advice` (up to 3); wins over the legacy tips when the feed answered. */
  advice: FeedItem[];
  loc: Locale;
  /** Today's phase — the pill beside the title (none while unknown). */
  phase: CyclePhase | null;
}) {
  const items = advice.length > 0
    ? advice.slice(0, 3).map(a => ({
        icon: 'sparkle' as IconName,
        title: localizeFeedText(a.title || a.body, loc),
        desc: a.title && a.body ? localizeFeedText(a.body, loc) : undefined,
      }))
    : tips.length > 0
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
  // Admin module switches: no TTC layout while «اقدام به بارداری» is off, no parent's teen cards without «نوجوان».
  const modules = useAppModules();
  const ttcAllowed = modules.ttc && (lifeMode === null || lifeMode === 'ttc' || lifeMode === 'cycle');
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
  // The banner's «پایان پریود» opens the editor on the open period; every other entry uses its default.
  const [dateEditorIntent, setDateEditorIntent] = useState<'edit' | undefined>(undefined);
  const dateEditorEver = useEverOpened(dateEditorOpen);
  // The day the user tapped in the week strip (defaults to today). Only the
  // ring and the phase card reflect it; the rest of the page stays on today.
  const [selectedDate, setSelectedDate] = useState<Date>(() => today());

  // Server state (§8) — cycle math + personalized message for today.
  const todayQuery = useCycleToday();
  const todayData = todayQuery.data;
  // The message engine (SPEC §5.4): one feed for the banner, tip, advice, challenge and the two modals. When it does
  // not answer (Laravel 404, network error, offline) the home is exactly what it was: built-in copy + legacy reads.
  const feedQuery = useMessageFeed();
  const feedStatus = feedState(feedQuery);
  const feed = feedQuery.data ?? null;
  const legacyDaily = legacyDailyEnabled(feedStatus);
  const { data: daily } = useDailyMessage(undefined, legacyDaily);
  const router = useRouter();
  const sendEvent = useMessageEvent();
  const ackAlert = useAckAlert();
  // Served messages she closed (or acted on) this session; the server's cap/cooldown takes over after the refetch.
  const [closedFeed, setClosedFeed] = useState<ReadonlySet<string>>(() => new Set());
  const profileQuery = useUserProfile();
  // The layout the last fresh profile decided (read after mount: localStorage is
  // client-only). Lets a cold load skip waiting on `/profile` (T-M5-12).
  const ttcHint = useMemo(() => (mounted ? readTtcHint() : null), [mounted]);
  // TTC home (`v19_Main`): only while trying to conceive; pregnancy mode never matches.
  // The fresh profile wins; until it lands, the remembered layout stands in.
  const ttcProfile =
    ttcAllowed &&
    (profileQuery.data ? profileQuery.data.health?.pregnancyIntention === 'trying' : ttcHint === true);
  useEffect(() => {
    if (profileQuery.data) writeTtcHint(ttcProfile);
  }, [profileQuery.data, ttcProfile]);
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

  const {
    base,
    engine,
    schedule,
    fertilitySuppressed,
    limits,
    fmt,
    overview,
    windowSlot,
    pmsSlot,
    nextPeriodDate,
    ovulationDate,
    toPrediction,
  } = usePredictionSlots();
  // A suppressed user gets the plain cycle home: no TTC chance card, tiles, LH tip or ovulation ring.
  const isTtc = ttcProfile && !limits.hideFertility;

  const message: DailyMessage | null = daily ?? null;
  const dos = message?.primary.dos ?? [];

  // ── Selected-day info for the ring and the phase card ──
  const selApiDate = toApiDate(selectedDate);
  const isToday = selApiDate === toApiDate(base);

  // A tapped past/future day loads its own calculation + message; today reuses
  // the queries above (same cache keys — no extra fetch).
  const { data: dateData, isFetching: dateFetching } = useCycleForDate(selApiDate, !isToday);
  const { data: infoMessage } = useDailyMessage(isToday ? undefined : selApiDate, !isToday || legacyDaily);

  const infoCalc = (isToday ? todayData : dateData)?.calculation ?? null;
  const infoPred = infoCalc ? deriveCyclePredictions(infoCalc) : null;

  // The countdown measures the selected day against *its own* cycle's predicted
  // start: the engine's anchors for that day once they load (a past day then
  // shows the period it actually led to), and until then today's schedule rolled
  // whole cycles — so scrubbing counts down smoothly instead of jumping.
  const selectedSchedule: CycleSchedule | null =
    engine
      ? null
      : (isToday ? null : deriveCycleSchedule(dateData?.cycleView ?? null, dateData?.calculation ?? null)) ??
        (schedule ? cycleScheduleFor(schedule, selectedDate) : null);
  // The engine's own read-out for the selected day — the very same `cycle_view`
  // that renders the day-status card — so the ring can never disagree with it.
  const infoView = (isToday ? todayData : dateData)?.cycleView ?? null;
  // The engine's main phase wins over the legacy calculation's phase.
  // A v6 payload that says `unknown` stays neutral: no legacy-phase guess behind the engine's back.
  const infoPhase: CyclePhase | null =
    phaseFromMainPhase(infoView?.mainPhase) ?? (isEnginePayload(infoView) ? null : (infoPred?.phase ?? null));
  const infoDaysUntilNextPeriod =
    infoView?.daysToPeriod ??
    (engine ? null : selectedSchedule ? daysUntilNextPeriod(selectedSchedule, selectedDate) : null);
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
    !teen && !limits.hideFertility && infoFertilityLevel && infoFertilityLevel !== 'unknown'
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
  const rawMarkOf = useDayMarks(
    [...weekDays, ...ringDates],
    schedule ?? selectedSchedule,
    periodLength,
    engine,
    fertilitySuppressed,
  );
  // Teen (B-N2-10 audit): no fertile-window / ovulation dots on the ring or the week strip either.
  const markOf = (date: Date): CycleDayMarker | null => {
    const mark = rawMarkOf(date);
    return teen && (mark === 'fertile' || mark === 'ovulation') ? null : mark;
  };
  const levelOf = (date: Date): FertileLevel | null =>
    fertileLevelAt((offset) => markOf(addDays(date, offset)));
  const ringDays = ringDates.map(date => ({
    marker: markOf(date),
    level: levelOf(date),
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
  const pregnancyPromptDismissed = mounted && dismissTick >= 0 && isPromptDismissed('pregnancy', todayIso);
  const dismiss = (kind: 'start' | 'end' | 'pregnancy') => {
    dismissPrompt(kind, todayIso);
    setDismissTick((n) => n + 1);
  };
  const hero = todayHero({
    isToday: isToday && !loadingInfo && !noPeriodData,
    inPeriod,
    daysLeft,
    daysLate: infoView?.daysLate ?? null,
    periodOpen: infoView?.anchors?.currentPeriodEndIsConfirmed === false,
    cycleDay: infoCycleDay,
    periodLength,
    startDismissed,
    endDismissed,
    todayUi: infoView?.todayUi ?? null,
  });
  const startPeriod = useStartPeriod();
  const endPeriod = useEndPeriod();
  const [skipOpen, setSkipOpen] = useState(false);
  const skipEver = useEverOpened(skipOpen);
  const daysLate = infoView?.daysLate ?? 0;
  const lateTier = hero.tier;

  // TTC ring centre (`v19_Main`): «تخمک‌گذاری تا N روز» while ovulation is still
  // ahead in this cycle; once it has passed (or during the period) the ring
  // counts down to the next period like every other cycle home.
  const daysToOvulation = infoView?.daysToOvulation ?? null;
  const ttcRing =
    isTtc && !noPeriodData && !loadingInfo && !inPeriod && daysToOvulation != null && daysToOvulation >= 0;
  const skipExpected = infoView?.anchors?.predictedNextPeriodStart ?? null;
  const skipOffered = canOfferSkip({
    isToday,
    todayUi: infoView?.todayUi ?? null,
    predictedNext: skipExpected,
    todayIso,
    inPeriod,
    ttcRing,
  });
  // No countdown at all (postpartum before periods return, a pill user, an unknown day):
  // a calm label instead of a huge dash, and never «پریود بعدی» without a prediction.
  const noCountdown =
    !loadingInfo && !noPeriodData && !ttcRing && !inPeriod && hero.variant !== 'late' && daysLeft == null;
  // Off the period, the centre counts to the next coloured stretch on the ring
  // (fertile window → ovulation → PMS); only when the period is next does it
  // keep the «پریود بعدی» countdown.
  const milestone =
    loadingInfo || noPeriodData || noCountdown || ttcRing || inPeriod || hero.variant === 'late'
      ? null
      : ringNextMilestone(ringDays.map(d => d.marker), ringNowIndex);
  const ringOverline = noPeriodData
    ? t('ring.noData')
    : noCountdown
      ? t('ring.noPrediction')
    : ttcRing
      ? tf('home.ovulationIn')
      : milestone
        ? milestone.days === 0
          ? t(`ring.milestoneOn.${milestone.kind}`)
          : t(`ring.milestoneIn.${milestone.kind}`)
      : inPeriod && !loadingInfo
        ? t('ring.period')
        : hero.variant === 'late'
          ? t('nb.hero.late')
          : hero.variant === 'near' && daysLeft !== 0
            ? t('nb.hero.periodIn')
            : t('ring.nextPeriod');
  const ringNumber = loadingInfo || noCountdown
    ? dash
    : ttcRing
      ? daysToOvulation === 0
        ? t('ring.today')
        : formatNumber(daysToOvulation, loc)
      : milestone
        ? milestone.days === 0
          ? (isToday ? t('ring.today') : t('ring.thisDay'))
          : formatNumber(milestone.days, loc)
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
      : milestone
        ? milestone.days > 0
          ? t('ring.daysLeft', { n: milestone.days })
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
  // A null v6 phase (combined pill, unknown day, stale anchor) gets a neutral header that implies no natural phase.
  const phaseTitle = loadingInfo
    ? dash
    : !infoPhase
      ? infoCycleDay != null
        ? t('ring.cycleDay', { n: infoCycleDay })
        : t(isToday ? 'phaseCard.title.neutral' : 'phaseCard.title.neutralDay')
      : nearPeriod
      ? t('phaseCard.nearPeriod')
      : t(`phaseCard.title.${infoPhase}`, { n: infoCycleDay ?? infoPred?.cycleDay ?? 1 });
  const phaseDot: CyclePhase | null = loadingInfo || !infoPhase ? null : nearPeriod ? 'period' : infoPhase;
  const phaseText =
    !isToday || loadingInfo
      ? infoPhaseDesc
      : hero.variant === 'late'
        ? t(lateTier === 'calm' ? 'nb.lateText' : 'nb.lateTextLong', { n: formatNumber(daysLate, loc) })
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



  // Server pass / first client render: backdrop only, so both sides match.
  if (!mounted) {
    return (
      <div className="view">
        <div className="home-grad home-grad-fill" />
      </div>
    );
  }

  // ── The message feed (T-M10-14) ──
  const closeFeedItem = (item: FeedItem) => setClosedFeed((prev) => new Set(prev).add(feedItemKey(item)));
  const dismissFeedItem = (item: FeedItem) => {
    closeFeedItem(item);
    sendEvent.mutate({ item, event: 'dismissed' });
  };
  // A CTA runs the home's own editor for the three banner links, else navigates; a link into a switched-off
  // module (or an external one) gives no button at all.
  const runFeedTarget = (target: FeedLinkTarget) => {
    if (target.kind === 'route') {
      router.push(target.path as never);
      return;
    }
    if (target.action === 'edit_end') {
      setDateEditorIntent('edit');
      setDateEditorOpen(true);
    } else if (target.action === 'edit_last_period') {
      setDateEditorIntent(undefined);
      setDateEditorOpen(true);
    } else {
      openSheet('log', `${toApiDate(base)}:pregnancy_test`);
    }
  };
  const feedCta = (item: FeedItem, afterClick?: () => void): (() => void) | null => {
    const target = usableLink(feedLinkTarget(item.ctaDeeplink), modules);
    if (!target) return null;
    return () => {
      sendEvent.mutate({ item, event: 'clicked' });
      afterClick?.();
      runFeedTarget(target);
    };
  };
  const feedBannerAll = slotItems(feed, 'status_banner')[0] ?? null;
  const feedBanner = firstVisible(feedBannerAll ? [feedBannerAll] : [], closedFeed);
  const feedTip = firstVisible(slotItems(feed, 'smart_tip').slice(0, 1), closedFeed);
  const urgentItem = firstVisible(slotItems(feed, 'urgent_modal').slice(0, 1), closedFeed);
  const modalItem = urgentItem ? null : firstVisible(slotItems(feed, 'modal').slice(0, 1), closedFeed);

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
            levelOf={levelOf}
          />
          <section className={clsx('home-ring', loadingInfo && 'is-loading')}>
            <div
              className={clsx('home-ring-glow', ttcRing && 'is-ttc', !ttcRing && hero.variant !== 'normal' && 'is-period')}
              aria-hidden
            />
            <CycleRing
              days={ringDays}
              nowIndex={ringNowIndex}
              label={ringLabel}
              compact={!ttcRing && hero.startPrompt && skipOffered}
            >
              {/* `--period` red means menstruation (§10.2); "nothing logged yet"
                  is not a period, so the empty state takes the neutral caption. */}
              <span className={noPeriodData || noCountdown ? 'home-ring-sub' : clsx('home-ring-over', ttcRing && 'is-ttc')}>
                {ringOverline}
              </span>
              {!noPeriodData && (
                <div className="home-ring-big">
                  {noCountdown ? (
                    <span className="home-ring-unit">{t('ring.noPredictionValue')}</span>
                  ) : (
                    <span className="home-ring-num">{ringNumber}</span>
                  )}
                  {ringUnit && <span className="home-ring-unit">{ringUnit}</span>}
                </div>
              )}
              {ringSub && !noPeriodData && <span className="home-ring-sub">{ringSub}</span>}
              {ttcRing ? (
                <Link href={logHref} className="home-ring-edit is-ink" {...pilot('home.log-today')}>
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
                    {...pilot('home.period-started')}
                    disabled={startPeriod.isPending}
                  >
                    <Icon name="drop" size={16} strokeWidth={2.2} />
                    {t('nb.hero.started')}
                  </button>
                  <button type="button" className="home-ring-later" onClick={() => dismiss('start')} {...pilot('home.period-not-yet')}>
                    {t('nb.hero.notYet')}
                  </button>
                  {skipOffered && (
                    <button type="button" className="home-ring-later" data-action="skip_period" onClick={() => setSkipOpen(true)}>
                      {t('skip.action')}
                    </button>
                  )}
                </>
              ) : hero.endPrompt ? (
                // «پریودم تموم شد / هنوز ادامه دارد» (nbl_Cycle_Home_During): closes the open period today.
                <>
                  <button
                    type="button"
                    className="home-ring-cta"
                    onClick={() => endPeriod.mutate()}
                    {...pilot('home.period-ended')}
                    disabled={endPeriod.isPending}
                  >
                    <Icon name="check" size={16} strokeWidth={2.4} />
                    {t('nb.hero.ended')}
                  </button>
                  <button type="button" className="home-ring-later" onClick={() => dismiss('end')} {...pilot('home.period-still-going')}>
                    {t('nb.hero.stillGoing')}
                  </button>
                </>
              ) : (
                // The date editor rises over the home screen itself (§4.1).
                <button type="button" className="home-ring-edit" onClick={() => setDateEditorOpen(true)} {...pilot('home.edit-period')}>
                  <Icon name="pen" size={16} strokeWidth={2.2} />
                  {noPeriodData ? t('ring.logPeriod') : t('ring.editPeriod')}
                </button>
              )}
            </CycleRing>
          </section>
          {/* «ثبت امروز» + streak (nbl_Cycle_Home), right under the cycle ring: three cards — «ثبت پریود»
              (→ calendar editor), «علائم», «حال و روحیه» (user request, 2026-11-22; «دسترسی سریع» removed).
              The TTC home logs through its tiles. */}
          {!noPeriodData && !loadFailed && !isTtc && <TodayLogCard streakDays={overview?.streakDays ?? null} onEditPeriod={() => setDateEditorOpen(true)} />}
          {feedBanner ? (
            // M-14: the engine's `status_banner` copy and CTA win over the built-in banner below.
            <FeedBanner
              item={feedBanner}
              onCta={feedCta(feedBanner)}
              onSkip={skipOffered ? () => setSkipOpen(true) : undefined}
              onDismiss={() => dismissFeedItem(feedBanner)}
            />
          ) : feedBannerAll ? null : hero.fromEngine && hero.banner ? (
            // §2.43: one banner at a time, chosen by the engine (`today_ui.banner`).
            <TodayBanner
              banner={hero.banner}
              daysLate={daysLate}
              tier={hero.tier}
              showPregnancyTest={hero.showPregnancyTest}
              onEditEnd={() => {
                setDateEditorIntent('edit');
                setDateEditorOpen(true);
              }}
              onAskLastPeriod={() => {
                setDateEditorIntent(undefined);
                setDateEditorOpen(true);
              }}
              onLogPregnancyTest={() => openSheet('log', `${toApiDate(base)}:pregnancy_test`)}
              onSkip={skipOffered ? () => setSkipOpen(true) : undefined}
            />
          ) : (
            hero.endMissing && (
              <div role="status" className="ch-alert">
                <IconCircle icon="info" tone="warm" size="sm" />
                <span className="ch-alert-body">
                  <b className="ch-alert-title">{t('nb.endMissing.title')}</b>
                  <span className="ch-alert-text">{t('nb.endMissing.text')}</span>
                </span>
              </div>
            )
          )}
          {isToday && <SkipUndo />}
          {limits.promptPregnancy && isToday && !pregnancyPromptDismissed && !loadFailed && (
            <PregnancyPromptCard onDismiss={() => dismiss('pregnancy')} />
          )}
          {/* «توصیه‌های امروز» — under «ثبت امروز» and right above the phase card (user request,
              2026-11-23; it used to sit near the end of the feed). */}
          {!loadFailed && !booting && (
            <Recommendations
              t={t}
              tips={calc?.dailyTips ?? []}
              dos={dos}
              phase={todayPhase}
              advice={dailyAdvice(feed)}
              loc={loc}
            />
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
            {!teen && modules.teen && <LinkedTeenCards />}
            {/* companion-v2: «همراهت را به ریتمی دعوت کن» — only with no companion yet, dismissible per session. */}
            {!teen && <CompanionInviteBanner />}
            {!teen && <BannerSlideshow position="home_top" />}
            {/* B-N10-02: sponsored slots, targeted by mode only; never teen (the slot also refuses loss/health paths). */}
            {!teen && <AdSlot slot="ad_slider" mode={lifeMode} />}
            {/* «یادآورهای امروز» — today's doses + next appointment (M3, /care/today),
                right under the hero like the reminders artboard (T-M3-10 audit L-6). */}
            {/* CB-NAV-03 (nbd_Nav_Today): «برای امروز» = the M3 reminders card + today's to-do count. */}
            <ForToday reminders={<TodayRemindersCard />} />
            {/* CB-COND-07b: soft condition suggestion (heavy pain / flow) — nothing unless /messages/nudges hits. */}
            {!teen && modules.programs && <HomeConditionNudge />}
            {/* B-N8-03 (Option A): «ادامه یادگیری» / «جدید · از مدرس» — nothing without a course. */}
            {!teen && modules.learning && <LearnHomeCard />}
            {/* «چکاپ‌های دوره‌ای» (M4, /checkups/home). The «پیش‌بینی‌ها» card lives on /analysis. */}
            <CheckupsCard />
            {feedTip && (
              <FeedSmartTipCard
                item={feedTip}
                onCta={feedCta(feedTip)}
                onDismiss={() => dismissFeedItem(feedTip)}
              />
            )}
            {!teen && <BannerSlideshow position="home_middle" />}
            {!teen && <HomeLabCards />}
            <TodayChallengeCard pick={feed?.dailyChallenge ?? null} />
            {!noPeriodData && !loadFailed && !teen && (
              <PmsInsightCard pms={toPrediction(pmsSlot)} basedOnCycles={overview?.cycleLength.basedOnCycles ?? null} />
            )}
            {modules.home_articles && <Articles t={t} locale={loc} />}
            {!teen && <AdSlot slot="ad_square" mode={lifeMode} />}
            {!teen && <BannerSlideshow position="home_bottom" />}
            {!teen && <AdSlot slot="ad_strip" mode={lifeMode} />}
          </div>
        )}
        <div className="page-tail" />
      </div>

      {/* It portals to `.app-shell`, so mounting it here still covers the
          whole screen. */}
      {dateEditorEver && <PeriodDateEditor
        open={dateEditorOpen}
        intent={dateEditorIntent}
        onClose={() => {
          setDateEditorOpen(false);
          setDateEditorIntent(undefined);
        }}
      />}

      {/* SPEC §5.4.1: the urgent message cannot be closed — only «دیدم» (POST /alerts/{id}/ack) clears it. */}
      {urgentItem && (
        <UrgentModal
          item={urgentItem}
          pending={ackAlert.isPending}
          failed={ackAlert.isError}
          onAck={() => ackAlert.mutate({ ruleId: urgentItem.id, alertId: urgentItem.alertId }, { onSuccess: () => closeFeedItem(urgentItem) })}
        />
      )}
      {modalItem && (
        <InAppModal
          item={modalItem}
          onCta={feedCta(modalItem, () => closeFeedItem(modalItem))}
          onClose={() => dismissFeedItem(modalItem)}
        />
      )}

      {skipExpected && skipEver && <SkipPeriodSheet open={skipOpen} onClose={() => setSkipOpen(false)} expectedStart={skipExpected} />}

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
          <TeenHome
            actions={<HeaderActions className="home-hdr-btn" />}
            // CB-NAV-04b: same Today sections as every home; the doctor/labs tiles need a parent link (no shop/ads).
            today={<ForToday reminders={<TodayRemindersCard />} />}
          />
          <div className="page-tail" />
        </div>
        <BottomNav />
      </div>
    );
  }
  return <CycleHome lifeMode={mode} />;
}
