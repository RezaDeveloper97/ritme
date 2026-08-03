'use client';

import { useLayoutEffect, useRef } from 'react';

import { useTranslations } from 'next-intl';

import { useUserMode } from '@/entities/message';
import { Link, usePathname, useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { getNavGoo, restFrame } from '../lib/goo-motion';
import type { BlobPos, Frame, Rect } from '../lib/goo-motion';

import { navItemsForMode } from '../model/nav-items';

/**
 * Floating bottom navigation shared across the main tabs. The active-tab
 * highlight is a gradient goo blob rendered under an SVG metaball filter
 * together with an echo circle behind the FAB: crossing the center "+" the
 * blob is sucked into the FAB (which inflates and leans toward it), holds a
 * beat, then a droplet is ejected the other side and springs into the target
 * tab. Motion is a persistent spring simulation in `../lib/goo-motion` — each
 * screen mounts its own <BottomNav />, so the sim lives at module scope and
 * navigation merely retargets it: velocity carries across remounts, which is
 * what makes rapid tab hopping feel continuous instead of restarting. A rAF
 * loop writes frames straight to the DOM so nothing re-renders per frame.
 * Positions are measured physical offsets, so the same code works in both
 * `fa` (RTL) and `en` (LTR).
 */
export function BottomNav() {
  const t = useTranslations('nav');
  const pathname = usePathname();
  const router = useRouter();
  const modeQuery = useUserMode();
  const items = navItemsForMode(modeQuery.data?.mode);

  /**
   * The highlight is a circle the exact diameter of the FAB, centred on the
   * tab — tabs are wider than the FAB, so we can't just use the tab's box.
   */
  const pillRect = (tabEl: HTMLElement, fabWidth: number): Rect => {
    const size = Math.min(fabWidth, tabEl.offsetWidth);
    return {
      left: tabEl.offsetLeft + (tabEl.offsetWidth - size) / 2,
      width: size,
    };
  };

  const barRef = useRef<HTMLElement>(null);
  const blobRef = useRef<HTMLSpanElement>(null);
  const echoRef = useRef<HTMLSpanElement>(null);
  const lensRef = useRef<HTMLSpanElement>(null);

  /**
   * iOS 26-style "selection bubble": tapping the tab that is already active
   * inflates a liquid-glass lens over the goo blob while the blob jellies
   * underneath. Restarted via class toggle + reflow so rapid taps replay it.
   */
  const popBubble = (tabEl: HTMLElement) => {
    const lens = lensRef.current;
    const blob = blobRef.current;
    const fabEl = barRef.current?.querySelector<HTMLElement>('.fab');
    if (!lens || !blob || !fabEl) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const pill = pillRect(tabEl, fabEl.offsetWidth);
    lens.style.insetInlineStart = 'auto';
    lens.style.left = `${pill.left}px`;
    lens.style.width = `${pill.width}px`;
    lens.classList.remove('pop');
    blob.classList.remove('jelly');
    void lens.offsetWidth;
    lens.classList.add('pop');
    blob.classList.add('jelly');
  };

  const activeKey = items.find((item) => pathname === item.href)?.key;
  const isFabRoute = items.some((item) => item.fab && pathname === item.href);

  const rafRef = useRef(0);

  const paint = (f: Frame) => {
    const blob = blobRef.current;
    const echo = echoRef.current;
    const fabEl = barRef.current?.querySelector<HTMLElement>('.fab');
    if (!blob || !echo || !fabEl) return;
    blob.style.insetInlineStart = 'auto';
    blob.style.left = `${f.left}px`;
    blob.style.width = `${f.width}px`;
    blob.style.transform = `scaleY(${f.scaleY}) skewX(${f.skewX}deg)`;
    blob.style.opacity = `${f.blobOpacity}`;
    echo.style.opacity = `${f.echoOpacity}`;
    fabEl.style.transform = `translateX(${f.fabX}px) scale(${f.fabScale})`;
  };

  const runLoop = () => {
    cancelAnimationFrame(rafRef.current);
    const goo = getNavGoo();
    const step = (now: number) => {
      paint(goo.frame(now));
      if (!goo.settled()) rafRef.current = requestAnimationFrame(step);
    };
    rafRef.current = requestAnimationFrame(step);
  };

  /**
   * Response latency killer: the route transition (unmount + remount +
   * effect) costs ~180ms, which reads as lag. So the sim is retargeted the
   * instant the tab is clicked; the remounted nav's effect re-issues the same
   * retarget (a no-op for a spring) and takes over the loop.
   */
  const kick = (el: HTMLElement, fab: boolean) => {
    const goo = getNavGoo();
    const fabEl = barRef.current?.querySelector<HTMLElement>('.fab');
    if (!fabEl || !goo.initialized) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const fabRect: Rect = { left: fabEl.offsetLeft, width: fabEl.offsetWidth };
    goo.retarget(fab ? 'fab' : pillRect(el, fabRect.width), fabRect);
    runLoop();
  };

  /* ── Liquid-glass drag (iOS 26 tab bar) ──
     Press-and-hold the bar and the highlight detaches and chases the finger
     (springs make it lag and wobble like liquid); the glass lens rides on
     top. Release anywhere and the nearest tab becomes the active screen. */
  const drag = useRef({
    id: -1,
    active: false,
    moved: false,
    timer: 0,
    startX: 0,
    startY: 0,
    lastX: 0,
  });

  const moveDrag = (clientX: number) => {
    const bar = barRef.current;
    const lens = lensRef.current;
    const fabEl = bar?.querySelector<HTMLElement>('.fab');
    if (!bar || !fabEl || !lens) return;
    const size = fabEl.offsetWidth;
    const rect = bar.getBoundingClientRect();
    const x = Math.min(
      rect.width - size / 2 - 6,
      Math.max(size / 2 + 6, clientX - rect.left),
    );
    const pos: Rect = { left: x - size / 2, width: size };
    getNavGoo().follow(pos, { left: fabEl.offsetLeft, width: fabEl.offsetWidth });
    lens.style.insetInlineStart = 'auto';
    lens.style.left = `${pos.left}px`;
    lens.style.width = `${pos.width}px`;
    runLoop();
  };

  const beginDrag = (pointerId: number) => {
    const bar = barRef.current;
    if (!bar || !getNavGoo().initialized) return;
    drag.current.active = true;
    drag.current.moved = true;
    try {
      bar.setPointerCapture(pointerId);
    } catch {
      /* pointer already gone */
    }
    bar.classList.add('dragging');
    lensRef.current?.classList.remove('pop');
    lensRef.current?.classList.add('drag');
    blobRef.current?.classList.remove('jelly');
    moveDrag(drag.current.lastX);
  };

  const endDrag = (clientX: number, commit: boolean) => {
    const bar = barRef.current;
    const d = drag.current;
    window.clearTimeout(d.timer);
    d.id = -1;
    if (!d.active || !bar) return;
    d.active = false;
    bar.classList.remove('dragging');
    lensRef.current?.classList.remove('drag');
    const fabEl = bar.querySelector<HTMLElement>('.fab');
    if (!fabEl) return;
    // Snap to the nearest item — DOM order matches `items` order.
    const els = Array.from(bar.querySelectorAll<HTMLElement>('.tab, .fab'));
    const rect = bar.getBoundingClientRect();
    const x = clientX - rect.left;
    let best = 0;
    let bestDist = Infinity;
    els.forEach((el, i) => {
      const dist = Math.abs(el.offsetLeft + el.offsetWidth / 2 - x);
      if (dist < bestDist) {
        bestDist = dist;
        best = i;
      }
    });
    const item = commit ? items[best] : items.find((it) => pathname === it.href);
    const el = commit ? els[best] : undefined;
    const fabRect: Rect = { left: fabEl.offsetLeft, width: fabEl.offsetWidth };
    const goo = getNavGoo();
    if (item?.fab) goo.retarget('fab', fabRect);
    else if (el) goo.retarget(pillRect(el, fabRect.width), fabRect);
    else {
      // Cancelled drag: spring home to whatever is currently active.
      const home = bar.querySelector<HTMLElement>('.tab.on');
      goo.retarget(
        home ? pillRect(home, fabRect.width) : isFabRoute ? 'fab' : fabRect,
        fabRect,
      );
    }
    runLoop();
    if (commit && item && pathname !== item.href) router.push(item.href);
  };

  const onPointerDown = (e: React.PointerEvent) => {
    if (e.pointerType === 'mouse' && e.button !== 0) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const d = drag.current;
    window.clearTimeout(d.timer);
    d.id = e.pointerId;
    d.active = false;
    d.moved = false;
    d.startX = e.clientX;
    d.startY = e.clientY;
    d.lastX = e.clientX;
    d.timer = window.setTimeout(() => beginDrag(e.pointerId), 220);
  };

  const onPointerMove = (e: React.PointerEvent) => {
    const d = drag.current;
    if (e.pointerId !== d.id) return;
    d.lastX = e.clientX;
    if (!d.active) {
      // Moved before the hold matured → it's a tap/scroll, not a drag.
      if (Math.hypot(e.clientX - d.startX, e.clientY - d.startY) > 12) {
        window.clearTimeout(d.timer);
        d.id = -1;
      }
      return;
    }
    moveDrag(e.clientX);
  };

  const onPointerUp = (e: React.PointerEvent) => {
    if (e.pointerId === drag.current.id) endDrag(e.clientX, true);
  };

  const onPointerCancel = (e: React.PointerEvent) => {
    if (e.pointerId === drag.current.id) endDrag(e.clientX, false);
  };

  /** A completed drag must not also fire the underlying Link's click. */
  const onClickCapture = (e: React.MouseEvent) => {
    if (drag.current.moved) {
      e.preventDefault();
      e.stopPropagation();
      drag.current.moved = false;
    }
  };

  useLayoutEffect(() => {
    const bar = barRef.current;
    const blob = blobRef.current;
    const echo = echoRef.current;
    if (!bar || !blob || !echo) return;
    const fabEl = bar.querySelector<HTMLElement>('.fab');
    if (!fabEl) return;

    const measure = (): { fab: Rect; target: BlobPos | null } => {
      const fab: Rect = { left: fabEl.offsetLeft, width: fabEl.offsetWidth };
      if (isFabRoute) return { fab, target: 'fab' };
      const el = bar.querySelector<HTMLElement>('.tab.on');
      return { fab, target: el ? pillRect(el, fab.width) : null };
    };

    const paintRest = () => {
      const { fab, target } = measure();
      echo.style.left = `${fab.left}px`;
      echo.style.top = `${fabEl.offsetTop}px`;
      echo.style.width = `${fab.width}px`;
      echo.style.height = `${fabEl.offsetHeight}px`;
      if (!target) {
        blob.style.opacity = '0';
        return;
      }
      paint(restFrame(target, fab));
    };

    const { fab, target } = measure();
    echo.style.left = `${fab.left}px`;
    echo.style.top = `${fabEl.offsetTop}px`;
    echo.style.width = `${fab.width}px`;
    echo.style.height = `${fabEl.offsetHeight}px`;

    window.addEventListener('resize', paintRest);
    const cleanupResize = () => window.removeEventListener('resize', paintRest);

    const goo = getNavGoo();

    if (!target) {
      blob.style.opacity = '0';
      return cleanupResize;
    }

    const reduced = window.matchMedia(
      '(prefers-reduced-motion: reduce)',
    ).matches;

    if (reduced || !goo.initialized) {
      goo.snap(target, fab);
      paint(restFrame(target, fab));
      return cleanupResize;
    }

    // The sim persists across remounts: this just moves the springs' target.
    // Position and velocity carry over (the click handler usually already
    // retargeted it before navigation), so mid-flight remounts are seamless.
    goo.retarget(target, fab);
    runLoop();

    return () => {
      cancelAnimationFrame(rafRef.current);
      fabEl.style.transform = '';
      cleanupResize();
    };
    // runLoop only touches refs and the module-scope sim — safe to omit.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeKey, isFabRoute, items.length]);

  return (
    <nav
      className="tabbar"
      aria-label={t('label')}
      ref={barRef}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerCancel}
      onClickCapture={onClickCapture}
    >
      <svg className="goo-defs" aria-hidden focusable="false">
        <defs>
          <filter id="nav-goo">
            {/* stdDeviation sets how far apart two shapes start bridging —
                bigger blur = the bubbly liquid neck forms earlier. */}
            <feGaussianBlur in="SourceGraphic" stdDeviation="11" result="blur" />
            <feColorMatrix
              in="blur"
              mode="matrix"
              values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 17 -6.5"
              result="goo"
            />
          </filter>
        </defs>
      </svg>
      <span className="goo" aria-hidden>
        <span ref={echoRef} className="goo-fab-echo" />
        <span ref={blobRef} className="goo-blob" />
      </span>
      <span ref={lensRef} className="goo-lens" aria-hidden />
      {items.map((item) => {
        if (item.fab) {
          return (
            <Link
              key={item.key}
              href={item.href}
              className="fab"
              aria-label={t(item.key)}
              onClick={(e) => kick(e.currentTarget, true)}
            >
              <Icon name={item.icon} size={26} className="ic" />
            </Link>
          );
        }

        const active = pathname === item.href;

        return (
          <Link
            key={item.key}
            href={item.href}
            className={active ? 'tab on' : 'tab'}
            aria-current={active ? 'page' : undefined}
            onClick={(e) =>
              active ? popBubble(e.currentTarget) : kick(e.currentTarget, false)
            }
          >
            <Icon name={item.icon} size={24} className="ic" />
            <span>{t(item.key)}</span>
          </Link>
        );
      })}
    </nav>
  );
}
