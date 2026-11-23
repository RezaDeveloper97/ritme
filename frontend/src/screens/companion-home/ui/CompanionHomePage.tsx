'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { useModuleEnabled } from '@/entities/app-module';
import { ChildAvatar, parseCompanionChildren } from '@/entities/child';
import {
  canRecordFor,
  type CompanionPartner,
  type CompanionSection,
  type SharedAppointment,
  useCompanionHome,
} from '@/entities/companion';
import { useLifeStage } from '@/entities/user';
import { pilot } from '@/shared/pilot';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, formatWeekday, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { AppSheet } from '@/shared/sheet';
import {
  EmptyState,
  HeaderButton,
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
import {
  ActionCard,
  dailyIndex,
  homeActions,
  LEARN_TOPICS,
  LearnTile,
  learnTopic,
  useSuggestionsMuted,
} from '@/widgets/companion-support';
import { LinkedTeenCards } from '@/widgets/linked-teen-card';

import { statusCopy } from '../model/status';
import '@/shared/ui/styles/cmh.css';

type T = ReturnType<typeof useTranslations<'companionHome'>>;

/** «نکته‌ی امروز» rotation (compPersonal): neutral, non-medical lines under `companionHome.personal.tips`. */
const DAILY_TIPS = ['listen', 'ask', 'small', 'space', 'notice', 'together', 'patience'] as const;

/** Static «امروز چه کمکی می‌توانی بکنی؟» lines when the server sends no tips. */
const HELP_LINES = [
  { key: 'a', icon: 'heart' },
  { key: 'b', icon: 'glass' },
  { key: 'c', icon: 'chat' },
] as const satisfies readonly { key: string; icon: IconName }[];

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

// ── «وضعیت کلی امروز» ────────────────────────────────────────────────────────
function StatusCard({ partner, t }: { partner: CompanionPartner; t: T }) {
  const copy = statusCopy(partner.phase, partner.signals);
  return (
    <section className="cmh2-status" aria-labelledby={`cmh2-status-${partner.link.id}`}>
      <span id={`cmh2-status-${partner.link.id}`} className="cmh2-status-lbl">
        <Icon name="clock" size={15} />
        {t('status.label')}
      </span>
      <p className="cmh2-status-main">{t(`status.${copy.phase}.main`)}</p>
      <p className="cmh2-status-sub">{t(`status.${copy.phase}.sub`)}</p>
      {copy.signal ? <p className="cmh2-status-extra">{t(`status.signals.${copy.signal}`)}</p> : null}
    </section>
  );
}

// ── «امروز چه کمکی می‌توانی بکنی؟» ─────────────────────────────────────────────
function HelpCard({ partner, t }: { partner: CompanionPartner; t: T }) {
  return (
    <section className="cmh2-help" aria-labelledby="cmh2-help-title">
      <h2 id="cmh2-help-title" className="cmh2-help-title">
        {t('help.title')}
      </h2>
      <ul className="cmh2-help-list">
        {partner.tips.length
          ? partner.tips.slice(0, 3).map((tip, i) => (
              <li key={tip.key} className="cmh2-help-item">
                <span className="cmh2-help-ic" aria-hidden>
                  <Icon name={HELP_LINES[i % HELP_LINES.length]!.icon} size={15} />
                </span>
                <span>
                  {tip.title}
                  {tip.body ? <span className="cmh2-help-body">{tip.body}</span> : null}
                </span>
              </li>
            ))
          : HELP_LINES.map((line) => (
              <li key={line.key} className="cmh2-help-item">
                <span className="cmh2-help-ic" aria-hidden>
                  <Icon name={line.icon} size={15} />
                </span>
                <span>{t(`help.${partner.phase}.${line.key}`)}</span>
              </li>
            ))}
      </ul>
    </section>
  );
}

// ── «اقدام‌های پیشنهادی» ──────────────────────────────────────────────────────
function SuggestedActions({ partner, t }: { partner: CompanionPartner; t: T }) {
  const router = useRouter();
  const picked = homeActions(partner.link.settings);
  return (
    <section className="cmh-sec" aria-labelledby="cmh2-acts-title">
      <SectionTitle
        id="cmh2-acts-title"
        title={t('actions.homeTitle')}
        actionLabel={t('actions.all')}
        onAction={() => router.push('/companion/actions')}
      />
      <ul className="cmh2-acts-scroll">
        {picked.map((a) => (
          <ActionCard key={a.key} cardKey={a.key} icon={a.icon} tone={a.tone} />
        ))}
      </ul>
    </section>
  );
}

// ── «چیزهایی که سارا با تو به اشتراک گذاشته» (meds / appointments only) ─────────
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

/** «افزودن برای سارا» (B-N4-06): opens the form with her preselected in «برای چه کسی؟». */
function AddForRow({ href, label }: { href: string; label: string }) {
  return (
    <li className="cmh-shared-add">
      <Link href={href} className="cmh-add" {...pilot('companion.add-for')}>
        <Icon name="plus" size={16} strokeWidth={2.2} />
        {label}
      </Link>
    </li>
  );
}

function SharedSection({ partner, name, t, locale }: { partner: CompanionPartner; name: string; t: T; locale: Locale }) {
  const grants = partner.link.grants;
  const level = (s: CompanionSection) => (grants[s] === 'edit' ? 'edit' : 'view') as 'view' | 'edit';
  const rows: React.ReactNode[] = [];
  const ownerId = partner.link.owner.id;

  // A section is null unless she shares it (grant ≠ none), so nothing unshared ever renders here.
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
    if (canRecordFor(partner.link, 'meds')) {
      rows.push(
        <AddForRow key="meds-add" href={`/reminders/medication/new?for=${ownerId}&from=companion`} label={t('shared.addMed', { name })} />,
      );
    }
  }
  if (partner.appointments) {
    const first = partner.appointments[0];
    const when = first ? appointmentWhen(first, locale) : null;
    const sub = !first ? t('shared.noAppointments') : when ? t('shared.line', { title: first.title, when }) : first.title;
    rows.push(
      <SharedRow key="appts" icon="calendar" tone="brand" title={t('shared.appointments', { name })} sub={sub} level={level('appointments')} t={t} />,
    );
    if (canRecordFor(partner.link, 'appointments')) {
      rows.push(
        <AddForRow
          key="appts-add"
          href={`/reminders/appointment/new?for=${ownerId}&from=companion`}
          label={t('shared.addAppointment', { name })}
        />,
      );
    }
  }
  if (!rows.length) return null;

  return (
    <section className="cmh-sec" aria-labelledby={`cmh-shared-${partner.link.id}`}>
      <SectionTitle id={`cmh-shared-${partner.link.id}`} title={t('shared.title', { name })} />
      <div className="nb-card cmh-shared">
        <ul className="cmh-shared-list">{rows}</ul>
      </div>
    </section>
  );
}

// ── Child (B-N5-05: the shared children of a spouse link, read-only) ─────
function SharedChildren({ raw }: { raw: unknown }) {
  const tc = useTranslations('children');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const kids = parseCompanionChildren(raw);
  const childrenOn = useModuleEnabled('children'); // admin switch «کودک»
  if (kids.length === 0 || !childrenOn) return null;
  return (
    <section className="cmh-sec" aria-labelledby="cmh-child">
      <SectionTitle id="cmh-child" title={tc('companion.title')} />
      {kids.map((c) => (
        <Link key={c.id} href={`/children/${c.id}`} className="nb-card cmh-child" aria-label={tc('companion.open', { name: c.name })}>
          <ChildAvatar id={c.id} name={c.name} initial={c.initial} sex={c.sex} size={48} />
          <span className="cmh-child-text">
            <b className="cmh-child-title">{c.name}</b>
            <span className="cmh-child-sub">
              {[c.ageLabel, c.nextVaccine ? tc('companion.nextVaccine', { date: formatDayMonth(fromApiDate(c.nextVaccine.dueDate), locale) }) : null]
                .filter(Boolean)
                .join(' · ')}
            </span>
          </span>
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} />
        </Link>
      ))}
    </section>
  );
}

// ── compHome: linked ─────────────────────────────────────────────────────────
function PartnerPanel({ partners, t, locale }: { partners: CompanionPartner[]; t: T; locale: Locale }) {
  const [selected, setSelected] = useState<number | null>(null);
  const [about, setAbout] = useState(false);
  const [muted] = useSuggestionsMuted();
  const partner = partners.find((p) => p.link.id === selected) ?? partners[0]!;
  const name = partner.partnerName ?? t('someone');

  return (
    <>
      <header className="cmh2-hero">
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
                  {...pilot(`companion.partner:${p.link.id}`)}
                >
                  {p.partnerName ?? t('someone')}
                </button>
              );
            })}
          </div>
        ) : null}
        <div className="cmh2-hero-head">
          <div className="cmh2-hero-text">
            <span className="cmh2-overline">{t('companionOfLabel')}</span>
            <h1 className="cmh2-name">{name}</h1>
          </div>
          <HeaderButton label={t('about.open')} icon="info" onClick={() => setAbout(true)} />
        </div>
        <StatusCard partner={partner} t={t} />
      </header>

      <div className="cmh-body">
        {/* CB-TEEN-03: a father's read-only teen cards (parent links never appear as partners). */}
        <LinkedTeenCards />
        {muted ? (
          <Link href="/companion/actions" className="cmh2-muted" {...pilot('companion.open:companion-actions')}>
            <Icon name="bellPlain" size={16} />
            {t('actions.mutedNote')}
          </Link>
        ) : (
          <>
            <HelpCard key={partner.link.id} partner={partner} t={t} />
            <SuggestedActions partner={partner} t={t} />
          </>
        )}
        <section className="cmh-sec" aria-labelledby="cmh2-learn-title">
          <SectionTitle id="cmh2-learn-title" title={t('learn.homeTitle')} />
          <div className="cmh2-tiles">
            <LearnTile topic={learnTopic('cycle')} href="/companion/learn" />
            <LearnTile topic={learnTopic('support')} href="/companion/learn" />
          </div>
        </section>
        <SharedSection partner={partner} name={name} t={t} locale={locale} />
        <SharedChildren raw={partner.child} />
      </div>

      <AppSheet open={about} onClose={() => setAbout(false)} size="half" title={t('about.title')}>
        <div className="cmh2-sheet-copy">
          <p>{t('about.sees', { name })}</p>
          <p>{t('about.words')}</p>
          <p>{t('about.control', { name })}</p>
        </div>
      </AppSheet>
    </>
  );
}

// ── compPersonal: no link (also the «ورود بدون اتصال» of the male onboarding) ──
function PersonalPanel({ viewer, t }: { viewer: string | null; t: T }) {
  const rtl = useDirection() === 'rtl';
  const tip = DAILY_TIPS[dailyIndex(toApiDate(today()), DAILY_TIPS.length)]!;
  return (
    <>
      <header className="cmh2-hero is-personal">
        <span className="cmh-avatar" aria-hidden>
          {viewer ? firstLetter(viewer) : <Icon name="user" size={20} />}
        </span>
        <h1 className="cmh2-hello">{viewer ? t('personal.hello', { name: viewer }) : t('personal.helloNoName')}</h1>
        <p className="cmh2-hello-sub">{t('personal.sub')}</p>
      </header>
      <div className="cmh-body">
        <LinkedTeenCards />
        <Link href="/companion/links" className="nb-card cmh2-banner" {...pilot('companion.open:companion-links')}>
          <span className="cmh2-banner-ic" aria-hidden>
            <Icon name="users" size={22} />
          </span>
          <span className="cmh2-banner-text">
            <b>{t('personal.bannerTitle')}</b>
            <span>{t('personal.bannerSub')}</span>
          </span>
          <span className="cmh2-banner-go" aria-hidden>
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={16} />
          </span>
        </Link>
        <section className="cmh-sec" aria-labelledby="cmh2-foryou">
          <SectionTitle id="cmh2-foryou" title={t('personal.forYou')} />
          <div className="cmh2-tiles">
            {LEARN_TOPICS.map((topic) => (
              <LearnTile key={topic.id} topic={topic} href="/companion/learn" tall />
            ))}
          </div>
        </section>
        <aside className="nb-card cmh2-tip" aria-labelledby="cmh2-tip-title">
          <IconCircle icon="info" tone="brand" size="md" />
          <p className="cmh2-tip-text">
            <b id="cmh2-tip-title">{t('personal.tipLabel')}</b> {t(`personal.tips.${tip}`)}
          </p>
        </aside>
      </div>
    </>
  );
}

function HomeSkeleton({ t }: { t: T }) {
  return (
    <div className="cmh-body">
      <SkeletonGroup label={t('loading')} className="cmh-skel">
        <Skeleton width="short" />
        <Skeleton width="medium" />
        <Skeleton shape="card" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    </div>
  );
}

/**
 * `/companion` — «ریتمی همراه» v2, member side. Linked: compHome — her coarse phase in probabilistic words
 * (no calendar, no day numbers, F-01), «امروز چه کمکی می‌توانی بکنی؟», suggested action cards ordered by her
 * preferences, two learn tiles, and only the meds / appointments / children she shares. Not linked: compPersonal —
 * his own panel (learn tiles, today's tip) with the «کد همراه داری؟» banner. Companion nav, no FAB.
 */
export function CompanionHomePage() {
  const t = useTranslations('companionHome');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const life = useLifeStage();
  const home = useCompanionHome();

  // A woman's account has no companion home (403) — send her to her own.
  const notCompanion = getApiErrorStatus(home.error) === 403 || (life.data != null && !life.data.companion);
  useEffect(() => {
    if (notCompanion) router.replace('/home');
  }, [notCompanion, router]);

  const data = home.data;

  let body: React.ReactNode;
  if (!mounted || home.isPending || notCompanion) {
    body = <HomeSkeleton t={t} />;
  } else if (home.isError || !data) {
    body = (
      <div className="cmh-body">
        <EmptyState
          icon="refresh"
          title={t('error.title')}
          body={t('error.body')}
          action={<PrimaryButton onClick={() => void home.refetch()}>{t('error.retry')}</PrimaryButton>}
        />
      </div>
    );
  } else if (!data.partners.length) {
    body = <PersonalPanel viewer={data.viewer.name} t={t} />;
  } else {
    body = <PartnerPanel partners={data.partners} t={t} locale={locale} />;
  }

  return (
    <div className="view cmh-page">
      <SkyLayer />
      <div className="scroll cmh-scroll">{body}</div>
      <BottomNav mode="companion" />
    </div>
  );
}
