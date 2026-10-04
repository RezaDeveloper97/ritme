'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type KeyboardEvent, useRef, useState } from 'react';

import {
  type IvfCycle,
  type IvfDose,
  type IvfDoseDay,
  type IvfInjectionSite,
  type IvfMed,
  type IvfMedsView,
  type IvfTrigger,
  useIvfGuidance,
  useIvfInjectionSites,
  useIvfMeds,
  useLogIvfScheduleDose,
  useUnlogIvfScheduleDose,
} from '@/entities/ivf';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, formatWeekday, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  Icon,
  IconCircle,
  ListGroup,
  ListRow,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import {
  chosenSite,
  isKnownSite,
  rotationNote,
  runsOutLabel,
  siteForLog,
  siteHint,
  siteTitle,
  sortForInventory,
  stockState,
} from '../model/schedule';

type T = ReturnType<typeof useTranslations<'ivf'>>;

/** Units the dose copy knows («۱۵۰ واحد»); same keys as the IVF home. */
const KNOWN_UNITS = ['iu', 'mg', 'mcg', 'ml'] as const;
type UnitKey = (typeof KNOWN_UNITS)[number] | 'other';
function unitKey(unit: string | null): UnitKey {
  const u = unit?.trim().toLowerCase() ?? '';
  return (KNOWN_UNITS as readonly string[]).includes(u) ? (u as UnitKey) : 'other';
}

/** First-strong isolate: a Latin medicine name («FSH», «hCG») must not reorder the RTL line around it. */
function isolate(text: string): string {
  return `\u2068${text}\u2069`;
}

function useSiteName(catalog: readonly IvfInjectionSite[] | undefined, t: T) {
  return (code: string) => siteTitle(code, catalog) ?? (isKnownSite(code) ? t(`meds.sites.names.${code}`) : code);
}

function Shell({ cycle, t, children }: { cycle: IvfCycle | null; t: T; children: React.ReactNode }) {
  const router = useRouter();
  const locale = useLocale() as Locale;
  return (
    <div className="view ivf-screen ivfm-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('meds.title')}
          subtitle={
            cycle
              ? t('meds.subtitle', {
                  cycle: t('home.cycle', { n: cycle.number, num: formatNumber(cycle.number, locale) }),
                  stage: t(`stages.${cycle.stage}.short`),
                })
              : undefined
          }
          onBack={() => router.push('/ivf')}
          backLabel={t('meds.back')}
        />
        {children}
      </div>
      <BottomNav />
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
/**
 * «برنامه تزریق» — `/ivf/meds` (nbl_IVF_Meds, CB-IVF-03), the IVF stage tab
 * «درمان». Trigger card, today's (log with the injection site) and tomorrow's
 * doses, the 8-site rotation picker (the API's least-recently-used site is
 * pre-selected) and the medicine stock with a «کم است» badge; «افزودن دارو از
 * روی نسخه» opens the form. A tab root: the bottom nav stays.
 */
export function IvfMedsPage() {
  const t = useTranslations('ivf');
  const mounted = useMounted();
  const query = useIvfMeds();

  if (!mounted || query.isPending) {
    return (
      <Shell cycle={null} t={t}>
        <SkeletonGroup label={t('meds.loading')} className="ivf-body">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
          <Skeleton shape="card" className="ivfm-skel-sites" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell cycle={null} t={t}>
        <div className="ivf-body">
          <Card className="ivf-state" role="alert">
            <IconCircle icon="warning" tone="danger" size="lg" />
            <p className="ivf-state-text">{t('meds.loadError')}</p>
            <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
              {t('meds.retry')}
            </SecondaryButton>
          </Card>
        </div>
      </Shell>
    );
  }

  const view = query.data;
  if (!view.cycle) {
    return (
      <Shell cycle={null} t={t}>
        <div className="ivf-body">
          <Card>
            <EmptyState
              icon="syringe"
              title={t('meds.noCycleTitle')}
              body={t('meds.noCycleBody')}
              action={
                <Link href="/ivf" className="nb-btn is-outline is-block">
                  {t('meds.noCycleCta')}
                </Link>
              }
            />
          </Card>
        </div>
      </Shell>
    );
  }

  return (
    <Shell cycle={view.cycle} t={t}>
      <Schedule view={view} t={t} />
    </Shell>
  );
}

function Schedule({ view, t }: { view: IvfMedsView; t: T }) {
  const locale = useLocale() as Locale;
  const sitesCatalog = useIvfInjectionSites(locale);
  const [picked, setPicked] = useState<string | null>(null);
  const chosen = chosenSite(view.sites, picked);
  const siteName = useSiteName(sitesCatalog.data, t);
  const log = useLogIvfScheduleDose();
  const unlog = useUnlogIvfScheduleDose();
  const busy = log.isPending || unlog.isPending;

  const toggle = (dose: IvfDose) => {
    const input = { medId: dose.medId, date: dose.date, slot: dose.slot };
    if (dose.taken) {
      unlog.mutate(input);
      return;
    }
    // The pick is used once: after the log the API suggests the next site in rotation.
    log.mutate({ ...input, site: siteForLog(dose, chosen) }, { onSuccess: () => setPicked(null) });
  };

  return (
    <div className="ivf-body">
      {view.trigger ? <TriggerCard trigger={view.trigger} t={t} /> : null}

      <DayList
        day={view.today}
        title={t('meds.today')}
        empty={t('meds.todayEmpty')}
        id="ivfm-today"
        t={t}
        siteName={siteName}
        action={{ busy, onToggle: toggle }}
      />
      {log.isError || unlog.isError ? (
        <p className="ivf-error" role="alert">
          {t('doses.error')}
        </p>
      ) : null}
      <DayList
        day={view.tomorrow}
        title={t('meds.tomorrow')}
        empty={t('meds.tomorrowEmpty')}
        id="ivfm-tomorrow"
        t={t}
        siteName={siteName}
      />

      {view.sites.codes.length ? (
        <SitePicker
          view={view}
          chosen={chosen}
          onPick={setPicked}
          catalog={sitesCatalog.data}
          siteName={siteName}
          t={t}
        />
      ) : null}

      <Inventory meds={view.meds} today={view.today.date} t={t} />

      <Link href="/ivf/meds/new" className="ivfm-add">
        <Icon name="plus" size={18} />
        {t('meds.add')}
      </Link>
    </div>
  );
}

// ── Trigger ────────────────────────────────────────────────────
function TriggerCard({ trigger, t }: { trigger: IvfTrigger; t: T }) {
  const locale = useLocale() as Locale;
  const guidance = useIvfGuidance(locale);
  const copy = guidance.data?.find((g) => g.code === 'trigger_timing');
  const time = formatNumber(trigger.triggerAt.slice(11, 16), locale);
  const date = formatDayMonth(fromApiDate(trigger.triggerAt.slice(0, 10)), locale);
  return (
    <section className={clsx('ivfm-trigger', trigger.taken && 'is-done')} aria-labelledby="ivfm-trigger-title">
      <IconCircle icon="clock" tone="bloom" size="lg" />
      <div className="ivfm-trigger-text">
        <h2 id="ivfm-trigger-title" className="ivfm-trigger-title">
          {copy?.title ?? t('meds.trigger.title')}
        </h2>
        <p className="ivfm-trigger-body">{copy?.body ?? t('meds.trigger.body')}</p>
        <p className="ivfm-trigger-when">
          <span>{t('meds.trigger.when', { name: isolate(trigger.name), date, time })}</span>
          {trigger.taken ? <span className="ivfm-pill is-ok">{t('meds.trigger.done')}</span> : null}
        </p>
      </div>
    </section>
  );
}

// ── Today / tomorrow ───────────────────────────────────────────
function DayList({
  day,
  title,
  empty,
  id,
  t,
  siteName,
  action,
}: {
  day: IvfDoseDay;
  title: string;
  empty: string;
  id: string;
  t: T;
  siteName: (code: string) => string;
  action?: { busy: boolean; onToggle: (dose: IvfDose) => void };
}) {
  return (
    <section className="ivf-section" aria-labelledby={id}>
      <h2 id={id} className="ivfm-day-title">
        {title}
      </h2>
      {day.doses.length === 0 ? (
        <Card className="ivf-empty-line">
          <p className="ivf-muted">{empty}</p>
        </Card>
      ) : (
        <ListGroup className="ivf-doses">
          {day.doses.map((dose) => (
            <DoseRow key={`${dose.medId}-${dose.slot}`} dose={dose} t={t} siteName={siteName} action={action} />
          ))}
        </ListGroup>
      )}
    </section>
  );
}

function DoseRow({
  dose,
  t,
  siteName,
  action,
}: {
  dose: IvfDose;
  t: T;
  siteName: (code: string) => string;
  action?: { busy: boolean; onToggle: (dose: IvfDose) => void };
}) {
  const locale = useLocale() as Locale;
  const time = formatNumber(dose.slot, locale);
  const amount = dose.dose?.trim()
    ? t(`doses.unit.${unitKey(dose.unit)}`, { dose: formatDecimal(dose.dose, locale), unit: dose.unit ?? '' }).trim()
    : null;
  const title = amount ? `${isolate(dose.name)} · ${amount}` : dose.name;
  const meta = time;
  return (
    <ListRow
      className="ivf-dose"
      icon="syringe"
      iconTone={dose.taken ? 'data' : 'brand'}
      title={title}
      description={dose.taken && dose.site ? t('meds.withSite', { meta, site: siteName(dose.site) }) : meta}
      trailing={
        action ? (
          <button
            type="button"
            className={dose.taken ? 'ivf-dose-btn is-done' : 'ivf-dose-btn'}
            aria-pressed={dose.taken}
            aria-label={
              dose.taken ? t('doses.undoLabel', { name: dose.name, time }) : t('doses.logLabel', { name: dose.name, time })
            }
            disabled={action.busy}
            onClick={() => action.onToggle(dose)}
          >
            {dose.taken ? t('doses.done') : t('doses.log')}
          </button>
        ) : undefined
      }
    />
  );
}

// ── Site rotation ──────────────────────────────────────────────
/** Next radio index for an arrow / Home / End key in a 2-column grid (APG radiogroup). */
function radioStep(key: string, index: number, count: number, rtl: boolean): number | null {
  if (key === (rtl ? 'ArrowLeft' : 'ArrowRight') || key === 'ArrowDown') return (index + 1) % count;
  if (key === (rtl ? 'ArrowRight' : 'ArrowLeft') || key === 'ArrowUp') return (index - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return null;
}

function SitePicker({
  view,
  chosen,
  onPick,
  catalog,
  siteName,
  t,
}: {
  view: IvfMedsView;
  chosen: string | null;
  onPick: (code: string) => void;
  catalog: readonly IvfInjectionSite[] | undefined;
  siteName: (code: string) => string;
  t: T;
}) {
  const rtl = useDirection() === 'rtl';
  const refs = useRef<Array<HTMLButtonElement | null>>([]);
  const codes = view.sites.codes;
  const note = rotationNote(view.sites, chosen);
  const hint = siteHint(chosen, catalog);
  const checkedIndex = chosen ? codes.indexOf(chosen) : -1;
  const tabStop = checkedIndex >= 0 ? checkedIndex : 0;

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const next = radioStep(event.key, index, codes.length, rtl);
    if (next === null) return;
    event.preventDefault();
    onPick(codes[next]!);
    refs.current[next]?.focus();
  };

  return (
    <section className="ivf-section" aria-labelledby="ivfm-sites-title">
      <h2 id="ivfm-sites-title" className="ivfm-section-title">
        {t('meds.sites.title')}
      </h2>
      <Card className="ivfm-sites">
        <div role="radiogroup" aria-label={t('meds.sites.label')} className="ivfm-site-grid">
          {codes.map((code, index) => {
            const isLast = note.last === code;
            return (
              <button
                key={code}
                ref={(el) => {
                  refs.current[index] = el;
                }}
                type="button"
                role="radio"
                aria-checked={code === chosen}
                tabIndex={index === tabStop ? 0 : -1}
                className={clsx('ivfm-site', isLast && 'is-last')}
                onClick={() => onPick(code)}
                onKeyDown={(event) => onKeyDown(event, index)}
              >
                <span className="ivfm-site-name">{siteName(code)}</span>
                {isLast ? (
                  <span className="ivfm-site-tag">
                    <Icon name="history" size={12} />
                    {t('meds.sites.lastTag')}
                  </span>
                ) : null}
              </button>
            );
          })}
        </div>
        <p className="ivfm-site-note" aria-live="polite">
          {note.last ? `${t('meds.sites.last', { site: siteName(note.last) })} ` : null}
          {note.today ? `${t(`meds.sites.${note.today.kind}`, { site: siteName(note.today.site) })} ` : null}
          {hint}
        </p>
      </Card>
    </section>
  );
}

// ── Inventory ──────────────────────────────────────────────────
function Inventory({ meds, today, t }: { meds: readonly IvfMed[]; today: string; t: T }) {
  const router = useRouter();
  const locale = useLocale() as Locale;
  return (
    <section className="ivf-section" aria-labelledby="ivfm-stock-title">
      <h2 id="ivfm-stock-title" className="ivfm-section-title">
        {t('meds.inventory.title')}
      </h2>
      {meds.length === 0 ? (
        <Card className="ivf-empty-line">
          <p className="ivf-muted">{t('meds.empty')}</p>
        </Card>
      ) : (
        <ListGroup className="ivfm-stock">
          {sortForInventory(meds).map((med) => {
            const state = stockState(med.inventory);
            const inv = med.inventory;
            const runsOut = runsOutLabel(inv, today);
            const left = inv
              ? t('meds.inventory.left', {
                  units: t(`meds.inventory.units.${inv.stockUnit}`, {
                    count: inv.unitsLeft,
                    num: formatNumber(inv.unitsLeft, locale),
                  }),
                })
              : t('meds.inventory.none');
            const until = runsOut
              ? t('meds.inventory.until', {
                  day:
                    runsOut.kind === 'weekday'
                      ? formatWeekday(fromApiDate(runsOut.on), locale)
                      : formatDayMonth(fromApiDate(runsOut.on), locale),
                })
              : null;
            return (
              <ListRow
                key={med.id}
                className="ivfm-stock-row"
                icon="box"
                iconTone={state === 'low' ? 'bloom' : 'data'}
                title={med.name}
                description={until && state === 'low' ? `${left} · ${until}` : left}
                trailing={
                  state === 'none' ? undefined : (
                    <span className={clsx('ivfm-pill', state === 'low' ? 'is-low' : 'is-ok')}>
                      {t(`meds.inventory.${state}`)}
                    </span>
                  )
                }
                onClick={() => router.push(`/ivf/meds/${med.id}`)}
              />
            );
          })}
        </ListGroup>
      )}
    </section>
  );
}
