'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';

import {
  type IvfCompanion,
  type IvfCycle,
  type IvfDose,
  type IvfDoseDay,
  type IvfNextAppointment,
  useIvfHome,
  useIvfStages,
  useLogIvfDose,
  useSetIvfCompanionNotify,
  useStartIvfCycle,
  useUnlogIvfDose,
} from '@/entities/ivf';
import { lifeStageKeys } from '@/entities/user';
import { Link, type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  HubHeader,
  Icon,
  IconCircle,
  ListGroup,
  ListRow,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StepTimeline,
  Switch,
  type TimelineStep,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import {
  clockOf,
  dayOf,
  DOSES_ANCHOR,
  hasAmount,
  IVF_SCREENS_READY,
  stageCopy,
  stageDayChip,
  stepMeta,
  unitKey,
  whenKey,
} from '../model/home';

type T = ReturnType<typeof useTranslations<'ivf'>>;

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="view ivf-screen">
      <SkyLayer />
      <div className="scroll">{children}</div>
      <BottomNav />
    </div>
  );
}

function Header({ cycle, t }: { cycle: IvfCycle | null; t: T }) {
  const locale = useLocale() as Locale;
  return (
    <HubHeader
      className="ivf-hub"
      date={t('home.eyebrow')}
      greeting={
        cycle
          ? t('home.title', { cycle: t('home.cycle', { n: cycle.number, num: formatNumber(cycle.number, locale) }) })
          : t('home.titleBare')
      }
      actions={<IconCircle icon="drop" tone="bloom" size="lg" />}
    />
  );
}

// ── Main export ────────────────────────────────────────────────
/**
 * IVF «امروز» — `/ivf` (nbl_IVF_Home, CB-IVF-02). The home of the TTC IVF
 * sub-mode: `/home` sends a `ttc` user with bloom's «IVF/IUI» switch here, and
 * the bottom nav's «امروز» points at it. Six-stage timeline, today's
 * injections (done / log), the next appointment and the companion reminder
 * toggle (hidden without a linked companion).
 */
export function IvfHomePage() {
  const t = useTranslations('ivf');
  const mounted = useMounted();
  const query = useIvfHome();

  if (!mounted || query.isPending) {
    return (
      <Shell>
        <Header cycle={null} t={t} />
        <SkeletonGroup label={t('home.loading')} className="ivf-body">
          <Skeleton shape="card" className="ivf-skel-timeline" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell>
        <Header cycle={null} t={t} />
        <div className="ivf-body">
          <Card className="ivf-state" role="alert">
            <IconCircle icon="warning" tone="danger" size="lg" />
            <p className="ivf-state-text">{t('home.loadError')}</p>
            <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
              {t('home.retry')}
            </SecondaryButton>
          </Card>
        </div>
      </Shell>
    );
  }

  const home = query.data;
  if (!home.cycle) {
    return (
      <Shell>
        <Header cycle={null} t={t} />
        <div className="ivf-body">{home.enabled ? <StartCard t={t} /> : <OffCard t={t} />}</div>
      </Shell>
    );
  }

  return (
    <Shell>
      <Header cycle={home.cycle} t={t} />
      <div className="ivf-body">
        <TimelineCard cycle={home.cycle} t={t} />
        <DosesSection today={home.today} t={t} />
        {home.nextAppointment ? <NextAppointment next={home.nextAppointment} t={t} /> : null}
        {home.companion.linked ? <CompanionCard companion={home.companion} t={t} /> : null}
        <QuickActions t={t} />
      </div>
    </Shell>
  );
}

// ── Empty states ───────────────────────────────────────────────
function OffCard({ t }: { t: T }) {
  return (
    <Card>
      <EmptyState
        icon="drop"
        title={t('empty.offTitle')}
        body={t('empty.offBody')}
        action={
          <Link href="/profile/mode" className="nb-btn is-outline is-block">
            {t('empty.offCta')}
          </Link>
        }
      />
    </Card>
  );
}

function StartCard({ t }: { t: T }) {
  const queryClient = useQueryClient();
  const start = useStartIvfCycle();
  return (
    <Card>
      <EmptyState
        icon="drop"
        title={t('empty.startTitle')}
        body={t('empty.startBody')}
        action={
          <>
            <button
              type="button"
              className="nb-btn is-primary is-block"
              disabled={start.isPending}
              aria-busy={start.isPending || undefined}
              onClick={() =>
                start.mutate(undefined, {
                  // Starting a cycle switches «IVF/IUI» on server-side — the nav/home read it.
                  onSuccess: () => void queryClient.invalidateQueries({ queryKey: lifeStageKeys.all }),
                })
              }
            >
              {t('empty.startCta')}
            </button>
            {start.isError ? (
              <p className="ivf-error" role="alert">
                {t('empty.startError')}
              </p>
            ) : null}
          </>
        }
      />
    </Card>
  );
}

// ── Timeline ───────────────────────────────────────────────────
function TimelineCard({ cycle, t }: { cycle: IvfCycle; t: T }) {
  const locale = useLocale() as Locale;
  const stages = useIvfStages(locale);
  const chip = stageDayChip(cycle);
  const steps: TimelineStep[] = cycle.timeline.map((step) => {
    const copy = stageCopy(step.stage, stages.data);
    const meta = stepMeta(step);
    return {
      id: step.stage,
      state: step.status,
      title: copy.title ?? t(`stages.${step.stage}.title`),
      meta:
        meta === 'hint'
          ? (copy.hint ?? t(`stages.${step.stage}.hint`))
          : meta === 'date' && step.date
            ? formatDayMonth(fromApiDate(step.date), locale)
            : undefined,
    };
  });
  return (
    <Card as="section" className="ivf-timeline" aria-labelledby="ivf-stage-title">
      <div className="ivf-card-head">
        <h2 id="ivf-stage-title" className="ivf-card-title">
          {t('timeline.title')}
        </h2>
        {chip ? (
          <span className="ivf-day-chip">
            {t('timeline.stageDay', {
              day: formatNumber(chip.day, locale),
              stage: t(`stages.${chip.stage}.short`),
            })}
          </span>
        ) : null}
      </div>
      <StepTimeline
        steps={steps}
        label={t('timeline.label')}
        stateLabels={{ done: t('timeline.state.done'), current: t('timeline.state.current'), todo: t('timeline.state.todo') }}
        locale={locale}
      />
    </Card>
  );
}

// ── Today's injections ─────────────────────────────────────────
function DosesSection({ today, t }: { today: IvfDoseDay; t: T }) {
  const router = useRouter();
  const log = useLogIvfDose();
  const unlog = useUnlogIvfDose();
  const busy = log.isPending || unlog.isPending;
  return (
    <section className="ivf-section" id={DOSES_ANCHOR} aria-labelledby="ivf-doses-title">
      <SectionTitle
        id="ivf-doses-title"
        title={t('doses.title')}
        actionLabel={IVF_SCREENS_READY.meds ? t('doses.plan') : undefined}
        onAction={IVF_SCREENS_READY.meds ? () => router.push('/ivf/meds') : undefined}
      />
      {today.doses.length === 0 ? (
        <Card className="ivf-empty-line">
          <p className="ivf-muted">{t('doses.empty')}</p>
        </Card>
      ) : (
        <ListGroup className="ivf-doses">
          {today.doses.map((dose) => (
            <DoseRow
              key={`${dose.medId}-${dose.slot}`}
              dose={dose}
              busy={busy}
              t={t}
              onToggle={() => {
                const input = { medId: dose.medId, date: dose.date, slot: dose.slot };
                if (dose.taken) unlog.mutate(input);
                else log.mutate(input);
              }}
            />
          ))}
        </ListGroup>
      )}
      {log.isError || unlog.isError ? (
        <p className="ivf-error" role="alert">
          {t('doses.error')}
        </p>
      ) : null}
    </section>
  );
}

function DoseRow({ dose, busy, onToggle, t }: { dose: IvfDose; busy: boolean; onToggle: () => void; t: T }) {
  const locale = useLocale() as Locale;
  const time = formatNumber(dose.slot, locale);
  const route = t(`doses.route.${dose.route}`);
  const amount = hasAmount(dose)
    ? t(`doses.unit.${unitKey(dose.unit)}`, { dose: formatNumber(dose.dose ?? '', locale), unit: dose.unit ?? '' }).trim()
    : null;
  return (
    <ListRow
      className="ivf-dose"
      icon="syringe"
      iconTone={dose.taken ? 'data' : 'brand'}
      title={dose.name}
      description={amount ? t('doses.metaDose', { time, route, dose: amount }) : t('doses.meta', { time, route })}
      trailing={
        <button
          type="button"
          className={dose.taken ? 'ivf-dose-btn is-done' : 'ivf-dose-btn'}
          aria-pressed={dose.taken}
          aria-label={
            dose.taken ? t('doses.undoLabel', { name: dose.name, time }) : t('doses.logLabel', { name: dose.name, time })
          }
          disabled={busy}
          onClick={onToggle}
        >
          {dose.taken ? t('doses.done') : t('doses.log')}
        </button>
      }
    />
  );
}

// ── Next appointment ───────────────────────────────────────────
function NextAppointment({ next, t }: { next: IvfNextAppointment; t: T }) {
  const router = useRouter();
  const locale = useLocale() as Locale;
  const time = formatNumber(clockOf(next.scheduledAt), locale);
  const key = whenKey(next.daysUntil);
  const when =
    key === 'date'
      ? t('next.date', { date: formatDayMonth(fromApiDate(dayOf(next.scheduledAt)), locale), time })
      : t(`next.${key}`, { time });
  return (
    <section className="ivf-section" aria-labelledby="ivf-next-title">
      <SectionTitle id="ivf-next-title" title={t('next.title')} />
      <ListGroup className="ivf-next">
        <ListRow
          icon="calendar"
          iconTone="bloom"
          title={next.title}
          description={next.prep ? t('next.withPrep', { when, prep: next.prep }) : when}
          onClick={() => router.push(`/reminders/appointment/${next.id}`)}
        />
      </ListGroup>
    </section>
  );
}

// ── Companion reminders ────────────────────────────────────────
function CompanionCard({ companion, t }: { companion: IvfCompanion; t: T }) {
  const notify = useSetIvfCompanionNotify();
  return (
    <ListGroup className="ivf-companion">
      <ListRow
        id="ivf-companion"
        icon="heart"
        iconTone="bloom"
        title={t('companion.title')}
        description={t('companion.body')}
        trailing={
          <Switch
            checked={companion.notify}
            onCheckedChange={(next) => notify.mutate(next)}
            labelledBy="ivf-companion-title"
            disabled={notify.isPending}
          />
        }
      />
      {notify.isError ? (
        <p className="ivf-error" role="alert">
          {t('companion.error')}
        </p>
      ) : null}
    </ListGroup>
  );
}

// ── Quick actions (scan / two-week wait) ───────────────────────
/** Shown as CB-IVF-04/05 land their routes (IVF_SCREENS_READY) — never a 404 link. */
function QuickActions({ t }: { t: T }) {
  const actions = [
    IVF_SCREENS_READY.scan ? { href: '/ivf/scan', icon: 'note' as const, label: t('actions.scan') } : null,
    IVF_SCREENS_READY.tww ? { href: '/ivf/tww', icon: 'clock' as const, label: t('actions.tww') } : null,
  ].filter((a) => a !== null);
  if (actions.length === 0) return null;
  return (
    <div className="ivf-actions">
      {actions.map((a) => (
        <Link key={a.href} href={a.href} className="ivf-action">
          <Icon name={a.icon} size={18} />
          {a.label}
        </Link>
      ))}
    </div>
  );
}
