'use client';

import { useEffect, useRef } from 'react';

interface WheelPickerProps {
  id: string;
  items: string[];
  selectedIndex: number;
  width?: number;
  onChange: (index: number) => void;
}

/** Vertical scroll-snap wheel picker (birthday, period length). */
export function WheelPicker({ id, items, selectedIndex, width = 80, onChange }: WheelPickerProps) {
  const ref = useRef<HTMLDivElement>(null);
  const ITEM_H = 44;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.scrollTop = selectedIndex * ITEM_H;

    let raf = 0;
    const update = () => {
      const idx = Math.max(0, Math.min(items.length - 1, Math.round(el.scrollTop / ITEM_H)));
      el.querySelectorAll('.wi').forEach((it, k) => it.classList.toggle('on', k === idx));
      onChange(idx);
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

  return (
    <div id={id} ref={ref} className="wheel" style={{ width }}>
      <div className="wpad" />
      {items.map((item, i) => (
        <div key={i} className={`wi${i === selectedIndex ? ' on' : ''}`}>
          {item}
        </div>
      ))}
      <div className="wpad" />
    </div>
  );
}
