import type { CSSProperties } from 'react';

export type IconName =
  | 'bell' | 'sparkle' | 'chevronRight' | 'chevronLeft' | 'chevronDown'
  | 'pencil' | 'check' | 'checkCircle' | 'loader' | 'drop'
  | 'calendar' | 'plus' | 'heart' | 'moon' | 'zap' | 'smile'
  | 'book' | 'alarm' | 'pill' | 'flame' | 'info' | 'x'
  | 'user' | 'chart' | 'grid' | 'arrowL' | 'refresh'
  | 'walk' | 'thermo' | 'glass' | 'stetho'
  | 'home' | 'bookOpen' | 'apple' | 'brain'
  | 'globe' | 'shield' | 'logout' | 'download' | 'trash';

const PATHS: Record<IconName, string> = {
  bell:         '<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>',
  sparkle:      '<path d="M12 3l1.6 4.6L18 9.2l-4.4 1.6L12 15l-1.6-4.2L6 9.2l4.4-1.6z"/><path d="M19 13l.7 2 .3 .5 2 .7-2 .7-.3 .5-.7 2-.7-2-.3-.5-2-.7 2-.7 .3-.5z"/>',
  chevronRight: '<path d="M9 18l6-6-6-6"/>',
  chevronLeft:  '<path d="M15 18l-6-6 6-6"/>',
  chevronDown:  '<path d="M6 9l6 6 6-6"/>',
  pencil:       '<path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  check:        '<path d="M20 6L9 17l-5-5"/>',
  checkCircle:  '<circle cx="12" cy="12" r="9"/><path d="M9 12l2 2 4-4"/>',
  loader:       '<path d="M12 3v3M12 18v3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M3 12h3M18 12h3M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1"/>',
  drop:         '<path d="M12 2.7s6 6.2 6 10.3a6 6 0 0 1-12 0c0-4.1 6-10.3 6-10.3z"/>',
  calendar:     '<rect x="3" y="4.5" width="18" height="17" rx="3"/><path d="M3 9h18M8 2.5v4M16 2.5v4"/>',
  plus:         '<path d="M12 5v14M5 12h14"/>',
  heart:        '<path d="M19 5.6a4.6 4.6 0 0 0-7-.6l-1 1-1-1a4.6 4.6 0 1 0-6.5 6.5L12 20l8.5-8.5A4.6 4.6 0 0 0 19 5.6z"/>',
  moon:         '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
  zap:          '<path d="M13 2L4.5 13.5H11l-1 8.5L19.5 10H13z"/>',
  smile:        '<circle cx="12" cy="12" r="9"/><path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/>',
  book:         '<path d="M3 5.5A2.5 2.5 0 0 1 5.5 3H20v15H5.5A2.5 2.5 0 0 0 3 20.5z"/><path d="M3 5.5V20.5"/>',
  bookOpen:     '<path d="M12 7v14"/><path d="M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z"/>',
  home:         '<path d="M3 10.7 12 3l9 7.7"/><path d="M5.2 9.5V19a1.5 1.5 0 0 0 1.5 1.5h10.6A1.5 1.5 0 0 0 18.8 19V9.5"/>',
  alarm:        '<circle cx="12" cy="13" r="8"/><path d="M12 9v4l2.5 1.5M5 3L2 6M19 3l3 3"/>',
  pill:         '<path d="M10.5 20.5a5 5 0 0 1-7-7l6-6a5 5 0 0 1 7 7z"/><path d="M8.5 8.5l7 7"/>',
  stetho:       '<path d="M4.5 3v5a4.5 4.5 0 0 0 9 0V3"/><path d="M4.5 3h-1M13.5 3h1M9 17v1a4 4 0 0 0 8 0v-2"/><circle cx="18" cy="14" r="2.4"/>',
  flame:        '<path d="M12 2s4 4 4 8a4 4 0 0 1-8 0c0-1 .4-2 1-2.6C9 8 12 6 12 2z"/><path d="M12 22a6 6 0 0 0 6-6c0-2-1-3.5-2-4.5.2 2.5-1.6 3.8-2.6 4.2.4-1.8-.4-3.7-1.4-4.7-.2 3-3 3.6-3 6.2A3.8 3.8 0 0 0 12 22z"/>',
  info:         '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>',
  x:            '<path d="M18 6L6 18M6 6l12 12"/>',
  glass:        '<path d="M6 3h12l-1.4 16a2 2 0 0 1-2 1.8H9.4a2 2 0 0 1-2-1.8z"/><path d="M6.7 9h10.6"/>',
  walk:         '<circle cx="13" cy="4.5" r="1.6"/><path d="M9 21l2.5-6L9.5 12l-1 4M11.5 15l3 1.5 1.5 3.5M11.5 9l3 1 1.5 3"/>',
  thermo:       '<path d="M14 14.8V5a2 2 0 0 0-4 0v9.8a4 4 0 1 0 4 0z"/>',
  user:         '<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4.4 3.6-7 8-7s8 2.6 8 7"/>',
  chart:        '<path d="M3 3v18h18"/><path d="M7 14l3-4 3 3 4-6"/>',
  grid:         '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
  arrowL:       '<path d="M19 12H5M12 19l-7-7 7-7"/>',
  refresh:      '<path d="M3 12a9 9 0 0 1 15-6.7L21 8M21 3v5h-5M21 12a9 9 0 0 1-15 6.7L3 16M3 21v-5h5"/>',
  globe:        '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.5 3.8 5.6 3.8 9S14.5 18.5 12 21c-2.5-2.5-3.8-5.6-3.8-9S9.5 5.5 12 3z"/>',
  shield:       '<path d="M12 3l7 3v5c0 4.4-3 8.2-7 10-4-1.8-7-5.6-7-10V6z"/><path d="M9.2 12l2 2 3.6-4"/>',
  logout:       '<path d="M15 3h3a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-3"/><path d="M10 17l5-5-5-5M15 12H3"/>',
  download:     '<path d="M12 3v12M7 10l5 5 5-5"/><path d="M4 20h16"/>',
  apple:        '<path d="M12 8c-1.2-1-2.6-1.4-4-1-2 .6-3 2.7-3 5.2 0 3.6 2.3 8 4.6 8 .9 0 1.6-.4 2.4-.4s1.5.4 2.4.4c2.3 0 4.6-4.4 4.6-8 0-2.5-1-4.6-3-5.2-1.4-.4-2.8 0-4 1z"/><path d="M12 8c0-2.2 1.3-4 3.5-4.5"/>',
  brain:        '<path d="M9.5 4a2.5 2.5 0 0 0-2.4 3.2A2.6 2.6 0 0 0 5 9.8c0 .9.4 1.7 1.1 2.2A2.6 2.6 0 0 0 5.4 14c0 1.2.8 2.2 2 2.5A2.5 2.5 0 0 0 12 18V5.9A2 2 0 0 0 9.5 4z"/><path d="M14.5 4a2.5 2.5 0 0 1 2.4 3.2A2.6 2.6 0 0 1 19 9.8c0 .9-.4 1.7-1.1 2.2.4.5.7 1.2.7 2 0 1.2-.8 2.2-2 2.5A2.5 2.5 0 0 1 12 18"/>',
  trash:        '<path d="M4 7h16M9 7V5a1.5 1.5 0 0 1 1.5-1.5h3A1.5 1.5 0 0 1 15 5v2M6 7l1 12.5A1.5 1.5 0 0 0 8.5 21h7a1.5 1.5 0 0 0 1.5-1.5L18 7"/>',
};

interface IconProps {
  name: IconName;
  size?: number;
  fill?: string;
  stroke?: string;
  strokeWidth?: number;
  style?: CSSProperties;
  className?: string;
}

export function Icon({
  name,
  size = 24,
  fill = 'none',
  stroke = 'currentColor',
  strokeWidth = 2,
  style,
  className,
}: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill={fill}
      stroke={stroke}
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      style={style}
      className={className}
      dangerouslySetInnerHTML={{ __html: PATHS[name] ?? '' }}
    />
  );
}

/** Solid drop shape for phase indicators. */
export function DropSolid({ size = 16, color }: { size?: number; color: string }) {
  return (
    <svg viewBox="0 0 24 24" width={size} height={size} fill={color}>
      <path d="M12 2.5s6.5 6.6 6.5 11A6.5 6.5 0 0 1 5.5 13.5C5.5 9.1 12 2.5 12 2.5z" />
    </svg>
  );
}
