'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import {
  ChildAvatar,
  type ChildHome,
  type IndicatorValue,
  isVisitUrgent,
  roundPercentile,
  useChild,
  useChildren,
  useDueText,
} from '@/entities/child';
import { BabyTodayList } from '@/features/baby-log';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatDecimal, formatLongDate, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  type IconName,
  IconCircle,
  PrimaryButton,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

type T = ReturnType<typeof useTranslations<'children'>>;

function Chevron({ className }: { className?: string }) {
  const rtl = useDirection() === 'rtl';
  return <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} className={className} />;
}

function Shell({ childIds, children }: { childIds?: readonly number[]; children: React.ReactNode }) {
  return (
    <div className="view chd-screen chd-home">
      <SkyLayer />
      <div className="scroll">{children}</div>
      <BottomNav childIds={childIds} />
    </div>
  );
}

function TopBar({ title, eyebrow, onEdit, t }: { title?: string; eyebrow?: string; onEdit?: () => void; t: T }) {
  const router = useRouter();
  const rtl = useDirection() === 'rtl';
  return (
    <header className="chd-top">
      <HeaderButton
        variant="soft"
        icon={rtl ? 'arrowR' : 'arrowL'}
        label={t('common.back')}
        onClick={() => router.push('/children')}
      />
      <div className="chd-top-title">
        {eyebrow ? <span className="chd-top-eyebrow">{eyebrow}</span> : null}
        {title ? (
          <h1 id="chd-name" className="chd-top-name">
            {title}
          </h1>
        ) : null}
      </div>
      {onEdit ? (
        <HeaderButton variant="soft" icon="pencil" label={t('home.edit')} onClick={onEdit} />
      ) : (
        <span className="chd-top-spacer" aria-hidden />
      )}
    </header>
  );
}

/**
 * `/children/[id]` — the child home (nbl_v16_ChildHome): age, latest
 * measurements with WHO percentile chips, next vaccine visit, tiles to growth /
 * vaccines / milestones / learn (B-N5-06), «این هفته {name}» and today's
 * feeds / sleep / diapers (links into the feeding screen, B-N5-07). A spouse sees a shared
 * child read-only (no edit, no «ثبت اندازه»). The «کودک» tab root.
 */
export function ChildHomePage({ id }: { id: number }) {
  const t = useTranslations('children');
  const router = useRouter();
  const mounted = useMounted();
  const query = useChild(Number.isFinite(id) && id > 0 ? id : null);
  const list = useChildren();
  const childIds = list.data?.children.map((c) => c.id);

  if (!Number.isFinite(id) || id <= 0 || getApiErrorStatus(query.error) === 404) {
    return (
      <Shell childIds={childIds}>
        <TopBar t={t} />
        <div className="chd-body">
          <EmptyState
            icon="sprout"
            title={t('home.notFoundTitle')}
            body={t('home.notFoundBody')}
            action={<PrimaryButton onClick={() => router.push('/children')}>{t('home.toList')}</PrimaryButton>}
          />
        </div>
      </Shell>
    );
  }

  if (!mounted || query.isPending) {
    return (
      <Shell childIds={childIds}>
        <section className="chd-hero">
          <TopBar eyebrow={t('home.eyebrow')} t={t} />
          <SkeletonGroup label={t('common.loading')} className="chd-hero-skel">
            <Skeleton shape="card" />
          </SkeletonGroup>
        </section>
        <div className="chd-body">
          <SkeletonGroup label={t('common.loading')}>
            <Skeleton shape="card" />
            <Skeleton shape="block" />
            <Skeleton shape="block" />
          </SkeletonGroup>
        </div>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell childIds={childIds}>
        <TopBar eyebrow={t('home.eyebrow')} t={t} />
        <div className="chd-body">
          <Card className="chd-state" role="alert">
            <IconCircle icon="warning" tone="danger" size="lg" />
            <p className="chd-state-text">{t('common.loadError')}</p>
            <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
              {t('common.retry')}
            </SecondaryButton>
          </Card>
        </div>
      </Shell>
    );
  }

  const child = query.data;
  return (
    <Shell childIds={childIds}>
      <Hero child={child} t={t} />
      <div className="chd-body">
        <NextVaccineCard child={child} t={t} />
        <Tiles child={child} t={t} />
        <ThisWeekCard child={child} t={t} />
        <TodayCard child={child} t={t} />
      </div>
    </Shell>
  );
}

// ── Hero ───────────────────────────────────────────────────────
function Hero({ child, t }: { child: ChildHome; t: T }) {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const latest = child.latest;
  return (
    <section className="chd-hero" aria-labelledby="chd-name">
      <TopBar
        eyebrow={t('home.eyebrow')}
        title={child.name}
        onEdit={child.canEdit ? () => router.push(`/children/${child.id}/edit`) : undefined}
        t={t}
      />

      <div className="chd-hero-main">
        <ChildAvatar id={child.id} name={child.name} initial={child.initial} sex={child.sex} size={84} className="chd-hero-avatar" />
        <div className="chd-hero-text">
          <span className="chd-hero-label">{t('home.age')}</span>
          <span className="chd-hero-age">{child.age.label}</span>
          <span className="chd-hero-chips">
            <span className="chd-hchip">
              <Icon name="cake" size={13} />
              {t('home.born', { date: formatLongDate(fromApiDate(child.birthDate), locale) })}
            </span>
            {child.sexLabel ? <span className="chd-hchip">{child.sexLabel}</span> : null}
            {child.role === 'shared' ? (
              <span className="chd-hchip">
                <Icon name="eye" size={13} />
                {child.ownerName ? t('home.sharedBy', { name: child.ownerName }) : t('home.readOnly')}
              </span>
            ) : null}
          </span>
        </div>
      </div>

      <div className="chd-measures">
        <Measure label={t('home.weight')} unit={t('home.kg')} value={latest?.weight ?? null} href={`/children/${child.id}/growth`} t={t} />
        <Measure label={t('home.length')} unit={t('home.cm')} value={latest?.length ?? null} href={`/children/${child.id}/growth`} t={t} />
        <Measure label={t('home.head')} unit={t('home.cm')} value={latest?.head ?? null} href={`/children/${child.id}/growth`} t={t} />
      </div>
      {latest ? (
        <p className="chd-measured-on">
          {latest.source === 'birth'
            ? t('home.fromBirth')
            : t('home.measuredOn', { date: formatLongDate(fromApiDate(latest.measuredOn), locale) })}
        </p>
      ) : null}
      <Link href={`/children/${child.id}/growth`} className="chd-hero-cta">
        <Icon name={child.canEdit ? 'plus' : 'chart'} size={18} />
        {child.canEdit ? t('home.addMeasurement') : t('home.viewGrowth')}
      </Link>
    </section>
  );
}

function Measure({ label, unit, value, href, t }: { label: string; unit: string; value: IndicatorValue | null; href: string; t: T }) {
  const locale = useLocale() as Locale;
  const p = roundPercentile(value?.percentile);
  return (
    <Link href={href} className="chd-measure">
      <span className="chd-measure-label">{label}</span>
      <span className="chd-measure-value">{value ? formatDecimal(value.value, locale) : t('home.noValue')}</span>
      <span className="chd-measure-unit">{unit}</span>
      {value && p !== null ? (
        <span className={clsx('chd-pchip', value.inBand === false ? 'nb-tone-warm' : 'nb-tone-data')}>
          {value.inBand === false ? t('home.outOfBand') : t('home.percentile', { p: formatNumber(p, locale) })}
        </span>
      ) : null}
    </Link>
  );
}

// ── Next vaccine ───────────────────────────────────────────────
function NextVaccineCard({ child, t }: { child: ChildHome; t: T }) {
  const locale = useLocale() as Locale;
  const due = useDueText();
  const next = child.vaccines.next;
  if (!next) {
    if (!child.vaccines.complete) return null;
    return (
      <Card as="section" className="chd-vaccine">
        <IconCircle icon="checkCircle" tone="success" size="lg" />
        <span className="chd-vaccine-text">
          <b className="chd-vaccine-title">{t('home.vaccinesDone')}</b>
          <span className="chd-vaccine-sub">
            {t('home.vaccinesDoneSub', {
              given: formatNumber(child.vaccines.given, locale),
              total: formatNumber(child.vaccines.total, locale),
            })}
          </span>
        </span>
      </Card>
    );
  }
  const parts = toParts(fromApiDate(next.dueDate), locale);
  const urgent = isVisitUrgent(next);
  return (
    <Card as="section" className="chd-vaccine" aria-labelledby="chd-vaccine-title">
      <span className={clsx('chd-date', urgent ? 'nb-tone-danger' : 'nb-tone-brand')} aria-hidden>
        <span className="chd-date-day">{formatNumber(parts.day, locale)}</span>
        <span className="chd-date-month">{monthName(parts.month, locale)}</span>
      </span>
      <span className="chd-vaccine-text">
        <b id="chd-vaccine-title" className="chd-vaccine-title">
          {t('home.vaccineTitle', { label: next.label })}
        </b>
        {next.doseNames.length > 0 ? <span className="chd-vaccine-sub">{next.doseNames.join('، ')}</span> : null}
        <span className={clsx('chd-vaccine-due', urgent && 'is-urgent')}>{due(next.daysLeft, next.statusLabel)}</span>
      </span>
      <Link href={`/children/${child.id}/vaccines`} className="chd-vaccine-link">
        {t('home.details')}
      </Link>
    </Card>
  );
}

// ── Tiles ──────────────────────────────────────────────────────
function Tile({ href, icon, tone, title, sub }: { href: string; icon: IconName; tone: Tone; title: string; sub: string }) {
  return (
    <Link href={href} className="chd-tile">
      <IconCircle icon={icon} tone={tone} />
      <b className="chd-tile-title">{title}</b>
      <span className="chd-tile-sub">{sub}</span>
    </Link>
  );
}

function Tiles({ child, t }: { child: ChildHome; t: T }) {
  const locale = useLocale() as Locale;
  const due = useDueText();
  const next = child.vaccines.next;
  const m = child.milestones;
  const base = `/children/${child.id}`;
  return (
    <nav className="chd-tiles" aria-label={child.name}>
      <Tile
        href={`${base}/vaccines`}
        icon="syringe"
        tone="brand"
        title={t('home.tiles.vaccines')}
        sub={
          next
            ? t('home.tiles.vaccinesNext', { label: next.label, due: due(next.daysLeft, next.statusLabel) })
            : t('home.tiles.vaccinesComplete')
        }
      />
      <Tile href={`${base}/growth`} icon="chart" tone="data" title={t('home.tiles.growth')} sub={child.growth.label} />
      <Tile
        href={`${base}/milestones`}
        icon="star"
        tone="warm"
        title={t('home.tiles.milestones')}
        sub={
          m
            ? t('home.tiles.milestonesSub', {
                checked: formatNumber(m.checked, locale),
                total: formatNumber(m.total, locale),
                label: m.label,
              })
            : t('home.tiles.milestonesNone')
        }
      />
      <Tile href={`${base}/learn`} icon="bookOpen" tone="success" title={t('home.tiles.learn')} sub={t('home.tiles.learnSub')} />
    </nav>
  );
}

// ── «این هفته {name}» ──────────────────────────────────────────
function ThisWeekCard({ child, t }: { child: ChildHome; t: T }) {
  const locale = useLocale() as Locale;
  const w = child.thisWeek;
  if (!w?.body) return null;
  return (
    <Card as="section" className="chd-week" aria-labelledby="chd-week-title">
      <div className="chd-week-head">
        <h2 id="chd-week-title" className="chd-card-title">
          {t('home.thisWeek', { name: child.name })}
        </h2>
        <span className="chd-chip nb-tone-brand">{t('home.weeks', { n: formatNumber(w.weeks, locale) })}</span>
      </div>
      <p className="chd-week-body">{w.body}</p>
      {child.milestones ? (
        <Link href={`/children/${child.id}/milestones`} className="chd-week-link">
          {t('home.milestonesLink', { label: child.milestones.label })}
          <Chevron />
        </Link>
      ) : null}
    </Card>
  );
}

// ── Today (B-N5-03 data, rows by features/baby-log, B-N5-07) ──
function TodayCard({ child, t }: { child: ChildHome; t: T }) {
  return (
    <Card as="section" className="chd-today" aria-labelledby="chd-today-title">
      <h2 id="chd-today-title" className="chd-card-title">
        {t('home.today')}
      </h2>
      <BabyTodayList childId={child.id} today={child.today} readOnly={!child.canEdit} />
    </Card>
  );
}
