'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { type PointerEvent as ReactPointerEvent, useRef, useState } from 'react';

import { useDirection } from '@/shared/i18n';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';

import { resolveSwipeIndex } from '../lib/swipe';
import { hasSlideBody, INTRO_SLIDES, SKIP_TARGET } from '../model/slides';
import { IntroIllustration } from './IntroIllustration';

interface Props {
  /** 0-based slide to open on (deep link / QA). */
  initialIndex?: number;
  /** «شروع کن» on the last slide. */
  onComplete: () => void;
  /** «حساب دارم · ورود» — an existing user going straight to sign-in. */
  onLogin: () => void;
}

/**
 * The five Night & Bloom intro slides (`Intro_1` … `Intro_5`). Owns swipe,
 * progress and the per-slide CTA; the screen decides where to go next.
 *
 * The track follows the reading direction, so in RTL the next slide comes in
 * from the left and a rightward swipe advances — the progress dots in the
 * header run the same way. Auto-play is intentionally absent.
 */
export function IntroCarousel({ initialIndex = 0, onComplete, onLogin }: Props) {
  const t = useTranslations('welcome');
  const dir = useDirection();
  const count = INTRO_SLIDES.length;
  // +1 in RTL: slide n sits n widths to the left, so the track moves right.
  const sign = dir === 'rtl' ? 1 : -1;

  const [index, setIndex] = useState(() => Math.min(Math.max(0, initialIndex), INTRO_SLIDES.length - 1));
  const [dragging, setDragging] = useState(false);
  const [dx, setDx] = useState(0);

  const viewportRef = useRef<HTMLDivElement>(null);
  const widthRef = useRef(0);
  const startXRef = useRef(0);

  const isLast = index === count - 1;

  const onPointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (e.pointerType === 'mouse' && e.button !== 0) return;
    widthRef.current = viewportRef.current?.clientWidth ?? 0;
    startXRef.current = e.clientX;
    setDragging(true);
    setDx(0);
  };

  const onPointerMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging) return;
    const delta = e.clientX - startXRef.current;
    // Capture only once it is clearly a swipe, so taps on buttons still click.
    if (Math.abs(delta) > 8 && !e.currentTarget.hasPointerCapture(e.pointerId)) {
      e.currentTarget.setPointerCapture(e.pointerId);
    }
    setDx(delta);
  };

  const endDrag = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging) return;
    setIndex(
      resolveSwipeIndex({
        // resolveSwipeIndex advances on a negative delta; mirror it in RTL.
        deltaX: (e.clientX - startXRef.current) * -sign,
        width: widthRef.current,
        index,
        count,
      }),
    );
    setDragging(false);
    setDx(0);
  };

  const advance = () => (isLast ? onComplete() : setIndex((i) => Math.min(count - 1, i + 1)));

  // Rubber-band resistance past the first / last slide (in logical terms).
  const swipe = dragging ? dx * -sign : 0; // < 0 = pulling towards the next slide
  let effectiveDx = dragging ? dx : 0;
  if ((index === 0 && swipe > 0) || (isLast && swipe < 0)) effectiveDx *= 0.35;

  return (
    <div className="view ib-page">
      <span className="ib-glow" aria-hidden />

      <header className="ib-hdr">
        <span className="ib-dots" aria-hidden>
          {INTRO_SLIDES.map((id, i) => (
            <span key={id} className={clsx('ib-dot-pg', i === index && 'is-on')} />
          ))}
        </span>
        {!isLast && (
          <button type="button" className="ib-skip" onClick={() => setIndex(SKIP_TARGET)}>
            {t('skip')}
          </button>
        )}
      </header>

      <div
        ref={viewportRef}
        role="group"
        aria-roledescription="carousel"
        aria-label={t('region')}
        className={clsx('ib-viewport', dragging && 'is-dragging')}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endDrag}
        onPointerCancel={endDrag}
      >
        <div
          className="ib-track"
          style={{ transform: `translateX(calc(${sign * index * 100}% + ${effectiveDx}px))` }}
        >
          {INTRO_SLIDES.map((id, i) => (
            <section
              key={id}
              className="ib-slide"
              aria-roledescription="slide"
              aria-label={t('slideOfCount', { n: i + 1, total: count })}
              aria-hidden={i !== index}
              inert={i !== index}
            >
              <div className="ib-copy">
                <h1 className="ib-title">{t(`slides.${id}.title`)}</h1>
                {hasSlideBody(id) && <p className="ib-body">{t(`slides.${id}.body`)}</p>}
              </div>
              <IntroIllustration id={id} />
            </section>
          ))}
        </div>
      </div>

      <span aria-live="polite" className="ib-sr">
        {t('slideOfCount', { n: index + 1, total: count })}
      </span>

      <footer className="ib-foot">
        <PrimaryButton onClick={advance}>{t(isLast ? 'start' : 'next')}</PrimaryButton>
        <SecondaryButton variant="text" className="ib-login" onClick={onLogin}>
          {t('login')}
        </SecondaryButton>
      </footer>
    </div>
  );
}
