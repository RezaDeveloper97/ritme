'use client';

import { useEffect, useRef } from 'react';

interface WheelPickerProps {
  id: string;
  items: string[];
  selectedIndex: number;
  width?: number;
  onChange: (index: number) => void;
}

const ITEM_H = 44;

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

/** Vertical scroll-snap wheel picker (birthday, period length). */
export function WheelPicker({ id, items, selectedIndex, width = 80, onChange }: WheelPickerProps) {
  const ref = useRef<HTMLDivElement>(null);

  // The scroll listener is bound once, so everything it reads goes through refs
  // rather than the mount-render closure.
  const onChangeRef = useRef(onChange);
  const countRef = useRef(items.length);
  const selectedRef = useRef(selectedIndex);
  onChangeRef.current = onChange;
  countRef.current = items.length;

  const paint = (el: HTMLElement, index: number) => {
    el.querySelectorAll('.wi').forEach((it, k) => it.classList.toggle('on', k === index));
  };

  // Park the wheel on the selected item — on mount and whenever the value
  // changes from outside (e.g. the item list switches calendars with the
  // locale, so the same date lands on a different index).
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    selectedRef.current = selectedIndex;
    if (Math.round(el.scrollTop / ITEM_H) !== selectedIndex) {
      el.scrollTop = selectedIndex * ITEM_H;
    }
    paint(el, selectedIndex);
  }, [selectedIndex, items.length]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    let raf = 0;
    const update = () => {
      const idx = clamp(Math.round(el.scrollTop / ITEM_H), 0, countRef.current - 1);
      paint(el, idx);
      // Only a *user* scroll is an edit. A programmatic park (mount, or an
      // externally changed value) must not echo back: it would overwrite the
      // real value with whatever index the wheel happened to clamp to.
      if (idx !== selectedRef.current) {
        selectedRef.current = idx;
        onChangeRef.current(idx);
      }
    };

    const onScroll = () => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(update);
    };

    el.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      el.removeEventListener('scroll', onScroll);
      cancelAnimationFrame(raf);
    };
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
