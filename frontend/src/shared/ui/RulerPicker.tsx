'use client';

import { useLocale } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

interface RulerPickerProps {
  min: number;
  max: number;
  value: number;
  unit: string;
  onChange: (value: number) => void;
  toDisplay?: (v: number) => string;
}

/** Vertical scroll-snap ruler for weight / height input (largest value on top). */
export function RulerPicker({ min, max, value, unit, onChange, toDisplay }: RulerPickerProps) {
  const loc = useLocale() as Locale;
  const rulerRef = useRef<HTMLDivElement>(null);
  const TICK_H = 12;
  // The scroll handler is bound once, so it reads the formatter from a ref
  // rather than the mount-render closure (digits differ per locale).
  const fmtRef = useRef((v: number) => (toDisplay ? toDisplay(v) : formatNumber(v.toFixed(1), loc)));
  fmtRef.current = (v: number) => (toDisplay ? toDisplay(v) : formatNumber(v.toFixed(1), loc));
  const [display, setDisplay] = useState(() => fmtRef.current(value));

  useEffect(() => {
    const el = rulerRef.current;
    if (!el) return;
    // Ticks are rendered max → min, so distance from the top is (max - value).
    el.scrollTop = (max - value) * TICK_H;

    let raf = 0;
    const update = () => {
      const idx = Math.round(el.scrollTop / TICK_H);
      const v = Math.max(min, Math.min(max, max - idx));
      setDisplay(fmtRef.current(v));
      onChange(v);
    };
    const onScroll = () => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(update);
    };
    el.addEventListener('scroll', onScroll, { passive: true });
    update();
    return () => {
      el.removeEventListener('scroll', onScroll);
      cancelAnimationFrame(raf);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const ticks = [];
  for (let i = max; i >= min; i--) {
    const major = i % 5 === 0;
    ticks.push(
      <div key={i} className={`tk${major ? ' major' : ''}`}>
        <div className="line" />
        <div className="num">{major ? formatNumber(i, loc) : ''}</div>
      </div>,
    );
  }

  return (
    <>
      <div className="rp-readout">
        <span className="rp-value">
          {display}
        </span>
        <span className="rp-unit">{unit}</span>
      </div>
      {/* dir is pinned so the tick column and the pointer stay on the same side in RTL. */}
      <div className="ruler-wrap" dir="ltr">
        <div className="rpoint" />
        <div ref={rulerRef} className="ruler">
          <div className="rpad" />
          {ticks}
          <div className="rpad" />
        </div>
        <div className="ruler-fade" />
      </div>
    </>
  );
}
