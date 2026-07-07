'use client';

import type { CSSProperties, ReactNode, RefObject } from 'react';

import type { BannerLinkType } from '@/entities/banner';
import { Link } from '@/shared/i18n';

interface Props {
  linkUrl: string | null;
  linkType: BannerLinkType | null;
  /**
   * Set to `true` by the carousel the moment a drag turns into a swipe, so the
   * click that fires on pointer-up after a swipe doesn't trigger navigation.
   * Read (and cleared) here on click.
   */
  swipeGuard: RefObject<boolean>;
  ariaLabel?: string;
  className?: string;
  style?: CSSProperties;
  children: ReactNode;
}

/**
 * Wraps a slide in the right kind of link:
 *  - internal → locale-aware in-app `Link` (SPA navigation, keeps the locale).
 *  - external → plain anchor opening a new tab (`noopener`).
 *  - none → a non-interactive container.
 *
 * A swipe that ends on the slide must not navigate, so every interactive
 * variant guards its click against {@link Props.swipeGuard}.
 */
export function BannerLink({
  linkUrl,
  linkType,
  swipeGuard,
  ariaLabel,
  className,
  style,
  children,
}: Props) {
  // Consume the swipe guard: if the pointer-up came from a swipe, cancel the
  // navigation and reset the flag for the next interaction.
  const guardClick = (event: { preventDefault: () => void }) => {
    if (swipeGuard.current) {
      event.preventDefault();
      swipeGuard.current = false;
    }
  };

  if (!linkUrl) {
    return (
      <div className={className} style={style}>
        {children}
      </div>
    );
  }

  if (linkType === 'external') {
    return (
      <a
        href={linkUrl}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={ariaLabel}
        className={className}
        style={style}
        draggable={false}
        onClickCapture={guardClick}
      >
        {children}
      </a>
    );
  }

  return (
    <Link
      href={linkUrl}
      aria-label={ariaLabel}
      className={className}
      style={style}
      draggable={false}
      onClickCapture={guardClick}
    >
      {children}
    </Link>
  );
}
