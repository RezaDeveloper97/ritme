'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, type FormEvent, type ReactNode } from 'react';

import { Button } from '@/shared/ui';

/** A modal form (native <dialog>: focus trap + Esc), styled like the shared confirm dialog. */
export function FormDialog({
  open,
  title,
  onClose,
  onSubmit,
  submitLabel,
  saving,
  tone = 'primary',
  children,
}: {
  open: boolean;
  title: string;
  onClose: () => void;
  onSubmit: () => void;
  submitLabel: string;
  saving: boolean;
  tone?: 'primary' | 'danger';
  children: ReactNode;
}) {
  const t = useTranslations('confirm');
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!saving) onSubmit();
  };

  return (
    <dialog
      ref={ref}
      className="dialog"
      aria-label={title}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      {open ? (
        <form className="flex flex-col gap-4 p-5" onSubmit={submit}>
          <h2 className="m-0 text-base font-bold">{title}</h2>
          {children}
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" onClick={onClose}>
              {t('cancel')}
            </Button>
            <Button type="submit" variant={tone} loading={saving}>
              {submitLabel}
            </Button>
          </div>
        </form>
      ) : null}
    </dialog>
  );
}
