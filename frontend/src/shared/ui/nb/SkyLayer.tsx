import { clsx } from 'clsx';

/**
 * The Night & Bloom background layer: violet glow (both themes) + the static
 * dark starfield (`.nb-sky`, B-N1-02). First child of a positioned screen;
 * content above it needs `position: relative`.
 */
export function SkyLayer({ className }: { className?: string }) {
  return <div aria-hidden className={clsx('nb-sky', className)} />;
}
