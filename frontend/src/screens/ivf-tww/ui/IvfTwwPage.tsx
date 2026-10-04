'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  IVF_TWW_MOODS,
  type IvfLutealMed,
  type IvfOutcome,
  type IvfOutcomeResult,
  type IvfTww,
  type IvfTwwMood,
  useIvfDangerSigns,
  useIvfGuidance,
  useIvfTww,
  useSetIvfTwwMood,
} from '@/entities/ivf';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  IconCircle,
  InfoNote,
  ListGroup,
  ListRow,
  PillChip,
  ProgressRing,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

import { countdown, dangerCopy, guidanceBody, waitProgress } from '../model/tww';

import { OutcomeSheet } from './OutcomeSheet';
import { OutcomeView } from './OutcomeView';

type T = ReturnType<typeof useTranslations<'ivf'>>;

/** Units the dose copy knows («۴۰۰ میلی‌گرم»); same keys as the IVF home. */
const KNOWN_UNITS = ['iu', 'mg', 'mcg', 'ml'] as const;

/** First-strong isolate: a Latin medicine name must not reorder the RTL line around it. */
function isolate(text: string): string {
  return `⁨${text}⁩`;
}

function Shell({ subtitle, children }: { subtitle?: string; children: React.ReactNode }) {
  const t = useTranslations('ivf');
  const router = useRouter();
  return (
    <div className="view ivf-screen tww-screen">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('tww.title')}
          subtitle={subtitle}
          onBack={() => router.push('/ivf')}
          backLabel={t('tww.back')}
        />
        {children}
      </div>
    </div>
  );
}

// ── Main export ────────────────────────────────────────────────
/**
 * «دو هفته انتظار» — `/ivf/tww` (nbl_IVF_TWW, CB-IVF-05). Countdown to the
 * beta blood test, today's mood chips, the early-home-test note, luteal-support
 * doses, the danger note (115) and the result buttons: positive → bloom's
 * pregnancy setup; negative → a calm screen that offers CB-LOSS-02's `/loss`
 * (never a second loss flow) or the IVF home; cancelled → the IVF home. A
 * sensitive screen with a back button: no tab bar, banners or shop.
 */
export function IvfTwwPage() {
  const t = useTranslations('ivf');
  const locale = useLocale() as Locale;
  const mounted = useMounted();
  const router = useRouter();
  const query = useIvfTww();
  const [pending, setPending] = useState<IvfOutcomeResult | null>(null);
  // The saved result stays on screen even after the refreshed read (no open cycle any more) lands.
  const [outcome, setOutcome] = useState<IvfOutcome | null>(null);

  if (outcome) {
    return (
      <Shell>
        <OutcomeView outcome={outcome} />
      </Shell>
    );
  }

  if (!mounted || query.isPending) {
    return (
      <Shell>
        <SkeletonGroup label={t('tww.loading')} className="ivf-body">
          <Skeleton shape="circle" className="tww-skel-ring" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (query.isError) {
    return (
      <Shell>
        <div className="ivf-body">
          <Card className="ivf-state" role="alert">
            <IconCircle icon="warning" tone="danger" size="lg" />
            <p className="ivf-state-text">{t('tww.loadError')}</p>
            <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
              {t('tww.retry')}
            </SecondaryButton>
          </Card>
        </div>
      </Shell>
    );
  }

  const tww = query.data;
  if (!tww.cycle) {
    return (
      <Shell>
        <EmptyState
          icon="clock"
          title={t('tww.noCycleTitle')}
          body={t('tww.noCycleBody')}
          action={
            <SecondaryButton block={false} onClick={() => router.push('/ivf')}>
              {t('tww.noCycleCta')}
            </SecondaryButton>
          }
        />
      </Shell>
    );
  }

  const subtitle = tww.transferOn
    ? t('tww.transfer', { date: formatDayMonth(fromApiDate(tww.transferOn), locale) })
    : undefined;

  return (
    <>
      <Shell subtitle={subtitle}>
        <TwwBody tww={tww} t={t} onResult={setPending} />
      </Shell>
      {/* The sheet sits outside the scroller: AppSheet is positioned against the screen. */}
      {pending ? (
        <OutcomeSheet
          result={pending}
          onClose={() => setPending(null)}
          onSaved={(saved) => {
            setPending(null);
            setOutcome(saved);
          }}
        />
      ) : null}
    </>
  );
}

function TwwBody({ tww, t, onResult }: { tww: IvfTww; t: T; onResult: (result: IvfOutcomeResult) => void }) {
  const locale = useLocale() as Locale;
  const guidance = useIvfGuidance(locale).data;
  return (
    <div className="ivf-body tww-body">
      <Countdown tww={tww} t={t} />
      <MoodCard tww={tww} t={t} note={guidanceBody(guidance, 'tww_feelings') ?? t('tww.mood.note')} />
      <InfoNote icon="info" className="tww-early">
        <strong className="tww-early-title">
          {guidance?.find((g) => g.code === 'early_test')?.title ?? t('tww.earlyTest.title')}
        </strong>{' '}
        {guidanceBody(guidance, 'early_test') ?? t('tww.earlyTest.body')}
      </InfoNote>
      <LutealMeds meds={tww.lutealSupport} t={t} />
      <DangerNote t={t} />
      <ResultButtons t={t} onResult={onResult} />
    </div>
  );
}

function Countdown({ tww, t }: { tww: IvfTww; t: T }) {
  const locale = useLocale() as Locale;
  const state = countdown(tww.daysToBeta, tww.daysSinceTransfer);
  let value: string;
  let caption: string;
  switch (state.kind) {
    case 'days':
      value = t('tww.ring.days', { n: state.days, num: formatNumber(state.days, locale) });
      caption = t('tww.ring.toBeta');
      break;
    case 'today':
      value = t('tww.ring.today');
      caption = t('tww.ring.todayCaption');
      break;
    case 'passed':
      value = t('tww.ring.passed');
      caption = t('tww.ring.passedCaption');
      break;
    case 'since':
      value = t('tww.ring.sinceTransfer', { num: formatNumber(state.day, locale) });
      caption = t('tww.ring.sinceCaption');
      break;
    default:
      value = '—';
      caption = t('tww.ring.noDate');
  }
  return (
    <section className="tww-ring-wrap">
      <ProgressRing
        value={waitProgress(tww.daysSinceTransfer, tww.daysToBeta)}
        label={t('tww.ring.label')}
        valueText={`${value} ${caption}`}
        size={184}
        thickness={18}
        className="tww-ring"
      >
        <span className={state.kind === 'days' ? 'tww-ring-value' : 'tww-ring-value is-text'}>{value}</span>
        <span className="tww-ring-caption">{caption}</span>
      </ProgressRing>
    </section>
  );
}

function MoodCard({ tww, t, note }: { tww: IvfTww; t: T; note: string }) {
  const ids = useId();
  const setMood = useSetIvfTwwMood();
  const choose = (mood: IvfTwwMood, on: boolean) => {
    setMood.mutate({ date: tww.today, mood: on ? mood : null });
  };
  return (
    <Card as="section" className="tww-mood" aria-labelledby={`${ids}-mood`}>
      <h2 id={`${ids}-mood`} className="tww-card-title">
        {t('tww.mood.title')}
      </h2>
      <ChipGroup label={t('tww.mood.label')} className="tww-chips">
        {IVF_TWW_MOODS.map((mood) => (
          <PillChip
            key={mood}
            pressed={tww.todayMood === mood}
            onPressedChange={(on) => choose(mood, on)}
          >
            {t(`tww.mood.options.${mood}`)}
          </PillChip>
        ))}
      </ChipGroup>
      {setMood.isError ? (
        <p className="ivf-error" role="alert">
          {t('tww.mood.error')}
        </p>
      ) : null}
      <p className="tww-note">{note}</p>
      <button type="button" className="tww-link" onClick={() => openSheet('log')}>
        <Icon name="pen" size={16} />
        {t('tww.mood.symptoms')}
      </button>
    </Card>
  );
}

function LutealMeds({ meds, t }: { meds: readonly IvfLutealMed[]; t: T }) {
  const ids = useId();
  const locale = useLocale() as Locale;
  const router = useRouter();
  const meta = (med: IvfLutealMed) => {
    const times = med.slots.map((s) => formatNumber(s.slot, locale)).join(' · ');
    const amount = med.dose?.trim();
    if (!amount) return t('tww.meds.meta', { times });
    const unit = med.unit?.trim().toLowerCase() ?? '';
    const known = (KNOWN_UNITS as readonly string[]).includes(unit) ? (unit as (typeof KNOWN_UNITS)[number]) : null;
    const dose = known
      ? t(`doses.unit.${known}`, { dose: formatNumber(amount, locale) })
      : t('doses.unit.other', { dose: formatNumber(amount, locale), unit: med.unit ?? '' });
    return t('tww.meds.metaDose', { times, dose: dose.trim() });
  };
  return (
    <section className="ivf-section" aria-labelledby={`${ids}-meds`}>
      <h2 id={`${ids}-meds`} className="tww-section-title">
        {t('tww.meds.title')}
      </h2>
      <ListGroup>
        {meds.length ? (
          meds.map((med) => (
            <ListRow
              key={med.medId}
              icon="pill"
              iconTone="data"
              title={isolate(med.name)}
              description={meta(med)}
              trailing={
                med.total > 0 ? (
                  <span
                    className={med.taken >= med.total ? 'tww-count is-done' : 'tww-count'}
                    aria-label={t('tww.meds.countLabel', {
                      taken: formatNumber(med.taken, locale),
                      total: formatNumber(med.total, locale),
                    })}
                  >
                    {t('tww.meds.count', {
                      taken: formatNumber(med.taken, locale),
                      total: formatNumber(med.total, locale),
                    })}
                  </span>
                ) : undefined
              }
            />
          ))
        ) : (
          <ListRow
            icon="plus"
            title={t('tww.meds.add')}
            description={t('tww.meds.empty')}
            onClick={() => router.push('/ivf/meds/new')}
          />
        )}
      </ListGroup>
    </section>
  );
}

function DangerNote({ t }: { t: T }) {
  const locale = useLocale() as Locale;
  const signs = useIvfDangerSigns(locale).data;
  const copy = dangerCopy(signs);
  const bodies = copy.bodies.length ? copy.bodies : [t('tww.danger.body')];
  return (
    <UrgentCard
      variant="note"
      title={copy.title ?? t('tww.danger.title')}
      hotlinesLabel={t('tww.danger.hotlines')}
      hotlines={copy.hotlines.map((number) => ({
        number,
        label: t('tww.danger.emergency'),
        display: formatNumber(number, locale),
      }))}
      className="tww-danger"
    >
      {bodies.map((body) => (
        <p key={body} className="tww-danger-body">
          {body}
        </p>
      ))}
    </UrgentCard>
  );
}

function ResultButtons({ t, onResult }: { t: T; onResult: (result: IvfOutcomeResult) => void }) {
  const ids = useId();
  return (
    <section className="tww-results" aria-labelledby={`${ids}-result`}>
      <h2 id={`${ids}-result`} className="sr-only">
        {t('tww.result.title')}
      </h2>
      <div className="tww-result-row">
        <button type="button" className="tww-result is-positive" onClick={() => onResult('positive')}>
          <Icon name="heart" size={18} />
          {t('tww.result.positive')}
        </button>
        <button type="button" className="tww-result" onClick={() => onResult('negative')}>
          <Icon name="chat" size={18} />
          {t('tww.result.negative')}
        </button>
      </div>
      <button type="button" className="tww-link is-quiet" onClick={() => onResult('cancelled')}>
        {t('tww.result.cancelled')}
      </button>
    </section>
  );
}
