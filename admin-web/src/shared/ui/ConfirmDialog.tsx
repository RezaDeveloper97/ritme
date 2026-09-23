'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef } from 'react';

import { Button } from './Button';
import { useConfirmStore } from './confirm';

/** Mounted once in app/providers; opened with `confirm({...})`. Native <dialog> = focus trap + Esc. */
export function ConfirmDialog() {
  const current = useConfirmStore((s) => s.current);
  const close = useConfirmStore((s) => s.close);
  const ref = useRef<HTMLDialogElement>(null);
  const t = useTranslations('confirm');

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (current && !dialog.open) dialog.showModal();
    if (!current && dialog.open) dialog.close();
  }, [current]);

  return (
    <dialog
      ref={ref}
      className="dialog"
      aria-labelledby="confirm-title"
      onCancel={(e) => {
        e.preventDefault();
        close(false);
      }}
    >
      {current ? (
        <div className="flex flex-col gap-3 p-5">
          <h2 id="confirm-title" className="m-0 text-base font-bold">
            {current.title ?? t('title')}
          </h2>
          <p className="m-0 text-ink-3">{current.message}</p>
          <div className="mt-2 flex justify-end gap-2">
            <Button onClick={() => close(false)} autoFocus>
              {t('cancel')}
            </Button>
            <Button variant={current.tone === 'danger' ? 'danger' : 'primary'} onClick={() => close(true)}>
              {current.confirmLabel ?? t('confirm')}
            </Button>
          </div>
        </div>
      ) : null}
    </dialog>
  );
}
