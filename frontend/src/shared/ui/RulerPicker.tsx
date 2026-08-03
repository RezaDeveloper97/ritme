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

/** Horizontal scroll-snap ruler for weight / height input. */
export function RulerPicker({ min, max, value, unit, onChange, toDisplay }: RulerPickerProps) {
  const loc = useLocale() as Locale;
  const rulerRef = useRef<HTMLDivElement>(null);
  const TICK_W = 12;
  // The scroll handler is bound once, so it reads the formatter from a ref
  // rather than the mount-render closure (digits differ per locale).
  const fmtRef = useRef((v: number) => (toDisplay ? toDisplay(v) : formatNumber(v.toFixed(1), loc)));
  fmtRef.current = (v: number) => (toDisplay ? toDisplay(v) : formatNumber(v.toFixed(1), loc));
  const [display, setDisplay] = useState(() => fmtRef.current(value));

  useEffect(() => {
    const el = rulerRef.current;
    if (!el) return;
    el.scrollLeft = (value - min) * TICK_W;

    let raf = 0;
    const update = () => {
      const idx = Math.round(el.scrollLeft / TICK_W);
      const v = Math.max(min, Math.min(max, min + idx));
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
  for (let i = min; i <= max; i++) {
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
      <div className="ruler-wrap">
        <div className="rpoint" />
        <div ref={rulerRef} className="ruler" dir="ltr">
          <div className="rpad" />
          {ticks}
          <div className="rpad" />
        </div>
        <div className="ruler-fade" />
      </div>
    </>
  );
}
