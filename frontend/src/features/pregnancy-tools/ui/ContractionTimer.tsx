'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useState } from 'react';

import type { PregnancyAlertV2 } from '@/entities/pregnancy';
import { getApiErrorCode, getApiErrorMessage } from '@/shared/api';
import { Link } from '@/shared/i18n';
import { Card, Icon, PrimaryButton, SectionTitle, StatusPill } from '@/shared/ui';

import { useAckToolAlert, useFinishContractions, useToggleContraction } from '../api/queries';
import { buzz } from '../model/haptics';
import { contractionRows, contractionView, isStaleContraction, secondsToMs } from '../model/timing';
import type { ContractionOverview, ContractionSession } from '../model/types';
import { useNow } from '../model/use-now';
import { useToolFormat } from './format';

interface ContractionTimerProps {
  overview: ContractionOverview;
}

/**
 * «زمان‌سنج انقباض» (Log_Contraction): tap when a contraction starts, tap
 * again when it ends. Live length and the interval building up come from the
 * server's timestamps (a reload resumes); averages over the last hour and the
 * 5-1-1 check are the server's. When the stop raises the 5-1-1 alert, its
 * admin-authored card (call / got it) shows here and in the alert centre.
 */
export function ContractionTimer({ overview }: ContractionTimerProps) {
  const t = useTranslations('pregnancyTools.contraction');
  const tc = useTranslations('pregnancyTools.common');
  const f = useToolFormat();
  const active = overview.active;
  const toggle = useToggleContraction();
  const finish = useFinishContractions();
  const [alerts, setAlerts] = useState<PregnancyAlertV2[]>([]);
  const [live, setLive] = useState('');
  const [saved, setSaved] = useState<ContractionSession | null>(null);
  const now = useNow(active !== null);
  const view = contractionView(active, now);
  const contracting = view.phase === 'contracting';
  const stale = isStaleContraction(view);
  const busy = toggle.isPending || finish.isPending;

  const onToggle = () => {
    if (busy) return;
    buzz(contracting ? [20, 40, 20] : 25);
    setSaved(null);
    finish.reset();
    toggle.mutate(
      { running: contracting },
      {
        onSuccess: ({ session, alerts: raised }) => {
          if (raised.length) {
            setAlerts(raised);
            buzz([60, 80, 60]);
          }
          const latest = session.contractions[0];
          setLive(
            session.running
              ? t('liveStarted')
              : latest?.durationSeconds != null
                ? t('liveEnded', { time: f.dur(latest.durationSeconds * 1000) })
                : '',
          );
        },
      },
    );
  };

  const onFinish = () => {
    if (!active || busy) return;
    finish.mutate(active.id, { onSuccess: (s) => setSaved(s) });
  };

  const error = toggle.error ?? finish.error;
  const notPregnant = getApiErrorCode(error) === 'pregnancy_not_active';
  const rows = active ? contractionRows(active, now) : [];
  const avgDuration = secondsToMs(active?.avgDurationSeconds ?? null);
  const avgInterval = secondsToMs(active?.avgIntervalSeconds ?? null);
  const met = Boolean(active && (active.fiveOneOne.met || active.alertAt));
  const runText = f.span(overview.params.runMinutes);

  return (
    <>
      <div className="ptl-body">
        <button
          type="button"
          className={clsx('ptl-ctr-btn', contracting && !stale && 'is-running')}
          aria-pressed={contracting}
          aria-label={stale ? t('staleLabel') : contracting ? t('endAria', { time: f.dur(view.runningMs) }) : t('startAria')}
          disabled={busy || stale}
          onClick={onToggle}
        >
          <span className="ptl-ctr-label" aria-hidden>
            {stale ? t('staleLabel') : contracting ? t('running') : view.phase === 'resting' ? t('sinceLast') : t('ready')}
          </span>
          <span className="ptl-ctr-time" aria-hidden>
            {stale ? '—' : f.dur(contracting ? view.runningMs : (view.sinceLastStartMs ?? 0))}
          </span>
          <span className="ptl-ctr-cta" aria-hidden>
            {stale ? t('staleCta') : contracting ? t('tapToEnd') : t('tapToStart')}
          </span>
        </button>
        <p className="sr-only" aria-live="polite" aria-atomic="true">
          {live}
        </p>

        <div className="ptl-stats ptl-ctr-stats">
          <Card className="ptl-stat is-center">
            <span className="ptl-stat-label">{t('avgDuration')}</span>
            <span className="ptl-stat-value">{avgDuration === null ? '—' : f.dur(avgDuration)}</span>
          </Card>
          <Card className="ptl-stat is-center">
            <span className="ptl-stat-label">{t('avgInterval')}</span>
            <span className="ptl-stat-value">{avgInterval === null ? '—' : f.dur(avgInterval)}</span>
          </Card>
        </div>
        {active && active.count > 0 ? <p className="ptl-window">{t('window', { run: runText })}</p> : null}

        {saved ? (
          <p className="ptl-saved" role="status">
            <Icon name="checkCircle" size={18} />
            {t('saved', { count: f.num(saved.count) })}
          </p>
        ) : null}

        {stale && active ? (
          <Card className="ptl-note" role="note">
            <Icon name="info" size={17} className="ptl-note-icon" />
            <span className="ptl-note-text">{t('staleBody', { time: f.dayTime(active.startedAt) })}</span>
          </Card>
        ) : null}

        {alerts.map((a) => (
          <AlertCard key={a.id} alert={a} onDone={() => setAlerts((list) => list.filter((x) => x.id !== a.id))} />
        ))}
        {alerts.length === 0 && met ? (
          <Card className="ptl-advice is-urgent" role="note">
            <div className="ptl-advice-head">
              <Icon name="warning" size={18} className="ptl-advice-icon" />
              <b className="ptl-note-title">{t('metTitle')}</b>
            </div>
            <p className="ptl-advice-body">{t('metBody')}</p>
            <a className="nb-btn is-primary is-block ptl-call" href="tel:">
              <Icon name="phone" size={18} />
              {t('call')}
            </a>
            <Link href="/pregnancy/alerts" className="ptl-link">
              {t('seeAlerts')}
            </Link>
          </Card>
        ) : null}

        {active ? (
          <Card padding="none" className="ptl-table" role="table" aria-label={t('tableLabel')}>
            <div className="ptl-tr is-head" role="row">
              <span role="columnheader">{t('colStart')}</span>
              <span role="columnheader">{t('colDuration')}</span>
              <span role="columnheader">{t('colInterval')}</span>
            </div>
            {rows.length === 0 ? <p className="ptl-table-empty">{t('empty')}</p> : null}
            {rows.map((r) => (
              <div key={r.id} className={clsx('ptl-tr', r.running && 'is-running')} role="row">
                <span role="cell">{f.num(r.start)}</span>
                <span role="cell">{r.durationMs === null || (r.running && stale) ? '—' : f.dur(r.durationMs)}</span>
                <span role="cell" className="ptl-td-muted">
                  {r.intervalMs === null ? '—' : f.dur(r.intervalMs)}
                </span>
              </div>
            ))}
          </Card>
        ) : (
          <Card className="ptl-empty">{t('empty')}</Card>
        )}

        <Card className="ptl-note" role="note">
          <Icon name="info" size={17} className="ptl-note-icon is-danger" />
          <span className="ptl-note-text">
            {t('guidance', { interval: f.num(overview.params.intervalMaxMinutes), run: runText })}
          </span>
        </Card>

        {error ? (
          notPregnant ? (
            <Card className="ptl-note is-error" role="alert">
              <Icon name="info" size={17} className="ptl-note-icon" />
              <div className="ptl-note-text">
                <b className="ptl-note-title">{tc('notPregnantTitle')}</b>
                <span>{tc('notPregnantBody')}</span>
                <Link href="/pregnancy/setup" className="ptl-link">
                  {tc('setup')}
                </Link>
              </div>
            </Card>
          ) : (
            <p className="ptl-error" role="alert">
              {getApiErrorMessage(error) ?? tc('error')}
            </p>
          )
        ) : null}

        <ContractionHistory overview={overview} />
      </div>

      {active ? (
        <div className="ptl-footer">
          <PrimaryButton onClick={onFinish} loading={finish.isPending} disabled={toggle.isPending}>
            {t('finish')}
          </PrimaryButton>
        </div>
      ) : null}
    </>
  );
}

/** The 5-1-1 alert as the server raised it: admin copy, a call CTA and «دیدم، ممنون». */
function AlertCard({ alert, onDone }: { alert: PregnancyAlertV2; onDone: () => void }) {
  const t = useTranslations('pregnancyTools.contraction');
  const ack = useAckToolAlert();
  const phone = alert.contact?.phone?.replace(/[^\d+]/g, '') ?? '';
  const call = alert.actions.find((a) => a.key === 'call');
  const ackAction = alert.actions.find((a) => a.key === 'ack');
  return (
    <Card as="article" className="ptl-advice is-urgent" role="alert" aria-labelledby={`ptl-alert-${alert.id}`}>
      <div className="ptl-advice-head">
        <Icon name="phone" size={18} className="ptl-advice-icon" />
        <b id={`ptl-alert-${alert.id}`} className="ptl-note-title">
          {alert.title}
        </b>
      </div>
      {alert.whatWeSaw ? <p className="ptl-advice-body">{alert.whatWeSaw}</p> : null}
      {alert.advice ? <p className="ptl-advice-body">{alert.advice}</p> : null}
      {alert.howSure ? <p className="ptl-advice-sub">{alert.howSure}</p> : null}
      {alert.contact?.text ? <p className="ptl-advice-sub is-strong">{alert.contact.text}</p> : null}
      {/* The call CTA always shows: with no number on the rule, `tel:` opens the dialer. */}
      <a className="nb-btn is-primary is-block ptl-call" href={`tel:${phone}`}>
        <Icon name="phone" size={18} />
        {call?.label ?? t('call')}
      </a>
      {ackAction ? (
        <button
          type="button"
          className="nb-btn is-text is-block"
          disabled={ack.isPending}
          onClick={() => ack.mutate(alert.id, { onSuccess: onDone })}
        >
          {ackAction.label ?? t('ack')}
        </button>
      ) : null}
    </Card>
  );
}

function ContractionHistory({ overview }: { overview: ContractionOverview }) {
  const t = useTranslations('pregnancyTools.contraction');
  const f = useToolFormat();
  if (overview.history.length === 0) return null;
  return (
    <section className="ptl-sec" aria-labelledby="ptl-ctr-history">
      <SectionTitle id="ptl-ctr-history" title={t('history')} />
      <Card padding="none" className="ptl-list">
        <ul className="ptl-rows">
          {overview.history.map((s) => {
            const interval = secondsToMs(s.avgIntervalSeconds);
            const duration = secondsToMs(s.avgDurationSeconds);
            return (
              <li key={s.id} className="ptl-row">
                <div className="ptl-row-main">
                  <span className="ptl-row-title">{t('historyCount', { count: f.num(s.count) })}</span>
                  <span className="ptl-row-sub">{f.dayTime(s.startedAt)}</span>
                  {interval !== null && duration !== null ? (
                    <span className="ptl-row-sub">
                      {t('historyAvg', { interval: f.dur(interval), duration: f.dur(duration) })}
                    </span>
                  ) : null}
                </div>
                {s.fiveOneOne.met || s.alertAt ? (
                  <StatusPill tone="danger" icon="warning">
                    {t('met')}
                  </StatusPill>
                ) : null}
              </li>
            );
          })}
        </ul>
      </Card>
    </section>
  );
}
