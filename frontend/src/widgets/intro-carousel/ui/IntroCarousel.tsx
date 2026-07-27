'use client';

import { useLocale, useTranslations } from 'next-intl';
import {
  type PointerEvent as ReactPointerEvent,
  useEffect,
  useRef,
  useState,
} from 'react';

import { getDirection, isLocale } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { INTRO_SLIDES, type IntroSlide } from '../model/slides';
import { resolveSwipeIndex } from '../lib/swipe';
import { IntroIllustration } from './IntroIllustration';

interface Props {
  /** Called when the user finishes or skips the intro. */
  onComplete: () => void;
}

/**
 * The pre-signup welcome carousel: a swipeable, three-slide introduction shown
 * to first-time visitors. Self-contained — it owns swipe, progress, and the
 * per-slide CTA, and reports a single `onComplete` so the screen decides where
 * to go next (and that the intro was seen).
 *
 * The track is laid out physically LTR so the transform math is
 * direction-independent (see `../lib/swipe`); each slide's text is flipped back
 * to the active locale's direction. Auto-play is intentionally absent — reading
 * copy shouldn't be on a timer.
 */
export function IntroCarousel({ onComplete }: Props) {
  const t = useTranslations('welcome');
  const locale = useLocale();
  const dir = getDirection(isLocale(locale) ? locale : 'fa');
  const count = INTRO_SLIDES.length;

  const [index, setIndex] = useState(0);
  const [dragging, setDragging] = useState(false);
  const [dx, setDx] = useState(0);
  const [reduceMotion, setReduceMotion] = useState(false);

  const viewportRef = useRef<HTMLDivElement>(null);
  const widthRef = useRef(0);
  const startXRef = useRef(0);

  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    const sync = () => setReduceMotion(mq.matches);
    sync();
    mq.addEventListener('change', sync);
    return () => mq.removeEventListener('change', sync);
  }, []);

  const isLast = index === count - 1;

  const onPointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    widthRef.current = viewportRef.current?.clientWidth ?? 0;
    startXRef.current = e.clientX;
    setDragging(true);
    setDx(0);
    e.currentTarget.setPointerCapture(e.pointerId);
  };

  const onPointerMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging) return;
    setDx(e.clientX - startXRef.current);
  };

  const endDrag = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging) return;
    setIndex(
      resolveSwipeIndex({
        deltaX: e.clientX - startXRef.current,
        width: widthRef.current,
        index,
        count,
      }),
    );
    setDragging(false);
    setDx(0);
  };

  const advance = () => (isLast ? onComplete() : setIndex((i) => Math.min(count - 1, i + 1)));

  // Rubber-band resistance when dragging past the first / last slide.
  let effectiveDx = dragging ? dx : 0;
  if ((index === 0 && effectiveDx > 0) || (isLast && effectiveDx < 0)) {
    effectiveDx *= 0.35;
  }

  return (
    <div className="view onb-page">
      {/* Top bar: brand mark + skip */}
      <div className="hdr ic-hdr">
        <span className="ic-brand">
          {t('brand')}
        </span>
        <button
          type="button"
          onClick={onComplete}
          className="ic-skip"
        >
          {t('skip')}
        </button>
      </div>

      {/* Swipe viewport */}
      <div
        ref={viewportRef}
        role="group"
        aria-roledescription="carousel"
        aria-label={t('region')}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endDrag}
        onPointerCancel={endDrag}
        style={{
          flex: 1,
          overflow: 'hidden',
          touchAction: 'pan-y',
          userSelect: 'none',
          cursor: dragging ? 'grabbing' : 'grab',
        }}
      >
        {/* Track — physical LTR so translate math is direction-independent */}
        <div
          style={{
            display: 'flex',
            direction: 'ltr',
            height: '100%',
            transform: `translateX(calc(${-index * 100}% + ${effectiveDx}px))`,
            transition: dragging || reduceMotion ? 'none' : 'transform .38s cubic-bezier(.22,.61,.36,1)',
          }}
        >
          {INTRO_SLIDES.map((slide, i) => (
            <div key={slide.id} className="ic-slide">
              <SlidePanel slide={slide} dir={dir} active={i === index} uid={slide.id} />
            </div>
          ))}
        </div>
      </div>

      {/* Live region for screen readers */}
      <span aria-live="polite" style={srOnly}>
        {t('slideOfCount', { n: index + 1, total: count })}
      </span>

      {/* Footer: progress dots + primary CTA */}
      <div className="ic-foot">
        <div className="ic-dots">
          {INTRO_SLIDES.map((slide, i) => {
            const on = i === index;
            return (
              <button
                key={slide.id}
                type="button"
                aria-label={t('goToSlide', { n: i + 1 })}
                aria-current={on}
                onClick={() => setIndex(i)}
                style={{
                  width: on ? 22 : 8,
                  height: 8,
                  borderRadius: 99,
                  border: 0,
                  padding: 0,
                  cursor: 'pointer',
                  background: on ? 'var(--brand)' : 'var(--field-border)',
                  transition: 'width .25s ease, background .25s ease',
                }}
              />
            );
          })}
        </div>
        <button className="btn btn-primary" onClick={advance}>
          {t(isLast ? 'start' : 'next')}
        </button>
      </div>
    </div>
  );
}

/* — One slide's content: illustration + copy, varying by slide — */
function SlidePanel({
  slide,
  dir,
  active,
  uid,
}: {
  slide: IntroSlide;
  dir: 'rtl' | 'ltr';
  active: boolean;
  uid: string;
}) {
  const t = useTranslations('welcome');
  return (
    <div
      dir={dir}
      className="scroll ic-panel"
    >
      {/* Illustration with a soft, theme-aware halo */}
      <div className="ic-media">
        <span
          style={{
            display: 'flex',
            width: '100%',
            justifyContent: 'center',
            transition: 'opacity .4s ease, transform .4s ease',
            opacity: active ? 1 : 0,
            transform: active ? 'translateY(0)' : 'translateY(8px)',
          }}
        >
          <IntroIllustration
            variant={slide.illustration}
            accentFrom={slide.accentFrom}
            accentTo={slide.accentTo}
            uid={uid}
          />
        </span>
      </div>

      {/* Copy */}
      <div className="ic-copy">
        <h2 className="titr ic-title">
          {t(`slides.${slide.id}.title`)}
        </h2>
        <p className="sub ic-body">
          {t(`slides.${slide.id}.body`)}
        </p>

        {slide.id === 'intro' && (
          <p
            className="sub ic-note"
          >
            {t('slides.intro.note')}
          </p>
        )}

        {slide.bullets && (
          <ul className="ic-bullets">
            <Bullet icon="heart">{t('slides.track.bullet1')}</Bullet>
            <Bullet icon="moon">{t('slides.track.bullet2')}</Bullet>
          </ul>
        )}

        {slide.disclaimer && (
          <>
            <p className="ic-disc">
              {t('disclaimer')}
            </p>
            <p className="ic-hint">
              {t('startHint')}
            </p>
          </>
        )}
      </div>
    </div>
  );
}

function Bullet({ icon, children }: { icon: 'heart' | 'moon'; children: React.ReactNode }) {
  return (
    <li className="ic-bullet">
      <span className="ic-bullet-ic">
        <Icon name={icon} size={18} stroke="var(--brand)" />
      </span>
      <span className="ic-bullet-t">
        {children}
      </span>
    </li>
  );
}

const srOnly = {
  position: 'absolute',
  width: 1,
  height: 1,
  padding: 0,
  margin: -1,
  overflow: 'hidden',
  clip: 'rect(0 0 0 0)',
  whiteSpace: 'nowrap',
  border: 0,
} as const;
