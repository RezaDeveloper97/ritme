'use client';

import { useTranslations } from 'next-intl';
import {
  type PointerEvent as ReactPointerEvent,
  useEffect,
  useRef,
  useState,
} from 'react';

import { type BannerPosition, useBanners } from '@/entities/banner';

import { resolveSwipeIndex, wrapIndex } from '../lib/carousel';
import { BannerLink } from './BannerLink';

const AUTOPLAY_MS = 5000;

interface Props {
  position: BannerPosition;
}

/**
 * A swipeable, auto-advancing banner slideshow for one home-page slot. Renders
 * nothing when the slot has no active banner, so unused slots are invisible.
 *
 * Swipe is finger/pointer driven; the track is laid out LTR internally so the
 * transform math is direction-independent, and the whole widget sits happily
 * inside the app's RTL layout. Auto-play pauses while dragging or hovering.
 */
export function BannerSlideshow({ position }: Props) {
  const t = useTranslations('banners');
  const banners = useBanners(position);
  const count = banners.length;

  const [index, setIndex] = useState(0);
  const [dragging, setDragging] = useState(false);
  const [hovering, setHovering] = useState(false);
  const [dx, setDx] = useState(0);

  const viewportRef = useRef<HTMLDivElement>(null);
  const widthRef = useRef(0);
  const startXRef = useRef(0);
  const swipeGuard = useRef(false);

  // Keep the index valid if the active banner set shrinks (e.g. one expired).
  useEffect(() => {
    if (index > count - 1) setIndex(Math.max(0, count - 1));
  }, [count, index]);

  // Auto-advance. Resets whenever the slide, count, or paused state changes;
  // stays off for a single banner or while the user is interacting.
  const paused = dragging || hovering;
  useEffect(() => {
    if (count <= 1 || paused) return;
    const id = setTimeout(() => setIndex((i) => wrapIndex(i, count, 1)), AUTOPLAY_MS);
    return () => clearTimeout(id);
  }, [index, count, paused]);

  if (count === 0) return null;

  const onPointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (count <= 1) return;
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
    const delta = e.clientX - startXRef.current;
    // A real swipe (beyond a few px) must not also fire a navigation click.
    if (Math.abs(delta) > 5) swipeGuard.current = true;

    setIndex(
      resolveSwipeIndex({ deltaX: delta, width: widthRef.current, index, count }),
    );
    setDragging(false);
    setDx(0);
  };

  // Rubber-band resistance when dragging past the first / last slide.
  let effectiveDx = dragging ? dx : 0;
  if ((index === 0 && effectiveDx > 0) || (index === count - 1 && effectiveDx < 0)) {
    effectiveDx *= 0.35;
  }

  return (
    <div style={{ padding: '16px 16px 0' }}>
      <div
        role="group"
        aria-roledescription="carousel"
        aria-label={t('region')}
        onMouseEnter={() => setHovering(true)}
        onMouseLeave={() => setHovering(false)}
      >
        {/* Viewport */}
        <div
          ref={viewportRef}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={endDrag}
          onPointerCancel={endDrag}
          style={{
            overflow: 'hidden',
            borderRadius: 12,
            touchAction: 'pan-y',
            cursor: count > 1 ? (dragging ? 'grabbing' : 'grab') : 'default',
            userSelect: 'none',
          }}
        >
          {/* Track — forced LTR so translate math is direction-independent */}
          <div
            style={{
              display: 'flex',
              direction: 'ltr',
              transform: `translateX(calc(${-index * 100}% + ${effectiveDx}px))`,
              transition: dragging ? 'none' : 'transform .35s cubic-bezier(.22,.61,.36,1)',
            }}
          >
            {banners.map((banner) => (
              <div key={banner.id} style={{ flex: '0 0 100%', minWidth: 0 }}>
                <BannerLink
                  linkUrl={banner.linkUrl}
                  linkType={banner.linkType}
                  swipeGuard={swipeGuard}
                  ariaLabel={banner.title ?? undefined}
                  style={{ display: 'block' }}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={banner.imageUrl}
                    alt={banner.title ?? ''}
                    draggable={false}
                    style={{
                      display: 'block',
                      width: '100%',
                      aspectRatio: '2 / 1',
                      objectFit: 'cover',
                      borderRadius: 12,
                      background: 'var(--surface-3)',
                    }}
                  />
                </BannerLink>
              </div>
            ))}
          </div>
        </div>

        {/* Dots */}
        {count > 1 && (
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              gap: 6,
              marginTop: 10,
            }}
          >
            {banners.map((banner, i) => {
              const active = i === index;
              return (
                <button
                  key={banner.id}
                  type="button"
                  aria-label={t('goToSlide', { n: i + 1 })}
                  aria-current={active}
                  onClick={() => setIndex(i)}
                  style={{
                    width: active ? 20 : 8,
                    height: 8,
                    borderRadius: 99,
                    border: 0,
                    padding: 0,
                    cursor: 'pointer',
                    background: active ? 'var(--brand)' : 'var(--track)',
                    transition: 'width .25s ease, background .25s ease',
                  }}
                />
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
