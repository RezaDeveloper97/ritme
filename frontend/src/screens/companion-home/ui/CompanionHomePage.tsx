'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  COMPANION_SECTIONS,
  type CompanionArticle,
  type CompanionPartner,
  type CompanionSection,
  type SharedAppointment,
  useCompanionHome,
} from '@/entities/companion';
import { useLifeStage } from '@/entities/user';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  IconCircle,
  type IconName,
  PrimaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  type Tone,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { phaseBar } from '../model/phase-bar';
import { CodeEntry } from './CodeEntry';

type T = ReturnType<typeof useTranslations<'companionHome'>>;

function firstLetter(name: string): string {
  return Array.from(name.trim())[0] ?? '';
}

/** «۲۱:۰۰» from «21:00» (or an ISO date-time). */
function clock(value: string, locale: Locale): string {
  const hhmm = value.length > 5 ? value.slice(11, 16) : value.slice(0, 5);
  return formatNumber(hhmm, locale);
}

function appointmentWhen(a: SharedAppointment, locale: Locale): string | null {
  if (!a.scheduledAt) return null;
  const day = fromApiDate(a.scheduledAt.slice(0, 10));
  const dayLabel = a.daysUntil != null && a.daysUntil >= 0 && a.daysUntil < 7 ? formatWeekday(day, locale) : formatDayMonth(day, locale);
  return `${dayLabel} ${clock(a.scheduledAt, locale)}`;
}

// ── Partner card (her day / phase / next period, or pregnancy week) ──────────
function PartnerCard({ partner, name, t, locale }: { partner: CompanionPartner; name: string; t: T; locale: Locale }) {
  const cycle = partner.cycle?.hasData ? partner.cycle : null;
  const pregnancy = partner.pregnancy?.isActive ? partner.pregnancy : null;
  const bar = cycle?.cycleDay && cycle.cycleLength ? phaseBar(cycle.cycleDay, cycle.cycleLength) : null;
  const n = (v: number) => formatNumber(v, locale);

  if (pregnancy) {
    return (
      <section className="nb-card cmh-hero" aria-label={t('pregnancy.overline', { name })}>
        <div className="cmh-hero-head">
          <div>
            <div className="cmh-overline">{t('pregnancy.overline', { name })}</div>
            {pregnancy.currentWeek ? <span className="cmh-display">{t('pregnancy.week', { week: n(pregnancy.currentWeek) })}</span> : null}
          </div>
          {pregnancy.trimester ? <StatusPill tone="bloom">{t('pregnancy.trimester', { n: n(pregnancy.trimester) })}</StatusPill> : null}
        </div>
        {partner.note ? <p className="cmh-hero-note">{partner.note}</p> : null}
      </section>
    );
  }

  if (cycle) {
    return (
      <section className="nb-card cmh-hero" aria-label={t('cycle.overline', { name })}>
        <div className="cmh-hero-head">
          <div>
            <div className="cmh-overline">{t('cycle.overline', { name })}</div>
            {cycle.cycleDay ? (
              <span className="cmh-display">
                {cycle.cycleLength
                  ? t('cycle.day', { day: n(cycle.cycleDay), length: n(cycle.cycleLength) })
                  : t('cycle.dayOnly', { day: n(cycle.cycleDay) })}
              </span>
            ) : null}
          </div>
          {partner.phase !== 'general' ? (
            <StatusPill tone="period" className="cmh-phase-pill">
              {t('cycle.phase', { phase: t(`phases.${partner.phase}`) })}
            </StatusPill>
          ) : null}
        </div>
        {bar && cycle.cycleDay && cycle.cycleLength ? (
          <div
            className="cmh-bar-wrap"
            role="img"
            aria-label={t('cycle.barLabel', { day: n(cycle.cycleDay), length: n(cycle.cycleLength) })}
          >
            <div className="cmh-bar">
              {bar.segments
                .filter((s) => s.days > 0)
                .map((s) => (
                  <span key={s.key} className={`cmh-bar-seg is-${s.key}`} style={{ flexGrow: s.days }} />
                ))}
            </div>
            <span className="cmh-bar-marker" style={{ insetInlineStart: `${bar.markerPercent}%` }} />
          </div>
        ) : null}
        {bar ? (
          <div className="cmh-bar-legend" aria-hidden>
            <span>{t('cycle.bar.period')}</span>
            <span>{t('cycle.bar.fertile')}</span>
            <span>{t('cycle.bar.luteal')}</span>
          </div>
        ) : null}
        {cycle.daysLate > 0 ? (
          <div className="cmh-next">
            <Icon name="drop" size={16} className="cmh-next-ic" />
            {t('cycle.late', { days: cycle.daysLate })}
          </div>
        ) : cycle.daysToPeriod != null && cycle.daysToPeriod >= 0 ? (
          <div className="cmh-next">
            <Icon name="drop" size={16} className="cmh-next-ic" />
            {t('cycle.nextPeriod', { days: cycle.daysToPeriod })}
          </div>
        ) : null}
        {partner.note ? <p className="cmh-hero-note">{partner.note}</p> : null}
      </section>
    );
  }

  // Cycle not shared (or nothing logged yet): her name and the general note.
  return (
    <section className="nb-card cmh-hero">
      <div className="cmh-overline">{t('generalOverline')}</div>
      {partner.cycle && !partner.cycle.hasData ? <p className="cmh-hero-note">{t('cycle.noData', { name })}</p> : null}
      {partner.note ? <p className="cmh-hero-note">{partner.note}</p> : null}
    </section>
  );
}

// ── «چیزهایی که سارا با تو به اشتراک گذاشته» ───────────────────────────────
function SharedRow({ icon, tone, title, sub, level, t }: { icon: IconName; tone: Tone; title: string; sub: string; level: 'view' | 'edit'; t: T }) {
  return (
    <li className="cmh-shared-row">
      <IconCircle icon={icon} tone={tone} size="md" />
      <span className="cmh-shared-text">
        <b className="cmh-shared-title">{title}</b>
        <span className="cmh-shared-sub">{sub}</span>
      </span>
      <StatusPill tone={level === 'edit' ? 'data' : 'brand'} className="cmh-level">
        {t(level === 'edit' ? 'shared.edit' : 'shared.view')}
      </StatusPill>
    </li>
  );
}

function SharedSection({ partner, name, t, locale }: { partner: CompanionPartner; name: string; t: T; locale: Locale }) {
  const grants = partner.link.grants;
  const level = (s: CompanionSection) => (grants[s] === 'edit' ? 'edit' : 'view') as 'view' | 'edit';
  const rows: React.ReactNode[] = [];

  if (partner.meds) {
    const first = partner.meds[0];
    const sub = !first
      ? t('shared.noMeds')
      : partner.meds.length > 1
        ? t('shared.more', { title: first.title, count: partner.meds.length - 1 })
        : first.times[0]
          ? t('shared.line', { title: first.title, when: clock(first.times[0], locale) })
          : first.title;
    rows.push(<SharedRow key="meds" icon="pill" tone="data" title={t('shared.meds', { name })} sub={sub} level={level('meds')} t={t} />);
  }
  if (partner.appointments) {
    const first = partner.appointments[0];
    const when = first ? appointmentWhen(first, locale) : null;
    const sub = !first ? t('shared.noAppointments') : when ? t('shared.line', { title: first.title, when }) : first.title;
    rows.push(
      <SharedRow key="appts" icon="calendar" tone="brand" title={t('shared.appointments', { name })} sub={sub} level={level('appointments')} t={t} />,
    );
  }
  if (partner.cycle) {
    rows.push(<SharedRow key="cycle" icon="drop" tone="period" title={t('shared.cycle')} sub={t('shared.aboveCard')} level={level('cycle')} t={t} />);
  }
  if (partner.pregnancy) {
    rows.push(
      <SharedRow key="preg" icon="heart" tone="bloom" title={t('shared.pregnancy')} sub={t('shared.aboveCard')} level={level('pregnancy')} t={t} />,
    );
  }
  if (partner.symptomDays != null) {
    rows.push(
      <SharedRow
        key="symptoms"
        icon="symptom"
        tone="warm"
        title={t('shared.symptoms')}
        sub={t('shared.symptomDays', { days: partner.symptomDays })}
        level={level('symptoms')}
        t={t}
      />,
    );
  }

  const hidden = COMPANION_SECTIONS.filter((s) => grants[s] === 'none').map((s) => t(`shared.sections.${s}`));
  const hiddenText = hidden.length ? new Intl.ListFormat(locale, { type: 'conjunction' }).format(hidden) : null;

  return (
    <section className="cmh-sec" aria-labelledby={`cmh-shared-${partner.link.id}`}>
      <SectionTitle id={`cmh-shared-${partner.link.id}`} title={t('shared.title', { name })} />
      <div className="nb-card cmh-shared">
        {rows.length ? <ul className="cmh-shared-list">{rows}</ul> : <p className="cmh-shared-note">{t('shared.nothing', { name })}</p>}
        {rows.length && hiddenText ? <p className="cmh-shared-note">{t('shared.notShared', { items: hiddenText })}</p> : null}
      </div>
    </section>
  );
}

// ── «امروز چه کار کنی؟» ────────────────────────────────────────────────────
const TIP_LOOK: readonly { icon: IconName; tone: Tone }[] = [
  { icon: 'heart', tone: 'period' },
  { icon: 'sparkle', tone: 'data' },
  { icon: 'walk', tone: 'brand' },
];

function Tips({ partner, t }: { partner: CompanionPartner; t: T }) {
  // Ticking a tip off is a private nudge for today — kept on screen only.
  const [done, setDone] = useState<ReadonlySet<string>>(new Set());
  if (!partner.tips.length) return null;
  return (
    <section className="cmh-sec" aria-labelledby={`cmh-tips-${partner.link.id}`}>
      <SectionTitle id={`cmh-tips-${partner.link.id}`} title={t('tips.title')} />
      <ul className="cmh-tips">
        {partner.tips.map((tip, i) => {
          const look = TIP_LOOK[i % TIP_LOOK.length]!;
          const on = done.has(tip.key);
          return (
            <li key={tip.key} className={clsx('nb-card cmh-tip', on && 'is-done')}>
              <IconCircle icon={look.icon} tone={look.tone} size="md" />
              <span className="cmh-tip-text">
                <b className="cmh-tip-title">{tip.title}</b>
                {tip.body ? <span className="cmh-tip-body">{tip.body}</span> : null}
              </span>
              <button
                type="button"
                className="cmh-tip-btn"
                aria-pressed={on}
                onClick={() =>
                  setDone((prev) => {
                    const next = new Set(prev);
                    if (on) next.delete(tip.key);
                    else next.add(tip.key);
                    return next;
                  })
                }
              >
                {on ? <Icon name="check" size={16} strokeWidth={2.4} /> : null}
                {on ? t('tips.doneOn') : t('tips.done')}
              </button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

// ── «برای خواندن» ─────────────────────────────────────────────────────────
function Articles({ articles, t }: { articles: CompanionArticle[]; t: T }) {
  if (!articles.length) return null;
  return (
    <section className="cmh-sec" aria-labelledby="cmh-articles">
      <SectionTitle id="cmh-articles" title={t('articles.title')} actionLabel={t('articles.all')} onAction={() => openSheet('articles')} />
      <ul className="cmh-articles">
        {articles.map((a, i) => (
          <li key={a.id}>
            <button type="button" className="nb-card cmh-article" onClick={() => openSheet('article', a.slug)}>
              <span className={clsx('cmh-article-art', i % 2 ? 'is-period' : 'is-bloom')} aria-hidden>
                <Icon name="bookOpen" size={22} />
              </span>
              <span className="cmh-article-body">
                <b className="cmh-article-title">{a.title ?? t('articles.article')}</b>
                <span className="cmh-article-meta">
                  {a.readTimeMinutes ? t('articles.readTime', { minutes: a.readTimeMinutes }) : t('articles.article')}
                </span>
              </span>
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}

// ── Child (B-N5 fills it) ─────────────────────────────────────────────────
function ChildPlaceholder({ t }: { t: T }) {
  return (
    <section className="cmh-sec" aria-labelledby="cmh-child">
      <SectionTitle id="cmh-child" title={t('child.title')} />
      <div className="nb-card cmh-child">
        <IconCircle icon="sprout" tone="bloom" size="lg" />
        <span className="cmh-child-text">
          <b className="cmh-child-title">{t('child.soon')}</b>
          <span className="cmh-child-sub">{t('child.soonSub')}</span>
        </span>
      </div>
    </section>
  );
}

function HomeSkeleton({ t }: { t: T }) {
  return (
    <SkeletonGroup label={t('loading')} className="cmh-skel">
      <Skeleton width="short" />
      <Skeleton width="medium" />
      <Skeleton shape="card" />
      <Skeleton shape="block" />
      <Skeleton shape="block" />
    </SkeletonGroup>
  );
}

/**
 * `/companion` — the companion panel home (B-N4-05, nbl_Hamdam_Home): per
 * linked partner her cycle/pregnancy card, what she shares (meds, appointments,
 * cycle, symptoms with the access level), «امروز چه کار کنی؟» tips, reading
 * suggestions and the child card (placeholder until B-N5). With no link the
 * empty state carries the code entry. Its own nav: امروز · خدمات · من.
 */
export function CompanionHomePage() {
  const t = useTranslations('companionHome');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const life = useLifeStage();
  const home = useCompanionHome();
  const [selected, setSelected] = useState<number | null>(null);

  // A woman's account has no companion home (403) — send her to her own.
  const notCompanion = getApiErrorStatus(home.error) === 403 || (life.data != null && !life.data.companion);
  useEffect(() => {
    if (notCompanion) router.replace('/home');
  }, [notCompanion, router]);

  const data = home.data;
  const partners = data?.partners ?? [];
  const partner = partners.find((p) => p.link.id === selected) ?? partners[0] ?? null;
  const name = partner?.partnerName ?? t('someone');
  const viewer = data?.viewer.name ?? null;

  let body: React.ReactNode;
  if (!mounted || home.isPending || notCompanion) {
    body = <HomeSkeleton t={t} />;
  } else if (home.isError || !data) {
    body = (
      <EmptyState
        icon="refresh"
        title={t('error.title')}
        body={t('error.body')}
        action={<PrimaryButton onClick={() => void home.refetch()}>{t('error.retry')}</PrimaryButton>}
      />
    );
  } else if (!partner) {
    body = (
      <section className="nb-card cmh-empty" aria-labelledby="cmh-empty-title">
        <IconCircle icon="users" tone="brand" size="lg" />
        <h2 id="cmh-empty-title" className="cmh-empty-title">
          {data.emptyState?.title ?? t('empty.title')}
        </h2>
        <p className="cmh-empty-body">{data.emptyState?.body ?? t('empty.body')}</p>
        <CodeEntry />
      </section>
    );
  } else {
    body = (
      <>
        {partners.length > 1 ? (
          <div className="cmh-switch" role="tablist" aria-label={t('partners')}>
            {partners.map((p) => {
              const on = p.link.id === partner.link.id;
              return (
                <button
                  key={p.link.id}
                  type="button"
                  role="tab"
                  aria-selected={on}
                  className={clsx('cmh-switch-chip', on && 'is-on')}
                  onClick={() => setSelected(p.link.id)}
                >
                  {p.partnerName ?? t('someone')}
                </button>
              );
            })}
          </div>
        ) : null}
        <PartnerCard partner={partner} name={name} t={t} locale={locale} />
        <SharedSection partner={partner} name={name} t={t} locale={locale} />
        <Tips key={partner.link.id} partner={partner} t={t} />
        <Articles articles={data.articles} t={t} />
        <ChildPlaceholder t={t} />
      </>
    );
  }

  return (
    <div className="view cmh-page">
      <SkyLayer />
      <div className="scroll cmh-scroll">
        <header className="cmh-hdr">
          <div className="cmh-hdr-text">
            {partner ? <div className="cmh-overline">{t('companionOf', { name })}</div> : null}
            <h1 className="cmh-greeting">{viewer ? t('greeting', { name: viewer }) : t('greetingNoName')}</h1>
          </div>
          <span className="cmh-avatar" aria-hidden>
            {viewer ? firstLetter(viewer) : <Icon name="user" size={20} />}
          </span>
        </header>
        <div className="cmh-body">{body}</div>
      </div>
      <BottomNav mode="companion" />
    </div>
  );
}
