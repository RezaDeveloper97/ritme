'use client';

import clsx from 'clsx';
import { useEffect } from 'react';

import { useMarkDoneToast } from '../model/toast';

const DURATION_MS = 3000;

/** The «ثبت شد» toast raised by the MarkDone sheet. Mount once per route. */
export function MarkDoneToast() {
  const { message, tone, clear } = useMarkDoneToast();

  useEffect(() => {
    if (!message) return;
    const timer = window.setTimeout(clear, DURATION_MS);
    return () => window.clearTimeout(timer);
  }, [message, clear]);

  useEffect(() => clear, [clear]);

  if (!message) return null;
  return (
    <div
      role="status"
      className={clsx(
        'fixed inset-x-4 bottom-[calc(24px+env(safe-area-inset-bottom,0px))] z-50 rounded-2xl px-4 py-3 text-start text-[13px] font-bold shadow-lg',
        tone === 'ok' ? 'bg-(--ink) text-(--surface)' : 'bg-(--danger) text-(--on-accent)',
      )}
      onClick={clear}
    >
      {message}
    </div>
  );
}
