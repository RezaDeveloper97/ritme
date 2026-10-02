'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  HOT_FLASH_SEVERITIES,
  HOT_FLASH_TRIGGERS,
  type HotFlashDetails,
  type MenopauseFlash,
  type MenopauseFlashDay,
  useHotFlashDay,
  useHotFlashTimer,
  useMenopauseTips,
} from '@/entities/menopause';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  CountdownRing,
  EmptyState,
  formatClock,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SectionTitle,
  SeverityScale,
  severityTone,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
  useTimer,
} from '@/shared/ui';

import {
  detailsOf,
  EMPTY_DETAILS,
  flashElapsedSeconds,
  flashLength,
  severityIndex,
  startClock,
  toggleTrigger,
} from '../model/flash';

/** Code of the `meno_tips` item placed on this screen (docs/canvas-build/menopause.md §5). */
const BREATHING_TIP = 'hot_flash_breathing';

/** mm:ss stays left-to-right inside RTL text (LRI … PDI isolate). */
const isolate = (text: string) => `⁦${text}⁩`;

/**
 * «گرگرفتگی» (`/menopause/hot-flash`, CB-MENO-07, nbl_Meno_HotFlash): one tap
 * starts the timer (persisted server-side, so a running one resumes here or
 * on the home), another stops it and saves the length. Severity, sweat and
 * causes ride along on start/stop, and edit the flash just stopped. Below:
 * today's tiles + list and the breathing tip. A flow: no bottom nav.
 */
export function MenopauseHotFlashPage() {
  const t = useTranslations('menopause.hotFlash');
  const router = useRouter();
  const query = useHotFlashDay();

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="mhf-skel">
        <Skeleton shape="block" className="mhf-skel-ring" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="flame"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <HotFlashBody day={query.data} fetchedAt={query.dataUpdatedAt} />;
  }

  return (
    <div className="view mhf-page">
      <SkyLayer />
      <div className="scroll mhf-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/home')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}

function HotFlashBody({ day, fetchedAt }: { day: MenopauseFlashDay; fetchedAt: number }) {
  const t = useTranslations('menopause.hotFlash');
  const { start, stop } = useHotFlashTimer();
  const running = day.running;
  const [details, setDetails] = useState<HotFlashDetails>(() => (running ? detailsOf(running) : EMPTY_DETAILS));
  const [syncedRunId, setSyncedRunId] = useState<number | null>(running?.id ?? null);
  /** The flash stopped on this visit: the chips now edit it. */
  const [saved, setSaved] = useState<MenopauseFlash | null>(null);

  // A timer started elsewhere (the home) or resumed after a reload brings its own details.
  if (running && running.id !== syncedRunId) {
    setSyncedRunId(running.id);
    setDetails(detailsOf(running));
    setSaved(null);
  }

  const busy = start.isPending || stop.isPending;

  const change = (next: HotFlashDetails) => {
    setDetails(next);
    // Running: sent on stop. Just stopped: the stop route edits a stopped flash in place.
    if (!running && saved) stop.mutate({ id: saved.id, details: next }, { onSuccess: (flash) => setSaved(flash) });
  };

  const toggle = () => {
    if (busy) return;
    if (running) {
      stop.mutate({ id: running.id, details }, { onSuccess: (flash) => setSaved(flash) });
      return;
    }
    // A new flash starts clean once the previous one is saved.
    const fresh = saved ? EMPTY_DETAILS : details;
    if (saved) setDetails(EMPTY_DETAILS);
    setSaved(null);
    start.mutate(fresh);
  };

  return (
    <>
      <section className="mhf-timer" aria-label={t('ring.label')}>
        {running ? (
          <RunningRing key={`${running.id}-${fetchedAt}`} flash={running} disabled={busy} onToggle={toggle} />
        ) : (
          <IdleRing saved={saved} disabled={busy} onToggle={toggle} />
        )}
        <p className="mhf-hint" aria-live="polite">
          {running ? t('ring.runningHint') : saved ? t('ring.savedHint') : t('ring.idleHint')}
        </p>
        {start.isError || stop.isError ? (
          <p className="mhf-error" role="alert">
            {t('error')}
          </p>
        ) : null}
      </section>

      <DetailsCard details={details} onChange={change} />

      <TodaySection day={day} />

      <BreathingTip />
    </>
  );
}

function RingFace({
  seconds,
  caption,
  ariaLabel,
  pressed,
  disabled,
  onToggle,
}: {
  seconds: number;
  caption: string;
  ariaLabel: string;
  pressed: boolean;
  disabled: boolean;
  onToggle: () => void;
}) {
  const t = useTranslations('menopause.hotFlash');
  const locale = useLocale() as Locale;
  return (
    <button
      type="button"
      className={clsx('mhf-ring-btn', pressed && 'is-running')}
      aria-pressed={pressed}
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={onToggle}
    >
      <CountdownRing
        elapsedMs={seconds * 1000}
        label={t('ring.label')}
        locale={locale}
        tone="bloom"
        glow
        size={220}
        thickness={6}
        caption={
          <span className="mhf-ring-cap">
            <Icon name="flame" size={30} className="mhf-ring-icon" />
            <span>{caption}</span>
          </span>
        }
      />
    </button>
  );
}

function RunningRing({ flash, disabled, onToggle }: { flash: MenopauseFlash; disabled: boolean; onToggle: () => void }) {
  const t = useTranslations('menopause.hotFlash');
  const locale = useLocale() as Locale;
  const { elapsedMs } = useTimer({ running: true });
  const seconds = flashElapsedSeconds(flash.elapsedS, elapsedMs);
  const time = isolate(formatNumber(formatClock(seconds * 1000), locale));
  return (
    <RingFace
      seconds={seconds}
      caption={t('ring.runningCaption')}
      ariaLabel={t('ring.stopLabel', { time })}
      pressed
      disabled={disabled}
      onToggle={onToggle}
    />
  );
}

function IdleRing({ saved, disabled, onToggle }: { saved: MenopauseFlash | null; disabled: boolean; onToggle: () => void }) {
  const t = useTranslations('menopause.hotFlash');
  const seconds = saved?.durationS ?? 0;
  return (
    <RingFace
      seconds={seconds}
      caption={saved ? t('ring.savedCaption') : t('ring.idleCaption')}
      ariaLabel={t('ring.startLabel')}
      pressed={false}
      disabled={disabled}
      onToggle={onToggle}
    />
  );
}

function DetailsCard({ details, onChange }: { details: HotFlashDetails; onChange: (next: HotFlashDetails) => void }) {
  const t = useTranslations('menopause.hotFlash');
  const severityOptions = HOT_FLASH_SEVERITIES.map((value) => ({ value, label: t(`severity.${value}`) }));
  return (
    <Card as="section" className="mhf-card" aria-label={t('detailsLabel')}>
      <SeverityScale
        label={t('severity.title')}
        options={severityOptions}
        value={details.severity}
        withNone={false}
        onChange={(severity) => onChange({ ...details, severity })}
      />
      <ChipGroup label={t('sweatLabel')} className="mhf-sweat">
        <PillChip
          mode="multi"
          tone="data"
          icon="drop"
          pressed={details.sweat}
          onPressedChange={(sweat) => onChange({ ...details, sweat })}
        >
          {t('sweat')}
        </PillChip>
      </ChipGroup>
      <div className="mhf-causes">
        <h3 className="mhf-label">
          {t('triggers.title')}
        </h3>
        <ChipGroup label={t('triggers.title')}>
          {HOT_FLASH_TRIGGERS.map((code) => (
            <PillChip
              key={code}
              mode="multi"
              tone="brand"
              pressed={details.triggers.includes(code)}
              onPressedChange={() => onChange({ ...details, triggers: toggleTrigger(details.triggers, code) })}
            >
              {t(`triggers.${code}`)}
            </PillChip>
          ))}
        </ChipGroup>
      </div>
    </Card>
  );
}

function useLength() {
  const t = useTranslations('menopause.hotFlash.today');
  const locale = useLocale() as Locale;
  return (seconds: number) => {
    const { unit, value } = flashLength(seconds);
    return t(unit, { n: formatNumber(value, locale) });
  };
}

function Tile({ value, label, tone }: { value: string; label: string; tone: Tone }) {
  return (
    <div className={clsx('mhf-tile', `nb-tone-${tone}`)}>
      <b className="mhf-tile-value">{value}</b>
      <span className="mhf-tile-label">{label}</span>
    </div>
  );
}

function TodaySection({ day }: { day: MenopauseFlashDay }) {
  const t = useTranslations('menopause.hotFlash.today');
  const locale = useLocale() as Locale;
  const length = useLength();
  return (
    <section className="mhf-sec" aria-labelledby="mhf-today">
      <SectionTitle id="mhf-today" title={t('title')} />
      <Card className="mhf-today" padding="sm">
        <div className="mhf-tiles">
          <Tile value={formatNumber(day.count, locale)} label={t('count')} tone="period" />
          <Tile value={day.avgDurationS === null ? t('noValue') : length(day.avgDurationS)} label={t('avg')} tone="bloom" />
          <Tile value={formatNumber(day.nightCount, locale)} label={t('night')} tone="brand" />
        </div>
        {day.items.length === 0 ? (
          <p className="mhf-empty">{t('empty')}</p>
        ) : (
          <ul className="mhf-list" aria-label={t('listLabel')}>
            {day.items.map((flash) => (
              <FlashRow key={flash.id} flash={flash} />
            ))}
          </ul>
        )}
      </Card>
    </section>
  );
}

function FlashRow({ flash }: { flash: MenopauseFlash }) {
  const t = useTranslations('menopause.hotFlash');
  const locale = useLocale() as Locale;
  const length = useLength();
  const index = severityIndex(flash.severity);
  const clock = startClock(flash.startedAt);
  const title = [
    flash.severity ? t(`severity.${flash.severity}`) : null,
    flash.sweat ? t('today.withSweat') : null,
    flash.running ? t('today.running') : flash.durationS !== null ? length(flash.durationS) : null,
  ].filter(Boolean);
  const sub = [...flash.triggers.map((code) => t(`triggers.${code}`)), flash.night ? t('today.nightTag') : null].filter(
    Boolean,
  );
  return (
    <li className="mhf-row">
      <span className="mhf-time">{clock ? isolate(formatNumber(clock, locale)) : null}</span>
      <span
        className={clsx('mhf-dot', `nb-tone-${index < 0 ? 'neutral' : severityTone(index, false)}`)}
        aria-hidden="true"
      />
      <span className="mhf-row-text">
        <span className="mhf-row-title">{title.length ? title.join(' · ') : t('today.noDetails')}</span>
        {sub.length ? <span className="mhf-row-sub">{sub.join(' · ')}</span> : null}
      </span>
    </li>
  );
}

function BreathingTip() {
  const t = useTranslations('menopause.hotFlash');
  const locale = useLocale();
  const tips = useMenopauseTips(locale);
  if (tips.isPending) return null;
  const tip = tips.data?.find((item) => item.code === BREATHING_TIP && item.body);
  // Catalog unreachable or the item removed by an admin: the board's copy (needs clinical review too).
  return <InfoNote className="mhf-tip">{tip?.body ?? t('tipFallback')}</InfoNote>;
}
