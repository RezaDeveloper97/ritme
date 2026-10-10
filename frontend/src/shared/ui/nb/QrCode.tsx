'use client';

import { clsx } from 'clsx';
import { useMemo } from 'react';

import { encodeQr, qrSvgPath } from '@/shared/lib/qr';

interface QrCodeProps {
  /** The text to encode (a share URL). Encoded on the device — nothing is sent anywhere. */
  value: string;
  /** Accessible name, e.g. «کد QR برای پزشک». */
  label: string;
  className?: string;
}

const QUIET = 4;

/**
 * A QR code as one SVG path (CB-REC-05). Always dark modules on a light quiet zone — scanners misread inverted codes —
 * so the frame keeps that polarity in dark mode through `.nb-qr` tokens.
 */
export function QrCode({ value, label, className }: QrCodeProps) {
  const drawn = useMemo(() => {
    try {
      const m = encodeQr(value);
      return { size: m.size, path: qrSvgPath(m) };
    } catch {
      return null;
    }
  }, [value]);
  if (!drawn) return null;
  const box = drawn.size + QUIET * 2;
  return (
    <svg
      className={clsx('nb-qr', className)}
      viewBox={`${-QUIET} ${-QUIET} ${box} ${box}`}
      role="img"
      aria-label={label}
      shapeRendering="crispEdges"
    >
      <rect className="nb-qr-bg" x={-QUIET} y={-QUIET} width={box} height={box} rx={2} />
      <path className="nb-qr-ink" d={drawn.path} />
    </svg>
  );
}
