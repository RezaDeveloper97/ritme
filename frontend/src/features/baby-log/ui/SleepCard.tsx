'use client';

import { useId, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { type Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import { Card, IconCircle, PrimaryButton, SecondaryButton, Skeleton } from '@/shared/ui';

import { babyActionError, useSleepAction, useSleepDay } from '../api/queries';
import { clockText, manualSleepRange, sleepSeconds, tehranNow, wallTime } from '../model/live';
import { useNow } from '../model/use-now';

/** Hours with one decimal («۱۴٫۲»). */
export function hoursText(seconds: number, loc: Locale): string {
  return formatDecimal(String(Math.round((seconds / 3600) * 10) / 10), loc);
}

/**
 * Baby sleep on the feeding screen (B-N5-07): «خوابید» starts the server
 * timer (one per child, so it survives a reload), «بیدار شد» stops it; a
 * past sleep goes in by hand (two times ending today). Read-only for a spouse.
 */
export function SleepCard({ childId, name, readOnly }: { childId: number; name: string; readOnly: boolean }) {
  const t = useTranslations('babyLog');
  const loc = useLocale() as Locale;
  const day = useSleepDay(childId);
  const action = useSleepAction(childId);
  const active = day.data?.active ?? null;
  const now = useNow(!!active);
  const panelId = useId();
  const [open, setOpen] = useState(false);
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [invalid, setInvalid] = useState(false);

  const saveManual = () => {
    const range = manualSleepRange(from, to, tehranNow().date);
    if (!range) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    action.mutate(
      { kind: 'manual', input: range },
      {
        onSuccess: () => {
          setFrom('');
          setTo('');
          setOpen(false);
        },
      },
    );
  };

  const summary = day.data?.summary;
  const error = action.isError ? babyActionError(action.error) ?? t('errors.save') : null;

  return (
    <Card as="section" className="bfl-card" id="sleep" aria-labelledby="bfl-sleep-title">
      <div className="bfl-card-head">
        <IconCircle icon="moon" tone="brand" size="sm" />
        <h2 id="bfl-sleep-title" className="bfl-card-title">
          {t('sleep.title', { name })}
        </h2>
      </div>
      {day.isPending ? (
        <Skeleton shape="block" />
      ) : day.isError ? (
        <p className="bfl-error" role="alert">
          {t('errors.load')}{' '}
          <button type="button" className="bfl-link" onClick={() => void day.refetch()}>
            {t('errors.retry')}
          </button>
        </p>
      ) : (
        <>
          {active ? (
            <p className="bfl-live" role="timer">
              <span className="bfl-live-clock" dir="ltr">
                {formatNumber(clockText(sleepSeconds(active, now)), loc)}
              </span>
              <span className="bfl-live-sub">{t('sleep.since', { time: formatNumber(wallTime(active.startedAt), loc) })}</span>
            </p>
          ) : null}
          <p className="bfl-card-line">
            {summary && summary.count > 0
              ? t('sleep.today', { count: formatNumber(summary.count, loc), hours: hoursText(summary.seconds, loc) })
              : t('sleep.none')}
          </p>
          {readOnly ? null : active ? (
            <PrimaryButton icon="sun" onClick={() => action.mutate({ kind: 'stop', sleepId: active.id })} loading={action.isPending}>
              {t('sleep.stop')}
            </PrimaryButton>
          ) : (
            <SecondaryButton icon="moon" onClick={() => action.mutate({ kind: 'start' })} loading={action.isPending}>
              {t('sleep.start')}
            </SecondaryButton>
          )}
        </>
      )}
      {error ? (
        <p className="bfl-error" role="alert">
          {error}
        </p>
      ) : null}
      {readOnly || active ? null : (
        <div className="bfl-manual">
          <button type="button" className="bfl-link" aria-expanded={open} aria-controls={panelId} onClick={() => setOpen((o) => !o)}>
            {t('sleep.manual')}
          </button>
          {open ? (
            <div id={panelId} className="bfl-manual-body">
              <div className="bfl-manual-pair">
                <label className="fld-label">
                  <span className="fld-label-t">{t('sleep.from')}</span>
                  <input className="field fld-input" type="time" value={from} onChange={(e) => setFrom(e.target.value)} />
                </label>
                <label className="fld-label">
                  <span className="fld-label-t">{t('sleep.to')}</span>
                  <input className="field fld-input" type="time" value={to} onChange={(e) => setTo(e.target.value)} />
                </label>
              </div>
              {invalid ? (
                <p className="bfl-error" role="alert">
                  {t('sleep.needTimes')}
                </p>
              ) : null}
              <SecondaryButton onClick={saveManual} loading={action.isPending}>
                {t('manual.save')}
              </SecondaryButton>
            </div>
          ) : null}
        </div>
      )}
    </Card>
  );
}
