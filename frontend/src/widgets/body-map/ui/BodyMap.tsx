'use client';

import { clsx } from 'clsx';

import { ChipGroup, PillChip } from '@/shared/ui';

import { isMappedRegion } from '../model/regions';

export interface BodyMapRegion {
  code: string;
  label: string;
  selected: boolean;
  /** The region an intensity control next to the map is editing (thicker ring). */
  current?: boolean;
}

interface BodyMapProps {
  regions: readonly BodyMapRegion[];
  onToggle: (code: string) => void;
  /** Accessible name of the map, e.g. «نقشه بدن — جای درد را انتخاب کن». */
  label: string;
  className?: string;
}

/**
 * Front-facing body outline with one 44px toggle button per pain location (nbl_/nbd_Log_Pain). The SVG
 * is decoration; every region is a real `<button aria-pressed>` named by its label, so the map works
 * with a keyboard and a screen reader. The figure is anatomical, not text, so it is laid out LTR in
 * both directions (its left stays the body's right) — the labels inside still read in their own script.
 * Colours are tokens only: the body in `--brand-soft`, picked pins in `--bloom`.
 */
export function BodyMap({ regions, onToggle, label, className }: BodyMapProps) {
  const mapped = regions.filter((r) => isMappedRegion(r.code));
  const extra = regions.filter((r) => !isMappedRegion(r.code));
  return (
    <div className={clsx('bmap', className)}>
      <div className="bmap-figure" role="group" aria-label={label} dir="ltr">
        <svg className="bmap-svg" viewBox="0 0 240 300" aria-hidden focusable="false">
          <circle className="bmap-body" cx="120" cy="42" r="32" />
          <path className="bmap-body" d="M78 88 Q120 80 162 88 L170 210 Q120 232 70 210 Z" />
          <path className="bmap-limb" d="M78 90 L46 190 M162 90 L194 190 M96 222 L90 300 M144 222 L150 300" />
        </svg>
        {mapped.map((r) => (
          <button
            key={r.code}
            type="button"
            className={clsx('bmap-pin', `is-${r.code}`, r.current && 'is-current')}
            aria-pressed={r.selected}
            onClick={() => onToggle(r.code)}
          >
            <span className="bmap-dot" aria-hidden />
            <span className="bmap-label" dir="auto">
              {r.label}
            </span>
          </button>
        ))}
      </div>
      {extra.length ? (
        <ChipGroup label={label} className="bmap-extra">
          {extra.map((r) => (
            <PillChip key={r.code} mode="multi" tone="bloom" pressed={r.selected} onPressedChange={() => onToggle(r.code)}>
              {r.label}
            </PillChip>
          ))}
        </ChipGroup>
      ) : null}
    </div>
  );
}
