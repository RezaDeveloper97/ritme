'use client';

import { useEffect, useRef, useState } from 'react';

interface RulerPickerProps {
  min: number;
  max: number;
  value: number;
  unit: string;
  onChange: (value: number) => void;
  toDisplay?: (v: number) => string;
}

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const faNum = (n: string | number) =>
  String(n).replace(/[0-9]/g, (d) => FA[Number(d)]);

/** Horizontal scroll-snap ruler for weight / height input. */
export function RulerPicker({ min, max, value, unit, onChange, toDisplay }: RulerPickerProps) {
  const rulerRef = useRef<HTMLDivElement>(null);
  const TICK_W = 12;
  const [display, setDisplay] = useState(toDisplay ? toDisplay(value) : faNum(value.toFixed(1)));

  useEffect(() => {
    const el = rulerRef.current;
    if (!el) return;
    el.scrollLeft = (value - min) * TICK_W;

    let raf = 0;
    const update = () => {
      const idx = Math.round(el.scrollLeft / TICK_W);
      const v = Math.max(min, Math.min(max, min + idx));
      setDisplay(toDisplay ? toDisplay(v) : faNum(v.toFixed(1)));
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
        <div className="num">{major ? faNum(i) : ''}</div>
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
