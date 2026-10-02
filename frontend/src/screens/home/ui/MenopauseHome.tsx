'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, type ReactNode } from 'react';

import { checkupIcon, checkupItemSchema, type CheckupItem, type CheckupStatus } from '@/entities/checkup';
import {
  type MenopauseMessage,
  type MenopauseToday,
  type MenopauseTreatment,
  useHotFlashTimer,
  useMenopauseMessages,
  useMenopauseToday,
} from '@/entities/menopause';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  HeroCard,
  IconCircle,
  type IconName,
  LineChart,
  PrimaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  StatusPill,
  TileButton,
  type Tone,
  UrgentCard,
  formatClock,
  useTimer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { flashElapsedSeconds, listedMessages, routedLink, scoreTrend } from '../model/menopause';
import { PlusTrialOffer } from './PlusTrialOffer';

const STATUS_TONE: Record<CheckupStatus, Tone> = {
  overdue: 'danger',
  due: 'warm',
  soon: 'brand',
  not_yet: 'neutral',
  up_to_date: 'success',
  disabled: 'neutral',
};

const MESSAGE_LOOK: Record<MenopauseMessage['kind'], { icon: IconName; tone: Tone }> = {
  alert: { icon: 'chart', tone: 'warm' },
  reminder: { icon: 'bellPlain', tone: 'brand' },
  tip: { icon: 'sparkle', tone: 'bloom' },
};

/**
 * Menopause home (CB-MENO-05, nbl_Meno_Home / dark `Main`) — replaces bloom's
 * minimal B-N2-03 home. Under the shared Today header: the bleeding alert
 * (UrgentCard) when the API raises it, «بدون پریود N ماه» hero with the stage
 * chip (→ `/menopause/stage`), three quick actions (hot-flash timer inline,
 * today's log sheet, bleeding → log sheet), today's stats, the monthly score
 * with its 6-month sparkline, up to three upcoming checkups, the treatment card
 * and the menopause messages. Links to screens that don't exist yet
 * (score, treatment, alert, doctor report) are not rendered.
 */
export function MenopauseHome({ header }: { header: ReactNode }) {
  const t = useTranslations('menopause.home');
  const today = useMenopauseToday();

  let body: ReactNode;
  if (today.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="mh-skel">
        <Skeleton shape="card" className="mh-skel-hero" />
        <div className="mh-quick">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} shape="block" className="mh-skel-tile" />
          ))}
        </div>
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (today.isError) {
    body = (
      <EmptyState
        icon="moon"
        title={t('loadError')}
        body={t('loadErrorBody')}
        action={
          <PrimaryButton icon="refresh" loading={today.isFetching} onClick={() => void today.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <HomeBody data={today.data} fetchedAt={today.dataUpdatedAt} />;
  }

  return (
    <div className="view">
      <div className="home-grad home-grad-fill" />
      <div className="scroll page-scroll">
        <div className="mh-page">
          {header}
          {body}
        </div>
        <div className="page-tail" />
      </div>
      <PlusTrialOffer />
      <BottomNav />
    </div>
  );
}

function HomeBody({ data, fetchedAt }: { data: MenopauseToday; fetchedAt: number }) {
  const t = useTranslations('menopause.home');
  const messages = useMenopauseMessages();
  const list = messages.data ?? [];
  const bleedingMessage = list.find((m) => m.key === 'postmenopausal_bleeding') ?? null;
  const cards = listedMessages(list, data.profile.tip?.code ?? null);
  const checkups = useMemo(
    () =>
      data.checkups.flatMap((raw) => {
        const parsed = checkupItemSchema.safeParse(raw);
        return parsed.success ? [parsed.data] : [];
      }),
    [data.checkups],
  );

  return (
    <>
      {data.bleeding.alert || bleedingMessage ? <BleedingAlert data={data} message={bleedingMessage} /> : null}
      <StageHero data={data} />
      <QuickActions data={data} fetchedAt={fetchedAt} />
      <TodayStats data={data} />
      {cards.length ? (
        <section className="mh-sec" aria-label={t('messages.title')}>
          {cards.map((m) => (
            <MessageCard key={m.key} message={m} />
          ))}
        </section>
      ) : null}
      <ScoreCard data={data} />
      {checkups.length ? <Checkups items={checkups.slice(0, 3)} /> : null}
      {data.treatment.length ? <Treatment items={data.treatment} /> : null}
    </>
  );
}

function BleedingAlert({ data, message }: { data: MenopauseToday; message: MenopauseMessage | null }) {
  const t = useTranslations('menopause.home');
  const locale = useLocale() as Locale;
  const item = data.bleeding.alertItem;
  const title = message?.title ?? item?.title;
  const body = message?.body ?? item?.body;
  if (!title) return null;
  return (
    <UrgentCard title={title} icon="drop" className="mh-alert">
      {body ? <p className="mh-alert-body">{body}</p> : null}
      {data.bleeding.lastOn ? (
        <p className="mh-alert-meta">{t('alert.lastOn', { date: formatDayMonth(fromApiDate(data.bleeding.lastOn), locale) })}</p>
      ) : null}
    </UrgentCard>
  );
}

function StageHero({ data }: { data: MenopauseToday }) {
  const t = useTranslations('menopause.home.hero');
  const tStage = useTranslations('menopause.stages');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const { profile } = data;

  if (profile.needsStage || !profile.stage) {
    return (
      <HeroCard as="section" className="mh-hero" aria-labelledby="mh-hero-title">
        <span className="mh-over">{t('needsOverline')}</span>
        <h2 id="mh-hero-title" className="mh-need-title">
          {t('needsTitle')}
        </h2>
        <p className="mh-hero-body">{t('needsBody')}</p>
        <PrimaryButton block={false} className="mh-hero-cta" onClick={() => router.push('/menopause/stage')}>
          {t('needsAction')}
        </PrimaryButton>
      </HeroCard>
    );
  }

  const stageLabel = tStage(profile.stage);
  const months = profile.monthsWithoutPeriod;
  return (
    <HeroCard as="section" className="mh-hero" aria-labelledby="mh-hero-title">
      <div className="mh-hero-top">
        <div className="mh-hero-head">
          {months !== null ? <span className="mh-over">{t('overline')}</span> : null}
          <h2 id="mh-hero-title" className="mh-months">
            {months !== null ? t('months', { months: formatNumber(months, locale) }) : stageLabel}
          </h2>
        </div>
        <Link href="/menopause/stage" className="mh-stage-chip" aria-label={t('stageChip', { stage: stageLabel })}>
          {stageLabel}
        </Link>
      </div>
      {profile.tip?.body ? <p className="mh-hero-body">{profile.tip.body}</p> : null}
      {profile.suggestedStage ? (
        <p className="mh-suggest">
          {t('suggest')}{' '}
          <Link href="/menopause/stage" className="mh-link">
            {t('suggestAction')}
          </Link>
        </p>
      ) : null}
    </HeroCard>
  );
}

/** The running timer's live length: the server's elapsed at fetch + local ticks (remounted per fetch by its key). */
function RunningClock({ baseS, children }: { baseS: number; children: (time: string) => ReactNode }) {
  const locale = useLocale() as Locale;
  const { elapsedMs } = useTimer({ running: true });
  const seconds = flashElapsedSeconds(baseS, elapsedMs);
  // mm:ss stays left-to-right inside RTL text (LRI … PDI isolate).
  return <>{children(`\u2066${formatNumber(formatClock(seconds * 1000), locale)}\u2069`)}</>;
}

function QuickActions({ data, fetchedAt }: { data: MenopauseToday; fetchedAt: number }) {
  const t = useTranslations('menopause.home.quick');
  const { start, stop } = useHotFlashTimer();
  const running = data.hotFlashes.running;
  const busy = start.isPending || stop.isPending;

  const hotFlash = running ? (
    <RunningClock key={`${running.id}-${fetchedAt}`} baseS={running.elapsedS}>
      {(time) => (
        <TileButton
          layout="card"
          tone="period"
          icon="flame"
          className="mh-qa is-running"
          pressed
          disabled={busy}
          aria-label={t('hotFlashStopLabel', { time })}
          label={t('hotFlashRunning', { time })}
          sub={t('hotFlashStop')}
          onClick={() => stop.mutate(running.id)}
        />
      )}
    </RunningClock>
  ) : (
    <TileButton
      layout="card"
      tone="period"
      icon="flame"
      className="mh-qa"
      disabled={busy}
      label={t('hotFlash')}
      onClick={() => start.mutate()}
    />
  );

  return (
    <section className="mh-sec" aria-label={t('label')}>
      <div className="mh-quick">
        {hotFlash}
        <TileButton layout="card" tone="brand" icon="note" className="mh-qa" label={t('logToday')} onClick={() => openSheet('log')} />
        {/* No bleeding preset in the log sheet yet (features/log-day): opens today's log, where bleeding is a section. */}
        <TileButton layout="card" tone="period" icon="drop" className="mh-qa" label={t('bleeding')} onClick={() => openSheet('log')} />
      </div>
      {start.isError || stop.isError ? (
        <p className="mh-error" role="alert">
          {t('error')}
        </p>
      ) : null}
    </section>
  );
}

function Stat({ value, label, tone }: { value: string; label: string; tone: Tone }) {
  return (
    <div className={clsx('mh-stat', `nb-tone-${tone}`)}>
      <b className="mh-stat-value">{value}</b>
      <span className="mh-stat-label">{label}</span>
    </div>
  );
}

function TodayStats({ data }: { data: MenopauseToday }) {
  const t = useTranslations('menopause.home.today');
  const locale = useLocale() as Locale;
  const sleep = data.sleep ? t('hours', { hours: formatDecimal(data.sleep.hours, locale) }) : t('empty');
  return (
    <section className="mh-sec" aria-labelledby="mh-today">
      <SectionTitle id="mh-today" title={t('title')} actionLabel={t('log')} onAction={() => openSheet('log')} />
      <Card className="mh-stats" padding="sm">
        <Stat value={formatNumber(data.hotFlashes.count, locale)} label={t('flashes')} tone="period" />
        <Stat value={formatNumber(data.nightSweats.count, locale)} label={t('sweats')} tone="brand" />
        <Stat value={sleep} label={t('sleep')} tone="data" />
      </Card>
    </section>
  );
}

function ScoreCard({ data }: { data: MenopauseToday }) {
  const t = useTranslations('menopause.home.score');
  const locale = useLocale() as Locale;
  const latest = data.score.latest;
  // Oldest → newest left to right, like the board (the chart is `direction: ltr` in both locales).
  const series = data.score.trend;
  const trend = scoreTrend(latest?.delta ?? null);
  const hasTrend = series.some((p) => p.total !== null);

  return (
    <section className="mh-sec" aria-labelledby="mh-score">
      <div className="mh-sect-head">
        <SectionTitle id="mh-score" title={t('title')} />
        <p className="mh-sect-sub">{t('sub')}</p>
      </div>
      <Card className="mh-score">
        {latest ? (
          <>
            <div className="mh-score-top">
              <p className="mh-score-num">
                <b className="mh-score-total">{formatNumber(latest.total, locale)}</b>
                <span className="mh-score-of">
                  {t('of', { max: formatNumber(latest.max, locale) })}
                  {latest.band?.title ? ` · ${latest.band.title}` : null}
                </span>
              </p>
              {trend ? (
                <StatusPill tone={trend.kind === 'worse' ? 'warm' : 'data'} className="mh-score-pill">
                  {trend.kind === 'same' ? t('same') : t(trend.kind, { points: formatNumber(trend.points, locale) })}
                </StatusPill>
              ) : null}
            </div>
            {hasTrend ? (
              <LineChart
                label={t('chart')}
                className="mh-spark"
                height={110}
                min={0}
                series={[{ values: series.map((p) => p.total), tone: 'brand', points: true }]}
                xLabels={series.map((p) => monthName(toParts(fromApiDate(p.month), locale).month, locale))}
              />
            ) : null}
          </>
        ) : (
          <p className="mh-score-empty">{t('empty')}</p>
        )}
      </Card>
    </section>
  );
}

function Checkups({ items }: { items: CheckupItem[] }) {
  const t = useTranslations('menopause.home.checkups');
  const tStatus = useTranslations('checkups.status');
  const router = useRouter();
  return (
    <section className="mh-sec" aria-labelledby="mh-checkups">
      <SectionTitle id="mh-checkups" title={t('title')} actionLabel={t('all')} onAction={() => router.push('/checkups')} />
      <Card className="mh-list" padding="none">
        {items.map((item) => (
          <Link key={item.id} href={`/checkups/${item.id}`} className="mh-row">
            <IconCircle icon={checkupIcon(item.icon, { category: item.category })} tone="bloom" size="md" />
            <span className="mh-row-text">
              <b className="mh-row-title">{item.title}</b>
              {item.nextDueLabel || item.intervalLabel ? (
                <span className="mh-row-sub">{item.nextDueLabel ?? item.intervalLabel}</span>
              ) : null}
            </span>
            <StatusPill tone={STATUS_TONE[item.status]}>{tStatus(item.status)}</StatusPill>
          </Link>
        ))}
      </Card>
    </section>
  );
}

function Treatment({ items }: { items: MenopauseTreatment[] }) {
  const t = useTranslations('menopause.home.treatment');
  const locale = useLocale() as Locale;
  // HRT first (the board's card), then supplements and lifestyle.
  const sorted = [...items].sort((a, b) => Number(b.kind === 'hrt') - Number(a.kind === 'hrt'));
  return (
    <section className="mh-sec" aria-labelledby="mh-treatment">
      <SectionTitle id="mh-treatment" title={t('title')} />
      <Card className="mh-list" padding="none">
        {sorted.map((item) => {
          const parts = [
            t('week', { taken: formatNumber(item.daysTaken, locale), days: formatNumber(item.days, locale) }),
            item.reviewOn ? t('review', { month: monthName(toParts(fromApiDate(item.reviewOn), locale).month, locale) }) : null,
          ].filter(Boolean);
          return (
            <div key={item.id} className="mh-row">
              <IconCircle icon="pill" tone="data" size="md" />
              <span className="mh-row-text">
                <b className="mh-row-title">
                  {item.name} · {item.takenToday ? t('takenToday') : t('notToday')}
                </b>
                <span className="mh-row-sub">{parts.join(' · ')}</span>
              </span>
            </div>
          );
        })}
      </Card>
    </section>
  );
}

function MessageCard({ message }: { message: MenopauseMessage }) {
  const look = MESSAGE_LOOK[message.kind];
  const href = routedLink(message.link);
  return (
    <Card className="mh-msg" padding="sm">
      <IconCircle icon={look.icon} tone={look.tone} size="md" />
      <span className="mh-row-text">
        {message.title ? <b className="mh-row-title">{message.title}</b> : null}
        {message.body ? <span className="mh-msg-body">{message.body}</span> : null}
        {href && message.action ? (
          <Link href={href} className="mh-link">
            {message.action}
          </Link>
        ) : null}
      </span>
    </Card>
  );
}
