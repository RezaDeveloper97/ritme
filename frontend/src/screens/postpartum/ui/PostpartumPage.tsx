'use client';

import { useQueryClient } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { type Appointment, useAppointments } from '@/entities/care-reminder';
import { ChildrenStrip, useChildren, useDueText } from '@/entities/child';
import { useLogDay } from '@/entities/health-log';
import {
  type PostpartumAlert,
  type PostpartumOverview,
  type PostpartumStatus,
  postpartumKeys,
  usePostpartum,
} from '@/entities/postpartum';
import { useSaveLogDay } from '@/features/log-day';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import {
  addDays,
  formatDayMonth,
  formatDecimal,
  formatNumber,
  fromApiDate,
  monthName,
  toApiDate,
  toParts,
  today as todayDate,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet, useSheetStore } from '@/shared/sheet';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  type IconName,
  IconCircle,
  ProgressRing,
  SectionTitle,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
  UrgentCard,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { durationKey, headlineWeek, homeAlerts, sixWeekVisitDue, splitScheduled, tileSummary } from '../model/home';
import { MOOD_CHIPS, type MoodChip, pressedChips, toggleChip } from '../model/mood';

type T = ReturnType<typeof useTranslations<'postpartum'>>;

const CHIP_LOOK: Record<MoodChip, { icon: IconName; tone: Tone }> = {
  good: { icon: 'smile', tone: 'success' },
  tired: { icon: 'moon', tone: 'brand' },
  restless: { icon: 'warning', tone: 'warm' },
  sad: { icon: 'heart', tone: 'bloom' },
};

function Shell({ children }: { children: React.ReactNode }) {
  // B-N5-05: «کودک» opens the only child directly.
  const kids = useChildren();
  return (
    <div className="view pp-screen">
      <SkyLayer />
      <div className="scroll">{children}</div>
      <BottomNav childIds={kids.data?.children.map((c) => c.id)} />
    </div>
  );
}

function TopBar({ t }: { t: T }) {
  const router = useRouter();
  return (
    <header className="pp-top">
      <HeaderButton variant="soft" icon="bell" label={t('home.notifications')} onClick={() => openSheet('notifications')} />
      <div className="pp-brand">
        <span className="pp-brand-name">{t('home.brand')}</span>
        <span className="pp-brand-mode">{t('home.modeLabel')}</span>
      </div>
      <HeaderButton variant="soft" icon="user" label={t('home.profile')} onClick={() => router.push('/profile')} />
    </header>
  );
}

/** Refetch the home when the postpartum log sheet (FAB / tiles) closes — it writes the same taxonomy rows. */
function useRefreshAfterSheet() {
  const queryClient = useQueryClient();
  const depth = useSheetStore((s) => s.stack.length);
  const prev = useRef(depth);
  useEffect(() => {
    if (prev.current > 0 && depth === 0) void queryClient.invalidateQueries({ queryKey: postpartumKeys.all });
    prev.current = depth;
  }, [depth, queryClient]);
}

// ── Main export ────────────────────────────────────────────────
/**
 * Postpartum «امروز» — `/postpartum` (nbl_v15_Main / nbd_v15_Main, B-N5-04).
 * `/home` sends postpartum users here (stage bug B-4). Not active, or active
 * without a birth date (mode switched from Me) → the setup prompt.
 */
export function PostpartumPage() {
  const t = useTranslations('postpartum');
  const mounted = useMounted();
  const query = usePostpartum();
  useRefreshAfterSheet();

  if (!mounted || query.isPending) {
    return (
      <Shell>
        <TopBar t={t} />
        <SkeletonGroup label={t('common.loading')} className="pp-body">
          <Skeleton shape="card" className="pp-skel-hero" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell>
        <TopBar t={t} />
        <div className="pp-body">
          <Card className="pp-state" role="alert">
            <IconCircle icon="warning" tone="danger" size="lg" />
            <p className="pp-state-text">{t('common.loadError')}</p>
            <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
              {t('common.retry')}
            </SecondaryButton>
          </Card>
        </div>
      </Shell>
    );
  }

  const data = query.data;
  if (!data.active || data.setupRequired || !data.status || !data.profile) {
    return (
      <Shell>
        <TopBar t={t} />
        <div className="pp-body">
          <Card>
            <EmptyState
              icon="heart"
              title={data.setupRequired ? t('home.setup.title') : t('home.inactive.title')}
              body={data.setupRequired ? t('home.setup.body') : t('home.inactive.body')}
              action={
                <Link href="/postpartum/setup" className="nb-btn is-primary is-block">
                  {t('home.setup.cta')}
                </Link>
              }
            />
          </Card>
          <CallWhenCard data={data} t={t} />
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <Hero data={data} status={data.status} t={t} />
      <div className="pp-body">
        <AlertList alerts={homeAlerts(data.alerts, data.today)} t={t} />
        <MoodCard t={t} />
        <Tiles data={data} t={t} />
        <VisitsCard status={data.status} birthDate={data.profile.birthDate} t={t} />
        <CallWhenCard data={data} t={t} />
        {data.weekTip && (data.weekTip.title || data.weekTip.body) && (
          <Card as="section" className="pp-tip" aria-labelledby="pp-tip-title">
            <div className="pp-tip-head">
              <IconCircle icon="sparkle" tone="brand" size="sm" />
              <h2 id="pp-tip-title" className="pp-card-title">
                {t('home.tipTitle')}
              </h2>
            </div>
            {data.weekTip.title && <p className="pp-tip-lead">{data.weekTip.title}</p>}
            {data.weekTip.body && <p className="pp-tip-body">{data.weekTip.body}</p>}
          </Card>
        )}
      </div>
    </Shell>
  );
}

// ── Hero ───────────────────────────────────────────────────────
function Hero({ data, status, t }: { data: PostpartumOverview; status: PostpartumStatus; t: T }) {
  const locale = useLocale() as Locale;
  const n = (v: number) => formatNumber(v, locale);
  const key = durationKey(status);
  const duration =
    key === 'weeksDays'
      ? t('home.durationWeeksDays', { weeks: n(status.weeks), days: n(status.days) })
      : key === 'weeks'
        ? t('home.durationWeeks', { weeks: n(status.weeks) })
        : t('home.durationDays', { days: n(status.days) });
  const day = Math.min(status.daysSinceBirth, status.puerperiumDays);
  const totalWeeks = Math.round(status.puerperiumDays / 7);
  const profile = data.profile;
  const inPuerperium = status.daysSinceBirth < status.puerperiumDays;
  return (
    <section className="pp-hero" aria-labelledby="pp-duration">
      <TopBar t={t} />
      <HeroChildren />
      <div className="pp-hero-chips">
        <span className="pp-hchip">
          <Icon name="calendar" size={14} />
          {t('home.today', { date: formatDayMonth(todayDate(), locale) })}
        </span>
        <span className="pp-hchip">{t('home.weekChip', { week: n(headlineWeek(status)) })}</span>
      </div>
      <div className="pp-hero-main">
        <div className="pp-hero-text">
          {status.phaseLabel && <p className="pp-phase">{status.phaseLabel}</p>}
          <h1 id="pp-duration" className="pp-duration">
            {duration}
          </h1>
          <div className="pp-hero-tags">
            {profile?.deliveryTypeLabel && (
              <span className="pp-tag">
                <Icon name="heart" size={13} />
                {profile.deliveryTypeLabel}
              </span>
            )}
            {data.today?.feedsCount != null && data.today.feedsCount > 0 && (
              <span className="pp-tag">
                <Icon name="bottle" size={13} />
                {t('home.feedingChip')}
              </span>
            )}
            {profile && profile.babyCount > 1 && (
              <span className="pp-tag">{t('home.babiesChip', { count: n(profile.babyCount) })}</span>
            )}
          </div>
        </div>
        <ProgressRing
          value={status.progress}
          size={112}
          thickness={9}
          label={t('home.ringLabel')}
          valueText={inPuerperium ? t('home.ringValue', { day: n(day), total: n(status.puerperiumDays) }) : t('home.ringDone')}
          className="pp-ring"
        >
          {/* B-N5-10: past day 42 the ring says «۶ هفته · کامل شد» instead of a frozen «روز ۴۲». */}
          <span className="pp-ring-cap">{inPuerperium ? t('home.ringCaption') : t('home.ringWeeks')}</span>
          <span className="pp-ring-num">{n(inPuerperium ? day : totalWeeks)}</span>
          <span className="pp-ring-cap">{inPuerperium ? t('home.ringOf', { weeks: n(totalWeeks) }) : t('home.ringDone')}</span>
        </ProgressRing>
      </div>
      {inPuerperium && <p className="pp-hero-note">{t('home.heroNote')}</p>}
      <Link href="/postpartum/recovery" className="pp-hero-cta">
        <Icon name="mother" size={18} />
        {t('home.logToday')}
      </Link>
    </section>
  );
}

/** B-N5-05: the hero «فرزندان» row (avatars → child home, «+» → add). */
function HeroChildren() {
  const kids = useChildren();
  if (!kids.data) return null;
  return <ChildrenStrip items={kids.data.children} canAdd={kids.data.canAdd} className="pp-children" />;
}

// ── Alerts (server copy) ───────────────────────────────────────
function AlertList({ alerts, t }: { alerts: PostpartumAlert[]; t: T }) {
  const locale = useLocale() as Locale;
  if (!alerts.length) return null;
  return (
    <>
      {alerts.map((a) => {
        const action = a.action;
        if (a.level === 'warning' || a.level === 'urgent') {
          return (
            <UrgentCard
              key={a.key}
              title={a.title}
              action={
                action?.type === 'call' ? (
                  <a className="nb-btn is-primary is-block" href={`tel:${action.number}`}>
                    <Icon name="phone" size={18} />
                    {a.actionLabel ?? formatNumber(action.number, locale)}
                  </a>
                ) : undefined
              }
            >
              {a.body}
            </UrgentCard>
          );
        }
        return (
          <Card key={a.key} as="section" className="pp-alert" aria-label={a.title ?? undefined}>
            <IconCircle icon={action?.type === 'open_check' ? 'heartLine' : 'info'} tone="brand" size="md" />
            <div className="pp-alert-text">
              {a.title && <p className="pp-alert-title">{a.title}</p>}
              {a.body && <p className="pp-alert-body">{a.body}</p>}
            </div>
            {action?.type === 'open_check' && (
              <Link href={`/postpartum/mood?kind=${action.kind}`} className="pp-alert-cta">
                {a.actionLabel ?? t('home.moodCheck.start')}
              </Link>
            )}
            {action?.type === 'call' && (
              <a href={`tel:${action.number}`} className="pp-alert-cta">
                {a.actionLabel ?? formatNumber(action.number, locale)}
              </a>
            )}
          </Card>
        );
      })}
    </>
  );
}

// ── Mood chips ─────────────────────────────────────────────────
function MoodCard({ t }: { t: T }) {
  const date = toApiDate(todayDate());
  const day = useLogDay(date);
  const save = useSaveLogDay();
  const values = day.data?.categories;
  const pressed = pressedChips(values);
  const toggle = (chip: MoodChip) => {
    if (save.isPending || day.isPending) return;
    const { changes, draft } = toggleChip(values, chip);
    save.mutate({ date, changes, draft });
  };
  return (
    <Card as="section" className="pp-mood" aria-labelledby="pp-mood-title">
      <SectionTitle
        id="pp-mood-title"
        title={t('home.mood.title')}
        actionLabel={t('home.mood.log')}
        onAction={() => openSheet('log')}
      />
      <div className="pp-mood-chips" role="group" aria-labelledby="pp-mood-title">
        {MOOD_CHIPS.map((chip) => (
          <button
            key={chip}
            type="button"
            aria-pressed={pressed.has(chip)}
            disabled={day.isPending}
            className={clsx('pp-mchip', `nb-tone-${CHIP_LOOK[chip].tone}`)}
            onClick={() => toggle(chip)}
          >
            <Icon name={CHIP_LOOK[chip].icon} size={20} />
            <span>{t(`home.mood.${chip}`)}</span>
          </button>
        ))}
      </div>
      {save.isError && (
        <p className="pp-error" role="alert">
          {t('home.mood.saveError')}
        </p>
      )}
      <p className="pp-mood-note">{t('home.mood.note')}</p>
    </Card>
  );
}

// ── Bleeding / feeds / sleep tiles → postpartum log sheet ──────
function Tiles({ data, t }: { data: PostpartumOverview; t: T }) {
  const locale = useLocale() as Locale;
  const s = tileSummary(data.today);
  const tr = useTranslations('postpartum.recovery');
  const bleeding = s.bleeding
    ? s.bleeding.color
      ? t('home.tiles.bleedingValue', {
          amount: tr(`amounts.${s.bleeding.amount as 'light'}`),
          color: tr(`colorsShort.${s.bleeding.color as 'red'}`),
        })
      : tr(`amounts.${s.bleeding.amount as 'light'}`)
    : t('home.tiles.notLogged');
  const tiles: { key: string; icon: IconName; tone: Tone; label: string; value: string }[] = [
    { key: 'bleeding', icon: 'drop', tone: 'period', label: t('home.tiles.bleeding'), value: bleeding },
    {
      key: 'feeds',
      icon: 'bottle',
      tone: 'data',
      label: t('home.tiles.feeds'),
      value: s.feeds != null ? t('home.tiles.feedsValue', { count: formatNumber(s.feeds, locale) }) : t('home.tiles.notLogged'),
    },
    {
      key: 'sleep',
      icon: 'sleep',
      tone: 'brand',
      label: t('home.tiles.sleep'),
      value: s.sleep != null ? t('home.tiles.sleepValue', { hours: formatDecimal(s.sleep, locale) }) : t('home.tiles.notLogged'),
    },
  ];
  return (
    <div className="pp-tiles">
      {tiles.map((tile) => (
        <button key={tile.key} type="button" className={clsx('pp-tile', `nb-tone-${tile.tone}`)} onClick={() => openSheet('log')}>
          <span className="pp-tile-disc" aria-hidden>
            <Icon name={tile.icon} size={22} strokeWidth={1.8} />
          </span>
          <span className="pp-tile-label">{tile.label}</span>
          <span className="pp-tile-value">{tile.value}</span>
        </button>
      ))}
    </div>
  );
}

// ── Upcoming visits ────────────────────────────────────────────
function Chevron() {
  const rtl = useDirection() === 'rtl';
  return <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="pp-visit-chev" />;
}

function DateBadge({ date, tone }: { date: Date; tone: Tone }) {
  const locale = useLocale() as Locale;
  const p = toParts(date, locale);
  return (
    <span className={clsx('pp-date', `nb-tone-${tone}`)} aria-hidden>
      <span className="pp-date-day">{formatNumber(p.day, locale)}</span>
      <span className="pp-date-month">{monthName(p.month, locale)}</span>
    </span>
  );
}

function VisitRow({ appointment }: { appointment: Appointment }) {
  const locale = useLocale() as Locale;
  const when = splitScheduled(appointment.scheduledAt);
  const sub = [appointment.withWhom ?? appointment.location, when ? formatNumber(when.time, locale) : null]
    .filter(Boolean)
    .join(' · ');
  return (
    <li>
      <Link href={`/reminders/appointment/${appointment.id}`} className="pp-visit">
        {when ? <DateBadge date={fromApiDate(when.date)} tone="warm" /> : <IconCircle icon="stetho" tone="warm" />}
        <span className="pp-visit-text">
          <span className="pp-visit-title">{appointment.title}</span>
          {sub && <span className="pp-visit-sub">{sub}</span>}
        </span>
        <Chevron />
      </Link>
    </li>
  );
}

function VisitsCard({ status, birthDate, t }: { status: PostpartumStatus; birthDate: string; t: T }) {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const appointments = useAppointments('upcoming');
  const list = (appointments.data ?? []).slice(0, 3);
  const sixWeekDate = addDays(fromApiDate(birthDate), status.puerperiumDays);
  const showSixWeek = sixWeekVisitDue(status);
  return (
    <Card as="section" className="pp-visits" aria-labelledby="pp-visits-title">
      <SectionTitle
        id="pp-visits-title"
        title={t('home.visits.title')}
        actionLabel={t('home.visits.all')}
        onAction={() => router.push('/reminders')}
      />
      {appointments.isPending ? (
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
        </SkeletonGroup>
      ) : null}
      <ul className="pp-visit-list">
        {list.map((a) => (
          <VisitRow key={a.id} appointment={a} />
        ))}
        {showSixWeek && (
          <li>
            <Link href="/reminders/appointment/new" className="pp-visit">
              <DateBadge date={sixWeekDate} tone="warm" />
              <span className="pp-visit-text">
                <span className="pp-visit-title">{t('home.visits.sixWeek')}</span>
                <span className="pp-visit-sub">
                  {t('home.visits.sixWeekSub', { date: formatDayMonth(sixWeekDate, locale) })}
                </span>
              </span>
              <Chevron />
            </Link>
          </li>
        )}
        <ChildVaccineRows t={t} />
      </ul>
      {!appointments.isPending && list.length === 0 && !showSixWeek && (
        <Link href="/reminders/appointment/new" className="pp-visit-add">
          <Icon name="plus" size={16} />
          {t('home.visits.add')}
        </Link>
      )}
    </Card>
  );
}

/** B-N5-05: each child's next vaccine visit (→ its vaccines page); no child yet → add one. */
function ChildVaccineRows({ t }: { t: T }) {
  const tc = useTranslations('children');
  const due = useDueText();
  const kids = useChildren();
  if (!kids.data) return null;
  const rows = kids.data.children.filter((c) => c.vaccines.next).slice(0, 3);
  if (kids.data.children.length === 0) {
    return (
      <li>
        <Link href="/children/new" className="pp-visit">
          <IconCircle icon="syringe" tone="bloom" />
          <span className="pp-visit-text">
            <span className="pp-visit-title">{t('home.visits.vaccines')}</span>
            <span className="pp-visit-sub">{t('home.visits.vaccinesSub')}</span>
          </span>
          <Chevron />
        </Link>
      </li>
    );
  }
  return (
    <>
      {rows.map((c) => {
        const next = c.vaccines.next;
        if (!next) return null;
        return (
          <li key={c.id}>
            <Link href={`/children/${c.id}/vaccines`} className="pp-visit">
              <DateBadge date={fromApiDate(next.dueDate)} tone="bloom" />
              <span className="pp-visit-text">
                <span className="pp-visit-title">{tc('visits.vaccine', { label: next.label, name: c.name })}</span>
                <span className="pp-visit-sub">{due(next.daysLeft, next.statusLabel)}</span>
              </span>
              <Chevron />
            </Link>
          </li>
        );
      })}
    </>
  );
}

// ── «کی فوراً تماس بگیرم؟» ─────────────────────────────────────
function CallWhenCard({ data, t }: { data: PostpartumOverview; t: T }) {
  const locale = useLocale() as Locale;
  if (!data.callWhen?.title && !data.callWhen?.body) return null;
  return (
    <UrgentCard
      title={data.callWhen.title}
      urgent={false}
      hotlinesLabel={t('home.callWhen.hotlines')}
      hotlines={[{ label: t('home.callWhen.emergency'), number: '115', display: formatNumber('115', locale) }]}
      className="pp-callwhen"
    >
      {data.callWhen.body}
    </UrgentCard>
  );
}

