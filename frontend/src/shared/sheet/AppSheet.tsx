'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import {
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from 'react';

import { Icon } from '@/shared/ui';

import type { SheetSize } from './types';

/** How far the panel must be dragged down before releasing dismisses it. */
const DISMISS_THRESHOLD_PX = 90;
/** Must match the `osheetDown` / `osheetFadeOut` durations in globals.css. */
const EXIT_MS = 240;

interface AppSheetProps {
  /** Whether the sheet is showing. Flipping to `false` plays the exit first. */
  open: boolean;
  onClose: () => void;
  /** See {@link SheetSize}. Defaults to `full`, the safe size for long copy. */
  size?: SheetSize;
  /** Heading shown in the sticky header. Omit for content that titles itself. */
  title?: ReactNode;
  /** Sticky action row pinned below the body — buttons, a save bar. */
  footer?: ReactNode;
  /** Extra class on the panel, for per-sheet padding or a tinted backdrop. */
  className?: string;
  children: ReactNode;
}

/**
 * The one way a secondary screen enters this app: a panel that rises from the
 * bottom over whatever the user was already looking at.
 *
 * Presentation only — the caller owns `open`/`onClose`, whether that is local
 * state (an inline editor) or the URL-backed sheet router (`SheetHost`).
 *
 * The two sizes are genuinely different layouts, not two numbers:
 *
 * - **`full`** is a fixed-height panel and its body is the scroll container.
 * - **`half`** has no height of its own. It sits at `min-height` when the
 *   content is short and grows with the content instead of scrolling it, which
 *   is the behaviour that makes a half sheet feel like a card rather than a
 *   cropped page. Only when the content would outgrow the *screen* does the
 *   body become scrollable (`is-scrollable`) — clipping it instead would hide
 *   content with no way to reach it.
 *
 * RTL-safe by construction: the panel is full-width and every offset inside it
 * is a logical property (CLAUDE.md §12).
 */
export function AppSheet({
  open,
  onClose,
  size = 'full',
  title,
  footer,
  className,
  children,
}: AppSheetProps) {
  const t = useTranslations('common');
  const titleId = useId();

  // Kept mounted through the exit animation, then torn down.
  const [mounted, setMounted] = useState(open);
  const [dragY, setDragY] = useState(0);
  const [scrollable, setScrollable] = useState(false);

  const panelRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const dragStartY = useRef<number | null>(null);
  const restoreFocus = useRef<HTMLElement | null>(null);

  // The children belong to whichever sheet was last open. Holding on to them
  // lets the panel animate out with its content intact after the router has
  // already forgotten which sheet that was.
  const retained = useRef<ReactNode>(children);
  if (open) retained.current = children;

  useEffect(() => {
    if (open) {
      setMounted(true);
      // A sheet dismissed by a flick keeps its drag offset all the way out, so
      // clear it here rather than on release — otherwise the next sheet to open
      // would inherit the last one's parting position.
      setDragY(0);
      return;
    }
    if (!mounted) return;
    const timer = window.setTimeout(() => setMounted(false), EXIT_MS);
    return () => window.clearTimeout(timer);
  }, [open, mounted]);

  // Escape closes, like every other dialog in the app.
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  // Move focus into the panel on open and hand it back on close, so keyboard
  // and screen-reader users aren't left behind on the screen underneath.
  useEffect(() => {
    if (!open) return;
    restoreFocus.current = document.activeElement as HTMLElement | null;
    panelRef.current?.focus({ preventScroll: true });
    return () => restoreFocus.current?.focus?.({ preventScroll: true });
  }, [open]);

  /**
   * A half sheet only scrolls as a last resort — when its content has grown
   * past the tallest the panel is allowed to be. Measured rather than guessed,
   * because the content is data-driven and its height is unknowable up front.
   */
  useLayoutEffect(() => {
    if (!mounted || size !== 'half') {
      setScrollable(false);
      return;
    }
    const body = bodyRef.current;
    if (!body) return;

    const measure = () => setScrollable(body.scrollHeight - body.clientHeight > 1);
    measure();

    const observer = new ResizeObserver(measure);
    observer.observe(body);
    if (body.firstElementChild) observer.observe(body.firstElementChild);
    return () => observer.disconnect();
  }, [mounted, size, children]);

  const onPointerDown = (event: ReactPointerEvent<HTMLElement>) => {
    dragStartY.current = event.clientY;
    event.currentTarget.setPointerCapture(event.pointerId);
  };
  const onPointerMove = (event: ReactPointerEvent<HTMLElement>) => {
    if (dragStartY.current === null) return;
    // Downward only — dragging up must not detach the panel from the edge.
    setDragY(Math.max(0, event.clientY - dragStartY.current));
  };
  const endDrag = useCallback(() => {
    if (dragStartY.current === null) return;
    dragStartY.current = null;
    if (dragY > DISMISS_THRESHOLD_PX) {
      // Hold the offset: the exit keyframes declare no starting transform, so
      // the panel continues from where the finger left it.
      onClose();
      return;
    }
    setDragY(0);
  }, [dragY, onClose]);

  if (!mounted) return null;

  const closing = !open;

  return (
    <div
      className={clsx('osheet-backdrop', closing && 'is-closing')}
      onPointerDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        tabIndex={-1}
        className={clsx(
          'osheet',
          size === 'half' ? 'is-half' : 'is-full',
          scrollable && 'is-scrollable',
          className,
        )}
        // Data-driven: the live drag offset, which no class can know (§10.1).
        style={dragY ? { transform: `translateY(${dragY}px)` } : undefined}
      >
        <div
          className="osheet-handle"
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={endDrag}
          onPointerCancel={endDrag}
        >
          <span className="osheet-grip" aria-hidden />
        </div>

        {/* Always present, titled or not: a sheet with no visible way out is a
            trap for anyone who can't reach the drag handle. */}
        <header className="osheet-hdr">
          {title ? (
            <h2 id={titleId} className="osheet-title">
              {title}
            </h2>
          ) : (
            <span className="osheet-title" aria-hidden />
          )}
          <button
            type="button"
            className="iconbtn osheet-close"
            onClick={onClose}
            aria-label={t('actions.close')}
          >
            <Icon name="x" size={18} />
          </button>
        </header>

        <div ref={bodyRef} className="osheet-body">
          {open ? children : retained.current}
        </div>

        {footer ? <div className="osheet-foot">{footer}</div> : null}
      </div>
    </div>
  );
}
