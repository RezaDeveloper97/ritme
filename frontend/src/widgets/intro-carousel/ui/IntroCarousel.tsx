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
    <div className="view" style={{ background: 'var(--surface)' }}>
      {/* Top bar: brand mark + skip */}
      <div className="hdr" style={{ height: 56 }}>
        <span style={{ fontWeight: 900, fontSize: 18, color: 'var(--brand)', letterSpacing: 0.5 }}>
          {t('brand')}
        </span>
        <button
          type="button"
          onClick={onComplete}
          style={{
            border: 0,
            background: 'transparent',
            color: 'var(--muted)',
            fontSize: 14,
            fontWeight: 600,
            padding: '6px 8px',
            cursor: 'pointer',
          }}
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
            <div key={slide.id} style={{ flex: '0 0 100%', minWidth: 0, height: '100%' }}>
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
      <div style={{ padding: '10px 22px 22px' }}>
        <div style={{ display: 'flex', direction: 'ltr', justifyContent: 'center', gap: 6, marginBottom: 16 }}>
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
      className="scroll"
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        padding: '4px 26px 8px',
      }}
    >
      {/* Illustration with a soft, theme-aware halo */}
      <div
        style={{
          flex: '0 0 auto',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: 190,
          padding: '10px 0',
          background: 'radial-gradient(58% 58% at 50% 46%, var(--surface-2), transparent)',
        }}
      >
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
      <div style={{ marginTop: 6 }}>
        <h2 className="titr" style={{ fontSize: 23, fontWeight: 900, lineHeight: 1.4 }}>
          {t(`slides.${slide.id}.title`)}
        </h2>
        <p className="sub" style={{ margin: '12px 0 0', fontSize: 14 }}>
          {t(`slides.${slide.id}.body`)}
        </p>

        {slide.id === 'intro' && (
          <p
            className="sub"
            style={{ margin: '16px 0 0', fontSize: 13, color: 'var(--brand-deep)', fontWeight: 600 }}
          >
            {t('slides.intro.note')}
          </p>
        )}

        {slide.bullets && (
          <ul style={{ listStyle: 'none', margin: '18px 0 0', padding: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>
            <Bullet icon="heart">{t('slides.track.bullet1')}</Bullet>
            <Bullet icon="moon">{t('slides.track.bullet2')}</Bullet>
          </ul>
        )}

        {slide.disclaimer && (
          <>
            <p style={{ margin: '18px 0 0', fontSize: 12.5, lineHeight: 1.8, color: 'var(--muted)' }}>
              {t('disclaimer')}
            </p>
            <p style={{ margin: '14px 0 0', fontSize: 13, lineHeight: 1.8, color: 'var(--slate)', fontWeight: 600 }}>
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
    <li style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
      <span
        style={{
          flex: '0 0 auto',
          width: 34,
          height: 34,
          borderRadius: 12,
          background: 'var(--surface-2)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--brand)',
        }}
      >
        <Icon name={icon} size={18} stroke="var(--brand)" />
      </span>
      <span style={{ fontSize: 13.5, lineHeight: 1.7, color: 'var(--slate)', paddingTop: 6 }}>
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
