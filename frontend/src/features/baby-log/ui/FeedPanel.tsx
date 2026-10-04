'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { NumberStepper, PrimaryButton, SegmentedTabs, Skeleton, SkeletonGroup, type SegmentedTab } from '@/shared/ui';

import { babyActionError } from '../api/queries';
import { clockText, minutesOf, wallTime } from '../model/live';
import { FEED_SIDES, FEED_TYPES, MAX_AMOUNT_ML, type BabyFeed, type FeedSide, type FeedType } from '../model/types';
import type { FeedTimerState } from '../model/use-feed-timer';
import { ManualFeedForm } from './ManualFeedForm';

type T = ReturnType<typeof useTranslations<'babyLog'>>;

/** «۱۴:۲۰ · راست · ۱۲ دقیقه» / «۰۸:۰۰ · شیشه · ۹۰ میلی‌لیتر». */
export function feedLine(feed: BabyFeed, t: T, loc: Locale): string {
  const time = formatNumber(wallTime(feed.startedAt), loc);
  const min = formatNumber(minutesOf(feed.durationSeconds), loc);
  if (feed.type === 'breast') {
    const side = feed.lastSide ? t(`side.${feed.lastSide}`) : t('types.breast');
    return t('feed.lastBreast', { time, side, min });
  }
  if (feed.amountMl != null) {
    return t('feed.lastAmount', { time, type: t(`types.${feed.type}`), ml: formatNumber(feed.amountMl, loc) });
  }
  return t('feed.lastTimed', { time, type: t(`types.${feed.type}`), min });
}

function SideCard({
  side,
  state,
  t,
  seconds,
  disabled,
  readOnly,
  onTap,
}: {
  side: FeedSide;
  state: 'running' | 'paused' | 'idle' | 'suggested';
  t: T;
  seconds: number;
  disabled: boolean;
  readOnly: boolean;
  onTap: () => void;
}) {
  const loc = useLocale() as Locale;
  const time = formatNumber(clockText(seconds), loc);
  const stateText = t(`feed.state.${state}`);
  const body = (
    <>
      <span className="bfl-side-name">{t(`side.${side}`)}</span>
      <span className="bfl-side-clock" dir="ltr">
        {time}
      </span>
      <span className="bfl-side-state">{readOnly && state !== 'running' ? ' ' : stateText}</span>
    </>
  );
  const className = clsx('bfl-side', `is-${state}`);
  if (readOnly) {
    return (
      <div className={className} role="group" aria-label={t('feed.sideAria', { side: t(`side.${side}`), time, state: stateText })}>
        {body}
      </div>
    );
  }
  return (
    <button
      type="button"
      className={className}
      onClick={onTap}
      disabled={disabled}
      aria-pressed={state === 'running'}
      aria-label={t('feed.sideAria', { side: t(`side.${side}`), time, state: stateText })}
    >
      {body}
    </button>
  );
}

/**
 * The Log_Feed timer (nbl_/nbd_Log_Feed): سینه / شیشه / پمپ tabs, two side
 * cards for the breast timer (tap = start / switch, tap the running side =
 * pause), one timer card + ml for bottle and pump, «دفعه قبل» and «امروز».
 * The «پایان و ذخیره» bar is {@link FeedFinishBar}, mounted by the screen.
 * `readOnly` (a spouse on a shared child) shows the same live state, no actions.
 */
export function FeedPanel({ timer, readOnly }: { timer: FeedTimerState; readOnly: boolean }) {
  const t = useTranslations('babyLog');
  const loc = useLocale() as Locale;
  const { day, active, type, action } = timer;
  const busy = action.isPending;

  const tabs: SegmentedTab<FeedType>[] = FEED_TYPES.map((v) => ({ value: v, label: t(`types.${v}`) }));

  if (day.isPending) {
    return (
      <SkeletonGroup label={t('loading')} className="bfl-skel">
        <Skeleton shape="block" />
        <Skeleton shape="card" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  }

  const data = day.data;
  const sideState = (side: FeedSide): 'running' | 'paused' | 'idle' | 'suggested' => {
    if (active?.type === 'breast') return active.activeSide === side ? 'running' : active.activeSide ? 'idle' : 'paused';
    return data?.nextSide === side ? 'suggested' : 'idle';
  };
  const seconds = (side: FeedSide) => (side === 'left' ? timer.leftSeconds : timer.rightSeconds);
  const summary = data?.summary;
  const actionError = action.isError ? babyActionError(action.error) ?? t('errors.save') : null;

  return (
    <section className="bfl-feed" aria-label={t('feed.region')}>
      <SegmentedTabs tabs={tabs} value={type} onChange={timer.pickType} label={t('types.label')} />
      {timer.typeLocked ? (
        <p className="bfl-note" role="status">
          {t('feed.typeLocked')}
        </p>
      ) : null}

      {type === 'breast' ? (
        // Physical sides: left card on the left in both directions (the artboard's RTL order).
        <div className="bfl-sides" dir="ltr">
          {FEED_SIDES.map((side) => (
            <SideCard
              key={side}
              side={side}
              state={sideState(side)}
              t={t}
              seconds={seconds(side)}
              disabled={busy}
              readOnly={readOnly}
              onTap={() => timer.tapSide(side)}
            />
          ))}
        </div>
      ) : (
        <div className={clsx('bfl-bottle', active && 'is-running')}>
          <span className="bfl-side-name">{t(`types.${type}`)}</span>
          <span className="bfl-side-clock" dir="ltr">
            {formatNumber(clockText(timer.elapsedSeconds), loc)}
          </span>
          {active ? (
            <span className="bfl-side-state">{t(type === 'pump' ? 'feed.state.pumping' : 'feed.state.running')}</span>
          ) : readOnly ? null : (
            <button type="button" className="bfl-start" onClick={timer.startTimer} disabled={busy}>
              {t('feed.startTimer')}
            </button>
          )}
          {readOnly ? null : (
            <NumberStepper
              label={t('feed.amount')}
              value={timer.amountMl}
              onChange={timer.setAmountMl}
              min={0}
              max={MAX_AMOUNT_ML}
              step={10}
              unit={t('units.ml')}
              decrementLabel={t('feed.amountLess')}
              incrementLabel={t('feed.amountMore')}
              locale={loc}
              className="bfl-ml"
            />
          )}
        </div>
      )}

      {actionError ? (
        <p className="bfl-error" role="alert">
          {actionError}
        </p>
      ) : null}

      {day.isError ? (
        <p className="bfl-error" role="alert">
          {t('errors.load')}{' '}
          <button type="button" className="bfl-link" onClick={() => void day.refetch()}>
            {t('errors.retry')}
          </button>
        </p>
      ) : (
        <dl className="bfl-rows">
          <div className="bfl-row">
            <dt>{t('feed.last')}</dt>
            <dd>{data?.last ? feedLine(data.last, t, loc) : t('feed.lastNone')}</dd>
          </div>
          <div className="bfl-row">
            <dt>{t('feed.today')}</dt>
            <dd>
              {summary && summary.count > 0
                ? [
                    t('feed.todayValue', {
                      count: formatNumber(summary.count, loc),
                      min: formatNumber(minutesOf(summary.totalSeconds), loc),
                    }),
                    summary.bottleMl > 0 ? t('feed.todayBottle', { ml: formatNumber(summary.bottleMl, loc) }) : null,
                    summary.pumpMl > 0 ? t('feed.todayPump', { ml: formatNumber(summary.pumpMl, loc) }) : null,
                  ]
                    .filter(Boolean)
                    .join(t('dot'))
                : t('feed.todayNone')}
            </dd>
          </div>
        </dl>
      )}

      {readOnly ? null : active ? (
        <button type="button" className="bfl-link bfl-discard" onClick={timer.discard} disabled={busy}>
          {t('feed.discard')}
        </button>
      ) : (
        <ManualFeedForm type={type} onSave={(input) => timer.run({ kind: 'manual', input })} pending={busy} />
      )}
    </section>
  );
}

/** The fixed «پایان و ذخیره» bar of Log_Feed (disabled until a feed runs). */
export function FeedFinishBar({ timer }: { timer: FeedTimerState }) {
  const t = useTranslations('babyLog');
  const running = !!timer.active;
  return (
    <div className="bfl-foot">
      {!running ? <p className="bfl-foot-hint">{t(timer.type === 'breast' ? 'feed.hintBreast' : 'feed.hintTimer')}</p> : null}
      <PrimaryButton className="bfl-finish" onClick={timer.finish} disabled={!running} loading={running && timer.action.isPending}>
        {t('feed.finish')}
      </PrimaryButton>
    </div>
  );
}
