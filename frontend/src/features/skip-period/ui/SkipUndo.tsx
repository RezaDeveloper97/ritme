'use client';

import { useTranslations } from 'next-intl';
import { useEffect } from 'react';

import { toApiDate, today } from '@/shared/lib/date';

import { useUndoSkip } from '../api/mutations';
import { useSkipStore } from '../model/store';
import '@/shared/ui/styles/ch.css';

const TOAST_MS = 8000;

/**
 * After «پریود نداشتم»: a toast with «برگرداندن» right away, and a quiet note with the same link for the rest
 * of the day. Mount once on Today.
 */
export function SkipUndo() {
  const t = useTranslations('home.skip');
  const { recent, toast, hydrate, hideToast, clear } = useSkipStore();
  const undo = useUndoSkip();

  useEffect(hydrate, [hydrate]);
  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(hideToast, TOAST_MS);
    return () => window.clearTimeout(timer);
  }, [toast, hideToast]);

  const live = recent && recent.day === toApiDate(today()) ? recent : null;
  if (!live) return null;

  const onUndo = () => undo.mutate(live.expected, { onSuccess: clear });

  return (
    <>
      <div role="status" className="ch-alert" data-skip-note>
        <span className="ch-alert-body">
          <span className="ch-alert-text">{t('note.text')}</span>
          <button type="button" className="ch-link" onClick={onUndo} disabled={undo.isPending}>
            {t('undo')}
          </button>
          {undo.isError && <span className="ch-alert-text text-(--danger)">{t('errors.undo')}</span>}
        </span>
      </div>
      {toast && (
        <div
          role="status"
          className="fixed inset-x-4 bottom-[calc(116px+env(safe-area-inset-bottom,0px))] z-50 flex items-center justify-between gap-3 rounded-2xl bg-(--ink) px-4 py-3 text-start text-[13px] font-bold text-(--surface) shadow-lg"
        >
          <span>{t('toast.text')}</span>
          <button type="button" className="shrink-0 font-extrabold underline" onClick={onUndo} disabled={undo.isPending}>
            {t('undo')}
          </button>
        </div>
      )}
    </>
  );
}
