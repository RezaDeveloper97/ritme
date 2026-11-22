'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useState } from 'react';

import { isDoseDay, useAppointments, useMedications, type Appointment } from '@/entities/care-reminder';
import { useCompanionCalendar } from '@/entities/companion';
import { useLifeStage } from '@/entities/user';
import { getApiErrorStatus } from '@/shared/api';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import {
  formatLongDate,
  formatNumber,
  shiftMonth,
  toApiDate,
  today,
  todayParts,
  weekdayKeys,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { pilot } from '@/shared/pilot';
import {
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { MonthCard } from '@/widgets/cycle-calendar';

import { appointmentsByDay, monthWindow, periodDaySet, pickPartner } from '../model/calendar';

import '@/shared/ui/styles/cmh.css';
import './ccal.css';

type T = ReturnType<typeof useTranslations<'companionHome'>>;

/** `?from=` the reminder forms read to come back here after save. */
const FROM = 'companion-calendar';

/** «۰۹:۳۰» from a `Y-m-d H:i:s` wall-clock value or an `HH:MM` slot. */
function clock(value: string, locale: Locale): string {
  const hhmm = value.length > 5 ? value.slice(11, 16) : value.slice(0, 5);
  return formatNumber(hhmm, locale);
}

function Legend({ t, name, showPeriod }: { t: T; name: string; showPeriod: boolean }) {
  return (
    <div className="nb-card cc-legend">
      <ul className="cc-legend-list" aria-label={t('calendar.legendTitle')}>
        {showPeriod ? (
          <li className="cc-legend-item">
            <span aria-hidden className="cc-legend-dot is-predicted" />
            {t('calendar.legendPeriod', { name })}
          </li>
        ) : null}
        <li className="cc-legend-item">
          <span aria-hidden className="cc-legend-dot is-event" />
          {t('calendar.legendAppointment')}
        </li>
      </ul>
    </div>
  );
}

/** The selected day: her likely period (if shared) + his own appointments and doses. */
function DayCard({
  date,
  locale,
  t,
  partnerName,
  isPeriod,
  appointments,
  meds,
}: {
  date: Date;
  locale: Locale;
  t: T;
  partnerName: string;
  isPeriod: boolean;
  appointments: readonly Appointment[];
  meds: readonly { id: number; title: string; times: string[] }[];
}) {
  const empty = !isPeriod && appointments.length === 0 && meds.length === 0;
  return (
    <section className="nb-card cc-daycard" aria-labelledby="ccal-day-title">
      <h2 id="ccal-day-title" className="ccal-day-title">
        {formatLongDate(date, locale)}
      </h2>
      {isPeriod ? (
        <div className="ccal-period">
          <IconCircle icon="heart" tone="bloom" size="md" />
          <p className="ccal-period-text">
            <b>{t('calendar.periodTitle', { name: partnerName })}</b>
            <span>{t('calendar.periodBody', { name: partnerName })}</span>
          </p>
        </div>
      ) : null}
      {appointments.length || meds.length ? (
        <ul className="ccal-list">
          {appointments.map((a) => (
            <li key={`a${a.id}`}>
              <Link href={`/reminders/appointment/${a.id}`} className="ccal-row" {...pilot(`companion-calendar.appointment:${a.id}`)}>
                <IconCircle icon="calendar" tone="brand" size="md" />
                <span className="ccal-row-text">
                  <b>{a.title}</b>
                  <span>{a.scheduledAt ? t('calendar.at', { time: clock(a.scheduledAt, locale) }) : null}</span>
                </span>
              </Link>
            </li>
          ))}
          {meds.map((m) => (
            <li key={`m${m.id}`}>
              <Link href={`/reminders/medication/${m.id}`} className="ccal-row" {...pilot(`companion-calendar.medication:${m.id}`)}>
                <IconCircle icon="pill" tone="data" size="md" />
                <span className="ccal-row-text">
                  <b>{m.title}</b>
                  <span>{m.times.map((s) => clock(s, locale)).join(' · ')}</span>
                </span>
              </Link>
            </li>
          ))}
        </ul>
      ) : null}
      {empty ? <p className="ccal-empty">{t('calendar.dayEmpty')}</p> : null}
    </section>
  );
}

/**
 * `/companion/calendar` — a male (companion) account's calendar. His own medications and doctor appointments
 * (`/care/*`, the same reminders every account has) with an add button for each, and — only when his partner turned
 * on «روزهای احتمالی پریود» for this link and shares her cycle — the days her period is likely to fall on, drawn as the
 * soft «predicted» band. Predictions from today on only: never her logged history, cycle day or length (F-01 + her
 * explicit consent). Companion nav.
 */
export function CompanionCalendarPage() {
  const t = useTranslations('companionHome');
  const tw = useTranslations('companionHome.calendar.weekdays');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const life = useLifeStage();

  const start = todayParts(locale);
  const [view, setView] = useState({ year: start.year, month: start.month });
  const [selected, setSelected] = useState<Date>(() => today());
  const [linkId, setLinkId] = useState<number | null>(null);

  const { from, to } = monthWindow(view.year, view.month, locale);
  const overlay = useCompanionCalendar(from, to);
  const appointmentsQuery = useAppointments('all');
  const medsQuery = useMedications();

  // A woman's account has no companion calendar — send her to her own.
  const notCompanion = getApiErrorStatus(overlay.error) === 403 || (life.data != null && !life.data.companion);
  useEffect(() => {
    if (notCompanion) router.replace('/calendar');
  }, [notCompanion, router]);

  const partners = overlay.data?.partners ?? [];
  const partner = pickPartner(partners, linkId);
  const partnerName = partner?.partnerName ?? t('someone');
  const periodDays = periodDaySet(partner);
  const shared = partner?.periodDays != null;
  const byDay = useMemo(() => appointmentsByDay(appointmentsQuery.data ?? []), [appointmentsQuery.data]);
  const meds = medsQuery.data ?? [];

  const weekdays = weekdayKeys(locale).map((k) => tw(k));
  const step = (delta: number) => setView((v) => shiftMonth(v.year, v.month, delta));

  const dayInfo = (date: Date) => {
    const iso = toApiDate(date);
    const isPeriod = periodDays.has(iso);
    const appts = byDay.get(iso)?.length ?? 0;
    const parts = [formatLongDate(date, locale)];
    if (isPeriod) parts.push(t('calendar.a11yPeriod', { name: partnerName }));
    if (appts) parts.push(t('calendar.a11yAppointments', { count: appts }));
    return { tone: isPeriod ? ('predicted' as const) : null, ovulation: false, event: appts > 0, label: parts.join('، ') };
  };

  const selectedIso = toApiDate(selected);
  const selectedMeds = meds.filter((m) => isDoseDay(m, selected));

  let body: React.ReactNode;
  if (!mounted || notCompanion || (overlay.isPending && !overlay.data)) {
    body = (
      <SkeletonGroup label={t('loading')} className="cc-skel">
        <Skeleton shape="card" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else {
    body = (
      <>
        {partners.length > 1 ? (
          <div className="cmh-switch" role="tablist" aria-label={t('partners')}>
            {partners.map((p) => {
              const on = p.linkId === partner?.linkId;
              return (
                <button
                  key={p.linkId}
                  type="button"
                  role="tab"
                  aria-selected={on}
                  className={clsx('cmh-switch-chip', on && 'is-on')}
                  onClick={() => setLinkId(p.linkId)}
                  {...pilot(`companion-calendar.partner:${p.linkId}`)}
                >
                  {p.partnerName ?? t('someone')}
                </button>
              );
            })}
          </div>
        ) : null}

        <MonthCard
          year={view.year}
          month={view.month}
          locale={locale}
          weekdays={weekdays}
          dayInfo={dayInfo}
          formatDay={(d) => formatNumber(d, locale)}
          selected={selected}
          onSelect={setSelected}
          nav={{
            onPrev: () => step(-1),
            onNext: () => step(1),
            prevLabel: t('calendar.prevMonth'),
            nextLabel: t('calendar.nextMonth'),
          }}
        />
        <Legend t={t} name={partnerName} showPeriod={shared} />

        {partner && !shared ? (
          <InfoNote icon="lock">{t('calendar.notShared', { name: partnerName })}</InfoNote>
        ) : null}
        {partner && shared ? <InfoNote>{t('calendar.disclaimer')}</InfoNote> : null}
        {!partner && overlay.data ? (
          <Link href="/companion/links" className="nb-card cmh2-banner" {...pilot('companion-calendar.open:companion-links')}>
            <span className="cmh2-banner-ic" aria-hidden>
              <Icon name="users" size={22} />
            </span>
            <span className="cmh2-banner-text">
              <b>{t('calendar.noPartnerTitle')}</b>
              <span>{t('calendar.noPartnerBody')}</span>
            </span>
          </Link>
        ) : null}

        <DayCard
          date={selected}
          locale={locale}
          t={t}
          partnerName={partnerName}
          isPeriod={periodDays.has(selectedIso)}
          appointments={byDay.get(selectedIso) ?? []}
          meds={selectedMeds}
        />

        <section className="ccal-add" aria-labelledby="ccal-add-title">
          <SectionTitle id="ccal-add-title" title={t('calendar.addTitle')} />
          <PrimaryButton
            icon="plus"
            onClick={() => router.push(`/reminders/medication/new?from=${FROM}`)}
            {...pilot('companion-calendar.add-medication')}
          >
            {t('calendar.addMedication')}
          </PrimaryButton>
          <SecondaryButton
            icon="calendar"
            onClick={() => router.push(`/reminders/appointment/new?from=${FROM}`)}
            {...pilot('companion-calendar.add-appointment')}
          >
            {t('calendar.addAppointment')}
          </SecondaryButton>
        </section>
      </>
    );
  }

  return (
    <div className="view cmh-page ccal-page">
      <SkyLayer />
      <ScreenHeader title={t('calendar.title')} subtitle={t('calendar.subtitle')} />
      <div className="scroll cmh-scroll">
        <div className="cc-body">{body}</div>
      </div>
      <BottomNav mode="companion" />
    </div>
  );
}
