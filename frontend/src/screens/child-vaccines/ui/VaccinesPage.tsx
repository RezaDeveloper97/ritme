'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { type ChildHome, useChild, useDueText } from '@/entities/child';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatLongDate, formatMonthLabel, formatNumber, fromApiDate, toParts } from '@/shared/lib/date';
import { withHandoff } from '@/shared/lib/handoff';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  Checkbox,
  EmptyState,
  Icon,
  type IconName,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  type Tone,
} from '@/shared/ui';

import { isReadOnlyError, useVaccineSchedule, useVaccineWrite } from '../api/queries';
import { givenDoses, isActionable, notedDoses } from '../model/schedule';
import type { MarkTarget, VaccineSchedule, VaccineVisitDetail } from '../model/types';
import { MarkGivenSheet } from './MarkGivenSheet';

type T = ReturnType<typeof useTranslations<'children'>>;
type Tab = 'schedule' | 'card' | 'notes';
const TABS: readonly Tab[] = ['schedule', 'card', 'notes'];

function Shell({ header, children }: { header: ReactNode; children: ReactNode }) {
  return (
    <div className="view chd-screen cvx-screen">
      <SkyLayer />
      <div className="scroll">
        {header}
        <div className="chd-body">{children}</div>
      </div>
    </div>
  );
}

/** Pill tone of a visit status (artboard: done success, soon turquoise). */
function statusTone(status: string): Tone | null {
  switch (status) {
    case 'done':
      return 'success';
    case 'soon':
      return 'data';
    case 'due':
      return 'warm';
    case 'overdue':
      return 'danger';
    default:
      return null;
  }
}

/**
 * `/children/[id]/vaccines` (nbl_v16_Vaccines): progress of the national
 * schedule, tabs برنامه / کارت واکسن / یادداشت, a dose or a whole visit marked
 * as given (with its date and an optional note), un-ticking a dose removes it,
 * and «ثبت نوبت» opens the appointment form prefilled through a one-time
 * handoff (nothing about the child in the URL). A spouse sees it read-only.
 */
export function ChildVaccinesPage({ id }: { id: number }) {
  const t = useTranslations('children');
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const child = useChild(valid ? id : null);
  const schedule = useVaccineSchedule(valid ? id : 0);
  const write = useVaccineWrite(valid ? id : 0);
  const [tab, setTab] = useState<Tab>('schedule');
  const [mark, setMark] = useState<MarkTarget | null>(null);

  const name = child.data?.name ?? '';
  const header = (
    <ScreenHeader
      title={t('section.vaccines', { name })}
      subtitle={t('vaccines.subtitle')}
      onBack={() => router.push(valid ? `/children/${id}` : '/children')}
      backLabel={t('common.back')}
    />
  );

  if (!valid || getApiErrorStatus(child.error) === 404 || getApiErrorStatus(schedule.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="sprout"
          title={t('home.notFoundTitle')}
          body={t('home.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/children')}>{t('home.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }

  if (!mounted || child.isPending || schedule.isPending) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
          <Skeleton shape="block" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (child.isError || schedule.isError) {
    return (
      <Shell header={header}>
        <Card className="chd-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="chd-state-text">{t('common.loadError')}</p>
          <SecondaryButton
            icon="refresh"
            block={false}
            loading={schedule.isFetching}
            onClick={() => {
              void child.refetch();
              void schedule.refetch();
            }}
          >
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const home = child.data;
  const data = schedule.data;
  if (data.visits.length === 0) {
    return (
      <Shell header={header}>
        <EmptyState icon="syringe" title={t('vaccines.emptyTitle')} body={t('vaccines.emptyBody')} />
      </Shell>
    );
  }

  const book = (visit: VaccineVisitDetail) => {
    router.push(
      withHandoff('/reminders/appointment/new?kind=in_person', {
        topic: 'vaccine',
        title: t('visits.vaccine', { label: visit.label, name: home.name }),
        date: visit.dueDate,
      }),
    );
  };

  return (
    <Shell header={header}>
      <Summary data={data} t={t} />
      {home.role === 'shared' ? (
        <p className="cgr-readonly">
          <Icon name="eye" size={14} />
          {home.ownerName ? t('home.sharedBy', { name: home.ownerName }) : t('home.readOnly')}
        </p>
      ) : null}
      <SegmentedTabs<Tab>
        label={t('vaccines.tabsLabel')}
        value={tab}
        onChange={setTab}
        panelId={(v) => `cvx-panel-${v}`}
        tabs={TABS.map((v) => ({
          value: v,
          label: t(v === 'schedule' ? 'vaccines.tabSchedule' : v === 'card' ? 'vaccines.tabCard' : 'vaccines.tabNotes'),
        }))}
      />
      <div role="tabpanel" id={`cvx-panel-${tab}`} className="cvx-panel">
        {tab === 'schedule' ? (
          <ScheduleList
            data={data}
            home={home}
            busyCode={write.isPending ? (write.variables?.code ?? null) : null}
            onMark={setMark}
            onUnmark={(code) => write.mutate({ kind: 'unmark', code })}
            onBook={book}
            t={t}
          />
        ) : tab === 'card' ? (
          <VaccineCard data={data} name={home.name} t={t} />
        ) : (
          <Notes data={data} t={t} />
        )}
      </div>
      {write.isError && write.variables?.kind === 'unmark' ? (
        <p className="chd-error" role="alert">
          {isReadOnlyError(write.error) ? t('growth.form.readOnly') : t('vaccines.unmarkError')}
        </p>
      ) : null}
      {data.note ? (
        <InfoNote icon="info" className="cvx-note">
          {data.note}
        </InfoNote>
      ) : null}
      {home.canEdit ? (
        <MarkGivenSheet childId={home.id} birthDate={home.birthDate} target={mark} onClose={() => setMark(null)} />
      ) : null}
    </Shell>
  );
}

// ── Summary ────────────────────────────────────────────────────
function Summary({ data, t }: { data: VaccineSchedule; t: T }) {
  const locale = useLocale() as Locale;
  const due = useDueText();
  const s = data.summary;
  const next = s.next;
  const tone = next ? statusTone(next.status) : 'success';
  return (
    <Card as="section" className="cvx-summary" aria-labelledby="cvx-summary-title">
      <ProgressRing
        value={s.total > 0 ? s.given / s.total : 0}
        label={t('vaccines.ring', { given: formatNumber(s.given, locale), total: formatNumber(s.total, locale) })}
        size={64}
        thickness={7}
        tone="success"
        className="cvx-ring"
      >
        <span className="cvx-ring-num">
          {formatNumber(s.given, locale)}/{formatNumber(s.total, locale)}
        </span>
      </ProgressRing>
      <div className="cvx-summary-text">
        <b id="cvx-summary-title" className="cvx-summary-title">
          {t('vaccines.completed', { n: s.completedVisits, count: formatNumber(s.completedVisits, locale) })}
        </b>
        <span className="cvx-summary-sub">
          {next ? t('vaccines.next', { label: next.label, due: due(next.daysLeft, next.statusLabel) }) : t('vaccines.allDone')}
        </span>
      </div>
      {next && tone && next.statusLabel ? (
        <StatusPill tone={tone} icon={next.status === 'overdue' ? 'warning' : 'clock'} className="cvx-pill">
          {next.statusLabel}
        </StatusPill>
      ) : null}
    </Card>
  );
}

// ── Schedule ───────────────────────────────────────────────────
function visitIcon(visit: VaccineVisitDetail, isNext: boolean): { icon: IconName; tone: Tone } {
  if (visit.status === 'done') return { icon: 'check', tone: 'success' };
  if (visit.status === 'overdue') return { icon: 'warning', tone: 'danger' };
  if (isNext) return { icon: 'syringe', tone: 'brand' };
  return { icon: 'clock', tone: 'neutral' };
}

function ScheduleList({
  data,
  home,
  busyCode,
  onMark,
  onUnmark,
  onBook,
  t,
}: {
  data: VaccineSchedule;
  home: ChildHome;
  busyCode: string | null;
  onMark: (target: MarkTarget) => void;
  onUnmark: (code: string) => void;
  onBook: (visit: VaccineVisitDetail) => void;
  t: T;
}) {
  const locale = useLocale() as Locale;
  const due = useDueText();
  const nextCode = data.summary.next?.code ?? null;
  return (
    <Card as="section" className="cvx-visits" padding="none">
      <ol className="cvx-visit-list">
        {data.visits.map((visit) => {
          const isNext = visit.code === nextCode;
          const { icon, tone } = visitIcon(visit, isNext);
          const pillTone = statusTone(visit.status);
          const firstGiven = visit.doses.find((d) => d.givenOn)?.givenOn ?? null;
          const dueDate = fromApiDate(visit.dueDate);
          const parts = toParts(dueDate, locale);
          const sub =
            visit.status === 'done'
              ? formatLongDate(fromApiDate(firstGiven ?? visit.dueDate), locale)
              : visit.status === 'upcoming' && !isNext
                ? formatMonthLabel(parts.year, parts.month, locale)
                : t('vaccines.dueOn', { due: due(visit.daysLeft, visit.statusLabel), date: formatDayMonth(dueDate, locale) });
          const actionable = isActionable(visit, nextCode);
          return (
            <li key={visit.code} className={clsx('cvx-visit', isNext && 'is-next')}>
              <div className="cvx-visit-head">
                <IconCircle icon={icon} tone={tone} size="sm" />
                <div className="cvx-visit-titles">
                  <b className="cvx-visit-label">{visit.label}</b>
                  <span className="cvx-visit-sub">{sub}</span>
                </div>
                {pillTone && visit.statusLabel && (visit.status === 'done' || actionable) ? (
                  <StatusPill tone={pillTone} icon={visit.status === 'done' ? 'check' : 'clock'} className="cvx-pill">
                    {visit.statusLabel}
                  </StatusPill>
                ) : null}
              </div>
              <div className="cvx-doses">
                {visit.doses.map((dose) => (
                  <Checkbox
                    key={dose.code}
                    className={clsx('cvx-dose', !home.canEdit && 'is-readonly')}
                    checked={dose.givenOn !== null}
                    disabled={!home.canEdit || busyCode === dose.code}
                    label={dose.title}
                    description={
                      // A dose given on another day than the visit's date shows its own date (artboard: none).
                      dose.givenOn && dose.givenOn !== firstGiven
                        ? t('vaccines.givenOn', { date: formatLongDate(fromApiDate(dose.givenOn), locale) })
                        : undefined
                    }
                    onCheckedChange={(next) => (next ? onMark({ kind: 'dose', visit, dose }) : onUnmark(dose.code))}
                  />
                ))}
              </div>
              {actionable ? (
                <div className="cvx-actions">
                  <button type="button" className="cvx-btn is-solid" onClick={() => onBook(visit)}>
                    <Icon name="calendar" size={16} />
                    {t('vaccines.book')}
                  </button>
                  {home.canEdit ? (
                    <button type="button" className="cvx-btn" onClick={() => onMark({ kind: 'visit', visit })}>
                      <Icon name="check" size={16} />
                      {t('vaccines.markVisit')}
                    </button>
                  ) : null}
                </div>
              ) : null}
            </li>
          );
        })}
      </ol>
    </Card>
  );
}

// ── Card ───────────────────────────────────────────────────────
function VaccineCard({ data, name, t }: { data: VaccineSchedule; name: string; t: T }) {
  const locale = useLocale() as Locale;
  const given = givenDoses(data.visits);
  return (
    <Card as="section" className="cvx-card" aria-labelledby="cvx-card-title">
      <h2 id="cvx-card-title" className="chd-card-title">
        {t('vaccines.cardTitle', { name })}
      </h2>
      {given.length === 0 ? (
        <p className="cvx-empty">{t('vaccines.cardEmpty')}</p>
      ) : (
        <ul className="cvx-card-list">
          {given.map(({ visit, dose }) => (
            <li key={dose.code} className="cvx-card-row">
              <IconCircle icon="checkCircle" tone="success" size="sm" />
              <span className="cvx-card-text">
                <b className="cvx-card-name">{dose.title}</b>
                <span className="cvx-card-sub">
                  {dose.protectsAgainst
                    ? t('vaccines.doseLabel', { title: visit.label, protects: dose.protectsAgainst })
                    : visit.label}
                </span>
              </span>
              <span className="cvx-card-date">{formatLongDate(fromApiDate(dose.givenOn ?? visit.dueDate), locale)}</span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

// ── Notes ──────────────────────────────────────────────────────
function Notes({ data, t }: { data: VaccineSchedule; t: T }) {
  const locale = useLocale() as Locale;
  const noted = notedDoses(data.visits);
  return (
    <Card as="section" className="cvx-card" aria-labelledby="cvx-notes-title">
      <h2 id="cvx-notes-title" className="chd-card-title">
        {t('vaccines.notesTitle')}
      </h2>
      {data.reminderLabel ? (
        <p className="cvx-reminder">
          <Icon name="bellRing" size={16} />
          {data.reminderLabel}
        </p>
      ) : null}
      {noted.length === 0 ? (
        <p className="cvx-empty">{t('vaccines.notesEmpty')}</p>
      ) : (
        <ul className="cvx-card-list">
          {noted.map(({ dose }) => (
            <li key={dose.code} className="cvx-note-row">
              <b className="cvx-card-name">
                {t('vaccines.noteFor', {
                  title: dose.title,
                  date: formatLongDate(fromApiDate(dose.givenOn ?? ''), locale),
                })}
              </b>
              <p className="cvx-note-text">{dose.note}</p>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
