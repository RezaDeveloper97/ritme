'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useRef, useState } from 'react';

import { PELVIC_PATH, type PelvicLevel, usePelvic, useSavePelvicSession } from '@/entities/pelvic';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  CountdownRing,
  EmptyState,
  Icon,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  useTimer,
} from '@/shared/ui';

import { durationSeconds, phaseAt, phaseCount, repBarFilled, setsDone } from '../model/session';
import { useWakeLock } from '../model/wake-lock';

type Status = 'ready' | 'running' | 'finished' | 'stopped';

/**
 * «تمرین کف لگن» (`/programs/pelvic/kegel`, CB-PELV-02, nbl_Pelvic_Kegel): a full-screen guided session at today's
 * level — the CountdownRing counts each squeeze («منقبض کن ۳ از ۵ ثانیه») and relax, a bar shows the reps of the
 * set, the header the set; «توقف» ends it. A finished session (or a stop after at least one whole set) is saved as
 * today's session. The screen stays on (Wake Lock) while it runs. A flow: no bottom nav.
 */
export function PelvicKegelPage() {
  const t = useTranslations('pelvic.kegel');
  const router = useRouter();
  const query = usePelvic();
  const level = query.data?.program?.level ?? null;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('ringLabel')} className="pkg-skel">
        <Skeleton shape="circle" className="pkg-skel-ring" />
      </SkeletonGroup>
    );
  } else if (query.isError || !level) {
    body = (
      <EmptyState
        icon="target"
        title={t('noProgram')}
        action={
          <SecondaryButton block={false} onClick={() => router.push(PELVIC_PATH)}>
            {t('openPlan')}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = <Trainer level={level} />;
  }
  return <div className="view pkg-page">{body}</div>;
}

function Trainer({ level }: { level: PelvicLevel }) {
  const t = useTranslations('pelvic.kegel');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const save = useSavePelvicSession();
  const [status, setStatus] = useState<Status>('ready');
  const [index, setIndex] = useState(0);
  const startedAt = useRef(0);
  const total = phaseCount(level);
  const n = (value: number) => formatNumber(value, locale);

  useWakeLock(status === 'running');

  const record = (sets: number) => {
    if (sets < 1) return;
    save.mutate({ setsCompleted: sets, durationSec: durationSeconds(startedAt.current, Date.now()) });
  };
  const begin = () => {
    startedAt.current = Date.now();
    setIndex(0);
    setStatus('running');
  };
  const advance = () => {
    const next = index + 1;
    setIndex(next);
    if (next >= total) {
      setStatus('finished');
      record(level.sets);
    }
  };
  const stop = () => {
    setStatus('stopped');
    record(setsDone(index, level));
  };

  const phase = phaseAt(Math.min(index, total - 1), level);
  const ended = status === 'finished' || status === 'stopped';
  const filled = status === 'ready' ? 0 : repBarFilled(index, level);

  return (
    <>
      <SkyLayer />
      <div className="scroll pkg-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={t('subtitle', { level: n(level.number), set: n(phase.set), sets: n(level.sets) })}
          onBack={() => router.push(PELVIC_PATH)}
          backLabel={t('back')}
        />
        <div className="pkg-body">
          {status === 'running' ? (
            <PhaseRing
              key={index}
              kind={phase.kind}
              ms={phase.ms}
              seconds={phase.kind === 'hold' ? level.holdSec : level.restSec}
              onDone={advance}
            />
          ) : (
            <div className="pkg-ring is-idle">
              <CountdownRing
                elapsedMs={ended ? level.holdSec * 1000 : 0}
                totalMs={level.holdSec * 1000}
                mode="countdown"
                label={t('ringLabel')}
                locale={locale}
                glow
                size={252}
                caption={ended ? null : t('ready')}
                display={ended ? <Icon name="check" size={56} strokeWidth={2.4} /> : n(level.holdSec)}
                footer={ended ? t('doneLine', { sets: n(setsDone(index, level)) }) : t('ofSeconds', { seconds: n(level.holdSec) })}
              />
            </div>
          )}

          <ol className="pkg-reps" aria-label={t('repsLabel')}>
            {Array.from({ length: level.reps }, (_, i) => (
              <li key={i} className={clsx('pkg-rep', i < filled && 'is-done')} />
            ))}
          </ol>

          <p className="pkg-line" aria-live="polite">
            {status === 'ready'
              ? t('readyLine', { sets: n(level.sets), reps: n(level.reps) })
              : ended
                ? t(status === 'finished' ? 'doneTitle' : 'stoppedTitle')
                : t(phase.kind === 'hold' ? 'repLineHold' : 'repLineRelax', {
                    rep: n(phase.rep),
                    reps: n(level.reps),
                    seconds: n(level.restSec),
                  })}
          </p>

          {status === 'ready' ? (
            <button type="button" className="pkg-round" onClick={begin}>
              {t('begin')}
            </button>
          ) : status === 'running' ? (
            <button type="button" className="pkg-round" onClick={stop}>
              {t('stop')}
            </button>
          ) : (
            <div className="pkg-end">
              {save.isError ? (
                <p className="pkg-error" role="alert">
                  {t('saveError')}
                </p>
              ) : null}
              {save.isError ? (
                <SecondaryButton icon="refresh" loading={save.isPending} onClick={() => record(setsDone(index, level))}>
                  {t('retrySave')}
                </SecondaryButton>
              ) : null}
              <PrimaryButton loading={save.isPending} onClick={() => router.push(PELVIC_PATH)}>
                {save.isPending ? t('saving') : t('backToPlan')}
              </PrimaryButton>
            </div>
          )}

          <InfoNote className="pkg-note">{level.body ?? t('instruction')}</InfoNote>
        </div>
      </div>
    </>
  );
}

/** One squeeze or relax: its own clock (remounted per step), so equal hold and rest lengths restart cleanly. */
function PhaseRing({
  kind,
  ms,
  seconds,
  onDone,
}: {
  kind: 'hold' | 'relax';
  ms: number;
  seconds: number;
  onDone: () => void;
}) {
  const t = useTranslations('pelvic.kegel');
  const locale = useLocale() as Locale;
  const { elapsedMs } = useTimer({ running: true, totalMs: ms, onComplete: onDone });
  const left = Math.max(1, Math.ceil((ms - elapsedMs) / 1000));
  return (
    <div className={clsx('pkg-ring', kind === 'hold' ? 'is-hold' : 'is-relax')}>
      <CountdownRing
        elapsedMs={elapsedMs}
        totalMs={ms}
        mode="countdown"
        label={t('ringLabel')}
        locale={locale}
        tone={kind === 'hold' ? 'brand' : 'data'}
        glow
        size={252}
        caption={t(kind)}
        display={formatNumber(left, locale)}
        footer={t('ofSeconds', { seconds: formatNumber(seconds, locale) })}
      />
    </div>
  );
}
