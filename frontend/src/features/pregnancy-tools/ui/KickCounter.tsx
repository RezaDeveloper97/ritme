'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { getApiErrorCode, getApiErrorMessage } from '@/shared/api';
import { Link } from '@/shared/i18n';
import { Card, Icon, PrimaryButton, SectionTitle, StatusPill } from '@/shared/ui';

import { useStopKicks } from '../api/queries';
import { buzz } from '../model/haptics';
import { kickView } from '../model/timing';
import type { KickOverview, KickSession } from '../model/types';
import { useKickTaps } from '../model/use-kick-taps';
import { useNow } from '../model/use-now';
import { useToolFormat } from './format';

const RING = 240;
const STROKE = 14;
const RADIUS = (RING - STROKE) / 2;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

interface KickCounterProps {
  overview: KickOverview;
}

/**
 * «شمارش حرکات جنین» (Log_Kick): one big tap target per movement towards the
 * target (10), elapsed time from the server's start, the 2-hour guidance and,
 * past the window with fewer movements, the non-diagnostic «تماس بگیر» advice.
 * A running count lives on the server, so a reload picks it up where it was.
 */
export function KickCounter({ overview }: KickCounterProps) {
  const t = useTranslations('pregnancyTools.kick');
  const tc = useTranslations('pregnancyTools.common');
  const f = useToolFormat();
  const active = overview.active;
  const taps = useKickTaps(active);
  const stop = useStopKicks();
  const [saved, setSaved] = useState<KickSession | null>(null);
  const now = useNow(active !== null);

  const view = active ? kickView(active, now, taps.pending) : null;
  const target = view?.target ?? overview.target;
  const count = view?.count ?? Math.max(0, taps.pending);
  const progress = Math.min(1, count / Math.max(1, target));
  const windowText = f.span(active?.windowMinutes ?? overview.windowMinutes);
  const reached = view?.reached ?? false;

  // One longer buzz the moment the target is reached.
  const reachedRef = useRef(reached);
  useEffect(() => {
    if (reached && !reachedRef.current) buzz([40, 60, 40]);
    reachedRef.current = reached;
  }, [reached]);

  const error = taps.error ?? stop.error;
  const notPregnant = getApiErrorCode(error) === 'pregnancy_not_active';

  const onTap = () => {
    buzz(15);
    setSaved(null);
    stop.reset();
    void taps.tap();
  };

  const onStop = () => {
    if (!active || taps.busy) return;
    stop.mutate(active.id, { onSuccess: (s) => setSaved(s) });
  };

  const fmtTarget = f.num(target);

  return (
    <>
      <div className="ptl-body">
        <p className="ptl-lead">{t('lead')}</p>

        <div className="ptl-kick-ring">
          <svg width={RING} height={RING} viewBox={`0 0 ${RING} ${RING}`} aria-hidden className="ptl-kick-svg">
            <circle className="ptl-kick-track" cx={RING / 2} cy={RING / 2} r={RADIUS} strokeWidth={STROKE} />
            {progress > 0 ? (
              <circle
                className="ptl-kick-arc"
                cx={RING / 2}
                cy={RING / 2}
                r={RADIUS}
                strokeWidth={STROKE}
                strokeDasharray={`${CIRCUMFERENCE * progress} ${CIRCUMFERENCE}`}
                transform={`rotate(-90 ${RING / 2} ${RING / 2})`}
              />
            ) : null}
          </svg>
          <button
            type="button"
            className={clsx('ptl-kick-btn', reached && 'is-reached')}
            aria-label={t('tapLabel', { count: f.num(count), target: fmtTarget })}
            onClick={onTap}
          >
            <span className="ptl-kick-num" aria-hidden>
              {f.num(count)}
            </span>
            <span className="ptl-kick-of" aria-hidden>
              {active || taps.pending > 0 ? t('ofTarget', { target: fmtTarget }) : t('tapToStart')}
            </span>
          </button>
        </div>
        <p className="sr-only" aria-live="polite" aria-atomic="true">
          {active || taps.pending > 0 ? t('live', { count: f.num(count), target: fmtTarget }) : ''}
        </p>

        {active ? (
          <button
            type="button"
            className="ptl-undo"
            disabled={count <= 0}
            onClick={() => {
              buzz(10);
              taps.undo(count);
            }}
          >
            <Icon name="minus" size={16} />
            {t('undo')}
          </button>
        ) : null}

        <div className="ptl-stats">
          <Card className="ptl-stat is-center">
            <span className="ptl-stat-label">{t('elapsed')}</span>
            <span className="ptl-stat-value">{f.dur(view?.elapsedMs ?? 0)}</span>
          </Card>
          <Card className="ptl-stat is-center">
            <span className="ptl-stat-label">{t('started')}</span>
            <span className="ptl-stat-value">{active ? f.time(active.startedAt) : '—'}</span>
          </Card>
        </div>

        {saved ? (
          <p className="ptl-saved" role="status">
            <Icon name="checkCircle" size={18} />
            {t('saved', { count: f.num(saved.kicks), time: f.dur(saved.elapsedSeconds * 1000) })}
          </p>
        ) : null}

        {view?.reached && active ? (
          <Card className="ptl-note is-success" role="status">
            <Icon name="checkCircle" size={18} className="ptl-note-icon" />
            <div className="ptl-note-text">
              <b className="ptl-note-title">
                {t('reached', {
                  target: fmtTarget,
                  time: f.dur((active.timeToTargetSeconds ?? Math.floor(view.elapsedMs / 1000)) * 1000),
                })}
              </b>
              <span>{t('reachedBody')}</span>
            </div>
          </Card>
        ) : view?.lowCount || (saved && saved.lowCount) ? (
          <Card className="ptl-advice" role="note">
            <div className="ptl-advice-head">
              <Icon name="warning" size={18} className="ptl-advice-icon" />
              <b className="ptl-note-title">{t('lowTitle', { target: fmtTarget, hours: windowText })}</b>
            </div>
            <p className="ptl-advice-body">{t('lowBody')}</p>
            <a className="nb-btn is-outline is-block ptl-call" href="tel:">
              <Icon name="phone" size={18} />
              {t('call')}
            </a>
          </Card>
        ) : (
          <Card className="ptl-note" role="note">
            <Icon name="info" size={17} className="ptl-note-icon is-warm" />
            <span className="ptl-note-text">{t('guidance', { target: fmtTarget, hours: windowText })}</span>
          </Card>
        )}

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

        <KickHistory overview={overview} />
      </div>

      {active ? (
        <div className="ptl-footer">
          <PrimaryButton onClick={onStop} loading={stop.isPending} disabled={taps.busy}>
            {t('stop')}
          </PrimaryButton>
        </div>
      ) : null}
    </>
  );
}

function KickHistory({ overview }: { overview: KickOverview }) {
  const t = useTranslations('pregnancyTools.kick');
  const f = useToolFormat();
  return (
    <section className="ptl-sec" aria-labelledby="ptl-kick-history">
      <SectionTitle id="ptl-kick-history" title={t('history')} />
      {overview.today.kicks > 0 ? (
        <p className="ptl-today">{t('todayTotal', { count: f.num(overview.today.kicks) })}</p>
      ) : null}
      {overview.history.length === 0 ? (
        <Card className="ptl-empty">{t('historyEmpty')}</Card>
      ) : (
        <Card padding="none" className="ptl-list">
          <ul className="ptl-rows">
            {overview.history.map((s) => (
              <li key={s.id} className="ptl-row">
                <div className="ptl-row-main">
                  <span className="ptl-row-title">{t('historyCount', { count: f.num(s.kicks), target: f.num(s.target) })}</span>
                  <span className="ptl-row-sub">{f.dayTime(s.startedAt)}</span>
                </div>
                {s.reachedTarget && s.timeToTargetSeconds !== null ? (
                  <StatusPill tone="data" icon="check">
                    {t('historyReached', { target: f.num(s.target), time: f.dur(s.timeToTargetSeconds * 1000) })}
                  </StatusPill>
                ) : (
                  <span className="ptl-row-meta">{t('historyLength', { time: f.dur(s.elapsedSeconds * 1000) })}</span>
                )}
              </li>
            ))}
          </ul>
        </Card>
      )}
    </section>
  );
}
