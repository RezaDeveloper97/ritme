'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';

import { AUTOPILOT_CONFIG } from '../config';
import { cancelAutopilot, pauseAutopilot, resumeAutopilot, userInterrupted } from '../dom/engine';
import { useAutopilotStore } from '../core/store';

/**
 * What the user sees while the autopilot drives: a pointer that travels to each control, a ring around it,
 * a press ripple, the step's caption and a bar that can always stop it. Fills `.app-shell`, above any sheet.
 *
 * The inline styles are positions measured from the live DOM — data no class can know (§10.1).
 */
export function AutopilotOverlay() {
  const t = useTranslations('autopilot');
  const status = useAutopilotStore((s) => s.status);
  const step = useAutopilotStore((s) => s.step);
  const total = useAutopilotStore((s) => s.total);
  const caption = useAutopilotStore((s) => s.caption);
  const pointer = useAutopilotStore((s) => s.pointer);
  const ring = useAutopilotStore((s) => s.ring);
  const press = useAutopilotStore((s) => s.press);
  const confirm = useAutopilotStore((s) => s.confirm);
  const shield = useAutopilotStore((s) => s.shield);

  if (status === 'idle') return null;

  const { visuals } = AUTOPILOT_CONFIG;
  const paused = status === 'paused';
  const asking = confirm !== null;

  return (
    <div className={clsx('ap-layer', paused && 'is-paused')}>
      {/* While it drives, a touch means «wait» — it must not also land on the control underneath. */}
      {shield && !paused && !asking ? <div className="ap-shield" onPointerDown={userInterrupted} aria-hidden /> : null}

      {ring && !paused ? (
        <span
          className="ap-ring"
          aria-hidden
          style={{ transform: `translate(${ring.x}px, ${ring.y}px)`, width: `${ring.w}px`, height: `${ring.h}px` }}
        />
      ) : null}

      {pointer && !paused ? (
        <span
          className="ap-pointer"
          aria-hidden
          style={{ transform: `translate(${pointer.x}px, ${pointer.y}px)`, transitionDuration: `${pointer.travelMs}ms` }}
        >
          {press > 0 ? <span key={press} className="ap-ripple" /> : null}
          <span className="ap-dot" />
        </span>
      ) : null}

      {visuals.statusBar || asking ? (
        <div className="ap-bar" role="status">
          <div className="ap-bar-text">
            <span className="ap-bar-title">{asking ? t('confirm.question') : paused ? t('bar.paused') : t('bar.running')}</span>
            {visuals.caption && caption && !asking ? <span className="ap-bar-caption">{caption}</span> : null}
            {!asking && total > 1 ? (
              <span className="ap-bar-count">{t('bar.progress', { step: Math.min(step + 1, total), total })}</span>
            ) : null}
          </div>
          {asking ? (
            <>
              <button type="button" className="ap-btn" onClick={() => confirm.answer(false)}>
                {t('confirm.skip')}
              </button>
              <button type="button" className="ap-btn is-primary" onClick={() => confirm.answer(true)}>
                {t('confirm.allow')}
              </button>
            </>
          ) : (
            <>
              {paused ? (
                <button type="button" className="ap-btn is-primary" onClick={resumeAutopilot}>
                  {t('bar.resume')}
                </button>
              ) : (
                <button type="button" className="ap-btn" onClick={pauseAutopilot}>
                  {t('bar.pause')}
                </button>
              )}
              <button type="button" className="ap-btn" onClick={cancelAutopilot}>
                {t('bar.stop')}
              </button>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}
